package hermes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
)

type NodeHost struct {
	stateRoot   string
	nodeArchive string
	npmArchive  string
	home        func() string
	installDir  func() string
	nodeDir     func() string
	run         func(context.Context, installInvocation, time.Duration) commandResult
	operationID string
	diskFree    func(string) (uint64, error)
	downloadSources downloadsources.Provider
}

func (h *NodeHost) WithDownloadSources(provider downloadsources.Provider) *NodeHost {
	h.downloadSources = provider
	return h
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

func (h *NodeHost) Apply(ctx context.Context, operationID string, report func(operation.Stage, string)) error {
	h.operationID = operationID
	sources := downloadsources.Default()
	var err error
	if h.downloadSources != nil {
		sources, err = h.downloadSources.Get(ctx)
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
		}
	}
	if report != nil {
		report(operation.StageInstallNode, "")
	}
	if err := h.ensureNode(ctx, sources); err != nil {
		return err
	}
	if report != nil {
		report("install.npm", "")
	}
	if err := h.ensureNPM(ctx, sources); err != nil {
		return err
	}
	if !isRegularFile(filepath.Join(h.installDir(), "package-lock.json")) {
		return nil
	}
	if report != nil {
		report(operation.StageInstallNodeDeps, "")
	}
	return h.installDependencies(ctx, sources)
}

func (h *NodeHost) ensureNode(ctx context.Context, sources downloadsources.Config) error {
	if status := h.inspectNode(); status.State == PrereqReady {
		return nil
	}
	staging, err := operationPrivateDir(h.stateRoot, h.operationID+"-node")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := requireExtractBudget(staging, h.diskFree); err != nil {
		return err
	}
	archivePath := h.nodeArchive
	if archivePath == "" {
		archivePath = filepath.Join(staging, "node.zip")
		if err := downloadPinnedArtifact(ctx, sources.NodeArchiveURL, archivePath, archiveDownloadLimit, officialNodeArchiveSize, officialNodeArchiveSHA); err != nil {
			return err
		}
	} else if err := verifySizedDigest(archivePath, officialNodeArchiveSize, officialNodeArchiveSHA); err != nil {
		return installError(yorvaruntime.ErrorHermesNodeArchiveIntegrityFailed, err)
	}
	extracted := filepath.Join(staging, "tree")
	if err := extractPrefixedZip(ctx, archivePath, extracted, officialNodeZipRoot); err != nil {
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

func (h *NodeHost) ensureNPM(ctx context.Context, sources downloadsources.Config) error {
	if status := h.inspectNPM(); status.State == PrereqReady {
		return nil
	}
	staging, err := operationPrivateDir(h.stateRoot, h.operationID+"-npm")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	archivePath := h.npmArchive
	if archivePath == "" {
		archivePath = filepath.Join(staging, "npm.tgz")
		if err := downloadPinnedArtifact(ctx, sources.NPMArchiveURL, archivePath, archiveDownloadLimit, officialNpmArchiveSize, officialNpmArchiveSHA); err != nil {
			return err
		}
	} else if err := verifySizedDigest(archivePath, officialNpmArchiveSize, officialNpmArchiveSHA); err != nil {
		return installError(yorvaruntime.ErrorHermesNPMArchiveIntegrityFailed, err)
	}
	extracted := filepath.Join(staging, "npm")
	if err := extractNpmTarball(ctx, archivePath, extracted); err != nil {
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
