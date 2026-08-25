package backupmanagement

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

// DecryptToVerifiedFileWithX25519 writes one fully authenticated plaintext
// container to an empty private staging file and verifies that exact file.
func DecryptToVerifiedFileWithX25519(ctx context.Context, encrypted io.ReaderAt, encryptedSize int64, identity []byte, destination *os.File, limits Limits) (EncryptedArtifactVerification, error) {
	if ctx == nil || encrypted == nil || destination == nil || len(identity) == 0 || len(identity) > maxIdentityBytes || !limits.valid() {
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
	info, err := destination.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != 0 {
		return EncryptedArtifactVerification{}, verificationError(ErrorStagingFailed)
	}
	parsed, err := age.ParseX25519Identity(string(identity))
	if err != nil || parsed.String() != string(identity) {
		return EncryptedArtifactVerification{}, verificationError(ErrorCredentialInvalid)
	}
	frozen, err := snapshotAgeHeader(ctx, encrypted, encryptedSize, ageX25519Stanza)
	if err != nil {
		return EncryptedArtifactVerification{}, err
	}
	plaintext, plaintextSize, err := age.DecryptReaderAt(frozen, encryptedSize, parsed)
	if err != nil || plaintextSize <= 0 || plaintextSize > limits.MaxContainerBytes {
		return EncryptedArtifactVerification{}, verificationError(ErrorDecryptionFailed)
	}
	written, err := io.Copy(destination, io.NewSectionReader(contextReaderAt{ctx: ctx, reader: plaintext}, 0, plaintextSize))
	if err != nil || written != plaintextSize || destination.Sync() != nil {
		return EncryptedArtifactVerification{}, verificationError(ErrorDecryptionFailed)
	}
	verified, err := VerifyArtifact(contextReaderAt{ctx: ctx, reader: destination}, plaintextSize, limits)
	if err != nil {
		return EncryptedArtifactVerification{}, err
	}
	return EncryptedArtifactVerification{EncryptedSizeBytes: encryptedSize, PlaintextSizeBytes: plaintextSize, Artifact: verified}, nil
}

// ExtractVerifiedRuntimeSnapshot extracts only the fixed, already verified
// Hermes payload into an existing empty directory.
func ExtractVerifiedRuntimeSnapshot(ctx context.Context, container *os.File, containerSize int64, target string, limits Limits) (ArtifactMetadata, error) {
	if ctx == nil || container == nil || !filepath.IsAbs(target) || !limits.valid() {
		return ArtifactMetadata{}, verificationError(ErrorInputInvalid)
	}
	info, err := os.Lstat(target)
	if err != nil || !info.IsDir() || unsafeSnapshotInfo(info) {
		return ArtifactMetadata{}, verificationError(ErrorDestinationUnsafe)
	}
	children, err := os.ReadDir(target)
	if err != nil || len(children) != 0 {
		return ArtifactMetadata{}, verificationError(ErrorDestinationConflict)
	}
	verified, err := VerifyArtifact(contextReaderAt{ctx: ctx, reader: container}, containerSize, limits)
	if err != nil {
		return ArtifactMetadata{}, err
	}
	outer, err := zip.NewReader(container, containerSize)
	if err != nil {
		return ArtifactMetadata{}, verificationError(ErrorArchiveInvalid)
	}
	var payload *zip.File
	for _, member := range outer.File {
		if member.Name == PayloadArchivePath {
			payload = member
			break
		}
	}
	if payload == nil {
		return ArtifactMetadata{}, verificationError(ErrorPayloadMissing)
	}
	offset, err := payload.DataOffset()
	if err != nil {
		return ArtifactMetadata{}, verificationError(ErrorArchiveInvalid)
	}
	archive, err := zip.NewReader(io.NewSectionReader(container, offset, int64(payload.UncompressedSize64)), int64(payload.UncompressedSize64))
	if err != nil {
		return ArtifactMetadata{}, verificationError(ErrorArchiveInvalid)
	}
	for _, member := range archive.File {
		if err := contextError(ctx); err != nil {
			return ArtifactMetadata{}, err
		}
		if member.Name == PayloadArchiveRoot+"/" {
			continue
		}
		relative := strings.TrimPrefix(member.Name, PayloadArchiveRoot+"/")
		if relative == member.Name || relative == "" || !safeMemberName(strings.TrimSuffix(relative, "/")) {
			return ArtifactMetadata{}, verificationError(ErrorArchivePathUnsafe)
		}
		destination := filepath.Join(target, filepath.FromSlash(relative))
		if !pathWithinSnapshotRoot(target, destination) {
			return ArtifactMetadata{}, verificationError(ErrorArchivePathUnsafe)
		}
		if member.FileInfo().IsDir() {
			if err := os.Mkdir(destination, 0o700); err != nil {
				return ArtifactMetadata{}, verificationError(ErrorStagingFailed)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return ArtifactMetadata{}, verificationError(ErrorStagingFailed)
		}
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return ArtifactMetadata{}, verificationError(ErrorStagingFailed)
		}
		input, err := member.Open()
		if err != nil {
			_ = output.Close()
			return ArtifactMetadata{}, verificationError(ErrorArchiveInvalid)
		}
		written, copyErr := io.Copy(output, io.LimitReader(input, limits.MaxMemberExpandedBytes+1))
		closeInputErr := input.Close()
		syncErr := output.Sync()
		closeOutputErr := output.Close()
		if copyErr != nil || closeInputErr != nil || syncErr != nil || closeOutputErr != nil || written != int64(member.UncompressedSize64) {
			return ArtifactMetadata{}, verificationError(ErrorStagingFailed)
		}
	}
	return verified.Metadata, nil
}

// CanonicalHermesRuntimeRootPath resolves the fixed Hermes state path without
// requiring it to exist. Recovery needs this while the active tree is between
// two atomic renames.
func CanonicalHermesRuntimeRootPath() (string, error) {
	return canonicalHermesRuntimeRoot()
}

// CanonicalHermesRuntimeRoot is exposed only to the Hermes restore adapter.
func CanonicalHermesRuntimeRoot() (string, error) {
	root, err := CanonicalHermesRuntimeRootPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		return "", verificationError(ErrorSourceUnsafe)
	}
	return root, err
}
