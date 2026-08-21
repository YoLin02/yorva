package hermes

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	sourceOriginOfficial = "official-archive"
	sourceOriginBundled  = "bundled-archive"
	warningBundledUsed   = "HERMES_SOURCE_BUNDLED_USED"
	warningSourcePrepared = "HERMES_SOURCE_PREPARED"
)

type archiveClient struct {
	http      *http.Client
	url       string
	limit     int64
	diskFree  func(string) (uint64, error)
}

func newArchiveClient(sourceURL ...string) archiveClient {
	url := officialArchiveURL
	if len(sourceURL) > 0 && strings.TrimSpace(sourceURL[0]) != "" {
		url = strings.TrimSpace(sourceURL[0])
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   archiveConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   archiveConnectTimeout,
		ResponseHeaderTimeout: archiveConnectTimeout,
		ForceAttemptHTTP2:     true,
	}
	return archiveClient{
		http: &http.Client{
			Transport: transport,
			Timeout:   archiveFetchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 3 || !approvedArchiveRedirect(req) {
					return errRedirectRejected
				}
				return nil
			},
		},
		url:      url,
		limit:    archiveDownloadLimit,
		diskFree: volumeFreeBytes,
	}
}

func approvedArchiveRedirect(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	return req.URL.Scheme == "https" && req.URL.Hostname() != "" && req.URL.User == nil
}

func verifyArchiveFile(path string) error {
	if err := rejectReparsePoint(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	if !info.Mode().IsRegular() || info.Size() != officialArchiveSize {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive is not the expected regular file"))
	}
	file, err := os.Open(path)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	if !strings.EqualFold(hex.EncodeToString(sum.Sum(nil)), officialArchiveSHA256) {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive digest mismatch"))
	}
	return nil
}

func (c archiveClient) download(ctx context.Context, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return classifyArchiveTransport(ctx, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, fmt.Errorf("status %d", response.StatusCode))
	}
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, fmt.Errorf("status %d", response.StatusCode))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, fmt.Errorf("status %d", response.StatusCode))
	}
	part := destPath + ".part"
	file, err := os.OpenFile(part, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	sum := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, sum), io.LimitReader(response.Body, c.limit+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(part)
		return classifyArchiveTransport(ctx, copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(part)
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, closeErr)
	}
	if written > c.limit {
		_ = os.Remove(part)
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive exceeded download bound"))
	}
	if written != officialArchiveSize || !strings.EqualFold(hex.EncodeToString(sum.Sum(nil)), officialArchiveSHA256) {
		_ = os.Remove(part)
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive size or digest mismatch"))
	}
	if err := os.Rename(part, destPath); err != nil {
		_ = os.Remove(part)
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	return verifyArchiveFile(destPath)
}

func classifyArchiveTransport(ctx context.Context, err error) error {
	if errors.Is(err, errRedirectRejected) {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errRedirectRejected)
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return installError(yorvaruntime.ErrorRuntimeInstallTimeout, err)
	}
	return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
}

func isTransportArchiveError(err error) bool {
	code := installErrorCode(err)
	return code == yorvaruntime.ErrorRuntimeInstallSourceUnavailable || code == yorvaruntime.ErrorRuntimeInstallTimeout
}

func requireExtractBudget(path string, free func(string) (uint64, error)) error {
	if free == nil {
		free = volumeFreeBytes
	}
	available, err := free(path)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	need := uint64(archiveDiskBudget + archiveDiskMargin)
	if available < need {
		return installError(yorvaruntime.ErrorRuntimeInstallInsufficientDisk, errors.New("destination volume lacks free space"))
	}
	return nil
}

func extractOfficialArchive(ctx context.Context, archivePath, destDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	defer reader.Close()
	if len(reader.File) > archiveMaxEntries {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive entry count exceeded"))
	}
	var uncompressed int64
	var files int
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := rejectZipMember(file); err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			continue
		}
		files++
		if files > archiveMaxFiles {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive file count exceeded"))
		}
		uncompressed += int64(file.UncompressedSize64)
		if uncompressed > archiveMaxUncompressed || int64(file.UncompressedSize64) > archiveMaxMember {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive expansion limit exceeded"))
		}
	}
	info, err := os.Lstat(archivePath)
	if err == nil && info.Size() > 0 && uncompressed/info.Size() > archiveMaxRatio {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive expansion ratio exceeded"))
	}
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := rejectReparsePoint(destDir); err != nil {
		return err
	}
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := extractZipMember(destDir, file); err != nil {
			return err
		}
	}
	return verifyExtractedLayout(destDir)
}

