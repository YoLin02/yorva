package hermes

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	nodeArchiveMaxEntries      = 8000
	nodeArchiveMaxUncompressed = 256 << 20
	nodeArchiveMaxMember       = 96 << 20
	npmArchiveMaxEntries       = 8000
	npmArchiveMaxUncompressed  = 64 << 20
	nodeDepsOutputLimit        = 1 << 20
)

func verifySizedDigest(path string, size int64, digest string) error {
	if err := rejectReparsePoint(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return errors.New("archive is not the expected regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(sum.Sum(nil)), digest) {
		return errors.New("archive digest mismatch")
	}
	return nil
}

func extractPrefixedZip(ctx context.Context, archivePath, dest, prefix string) error {
	err := extractPrefixedZipInto(ctx, archivePath, dest, prefix)
	if err != nil {
		_ = os.RemoveAll(dest)
	}
	return err
}

func extractPrefixedZipInto(ctx context.Context, archivePath, dest, prefix string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) > nodeArchiveMaxEntries {
		return errors.New("zip entry count exceeded")
	}
	var uncompressed int64
	matched := 0
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := filepath.ToSlash(file.Name)
		if strings.Contains(name, "..") || strings.Contains(name, ":") || strings.HasPrefix(name, "/") {
			return errors.New("unsafe zip member")
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return errors.New("zip symlink rejected")
		}
		uncompressed += int64(file.UncompressedSize64)
		if uncompressed > nodeArchiveMaxUncompressed || int64(file.UncompressedSize64) > nodeArchiveMaxMember {
			return errors.New("zip expansion limit exceeded")
		}
		rel := strings.TrimPrefix(name, prefix+"/")
		if rel == name || rel == "" {
			continue
		}
		matched++
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if !pathWithin(dest, target) {
			return errors.New("zip member escaped destination")
		}
		if file.FileInfo().IsDir() || strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		if err := writeZipFile(file, target); err != nil {
			return err
		}
	}
	if matched == 0 {
		return errors.New("archive root prefix mismatch")
	}
	return nil
}

func writeZipFile(file *zip.File, target string) error {
	source, err := file.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, io.LimitReader(source, nodeArchiveMaxMember+1))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func extractNpmTarball(ctx context.Context, archivePath, dest string) error {
	err := extractNpmTarballInto(ctx, archivePath, dest)
	if err != nil {
		_ = os.RemoveAll(dest)
	}
	return err
}

func extractNpmTarballInto(ctx context.Context, archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	var entries int
	var uncompressed int64
	matched := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		entries++
		if entries > npmArchiveMaxEntries {
			return errors.New("tar entry count exceeded")
		}
		name := filepath.ToSlash(header.Name)
		if strings.Contains(name, "..") || strings.Contains(name, ":") || strings.HasPrefix(name, "/") {
			return errors.New("unsafe tar member")
		}
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return errors.New("tar symlink rejected")
		}
		uncompressed += header.Size
		if uncompressed > npmArchiveMaxUncompressed || header.Size > nodeArchiveMaxMember {
			return errors.New("tar expansion limit exceeded")
		}
		rel := strings.TrimPrefix(name, "package/")
		if rel == name || rel == "" {
			continue
		}
		matched++
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if !pathWithin(dest, target) {
			return errors.New("tar member escaped destination")
		}
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, io.LimitReader(reader, nodeArchiveMaxMember+1))
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if matched == 0 {
		return errors.New("archive root prefix mismatch")
	}
	return nil
}
