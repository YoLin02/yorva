package backupmanagement

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const maxSnapshotMemberNameBytes = 4 << 10

// SnapshotDescriptor is caller-observed immutable identity for one snapshot.
// It deliberately accepts no native path, Profile, account or credential data.
type SnapshotDescriptor struct {
	BackupID       string
	CreatedAt      string
	RuntimeVersion string
	Installation   InstallationIdentity
}

func (d SnapshotDescriptor) valid() bool {
	return validClosedID(d.BackupID, backupIDPrefix, backupIDBodyLength) &&
		validCanonicalCreatedAt(d.CreatedAt) &&
		d.RuntimeVersion == QualifiedSnapshotRuntimeVersion &&
		d.Installation.valid()
}

// SnapshotStaging consists of two distinct, empty, read/write regular files
// already created by the caller inside its access-restricted Operation
// staging. The builder creates no path and owns neither file's cleanup.
type SnapshotStaging struct {
	Payload   *os.File
	Container *os.File
}

// SnapshotResult contains only bounded, non-secret construction facts. It
// deliberately does not expose source paths, Profile names or member names.
type SnapshotResult struct {
	Artifact        VerificationResult
	SourceFileCount int
	SourceBytes     int64
}

// BuildCanonicalHermesRuntimeSnapshot creates one canonical plaintext backup
// container from the fixed canonical Hermes Runtime root. It is intentionally
// not wired to HTTP, Operations, encryption, publication, indexing or Restore.
//
// runtimeStopped is an explicit caller-proven prerequisite. This function does
// not infer process state. Until exact live-capture qualification exists, false
// fails closed before source bytes are read.
func BuildCanonicalHermesRuntimeSnapshot(ctx context.Context, descriptor SnapshotDescriptor, runtimeStopped bool, staging SnapshotStaging, limits Limits) (SnapshotResult, error) {
	root, err := canonicalHermesRuntimeRoot()
	if err != nil {
		return SnapshotResult{}, err
	}
	return buildHermesRuntimeSnapshot(ctx, root, descriptor, runtimeStopped, staging, limits)
}

