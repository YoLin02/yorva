package hermes

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	nodeArchiveMaxEntries      = 8000
	nodeArchiveMaxUncompressed = 256 << 20
	nodeArchiveMaxMember       = 32 << 20
	npmArchiveMaxEntries       = 8000
	npmArchiveMaxUncompressed  = 64 << 20
	nodeProbeTimeout           = 5 * time.Second
	nodeDepsOutputLimit        = 1 << 20
)

type PrerequisiteState string

const (
	PrereqReady         PrerequisiteState = "READY"
	PrereqMissing       PrerequisiteState = "MISSING"
	PrereqUnsupported   PrerequisiteState = "UNSUPPORTED"
	PrereqBroken        PrerequisiteState = "BROKEN"
	PrereqNotInstalled  PrerequisiteState = "NOT_INSTALLED"
	PrereqFailed        PrerequisiteState = "FAILED"
	PrereqTimedOut      PrerequisiteState = "TIMED_OUT"
)

type ComponentStatus struct {
	State     PrerequisiteState
	Version   string
	ErrorCode yorvaruntime.ErrorCode
	Retryable bool
}

type Prerequisites struct {
	Node             ComponentStatus
	NPM              ComponentStatus
	NodeDependencies ComponentStatus
	CheckedAt        time.Time
	ActiveOperation  string
}

type NodeHost struct {
	stateRoot    string
	nodeArchive  string
	npmArchive   string
	home         func() string
	installDir   func() string
	nodeDir      func() string
	run          func(context.Context, installInvocation, time.Duration) commandResult
	operationID  string
	diskFree     func(string) (uint64, error)
}

func NewNodeHost(stateRoot, nodeArchive, npmArchive string) *NodeHost {
	return &NodeHost{
		stateRoot:   stateRoot,
		nodeArchive: strings.TrimSpace(nodeArchive),
		npmArchive:  strings.TrimSpace(npmArchive),
		home:        officialHermesHome,
		installDir:  officialInstallDir,
		nodeDir:     officialNodeDir,
		run:         defaultInstallRun,
		diskFree:    volumeFreeBytes,
	}
}

func officialNodeDir() string {
	return filepath.Join(officialHermesHome(), "node")
}

func managedNpmCLI(nodeDir string) string {
	return filepath.Join(nodeDir, "node_modules", "npm", "bin", "npm-cli.js")
}

func (h *NodeHost) Inspect() Prerequisites {
	now := time.Now().UTC()
	node := h.inspectNode()
	npm := ComponentStatus{State: PrereqMissing, ErrorCode: yorvaruntime.ErrorHermesNPMMissing, Retryable: true}
	deps := ComponentStatus{State: PrereqNotInstalled, Retryable: true}
	if node.State == PrereqReady {
		npm = h.inspectNPM()
		if npm.State == PrereqReady {
			deps = h.inspectDeps()
		}
	}
	return Prerequisites{Node: node, NPM: npm, NodeDependencies: deps, CheckedAt: now}
}

func (h *NodeHost) inspectNode() ComponentStatus {
	exe := filepath.Join(h.nodeDir(), "node.exe")
	if !isRegularFile(exe) {
		return ComponentStatus{State: PrereqMissing, ErrorCode: yorvaruntime.ErrorHermesNodeMissing, Retryable: true}
	}
	version, err := h.probeVersion(exe, nil)
	if err != nil {
		return ComponentStatus{State: PrereqBroken, ErrorCode: yorvaruntime.ErrorHermesNodeMissing, Retryable: true}
	}
	if !nodeVersionSupported(version) {
		return ComponentStatus{State: PrereqUnsupported, Version: version, ErrorCode: yorvaruntime.ErrorHermesNodeUnsupported, Retryable: true}
	}
	return ComponentStatus{State: PrereqReady, Version: version}
}

func (h *NodeHost) inspectNPM() ComponentStatus {
	cli := managedNpmCLI(h.nodeDir())
	if !isRegularFile(cli) {
		return ComponentStatus{State: PrereqMissing, ErrorCode: yorvaruntime.ErrorHermesNPMMissing, Retryable: true}
	}
	version, err := h.probeVersion(filepath.Join(h.nodeDir(), "node.exe"), []string{cli, "--version"})
	if err != nil {
		return ComponentStatus{State: PrereqBroken, ErrorCode: yorvaruntime.ErrorHermesNPMMissing, Retryable: true}
	}
	if !npmVersionSupported(version) {
		return ComponentStatus{State: PrereqUnsupported, Version: version, ErrorCode: yorvaruntime.ErrorHermesNPMUnsupported, Retryable: true}
	}
	return ComponentStatus{State: PrereqReady, Version: version}
}

