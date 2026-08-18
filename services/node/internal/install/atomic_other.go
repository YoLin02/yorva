//go:build !windows

package install

import "os"

func replaceFileAtomic(tmp, dest string) error {
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, dest)
}

func syncDirectory(string) error {
	return nil
}
