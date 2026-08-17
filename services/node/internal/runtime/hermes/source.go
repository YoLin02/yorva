package hermes

import (
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

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var errRedirectRejected = errors.New("official source redirect rejected")

type fetchedScript struct {
	Directory string
	Path      string
	SHA256    string
	Size      int64
}

type sourceClient struct {
	http      *http.Client
	source    officialSource
	now       func() time.Time
	limit     int64
	fetchWait time.Duration
}

func newSourceClient(source officialSource) sourceClient {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   sourceConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   sourceConnectTimeout,
		ResponseHeaderTimeout: sourceConnectTimeout,
		ForceAttemptHTTP2:     true,
	}
	return sourceClient{
		http: &http.Client{
			Transport: transport,
			Timeout:   sourceFetchTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errRedirectRejected
			},
		},
		source:    source,
		now:       time.Now,
		limit:     sourceDownloadLimit,
		fetchWait: sourceFetchTimeout,
	}
}

func (c sourceClient) Fetch(ctx context.Context, stateRoot, operationID string) (fetchedScript, error) {
	directory, err := operationPrivateDir(stateRoot, operationID)
	if err != nil {
		return fetchedScript{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.source.URL, nil)
	if err != nil {
		return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	response, err := c.http.Do(request)
	if err != nil {
		if errors.Is(err, errRedirectRejected) {
			return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, errRedirectRejected)
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallTimeout, err)
		}
		return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, fmt.Errorf("status %d", response.StatusCode))
	}
	limited := io.LimitReader(response.Body, c.limit+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	if int64(len(payload)) > c.limit {
		return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("source exceeded download bound"))
	}
	if int64(len(payload)) != c.source.ExpectedSize {
		return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("source size mismatch"))
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	if !strings.EqualFold(digest, c.source.ExpectedSHA) {
		return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("source digest mismatch"))
	}

	scriptPath := filepath.Join(directory, "install.ps1")
	if err := os.WriteFile(scriptPath, payload, 0o600); err != nil {
		return fetchedScript{}, installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	if err := verifyRegularFile(scriptPath, c.source.ExpectedSize, c.source.ExpectedSHA); err != nil {
		_ = os.Remove(scriptPath)
		return fetchedScript{}, err
	}
	return fetchedScript{Directory: directory, Path: scriptPath, SHA256: digest, Size: int64(len(payload))}, nil
}

func verifyRegularFile(path string, expectedSize int64, expectedSHA string) error {
	if err := rejectReparsePoint(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	if !info.Mode().IsRegular() || info.Size() != expectedSize {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("script is not the expected regular file"))
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
	if !strings.EqualFold(hex.EncodeToString(sum.Sum(nil)), expectedSHA) {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("script digest changed"))
	}
	return nil
}

func operationPrivateDir(stateRoot, operationID string) (string, error) {
	if stateRoot == "" || operationID == "" || strings.Contains(operationID, "..") || strings.ContainsAny(operationID, `/\`) {
		return "", installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, errors.New("invalid operation storage path"))
	}
	root, err := filepath.Abs(stateRoot)
	if err != nil {
		return "", installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	if err := rejectReparsePoint(root); err != nil && !errors.Is(err, os.ErrNotExist) {
		if !os.IsNotExist(err) && !isNotExistInstallError(err) {
			// continue if root does not exist yet
		}
	}
	directory := filepath.Join(root, "operations", operationID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", installError(yorvaruntime.ErrorRuntimeInstallSourceUnavailable, err)
	}
	if err := rejectReparsePoint(directory); err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, directory)
	if err != nil || strings.HasPrefix(relative, "..") {
		return "", installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("operation directory escaped state root"))
	}
	return directory, nil
}

func cleanupFetchedScript(script fetchedScript) error {
	if script.Directory == "" {
		return nil
	}
	return os.RemoveAll(script.Directory)
}

type codedError struct {
	code yorvaruntime.ErrorCode
	err  error
}

func (e codedError) Error() string {
	if e.err == nil {
		return string(e.code)
	}
	return e.err.Error()
}

func (e codedError) Unwrap() error { return e.err }

func (e codedError) Code() yorvaruntime.ErrorCode { return e.code }

func installError(code yorvaruntime.ErrorCode, err error) error {
	return codedError{code: code, err: err}
}

func installErrorCode(err error) yorvaruntime.ErrorCode {
	var coded codedError
	if errors.As(err, &coded) {
		return coded.code
	}
	return yorvaruntime.ErrorRuntimeInstallSourceUnavailable
}

func isNotExistInstallError(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
