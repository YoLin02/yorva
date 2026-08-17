package applog

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/YoLin02/yorva/services/node/internal/buildinfo"
)

const (
	installLogDir  = "logs"
	InstallLogName = "install.ndjson"
)

func InstallLogPath(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	return filepath.Join(dataDir, installLogDir, InstallLogName)
}

func New(stderr io.Writer, dataDir string) (*slog.Logger, func()) {
	writers := make([]io.Writer, 0, 2)
	if stderr != nil {
		writers = append(writers, stderr)
	}
	closer := func() {}
	if path := InstallLogPath(dataDir); path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err == nil {
			if file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
				writers = append(writers, file)
				closer = func() { _ = file.Close() }
			}
		}
	}
	if len(writers) == 0 {
		writers = append(writers, io.Discard)
	}
	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(writers...), nil)).With(
		"service", buildinfo.Service,
		"version", buildinfo.Version,
	)
	return logger, closer
}

func ReadMatching(dataDir, needle string, limit int) string {
	if dataDir == "" || needle == "" || limit <= 0 {
		return ""
	}
	file, err := os.Open(InstallLogPath(dataDir))
	if err != nil {
		return ""
	}
	defer file.Close()
	var matched []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, needle) {
			matched = append(matched, line)
		}
	}
	text := strings.Join(matched, "\n")
	if len(text) > limit {
		return text[len(text)-limit:]
	}
	return text
}
