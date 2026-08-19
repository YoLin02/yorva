//go:build !windows

package hermes

import "os"

func atomicReplaceCredentialFile(from, to string) error {
	return os.Rename(from, to)
}