func rejectZipMember(file *zip.File) error {
	name := filepath.ToSlash(file.Name)
	if name == "" || !utf8.ValidString(name) {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive member name is invalid"))
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") || filepath.IsAbs(file.Name) {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive member has an absolute path"))
	}
	if strings.Contains(name, ":") {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive member has an alternate stream or drive path"))
	}
	parts := strings.Split(name, "/")
	if len(parts) == 0 || parts[0] != officialArchiveRoot {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive member escaped the official prefix"))
	}
	for _, part := range parts {
		if part == ".." {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive member contains a path escape"))
		}
	}
	mode := file.Mode()
	if mode&os.ModeSymlink != 0 || mode&os.ModeSocket != 0 || mode&os.ModeNamedPipe != 0 || mode&os.ModeDevice != 0 {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive member is not a regular file"))
	}
	return nil
}

func extractZipMember(destDir string, file *zip.File) error {
	relative, err := strippedArchivePath(file.Name)
	if err != nil {
		return err
	}
	target := filepath.Join(destDir, filepath.FromSlash(relative))
	if !pathWithin(destDir, target) {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive member escaped destination"))
	}
	if file.FileInfo().IsDir() || strings.HasSuffix(filepath.ToSlash(file.Name), "/") {
		if err := os.MkdirAll(target, 0o700); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		return rejectReparsePoint(target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	source, err := file.Open()
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	defer source.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	_, copyErr := io.Copy(out, io.LimitReader(source, archiveMaxMember+1))
	closeErr := out.Close()
	if copyErr != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, copyErr)
	}
	if closeErr != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, closeErr)
	}
	return rejectReparsePoint(target)
}

func strippedArchivePath(name string) (string, error) {
	slash := filepath.ToSlash(name)
	prefix := officialArchiveRoot + "/"
	if slash == officialArchiveRoot || slash == officialArchiveRoot+"/" {
		return ".", nil
	}
	if !strings.HasPrefix(slash, prefix) {
		return "", installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive member missing official prefix"))
	}
	return strings.TrimPrefix(slash, prefix), nil
}

func verifyExtractedLayout(root string) error {
	required := []string{
		"LICENSE",
		"pyproject.toml",
		filepath.Join("scripts", "install.ps1"),
		filepath.Join("hermes_cli", "main.py"),
	}
	for _, name := range required {
		path := filepath.Join(root, name)
		if !isRegularFile(path) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("materialized tree missing %s", name))
		}
		if err := rejectReparsePoint(path); err != nil {
			return err
		}
	}
	return nil
}

func verifyOfficialExtractedIdentity(root string) error {
	if err := verifyExtractedLayout(root); err != nil {
		return err
	}
	return verifyRegularFile(filepath.Join(root, "LICENSE"), officialLicenseSize, officialLicenseSHA256)
}

func officialScriptFromArchive(archivePath, destPath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	defer reader.Close()
	want := officialArchiveRoot + "/scripts/install.ps1"
	for _, file := range reader.File {
		if filepath.ToSlash(file.Name) != want {
			continue
		}
		source, err := file.Open()
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
		}
		payload, err := io.ReadAll(io.LimitReader(source, officialScriptSize+8192))
		source.Close()
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
		}
		normalized := normalizeNewlines(payload)
		if int64(len(normalized)) != officialScriptSize {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("normalized archive script size mismatch"))
		}
		sum := sha256.Sum256(normalized)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), officialScriptSHA256) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("normalized archive script digest mismatch"))
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
		}
		if err := os.WriteFile(destPath, normalized, 0o600); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
		}
		return verifyRegularFile(destPath, officialScriptSize, officialScriptSHA256)
	}
	return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("archive is missing scripts/install.ps1"))
}

func normalizeNewlines(payload []byte) []byte {
	payload = bytesReplaceAll(payload, []byte("\r\n"), []byte("\n"))
	return bytesReplaceAll(payload, []byte("\r"), []byte("\n"))
}

func bytesReplaceAll(payload, old, new []byte) []byte {
	return []byte(strings.ReplaceAll(string(payload), string(old), string(new)))
}

func placeMaterializedTree(staging, installDir string) error {
	if err := rejectReparsePoint(staging); err != nil {
		return err
	}
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := rejectReparsePoint(installDir); err != nil {
		return err
	}
	return filepath.WalkDir(staging, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		relative, relErr := filepath.Rel(staging, path)
		if relErr != nil || !pathWithin(staging, path) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("materialized tree escaped staging"))
		}
		target := filepath.Join(installDir, relative)
		if !pathWithin(installDir, target) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("materialized tree escaped install directory"))
		}
		if entry.IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
			}
			return rejectReparsePoint(target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		return copyRegularFile(path, target)
	})
}

func copyRegularFile(from, to string) error {
	if err := rejectReparsePoint(from); err != nil {
		return err
	}
	info, err := os.Lstat(from)
	if err != nil || !info.Mode().IsRegular() {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("refusing to copy a non-regular file"))
	}
	in, err := os.Open(from)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, copyErr)
	}
	if closeErr != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, closeErr)
	}
	return rejectReparsePoint(to)
}