func buildHermesRuntimeSnapshot(ctx context.Context, root string, descriptor SnapshotDescriptor, runtimeStopped bool, staging SnapshotStaging, limits Limits) (SnapshotResult, error) {
	if ctx == nil || !descriptor.valid() || !limits.valid() {
		return SnapshotResult{}, verificationError(ErrorInputInvalid)
	}
	if !runtimeStopped {
		return SnapshotResult{}, verificationError(ErrorSourceRuntimeLive)
	}
	if err := validateSnapshotStaging(root, staging); err != nil {
		return SnapshotResult{}, err
	}
	if err := contextError(ctx); err != nil {
		return SnapshotResult{}, err
	}

	inventory, err := inventorySnapshotSource(ctx, root, limits)
	if err != nil {
		return SnapshotResult{}, err
	}
	if snapshotPayloadUpperBound(inventory) > limits.MaxPayloadBytes {
		return SnapshotResult{}, verificationError(ErrorArchiveTotalSizeLimit)
	}
	for _, entry := range inventory.entries {
		if entry.isDir {
			continue
		}
		for _, stagingInfo := range mustSnapshotStagingInfo(staging) {
			if os.SameFile(entry.info, stagingInfo) {
				return SnapshotResult{}, verificationError(ErrorSourceUnsafe)
			}
		}
	}

	payloadStats, err := writeSnapshotPayload(ctx, staging.Payload, inventory, limits)
	if err != nil {
		return SnapshotResult{}, err
	}
	if err := verifySnapshotDirectoriesUnchanged(ctx, inventory.directories); err != nil {
		return SnapshotResult{}, err
	}
	if err := staging.Payload.Sync(); err != nil {
		return SnapshotResult{}, verificationError(ErrorStagingFailed)
	}
	payloadInfo, err := staging.Payload.Stat()
	if err != nil || payloadInfo.Size() <= 0 || payloadInfo.Size() > limits.MaxPayloadBytes {
		return SnapshotResult{}, verificationError(ErrorArchiveTotalSizeLimit)
	}
	payloadDigest, err := hashOpenStaging(ctx, staging.Payload, payloadInfo.Size())
	if err != nil {
		return SnapshotResult{}, err
	}

	manifest := Manifest{
		FormatVersion:      FormatVersion,
		SchemaVersion:      ManifestSchemaVersion,
		PolicyVersion:      ManifestPolicyVersion,
		BackupID:           descriptor.BackupID,
		CreatedAt:          descriptor.CreatedAt,
		Scope:              RuntimeScope,
		RuntimeKind:        RuntimeKind,
		RuntimeVersion:     descriptor.RuntimeVersion,
		Installation:       descriptor.Installation,
		InclusionPolicy:    InclusionPolicyID,
		IncludedCategories: slices.Clone(completeRuntimeSnapshotCategories),
		ExclusionPolicy:    ExclusionPolicyID,
		ExcludedCategories: slices.Clone(completeRuntimeSnapshotExcludedCategories),
		Payload: Payload{
			Path:               PayloadArchivePath,
			SizeBytes:          payloadInfo.Size(),
			SHA256:             payloadDigest,
			MemberCount:        payloadStats.memberCount,
			TotalExpandedBytes: payloadStats.expandedBytes,
		},
		Complete: true,
	}
	manifestBytes, err := manifest.MarshalCanonical()
	if err != nil {
		return SnapshotResult{}, err
	}
	if payloadInfo.Size() > limits.MaxContainerBytes-int64(len(manifestBytes))-(16<<10) {
		return SnapshotResult{}, verificationError(ErrorArchiveTotalSizeLimit)
	}
	if err := writeSnapshotContainer(ctx, staging.Container, staging.Payload, payloadInfo.Size(), manifestBytes); err != nil {
		return SnapshotResult{}, err
	}
	if err := staging.Container.Sync(); err != nil {
		return SnapshotResult{}, verificationError(ErrorStagingFailed)
	}
	containerInfo, err := staging.Container.Stat()
	if err != nil || containerInfo.Size() <= 0 || containerInfo.Size() > limits.MaxContainerBytes {
		return SnapshotResult{}, verificationError(ErrorArchiveTotalSizeLimit)
	}
	verified, err := VerifyArtifact(contextReaderAt{ctx: ctx, reader: staging.Container}, containerInfo.Size(), limits)
	if err != nil {
		return SnapshotResult{}, err
	}
	if verified.Metadata.PayloadSHA256 != payloadDigest || verified.Metadata.PayloadMemberCount != payloadStats.memberCount || verified.Metadata.TotalExpandedBytes != payloadStats.expandedBytes {
		return SnapshotResult{}, verificationError(ErrorSourceIncomplete)
	}
	return SnapshotResult{Artifact: verified, SourceFileCount: inventory.fileCount, SourceBytes: inventory.totalBytes}, nil
}

func validateSnapshotStaging(root string, staging SnapshotStaging) error {
	if staging.Payload == nil || staging.Container == nil {
		return verificationError(ErrorInputInvalid)
	}
	payloadInfo, payloadErr := staging.Payload.Stat()
	containerInfo, containerErr := staging.Container.Stat()
	if payloadErr != nil || containerErr != nil || !payloadInfo.Mode().IsRegular() || !containerInfo.Mode().IsRegular() || payloadInfo.Size() != 0 || containerInfo.Size() != 0 || os.SameFile(payloadInfo, containerInfo) {
		return verificationError(ErrorStagingFailed)
	}
	for _, file := range []*os.File{staging.Payload, staging.Container} {
		name, err := filepath.Abs(file.Name())
		if err != nil || pathWithinSnapshotRoot(root, name) {
			return verificationError(ErrorSourceUnsafe)
		}
		pathInfo, err := os.Lstat(name)
		if err != nil || unsafeSnapshotInfo(pathInfo) || !os.SameFile(pathInfo, mustFileInfo(file)) {
			return verificationError(ErrorStagingFailed)
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return verificationError(ErrorStagingFailed)
		}
	}
	return nil
}

func pathWithinSnapshotRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func mustFileInfo(file *os.File) fs.FileInfo {
	info, _ := file.Stat()
	return info
}

func mustSnapshotStagingInfo(staging SnapshotStaging) []fs.FileInfo {
	return []fs.FileInfo{mustFileInfo(staging.Payload), mustFileInfo(staging.Container)}
}

func writeSnapshotPayload(ctx context.Context, destination *os.File, inventory snapshotInventory, limits Limits) (archiveStats, error) {
	writer := zip.NewWriter(destination)
	closed := false
	defer func() {
		if !closed {
			_ = writer.Close()
		}
	}()

	stats := archiveStats{}
	for _, entry := range inventory.entries {
		if err := contextError(ctx); err != nil {
			return archiveStats{}, err
		}
		name := PayloadArchiveRoot + "/"
		mode := fs.FileMode(0o700) | fs.ModeDir
		if entry.relPath != "" {
			name += entry.relPath
			if entry.isDir {
				name += "/"
			} else {
				mode = 0o600
			}
		}
		if len(name) > maxSnapshotMemberNameBytes || !safeMemberName(name) {
			return archiveStats{}, verificationError(ErrorSourceUnsafe)
		}
		header := &zip.FileHeader{Name: name, Method: zip.Store, Modified: time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)}
		header.SetMode(mode)
		member, err := writer.CreateHeader(header)
		if err != nil {
			return archiveStats{}, verificationError(ErrorStagingFailed)
		}
		stats.memberCount++
		if entry.isDir {
			continue
		}
		if entry.info.Size() > limits.MaxMemberExpandedBytes || stats.expandedBytes > limits.MaxTotalExpandedBytes-entry.info.Size() {
			return archiveStats{}, verificationError(ErrorArchiveTotalSizeLimit)
		}
		if err := copyStableSnapshotFile(ctx, member, entry); err != nil {
			return archiveStats{}, err
		}
		stats.expandedBytes += entry.info.Size()
	}
	if err := writer.Close(); err != nil {
		return archiveStats{}, verificationError(ErrorStagingFailed)
	}
	closed = true
	return stats, nil
}

func snapshotPayloadUpperBound(inventory snapshotInventory) int64 {
	// Store-only ZIP members need local and central headers plus a data
	// descriptor. Five hundred and twelve bytes per member safely covers those
	// fixed records, timestamp extras and both copies of the bounded name.
	upper := inventory.totalBytes + 22
	for _, entry := range inventory.entries {
		nameLength := len(PayloadArchiveRoot) + 1 + len(entry.relPath)
		if entry.isDir && entry.relPath != "" {
			nameLength++
		}
		upper += 512 + int64(nameLength*2)
	}
	return upper
}

func copyStableSnapshotFile(ctx context.Context, destination io.Writer, entry snapshotSourceEntry) error {
	file, err := openSnapshotSourceFile(entry.absPath)
	if err != nil {
		return verificationError(ErrorSourceUnsafe)
	}
	before, err := file.Stat()
	if err != nil || !sameSnapshotFileState(entry.info, before) || unsafeSnapshotInfo(before) {
		_ = file.Close()
		return verificationError(ErrorSourceChanged)
	}
	firstDigest := sha256.New()
	written, copyErr := copySnapshotBounded(ctx, io.MultiWriter(destination, firstDigest), file, entry.info.Size())
	after, statErr := file.Stat()
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if statErr != nil || closeErr != nil || written != entry.info.Size() || !sameSnapshotFileState(before, after) {
		return verificationError(ErrorSourceChanged)
	}

	secondDigest, secondInfo, err := hashSnapshotSourceFile(ctx, entry.absPath, entry.info.Size())
	if err != nil || !sameSnapshotFileState(after, secondInfo) || secondDigest != hex.EncodeToString(firstDigest.Sum(nil)) {
		return verificationError(ErrorSourceChanged)
	}
	return nil
}

