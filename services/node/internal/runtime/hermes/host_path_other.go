//go:build !windows

package hermes

func userPathContainsDir(string) bool {
	return false
}

func applyUserPathPostcondition(string, string) error {
	return nil
}
