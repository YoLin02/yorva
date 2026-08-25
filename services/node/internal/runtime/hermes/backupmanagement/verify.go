package backupmanagement

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math"
	"os"
	"path"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	hardMaxContainerBytes      int64  = 2 << 30
	hardMaxPayloadBytes        int64  = 2 << 30
	hardMaxPayloadMembers             = 20_000
	hardMaxMemberExpandedBytes int64  = 512 << 20
	hardMaxTotalExpandedBytes  int64  = 4 << 30
	hardMaxCompressionRatio    uint64 = 200
	maxContainerMembers               = 3
	dosReparsePointAttribute   uint32 = 0x00000400
)

type Limits struct {
	MaxContainerBytes      int64
	MaxPayloadBytes        int64
	MaxPayloadMembers      int
	MaxMemberExpandedBytes int64
	MaxTotalExpandedBytes  int64
	MaxCompressionRatio    uint64
}

func DefaultLimits() Limits {
	return Limits{
		MaxContainerBytes:      hardMaxContainerBytes,
		MaxPayloadBytes:        hardMaxPayloadBytes,
		MaxPayloadMembers:      hardMaxPayloadMembers,
		MaxMemberExpandedBytes: hardMaxMemberExpandedBytes,
		MaxTotalExpandedBytes:  hardMaxTotalExpandedBytes,
		MaxCompressionRatio:    hardMaxCompressionRatio,
	}
}

func (limits Limits) valid() bool {
	return limits.MaxContainerBytes > 0 && limits.MaxContainerBytes <= hardMaxContainerBytes &&
		limits.MaxPayloadBytes > 0 && limits.MaxPayloadBytes <= hardMaxPayloadBytes &&
		limits.MaxPayloadMembers > 0 && limits.MaxPayloadMembers <= hardMaxPayloadMembers &&
		limits.MaxMemberExpandedBytes > 0 && limits.MaxMemberExpandedBytes <= hardMaxMemberExpandedBytes &&
		limits.MaxTotalExpandedBytes > 0 && limits.MaxTotalExpandedBytes <= hardMaxTotalExpandedBytes &&
		limits.MaxCompressionRatio > 0 && limits.MaxCompressionRatio <= hardMaxCompressionRatio
}

// VerifyArtifactBytes verifies a decrypted proposed container in memory. It
// does not retain the input or write any bytes.
func VerifyArtifactBytes(payload []byte, limits Limits) (VerificationResult, error) {
	return VerifyArtifact(bytes.NewReader(payload), int64(len(payload)), limits)
}

// VerifyArtifact verifies a decrypted proposed container using ReaderAt. It
// reads and hashes data but never extracts, writes, invokes Hermes, or mutates
// Runtime state.
func VerifyArtifact(reader io.ReaderAt, size int64, limits Limits) (VerificationResult, error) {
	if reader == nil || size <= 0 || !limits.valid() {
		return fail(ErrorInputInvalid)
	}
	if size > limits.MaxContainerBytes {
		return fail(ErrorArchiveTotalSizeLimit)
	}
	if !hasTerminalEOCD(reader, size) {
		return fail(ErrorArchiveInvalid)
	}
	container, err := zip.NewReader(reader, size)
	if err != nil {
		return fail(ErrorArchiveInvalid)
	}
	if len(container.File) == 0 || len(container.File) > maxContainerMembers {
		return fail(ErrorArchiveMemberLimit)
	}

	seenExact := make(map[string]struct{}, len(container.File))
	seenFolded := make(map[string]string, len(container.File))
	var manifestFile *zip.File
	var payloadFile *zip.File
	var containerExpanded uint64
	var containerCompressed uint64
	for _, member := range container.File {
		if code := validateMember(member, seenExact, seenFolded); code != "" {
			return fail(code)
		}
		if code := addArchiveBounds(member, &containerExpanded, &containerCompressed, uint64(MaxManifestBytes)+uint64(limits.MaxPayloadBytes), uint64(limits.MaxPayloadBytes), limits.MaxCompressionRatio); code != "" {
			return fail(code)
		}
		switch member.Name {
		case "manifest.json":
			if member.FileInfo().IsDir() {
				return fail(ErrorArchiveTypeUnsafe)
			}
			manifestFile = member
		case "payload/":
			if !member.FileInfo().IsDir() {
				return fail(ErrorArchiveTypeUnsafe)
			}
		case PayloadArchivePath:
			if member.FileInfo().IsDir() || member.Method != zip.Store {
				return fail(ErrorArchiveTypeUnsafe)
			}
			payloadFile = member
		default:
			return fail(ErrorArchiveRootUnsupported)
		}
	}
	if manifestFile == nil {
		return fail(ErrorManifestInvalid)
	}
	if payloadFile == nil {
		return fail(ErrorPayloadMissing)
	}

	manifestBytes, code := readMemberBounded(manifestFile, MaxManifestBytes)
	if code != "" {
		return fail(code)
	}
	manifest, parseErr := ParseCanonicalManifest(manifestBytes)
	if parseErr != nil {
		return fail(codeFromError(parseErr, ErrorManifestInvalid))
	}
	if int64(payloadFile.UncompressedSize64) != manifest.Payload.SizeBytes || int64(payloadFile.UncompressedSize64) > limits.MaxPayloadBytes {
		return fail(ErrorPayloadSizeMismatch)
	}
	if payloadFile.CompressedSize64 != payloadFile.UncompressedSize64 {
		return fail(ErrorArchiveTypeUnsafe)
	}
	offset, offsetErr := payloadFile.DataOffset()
	if offsetErr != nil || offset < 0 || manifest.Payload.SizeBytes > size-offset {
		return fail(ErrorArchiveInvalid)
	}
	payloadSection := io.NewSectionReader(reader, offset, manifest.Payload.SizeBytes)
	digest := sha256.New()
	written, hashErr := io.Copy(digest, payloadSection)
	if hashErr != nil || written != manifest.Payload.SizeBytes {
		return fail(ErrorArchiveInvalid)
	}
	if hex.EncodeToString(digest.Sum(nil)) != manifest.Payload.SHA256 {
		return fail(ErrorPayloadChecksumMismatch)
	}

	payloadSection = io.NewSectionReader(reader, offset, manifest.Payload.SizeBytes)
	payloadStats, code := verifyPayloadArchive(payloadSection, manifest.Payload.SizeBytes, limits)
	if code != "" {
		return fail(code)
	}
	if payloadStats.memberCount != manifest.Payload.MemberCount {
		return fail(ErrorPayloadMemberCountMismatch)
	}
	if payloadStats.expandedBytes != manifest.Payload.TotalExpandedBytes {
		return fail(ErrorPayloadExpandedMismatch)
	}

	result := unverifiedResult()
	result.State = ArtifactStructureVerified
	result.Metadata = ArtifactMetadata{
		ArtifactSizeBytes:  size,
		FormatVersion:      manifest.FormatVersion,
		SchemaVersion:      manifest.SchemaVersion,
		PolicyVersion:      manifest.PolicyVersion,
		Scope:              manifest.Scope,
		RuntimeKind:        manifest.RuntimeKind,
		RuntimeVersion:     manifest.RuntimeVersion,
		IncludedCategories: slices.Clone(manifest.IncludedCategories),
		PayloadSizeBytes:   manifest.Payload.SizeBytes,
		PayloadSHA256:      manifest.Payload.SHA256,
		PayloadMemberCount: manifest.Payload.MemberCount,
		TotalExpandedBytes: manifest.Payload.TotalExpandedBytes,
	}
	return result, nil
}