func (h *NodeHost) inspectDeps() ComponentStatus {
	lock := filepath.Join(h.installDir(), "package-lock.json")
	modules := filepath.Join(h.installDir(), "node_modules")
	if !isRegularFile(lock) {
		return ComponentStatus{State: PrereqNotInstalled, Retryable: true}
	}
	if !isDirectory(modules) {
		return ComponentStatus{State: PrereqNotInstalled, Retryable: true}
	}
	return ComponentStatus{State: PrereqReady}
}

func (h *NodeHost) probeVersion(executable string, args []string) (string, error) {
	if len(args) == 0 {
		args = []string{"--version"}
	}
	result := h.run(context.Background(), installInvocation{Executable: executable, Args: args, Dir: h.nodeDir()}, nodeProbeTimeout)
	if result.err != nil || result.timedOut || result.limited || result.exitCode != 0 {
		return "", errors.New("version probe failed")
	}
	return strings.TrimSpace(strings.TrimPrefix(result.stdout, "v")), nil
}

func nodeVersionSupported(version string) bool {
	return compareLooseVersion(version, officialNodeMinVersion) >= 0
}

func npmVersionSupported(version string) bool {
	return compareLooseVersion(version, officialNpmMinVersion) >= 0
}

func compareLooseVersion(got, min string) int {
	g := parseLoose(got)
	m := parseLoose(min)
	for i := 0; i < 3; i++ {
		if g[i] < m[i] {
			return -1
		}
		if g[i] > m[i] {
			return 1
		}
	}
	return 0
}

func parseLoose(value string) [3]int {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.Split(value, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		fmt.Sscanf(parts[i], "%d", &out[i])
	}
	return out
}

func (h *NodeHost) Apply(ctx context.Context, operationID string, report func(operation.Stage, string)) error {
	h.operationID = operationID
	if report != nil {
		report(operation.StageInstallNode, "")
	}
	if err := h.ensureNode(ctx); err != nil {
		return err
	}
	if report != nil {
		report("install.npm", "")
	}
	if err := h.ensureNPM(ctx); err != nil {
		return err
	}
	if report != nil {
		report(operation.StageInstallNodeDeps, "")
	}
	return h.installDependencies(ctx)
}

func (h *NodeHost) ensureNode(ctx context.Context) error {
	if status := h.inspectNode(); status.State == PrereqReady {
		return nil
	}
	if h.nodeArchive == "" {
		return installError(yorvaruntime.ErrorHermesNodeMissing, errors.New("bundled Node archive is not available"))
	}
	if err := verifySizedDigest(h.nodeArchive, officialNodeArchiveSize, officialNodeArchiveSHA); err != nil {
		return installError(yorvaruntime.ErrorHermesNodeArchiveIntegrityFailed, err)
	}
	staging, err := operationPrivateDir(h.stateRoot, h.operationID+"-node")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := requireExtractBudget(staging, h.diskFree); err != nil {
		return err
	}
	extracted := filepath.Join(staging, "tree")
	if err := extractPrefixedZip(ctx, h.nodeArchive, extracted, officialNodeZipRoot); err != nil {
		return installError(yorvaruntime.ErrorHermesNodeArchiveIntegrityFailed, err)
	}
	if !isRegularFile(filepath.Join(extracted, "node.exe")) {
		return installError(yorvaruntime.ErrorHermesNodeArchiveIntegrityFailed, errors.New("node.exe missing"))
	}
	target := h.nodeDir()
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	return placeMaterializedTree(extracted, target)
}

