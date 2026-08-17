//go:build !windows

package hermes

func userPathContainsDir(string) bool {
	return false
}
