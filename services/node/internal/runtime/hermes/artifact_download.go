package hermes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func downloadPinnedArtifact(ctx context.Context, sourceURL, destination string, limit, expectedSize int64, expectedSHA string) error {
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
	client := &http.Client{
		Transport: transport,
		Timeout:   archiveFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 3 || !approvedArchiveRedirect(req) {
				return errRedirectRejected
			}
			return nil
		},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	response, err := client.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return installError(yorvaruntime.ErrorRuntimeInstallTimeout, err)
		}
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, fmt.Errorf("status %d", response.StatusCode))
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	partial := destination + ".part"
	file, err := os.OpenFile(partial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, limit+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(partial)
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, firstNonNil(copyErr, closeErr))
	}
	if written > limit || written != expectedSize {
		_ = os.Remove(partial)
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("artifact size mismatch"))
	}
	if err := verifySizedDigest(partial, expectedSize, expectedSHA); err != nil {
		_ = os.Remove(partial)
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	if err := os.Rename(partial, destination); err != nil {
		_ = os.Remove(partial)
		return installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	return nil
}

func firstNonNil(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return errors.New("artifact download failed")
}