func (h *NodeHost) ensureNPM(ctx context.Context) error {
	if status := h.inspectNPM(); status.State == PrereqReady {
		return nil
	}
	if h.npmArchive == "" {
		return installError(yorvaruntime.ErrorHermesNPMMissing, errors.New("bundled npm archive is not available"))
	}
	if err := verifySizedDigest(h.npmArchive, officialNpmArchiveSize, officialNpmArchiveSHA); err != nil {
		return installError(yorvaruntime.ErrorHermesNPMArchiveIntegrityFailed, err)
	}
	staging, err := operationPrivateDir(h.stateRoot, h.operationID+"-npm")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	extracted := filepath.Join(staging, "npm")
	if err := extractNpmTarball(ctx, h.npmArchive, extracted); err != nil {
		return installError(yorvaruntime.ErrorHermesNPMArchiveIntegrityFailed, err)
	}
	target := filepath.Join(h.nodeDir(), "node_modules", "npm")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	_ = os.RemoveAll(target)
	if err := placeMaterializedTree(extracted, target); err != nil {
		return err
	}
	if status := h.inspectNPM(); status.State != PrereqReady {
		return installError(yorvaruntime.ErrorHermesNPMUnsupported, errors.New("managed npm postcondition failed"))
	}
	return nil
}

func (h *NodeHost) installDependencies(ctx context.Context) error {
	installDir := h.installDir()
	if !isRegularFile(filepath.Join(installDir, "package-lock.json")) {
		return installError(yorvaruntime.ErrorHermesNodeDepsFailed, errors.New("official package-lock.json is missing"))
	}
	node := filepath.Join(h.nodeDir(), "node.exe")
	cli := managedNpmCLI(h.nodeDir())
	if strings.EqualFold(filepath.Ext(cli), ".ps1") {
		return installError(yorvaruntime.ErrorHermesNPMUnsupported, errors.New("refusing npm.ps1"))
	}
	args := []string{cli, "ci", "--workspaces=false", "--omit=dev", "--ignore-scripts", "--no-audit", "--no-fund", "--progress=false"}
	result := h.run(ctx, installInvocation{Executable: node, Args: args, Dir: installDir}, nodeDepsTimeout)
	if result.timedOut {
		return installError(yorvaruntime.ErrorHermesNodeDepsTimeout, errors.New("npm ci timed out"))
	}
	if result.limited {
		return installError(yorvaruntime.ErrorRuntimeInstallOutputLimit, errOutputLimit)
	}
	if result.err != nil || result.exitCode != 0 {
		if errors.Is(result.err, context.Canceled) {
			return result.err
		}
		return installError(yorvaruntime.ErrorHermesNodeDepsFailed, errors.New("npm ci failed"))
	}
	return nil
}

func verifySizedDigest(path string, size int64, digest string) error {
	if err := rejectReparsePoint(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return errors.New("archive is not the expected regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(sum.Sum(nil)), digest) {
		return errors.New("archive digest mismatch")
	}
	return nil
}

func extractPrefixedZip(ctx context.Context, archivePath, dest, prefix string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) > nodeArchiveMaxEntries {
		return errors.New("zip entry count exceeded")
	}
	var uncompressed int64
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := filepath.ToSlash(file.Name)
		if strings.Contains(name, "..") || strings.Contains(name, ":") || strings.HasPrefix(name, "/") {
			return errors.New("unsafe zip member")
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return errors.New("zip symlink rejected")
		}
		uncompressed += int64(file.UncompressedSize64)
		if uncompressed > nodeArchiveMaxUncompressed || int64(file.UncompressedSize64) > nodeArchiveMaxMember {
			return errors.New("zip expansion limit exceeded")
		}
		rel := strings.TrimPrefix(name, prefix+"/")
		if rel == name || rel == "" {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if !pathWithin(dest, target) {
			return errors.New("zip member escaped destination")
		}
		if file.FileInfo().IsDir() || strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		if err := writeZipFile(file, target); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(file *zip.File, target string) error {
	source, err := file.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, io.LimitReader(source, nodeArchiveMaxMember+1))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func extractNpmTarball(ctx context.Context, archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	var entries int
	var uncompressed int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		entries++
		if entries > npmArchiveMaxEntries {
			return errors.New("tar entry count exceeded")
		}
		name := filepath.ToSlash(header.Name)
		if strings.Contains(name, "..") || strings.Contains(name, ":") || strings.HasPrefix(name, "/") {
			return errors.New("unsafe tar member")
		}
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return errors.New("tar symlink rejected")
		}
		uncompressed += header.Size
		if uncompressed > npmArchiveMaxUncompressed || header.Size > nodeArchiveMaxMember {
			return errors.New("tar expansion limit exceeded")
		}
		rel := strings.TrimPrefix(name, "package/")
		if rel == name || rel == "" {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if !pathWithin(dest, target) {
			return errors.New("tar member escaped destination")
		}
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, io.LimitReader(reader, nodeArchiveMaxMember+1))
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