func hashSnapshotSourceFile(ctx context.Context, path string, expectedSize int64) (string, fs.FileInfo, error) {
	file, err := openSnapshotSourceFile(path)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || unsafeSnapshotInfo(before) {
		return "", nil, verificationError(ErrorSourceChanged)
	}
	digest := sha256.New()
	written, err := copySnapshotBounded(ctx, digest, file, expectedSize)
	after, statErr := file.Stat()
	if err != nil {
		return "", nil, err
	}
	if statErr != nil || written != expectedSize || !sameSnapshotFileState(before, after) {
		return "", nil, verificationError(ErrorSourceChanged)
	}
	return hex.EncodeToString(digest.Sum(nil)), after, nil
}

func copySnapshotBounded(ctx context.Context, destination io.Writer, source io.Reader, expected int64) (int64, error) {
	buffer := make([]byte, 64<<10)
	limited := io.LimitReader(source, expected+1)
	var total int64
	for {
		if err := contextError(ctx); err != nil {
			return total, err
		}
		read, readErr := limited.Read(buffer)
		if read > 0 {
			written, writeErr := destination.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil || written != read {
				return total, verificationError(ErrorStagingFailed)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return total, verificationError(ErrorSourceIncomplete)
		}
	}
	if total != expected {
		return total, verificationError(ErrorSourceChanged)
	}
	return total, nil
}

func hashOpenStaging(ctx context.Context, file *os.File, expectedSize int64) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", verificationError(ErrorStagingFailed)
	}
	digest := sha256.New()
	written, err := copySnapshotBounded(ctx, digest, file, expectedSize)
	if err != nil {
		return "", err
	}
	if written != expectedSize {
		return "", verificationError(ErrorStagingFailed)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func writeSnapshotContainer(ctx context.Context, destination, payload *os.File, payloadSize int64, manifest []byte) error {
	if _, err := destination.Seek(0, io.SeekStart); err != nil {
		return verificationError(ErrorStagingFailed)
	}
	writer := zip.NewWriter(destination)
	closed := false
	defer func() {
		if !closed {
			_ = writer.Close()
		}
	}()
	manifestHeader := &zip.FileHeader{Name: "manifest.json", Method: zip.Store, Modified: time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)}
	manifestHeader.SetMode(0o600)
	manifestMember, err := writer.CreateHeader(manifestHeader)
	if err != nil {
		return verificationError(ErrorStagingFailed)
	}
	if _, err := manifestMember.Write(manifest); err != nil {
		return verificationError(ErrorStagingFailed)
	}
	directoryHeader := &zip.FileHeader{Name: "payload/", Method: zip.Store, Modified: time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)}
	directoryHeader.SetMode(fs.ModeDir | 0o700)
	if _, err := writer.CreateHeader(directoryHeader); err != nil {
		return verificationError(ErrorStagingFailed)
	}
	payloadHeader := &zip.FileHeader{Name: PayloadArchivePath, Method: zip.Store, Modified: time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)}
	payloadHeader.SetMode(0o600)
	payloadMember, err := writer.CreateHeader(payloadHeader)
	if err != nil {
		return verificationError(ErrorStagingFailed)
	}
	if _, err := payload.Seek(0, io.SeekStart); err != nil {
		return verificationError(ErrorStagingFailed)
	}
	if written, err := copySnapshotBounded(ctx, payloadMember, payload, payloadSize); err != nil {
		return err
	} else if written != payloadSize {
		return verificationError(ErrorStagingFailed)
	}
	if err := writer.Close(); err != nil {
		return verificationError(ErrorStagingFailed)
	}
	closed = true
	return nil
}