type archiveStats struct {
	memberCount   int
	expandedBytes int64
}

func verifyPayloadArchive(reader io.ReaderAt, size int64, limits Limits) (archiveStats, ErrorCode) {
	if !hasTerminalEOCD(reader, size) {
		return archiveStats{}, ErrorArchiveInvalid
	}
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return archiveStats{}, ErrorArchiveInvalid
	}
	if len(archive.File) == 0 || len(archive.File) > limits.MaxPayloadMembers {
		return archiveStats{}, ErrorArchiveMemberLimit
	}
	seenExact := make(map[string]struct{}, len(archive.File))
	seenFolded := make(map[string]string, len(archive.File))
	var expanded uint64
	var compressed uint64
	regularFiles := 0
	for _, member := range archive.File {
		if code := validateMember(member, seenExact, seenFolded); code != "" {
			return archiveStats{}, code
		}
		if member.Name != PayloadArchiveRoot+"/" && !strings.HasPrefix(member.Name, PayloadArchiveRoot+"/") {
			return archiveStats{}, ErrorArchiveRootUnsupported
		}
		if member.Name == PayloadArchiveRoot+"/" {
			if !member.FileInfo().IsDir() {
				return archiveStats{}, ErrorArchiveTypeUnsafe
			}
			continue
		}
		if code := addArchiveBounds(member, &expanded, &compressed, uint64(limits.MaxTotalExpandedBytes), uint64(limits.MaxMemberExpandedBytes), limits.MaxCompressionRatio); code != "" {
			return archiveStats{}, code
		}
		if member.FileInfo().IsDir() {
			continue
		}
		regularFiles++
		if code := consumeMember(member, limits.MaxMemberExpandedBytes); code != "" {
			return archiveStats{}, code
		}
	}
	if regularFiles == 0 {
		return archiveStats{}, ErrorPayloadMissing
	}
	if exceedsRatio(expanded, compressed, limits.MaxCompressionRatio) {
		return archiveStats{}, ErrorArchiveCompressionLimit
	}
	return archiveStats{memberCount: len(archive.File), expandedBytes: int64(expanded)}, ""
}

// The proposed canonical container permits no ZIP comment or trailing bytes.
// Requiring the terminal EOCD prevents a truncated outer container from being
// misread as an embedded nested ZIP. ZIP64 archives still end with this record.
func hasTerminalEOCD(reader io.ReaderAt, size int64) bool {
	const eocdSize = 22
	if size < eocdSize {
		return false
	}
	record := make([]byte, eocdSize)
	if _, err := reader.ReadAt(record, size-eocdSize); err != nil {
		return false
	}
	return binary.LittleEndian.Uint32(record[0:4]) == 0x06054b50 &&
		binary.LittleEndian.Uint16(record[20:22]) == 0
}

