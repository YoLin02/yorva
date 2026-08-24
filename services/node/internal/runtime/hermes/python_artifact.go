package hermes

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
)

func (h *HostInstaller) WithEmbeddedPython(path string) *HostInstaller {
	h.embeddedPythonPath = strings.TrimSpace(path)
	return h
}

func (h *HostInstaller) preparePythonMirror(ctx context.Context, workDir string, sources downloadsources.Config) (string, string, error) {
	if h.acquirePython != nil {
		mirror, err := h.acquirePython(ctx, workDir)
		return mirror, "", err
	}
	mirrorRoot := filepath.Join(workDir, "python-mirror")
	destinationDir := filepath.Join(mirrorRoot, officialPythonRelease)
	destination := filepath.Join(destinationDir, officialPythonArchiveName)
	if err := os.MkdirAll(destinationDir, 0o700); err != nil {
		return "", "", installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}

	useBundled := func() error {
		if h.embeddedPythonPath == "" {
			return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, os.ErrNotExist)
		}
		if err := verifySizedDigest(h.embeddedPythonPath, officialPythonArchiveSize, officialPythonArchiveSHA); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
		}
		return copyRegularFile(h.embeddedPythonPath, destination)
	}
	useOnline := func() error {
		return downloadPinnedArtifact(ctx, sources.PythonArchiveURL, destination, archiveDownloadLimit, officialPythonArchiveSize, officialPythonArchiveSHA)
	}

	var err error
	if sources.ArtifactPreference == downloadsources.PreferenceOnlineFirst {
		err = useOnline()
		if err != nil && isTransportArchiveError(err) && h.embeddedPythonPath != "" {
			h.debug("python.archive.online_unavailable", archiveLogFields(err, sourceOriginOfficial)...)
			err = useBundled()
		}
	} else if h.embeddedPythonPath != "" {
		err = useBundled()
	} else {
		err = useOnline()
	}
	if err != nil {
		return "", "", err
	}
	if err := verifySizedDigest(destination, officialPythonArchiveSize, officialPythonArchiveSHA); err != nil {
		return "", "", installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	metadata := map[string]any{
		"cpython-3.11.15-windows-x86_64-none": map[string]any{
			"name": "cpython", "arch": map[string]any{"family": "x86_64", "variant": nil},
			"os": "windows", "libc": "none", "major": 3, "minor": 11, "patch": 15,
			"prerelease": "", "url": sources.PythonArchiveURL, "sha256": officialPythonArchiveSHA,
			"variant": nil, "build": officialPythonRelease,
		},
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", "", installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	metadataPath := filepath.Join(mirrorRoot, "downloads.json")
	if err := os.WriteFile(metadataPath, metadataBytes, 0o600); err != nil {
		return "", "", installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	absolute, err := filepath.Abs(mirrorRoot)
	if err != nil {
		return "", "", installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	mirrorURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String()
	metadataAbsolute, err := filepath.Abs(metadataPath)
	if err != nil {
		return "", "", installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	metadataURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(metadataAbsolute)}).String()
	h.debug("python.archive.ready", "version", officialPythonVersion, "mirror", "local-verified")
	return mirrorURL, metadataURL, nil
}
