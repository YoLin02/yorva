//go:build !windows

package install

import "os"

func renameDirAtomic(from, to string) error {
	return os.Rename(from, to)
}