func validateMember(member *zip.File, seenExact map[string]struct{}, seenFolded map[string]string) ErrorCode {
	name := member.Name
	if !safeMemberName(name) {
		return ErrorArchivePathUnsafe
	}
	key := strings.TrimSuffix(name, "/")
	if _, exists := seenExact[key]; exists {
		return ErrorArchiveDuplicateMember
	}
	folded := strings.ToLower(key)
	if previous, exists := seenFolded[folded]; exists && previous != key {
		return ErrorArchiveCaseCollision
	}
	seenExact[key] = struct{}{}
	seenFolded[folded] = key

	mode := member.Mode()
	if member.ExternalAttrs&dosReparsePointAttribute != 0 ||
		mode&os.ModeSymlink != 0 || mode&os.ModeSocket != 0 || mode&os.ModeNamedPipe != 0 || mode&os.ModeDevice != 0 || mode&os.ModeCharDevice != 0 || mode&os.ModeIrregular != 0 {
		return ErrorArchiveTypeUnsafe
	}
	isDirectory := member.FileInfo().IsDir()
	if isDirectory != strings.HasSuffix(name, "/") {
		return ErrorArchiveTypeUnsafe
	}
	if !isDirectory && !mode.IsRegular() {
		return ErrorArchiveTypeUnsafe
	}
	if member.Method != zip.Store && member.Method != zip.Deflate {
		return ErrorArchiveTypeUnsafe
	}
	if isDirectory && (member.CompressedSize64 != 0 || member.UncompressedSize64 != 0) {
		return ErrorArchiveTypeUnsafe
	}
	return ""
}

func safeMemberName(name string) bool {
	if name == "" || !utf8.ValidString(name) || strings.ContainsRune(name, 0) || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || path.IsAbs(name) {
		return false
	}
	trimmed := strings.TrimSuffix(name, "/")
	if trimmed == "" || path.Clean(trimmed) != trimmed {
		return false
	}
	for _, component := range strings.Split(trimmed, "/") {
		if component == "" || component == "." || component == ".." || strings.ContainsAny(component, `<>:"\|?*`) || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
			return false
		}
		for _, character := range component {
			if character < 0x20 || character == 0x7f {
				return false
			}
		}
		if windowsDeviceName(component) {
			return false
		}
	}
	return true
}

func windowsDeviceName(component string) bool {
	base := strings.ToUpper(strings.SplitN(component, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CLOCK$" {
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) {
		return base[3] >= '1' && base[3] <= '9'
	}
	return false
}

func addArchiveBounds(member *zip.File, expanded, compressed *uint64, maxTotal, maxMember, maxRatio uint64) ErrorCode {
	if member.UncompressedSize64 > maxMember {
		return ErrorArchiveMemberSizeLimit
	}
	if member.UncompressedSize64 > math.MaxUint64-*expanded {
		return ErrorArchiveTotalSizeLimit
	}
	*expanded += member.UncompressedSize64
	if *expanded > maxTotal {
		return ErrorArchiveTotalSizeLimit
	}
	if member.CompressedSize64 > math.MaxUint64-*compressed {
		return ErrorArchiveInvalid
	}
	*compressed += member.CompressedSize64
	if exceedsRatio(member.UncompressedSize64, member.CompressedSize64, maxRatio) {
		return ErrorArchiveCompressionLimit
	}
	return ""
}

func exceedsRatio(expanded, compressed, maxRatio uint64) bool {
	if expanded == 0 {
		return false
	}
	if compressed == 0 {
		return true
	}
	if compressed > math.MaxUint64/maxRatio {
		return false
	}
	return expanded > compressed*maxRatio
}

func consumeMember(member *zip.File, limit int64) ErrorCode {
	reader, err := member.Open()
	if err != nil {
		return ErrorArchiveInvalid
	}
	written, copyErr := io.Copy(io.Discard, io.LimitReader(reader, limit+1))
	closeErr := reader.Close()
	if copyErr != nil || closeErr != nil || written != int64(member.UncompressedSize64) {
		return ErrorArchiveInvalid
	}
	if written > limit {
		return ErrorArchiveMemberSizeLimit
	}
	return ""
}

func readMemberBounded(member *zip.File, limit int64) ([]byte, ErrorCode) {
	if int64(member.UncompressedSize64) > limit {
		return nil, ErrorManifestInvalid
	}
	reader, err := member.Open()
	if err != nil {
		return nil, ErrorArchiveInvalid
	}
	payload, readErr := io.ReadAll(io.LimitReader(reader, limit+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || int64(len(payload)) != int64(member.UncompressedSize64) {
		return nil, ErrorArchiveInvalid
	}
	if int64(len(payload)) > limit {
		return nil, ErrorManifestInvalid
	}
	return payload, ""
}

func fail(code ErrorCode) (VerificationResult, error) {
	return failedResult(code), verificationError(code)
}

func codeFromError(err error, fallback ErrorCode) ErrorCode {
	if code, ok := ErrorCodeOf(err); ok {
		return code
	}
	return fallback
}
