//go:build !windows

package sqlite

import "os"

func replaceFile(from, to string) error { return os.Rename(from, to) }
