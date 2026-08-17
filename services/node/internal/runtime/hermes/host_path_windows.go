//go:build windows

package hermes

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func userPathContainsDir(dir string) bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue("Path")
	if err != nil {
		return false
	}
	want := filepath.Clean(dir)
	for _, part := range strings.Split(value, string(filepath.ListSeparator)) {
		expanded := osExpandLocalAppData(part)
		if filepath.Clean(expanded) == want {
			return true
		}
	}
	return false
}

func osExpandLocalAppData(value string) string {
	replaced := strings.ReplaceAll(value, "%LOCALAPPDATA%", osGetenvLocalAppData())
	replaced = strings.ReplaceAll(replaced, "%LocalAppData%", osGetenvLocalAppData())
	return replaced
}

func osGetenvLocalAppData() string {
	return strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
}
