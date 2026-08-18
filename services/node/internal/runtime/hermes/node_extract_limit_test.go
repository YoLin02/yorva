package hermes

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPrefixedZipRejectsRealBounds(t *testing.T) {
	t.Run("entry count", func(t *testing.T) {
		files := make(map[string]string, nodeArchiveMaxEntries+1)
		for i := 0; i < nodeArchiveMaxEntries+1; i++ {
			files[officialNodeZipRoot+"/f"+itoa(i)] = "x"
		}
		archive := writeZip(t, files)
		dest := filepath.Join(t.TempDir(), "out")
		if err := extractPrefixedZip(context.Background(), archive, dest, officialNodeZipRoot); err == nil {
			t.Fatal("entry overflow accepted")
		}
		if _, err := os.Stat(dest); !os.IsNotExist(err) {
			t.Fatal("failed zip extract left a destination")
		}
	})
	t.Run("member size", func(t *testing.T) {
		archive := writeClaimedSizeZip(t, officialNodeZipRoot+"/huge.bin", uint32(nodeArchiveMaxMember+1))
		dest := filepath.Join(t.TempDir(), "out")
		if err := extractPrefixedZip(context.Background(), archive, dest, officialNodeZipRoot); err == nil {
			t.Fatal("member overflow accepted")
		}
		if _, err := os.Stat(dest); !os.IsNotExist(err) {
			t.Fatal("failed zip extract left a destination")
		}
	})
	t.Run("uncompressed total", func(t *testing.T) {
		archive := writeClaimedSizeZip(t, officialNodeZipRoot+"/huge.bin", uint32(nodeArchiveMaxUncompressed+1))
		dest := filepath.Join(t.TempDir(), "out")
		if err := extractPrefixedZip(context.Background(), archive, dest, officialNodeZipRoot); err == nil {
			t.Fatal("uncompressed overflow accepted")
		}
	})
}

func TestExtractPrefixedZipRejectsAggregateOnlyUncompressedLimit(t *testing.T) {
	const members = 9
	claimed := make([]claimedZipMember, members)
	var total uint64
	for i := 0; i < members; i++ {
		size := uint32(nodeArchiveMaxMember)
		claimed[i] = claimedZipMember{Name: officialNodeZipRoot + "/part" + itoa(i) + ".bin", Uncompressed: size}
		total += uint64(size)
	}
	if uint64(nodeArchiveMaxMember) >= uint64(nodeArchiveMaxUncompressed) {
		t.Fatal("member limit no longer smaller than aggregate limit")
	}
	if total <= uint64(nodeArchiveMaxUncompressed) {
		t.Fatalf("fixture total %d does not exceed aggregate %d", total, nodeArchiveMaxUncompressed)
	}
	archive := writeClaimedSizeZipMembers(t, claimed)
	dest := filepath.Join(t.TempDir(), "out")
	if err := extractPrefixedZip(context.Background(), archive, dest, officialNodeZipRoot); err == nil {
		t.Fatal("aggregate-only uncompressed overflow accepted")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("failed aggregate zip extract left a destination")
	}
}

func TestExtractNpmTarballRejectsRealBounds(t *testing.T) {
	t.Run("entry count", func(t *testing.T) {
		path := writeTarGZHeaders(t, npmArchiveMaxEntries+1, 1)
		dest := filepath.Join(t.TempDir(), "out")
		if err := extractNpmTarball(context.Background(), path, dest); err == nil {
			t.Fatal("tar entry overflow accepted")
		}
		if _, err := os.Stat(dest); !os.IsNotExist(err) {
			t.Fatal("failed tar extract left a destination")
		}
	})
	t.Run("member size", func(t *testing.T) {
		path := writeSingleTarGZ(t, "package/huge.bin", nodeArchiveMaxMember+1)
		dest := filepath.Join(t.TempDir(), "out")
		if err := extractNpmTarball(context.Background(), path, dest); err == nil {
			t.Fatal("tar member overflow accepted")
		}
	})
	t.Run("wrong prefix", func(t *testing.T) {
		path := writeTarGZ(t, map[string]string{"other/bin/npm-cli.js": "x"})
		if err := extractNpmTarball(context.Background(), path, filepath.Join(t.TempDir(), "out")); err == nil {
			t.Fatal("npm prefix mismatch accepted")
		}
	})
}

type claimedZipMember struct {
	Name         string
	Uncompressed uint32
}

func writeClaimedSizeZip(t *testing.T, name string, claimed uint32) string {
	t.Helper()
	return writeClaimedSizeZipMembers(t, []claimedZipMember{{Name: name, Uncompressed: claimed}})
}

func writeClaimedSizeZipMembers(t *testing.T, members []claimedZipMember) string {
	t.Helper()
	var body bytes.Buffer
	write := func(v any) {
		if err := binary.Write(&body, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	type localMeta struct {
		name  []byte
		start int
	}
	locals := make([]localMeta, 0, len(members))
	data := []byte{'x'}
	for _, member := range members {
		nameBytes := []byte(member.Name)
		start := body.Len()
		body.Write([]byte("PK\x03\x04"))
		write(uint16(20))
		write(uint16(0))
		write(uint16(0))
		write(uint16(0))
		write(uint16(0))
		write(uint32(0))
		write(uint32(len(data)))
		write(member.Uncompressed)
		write(uint16(len(nameBytes)))
		write(uint16(0))
		body.Write(nameBytes)
		body.Write(data)
		locals = append(locals, localMeta{name: nameBytes, start: start})
	}
	centralStart := body.Len()
	for i, member := range members {
		body.Write([]byte("PK\x01\x02"))
		write(uint16(20))
		write(uint16(20))
		write(uint16(0))
		write(uint16(0))
		write(uint16(0))
		write(uint16(0))
		write(uint32(0))
		write(uint32(len(data)))
		write(member.Uncompressed)
		write(uint16(len(locals[i].name)))
		write(uint16(0))
		write(uint16(0))
		write(uint16(0))
		write(uint16(0))
		write(uint32(0))
		write(uint32(locals[i].start))
		body.Write(locals[i].name)
	}
	centralSize := body.Len() - centralStart
	body.Write([]byte("PK\x05\x06"))
	write(uint16(0))
	write(uint16(0))
	write(uint16(len(members)))
	write(uint16(len(members)))
	write(uint32(centralSize))
	write(uint32(centralStart))
	write(uint16(0))
	path := filepath.Join(t.TempDir(), "claimed.zip")
	if err := os.WriteFile(path, body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("crafted zip unreadable: %v", err)
	}
	if len(reader.File) != len(members) {
		_ = reader.Close()
		t.Fatalf("crafted zip files = %d", len(reader.File))
	}
	_ = reader.Close()
	return path
}

func writeTarGZHeaders(t *testing.T, count int, size int64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "many.tgz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	payload := bytes.Repeat([]byte("x"), int(size))
	for i := 0; i < count; i++ {
		header := &tar.Header{Name: "package/f" + itoa(i), Mode: 0600, Size: size}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(payload); err != nil {
			t.Fatal(err)
		}
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = file.Close()
	return path
}

func writeSingleTarGZ(t *testing.T, name string, size int64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "one.tgz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	header := &tar.Header{Name: name, Mode: 0600, Size: size}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("x")); err != nil && size > 1 {
		// header claims more bytes than written; close anyway
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = file.Close()
	return path
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
