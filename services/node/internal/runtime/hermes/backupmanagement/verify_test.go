package backupmanagement

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"testing"
)

type testZipMember struct {
	name          string
	body          []byte
	method        uint16
	mode          os.FileMode
	externalAttrs uint32
}

func TestVerifyArtifactNormal(t *testing.T) {
	members := []testZipMember{
		{name: PayloadArchiveRoot + "/", method: zip.Store, mode: os.ModeDir | 0o700},
		{name: PayloadArchiveRoot + "/config.yaml", body: []byte("model: test\n"), method: zip.Deflate, mode: 0o600},
		{name: PayloadArchiveRoot + "/profiles/work/state.db", body: []byte("sqlite fixture"), method: zip.Store, mode: 0o600},
	}
	artifact := buildTestArtifact(t, members, "")

	result, err := VerifyArtifact(bytes.NewReader(artifact), int64(len(artifact)), DefaultLimits())
	if err != nil {
		t.Fatalf("VerifyArtifact() error = %v", err)
	}
	if result.State != ArtifactStructureVerified {
		t.Fatalf("state = %q", result.State)
	}
	if result.Metadata.RuntimeKind != RuntimeKind || result.Metadata.RuntimeVersion != "0.20.5" {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
	if result.Metadata.PayloadMemberCount != len(members) {
		t.Fatalf("member count = %d, want %d", result.Metadata.PayloadMemberCount, len(members))
	}
	if result.EncryptionQualified || result.RestoreMutationQualified || EncryptionQualified || RestoreMutationQualified {
		t.Fatal("structural verification must not qualify encryption or Restore mutation")
	}
}

func TestVerifyArtifactRejectsTamperedPayloadHash(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), strings.Repeat("0", 64))
	result, err := VerifyArtifactBytes(artifact, DefaultLimits())
	assertErrorCode(t, err, ErrorPayloadChecksumMismatch)
	if result.State != ArtifactInvalid || result.ErrorCode != ErrorPayloadChecksumMismatch {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyArtifactRejectsTruncatedArchive(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	artifact = artifact[:len(artifact)-8]
	_, err := VerifyArtifactBytes(artifact, DefaultLimits())
	assertErrorCode(t, err, ErrorArchiveInvalid)
}

func TestVerifyPayloadRejectsUnsafePathsAndTypes(t *testing.T) {
	tests := []struct {
		name    string
		members []testZipMember
		code    ErrorCode
	}{
		{name: "traversal", members: []testZipMember{{name: PayloadArchiveRoot + "/../escape", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "absolute", members: []testZipMember{{name: "/" + PayloadArchiveRoot + "/file", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "UNC", members: []testZipMember{{name: `\\server\share\file`, body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "ADS", members: []testZipMember{{name: PayloadArchiveRoot + "/config.yaml:secret", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "drive", members: []testZipMember{{name: `C:/file`, body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "question mark", members: []testZipMember{{name: PayloadArchiveRoot + "/a?.txt", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "asterisk", members: []testZipMember{{name: PayloadArchiveRoot + "/a*.txt", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "quote", members: []testZipMember{{name: PayloadArchiveRoot + `/a".txt`, body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "pipe", members: []testZipMember{{name: PayloadArchiveRoot + "/a|.txt", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "less than", members: []testZipMember{{name: PayloadArchiveRoot + "/a<.txt", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "greater than", members: []testZipMember{{name: PayloadArchiveRoot + "/a>.txt", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "device name", members: []testZipMember{{name: PayloadArchiveRoot + "/CON.txt", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchivePathUnsafe},
		{name: "unknown root", members: []testZipMember{{name: "external/file", body: []byte("x"), method: zip.Store, mode: 0o600}}, code: ErrorArchiveRootUnsupported},
		{name: "symlink", members: []testZipMember{{name: PayloadArchiveRoot + "/link", body: []byte("target"), method: zip.Store, mode: os.ModeSymlink | 0o777}}, code: ErrorArchiveTypeUnsafe},
		{name: "named pipe", members: []testZipMember{{name: PayloadArchiveRoot + "/pipe", method: zip.Store, mode: os.ModeNamedPipe | 0o600}}, code: ErrorArchiveTypeUnsafe},
		{name: "reparse attribute", members: []testZipMember{{name: PayloadArchiveRoot + "/reparse", body: []byte("x"), method: zip.Store, mode: 0o600, externalAttrs: dosReparsePointAttribute}}, code: ErrorArchiveTypeUnsafe},
		{name: "duplicate", members: []testZipMember{{name: PayloadArchiveRoot + "/file", body: []byte("a"), method: zip.Store, mode: 0o600}, {name: PayloadArchiveRoot + "/file", body: []byte("b"), method: zip.Store, mode: 0o600}}, code: ErrorArchiveDuplicateMember},
		{name: "case fold", members: []testZipMember{{name: PayloadArchiveRoot + "/Config.yaml", body: []byte("a"), method: zip.Store, mode: 0o600}, {name: PayloadArchiveRoot + "/config.yaml", body: []byte("b"), method: zip.Store, mode: 0o600}}, code: ErrorArchiveCaseCollision},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifact := buildTestArtifact(t, test.members, "")
			_, err := VerifyArtifactBytes(artifact, DefaultLimits())
			assertErrorCode(t, err, test.code)
			if strings.Contains(err.Error(), test.members[0].name) {
				t.Fatal("safe verification error exposed an archive member name")
			}
		})
	}
}

func TestVerifyPayloadAllowsUnicodeAndInteriorSpaces(t *testing.T) {
	members := []testZipMember{
		{name: PayloadArchiveRoot + "/资料/工作 文件.txt", body: []byte("valid"), method: zip.Deflate, mode: 0o600},
		{name: PayloadArchiveRoot + "/profiles/研发 团队/config.yaml", body: []byte("model: test\n"), method: zip.Store, mode: 0o600},
	}
	artifact := buildTestArtifact(t, members, "")
	result, err := VerifyArtifactBytes(artifact, DefaultLimits())
	if err != nil {
		t.Fatalf("VerifyArtifactBytes() error = %v", err)
	}
	if result.State != ArtifactStructureVerified || result.Metadata.PayloadMemberCount != len(members) {
		t.Fatalf("result = %#v", result)
	}
}

func TestVerifyPayloadRejectsBombAndPolicyLimits(t *testing.T) {
	tests := []struct {
		name    string
		members []testZipMember
		limits  func() Limits
		code    ErrorCode
	}{
		{
			name: "member count",
			members: []testZipMember{
				{name: PayloadArchiveRoot + "/a", body: []byte("a"), method: zip.Store, mode: 0o600},
				{name: PayloadArchiveRoot + "/b", body: []byte("b"), method: zip.Store, mode: 0o600},
			},
			limits: func() Limits { limits := DefaultLimits(); limits.MaxPayloadMembers = 1; return limits },
			code:   ErrorArchiveMemberLimit,
		},
		{
			name:    "per member",
			members: []testZipMember{{name: PayloadArchiveRoot + "/large", body: bytes.Repeat([]byte("x"), 16), method: zip.Store, mode: 0o600}},
			limits:  func() Limits { limits := DefaultLimits(); limits.MaxMemberExpandedBytes = 8; return limits },
			code:    ErrorArchiveMemberSizeLimit,
		},
		{
			name: "total expansion",
			members: []testZipMember{
				{name: PayloadArchiveRoot + "/a", body: bytes.Repeat([]byte("a"), 8), method: zip.Store, mode: 0o600},
				{name: PayloadArchiveRoot + "/b", body: bytes.Repeat([]byte("b"), 8), method: zip.Store, mode: 0o600},
			},
			limits: func() Limits {
				limits := DefaultLimits()
				limits.MaxMemberExpandedBytes = 8
				limits.MaxTotalExpandedBytes = 12
				return limits
			},
			code: ErrorArchiveTotalSizeLimit,
		},
		{
			name:    "compression ratio",
			members: []testZipMember{{name: PayloadArchiveRoot + "/bomb", body: bytes.Repeat([]byte{0}, 4096), method: zip.Deflate, mode: 0o600}},
			limits:  func() Limits { limits := DefaultLimits(); limits.MaxCompressionRatio = 2; return limits },
			code:    ErrorArchiveCompressionLimit,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifact := buildTestArtifact(t, test.members, "")
			_, err := VerifyArtifactBytes(artifact, test.limits())
			assertErrorCode(t, err, test.code)
		})
	}
}

func TestVerifyArtifactRejectsInvalidLimitsWithoutReading(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxCompressionRatio = hardMaxCompressionRatio + 1
	_, err := VerifyArtifact(nil, 1, limits)
	assertErrorCode(t, err, ErrorInputInvalid)
}

func normalPayloadMembers() []testZipMember {
	return []testZipMember{{name: PayloadArchiveRoot + "/config.yaml", body: []byte("model: test\n"), method: zip.Store, mode: 0o600}}
}

func buildTestArtifact(t *testing.T, members []testZipMember, checksumOverride string) []byte {
	t.Helper()
	payload := buildTestZip(t, members)
	sum := sha256.Sum256(payload)
	checksum := hex.EncodeToString(sum[:])
	if checksumOverride != "" {
		checksum = checksumOverride
	}
	var expanded int64
	for _, member := range members {
		if member.mode&os.ModeDir == 0 {
			expanded += int64(len(member.body))
		}
	}
	manifest := validTestManifest()
	manifest.Payload.SizeBytes = int64(len(payload))
	manifest.Payload.SHA256 = checksum
	manifest.Payload.MemberCount = len(members)
	manifest.Payload.TotalExpandedBytes = expanded
	manifestBytes, err := manifest.MarshalCanonical()
	if err != nil {
		t.Fatalf("MarshalCanonical() error = %v", err)
	}
	return buildTestZip(t, []testZipMember{
		{name: "manifest.json", body: manifestBytes, method: zip.Store, mode: 0o600},
		{name: PayloadArchivePath, body: payload, method: zip.Store, mode: 0o600},
	})
}

func buildTestZip(t *testing.T, members []testZipMember) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, member := range members {
		header := &zip.FileHeader{Name: member.name, Method: member.method}
		header.SetMode(member.mode)
		header.ExternalAttrs |= member.externalAttrs
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("CreateHeader(%q) error = %v", member.name, err)
		}
		if _, err := io.Copy(entry, bytes.NewReader(member.body)); err != nil {
			t.Fatalf("write member %q: %v", member.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}
