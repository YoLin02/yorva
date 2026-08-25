package backupmanagement

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	backupFileSuffix       = ".yorva-backup.age"
	stagingFilePrefix      = ".yorva-backup-staging-"
	stagingNameRandomBytes = 12
	maxDestinationBytes    = 32 << 10
	minimumSpaceReserve    = 1 << 20
)

// EncryptionMode identifies the one and only credential authority selected
// for a publication. There is no fallback between modes.
type EncryptionMode string

const (
	EncryptionModeDevice     EncryptionMode = "DEVICE_X25519"
	EncryptionModePassphrase EncryptionMode = "PORTABLE_PASSPHRASE"
)

type PublicationState string

const (
	PublicationVerified          PublicationState = "PUBLISHED_VERIFIED"
	PublicationReconcileRequired PublicationState = "PUBLISHED_RECONCILE_REQUIRED"
)

// CreateCredential is a one-shot, in-memory credential. Its formatting methods
// always redact it. Publication consumes and clears its secret bytes even when
// the operation fails.
type CreateCredential struct {
	mu        sync.Mutex
	mode      EncryptionMode
	recipient string
	secret    []byte
	consumed  bool
}

func NewDeviceCreateCredential(recipient string, identity []byte) (*CreateCredential, error) {
	if len(recipient) == 0 || len(recipient) > maxRecipientBytes || len(identity) == 0 || len(identity) > maxIdentityBytes {
		return nil, verificationError(ErrorInputInvalid)
	}
	return &CreateCredential{mode: EncryptionModeDevice, recipient: recipient, secret: bytesClone(identity)}, nil
}

func NewPassphraseCreateCredential(passphrase []byte) (*CreateCredential, error) {
	if len(passphrase) == 0 || len(passphrase) > maxPassphraseBytes {
		return nil, verificationError(ErrorInputInvalid)
	}
	return &CreateCredential{mode: EncryptionModePassphrase, secret: bytesClone(passphrase)}, nil
}

func (c *CreateCredential) String() string   { return "[REDACTED BACKUP CREDENTIAL]" }
func (c *CreateCredential) GoString() string { return "[REDACTED BACKUP CREDENTIAL]" }

func (c *CreateCredential) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, "[REDACTED BACKUP CREDENTIAL]")
}

func (c *CreateCredential) consume() (EncryptionMode, string, []byte, error) {
	if c == nil {
		return "", "", nil, verificationError(ErrorInputInvalid)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.consumed || len(c.secret) == 0 {
		return "", "", nil, verificationError(ErrorInputInvalid)
	}
	c.consumed = true
	secret := bytesClone(c.secret)
	clear(c.secret)
	c.secret = nil
	return c.mode, c.recipient, secret, nil
}

// LocalDestination is an opaque, locally-selected destination. It must be
// created inside the trusted local/native picker integration, never decoded
// from a remote API body. Formatting never exposes the path.
type LocalDestination struct {
	path string
}

// InspectLocalDestination validates a path selected by trusted local UI. This
// constructor is deliberately not wired to HTTP and grants no directory or
// arbitrary-file access beyond one absent .yorva-backup.age file.
func InspectLocalDestination(path string) (LocalDestination, error) {
	path, err := normalizeDestination(path)
	if err != nil {
		return LocalDestination{}, err
	}
	if _, err := inspectDestination(path); err != nil {
		return LocalDestination{}, err
	}
	return LocalDestination{path: path}, nil
}

func (d LocalDestination) String() string   { return "[LOCAL BACKUP DESTINATION]" }
func (d LocalDestination) GoString() string { return "[LOCAL BACKUP DESTINATION]" }

func (d LocalDestination) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, "[LOCAL BACKUP DESTINATION]")
}

// PublicationRequest accepts only an already constructed canonical plaintext
// container. This layer does not enumerate Hermes state or claim that the
// artifact is a complete Runtime backup.
type PublicationRequest struct {
	Destination  LocalDestination
	Artifact     io.ReaderAt
	ArtifactSize int64
	Credential   *CreateCredential
	Limits       Limits
}

// PublishedArtifactMetadata contains only safe metadata derived from the
// published ciphertext and its fully authenticated plaintext. A future
// Runtime-scope index may persist it together with its own internal destination
// and key references; neither is returned here.
type PublishedArtifactMetadata struct {
	State              PublicationState
	EncryptedSizeBytes int64
	ChecksumSHA256     string
	EncryptionMode     EncryptionMode
	Artifact           VerificationResult
}

type publicationOps struct {
	freeBytes     func(string) (uint64, error)
	createStaging func(string) (*os.File, error)
	publish       func(string, string) error
	syncDirectory func(string) error
	beforePublish func() error
}

func defaultPublicationOps() publicationOps {
	return publicationOps{
		freeBytes:     destinationFreeBytes,
		createStaging: createPrivateStaging,
		publish:       publishNoReplace,
		syncDirectory: syncDestinationDirectory,
	}
}

// PublishVerifiedArtifact verifies the plaintext container, consumes one
// request-scoped credential, and publishes one new encrypted file without
// overwriting an existing destination.
func PublishVerifiedArtifact(ctx context.Context, request PublicationRequest) (PublishedArtifactMetadata, error) {
	return publishVerifiedArtifact(ctx, request, defaultPublicationOps())
}

func publishVerifiedArtifact(ctx context.Context, request PublicationRequest, ops publicationOps) (result PublishedArtifactMetadata, err error) {
	if ctx == nil || request.Artifact == nil || request.ArtifactSize <= 0 || !request.Limits.valid() || request.Destination.path == "" {
		return PublishedArtifactMetadata{}, verificationError(ErrorInputInvalid)
	}
	mode, recipient, secret, err := request.Credential.consume()
	if err != nil {
		return PublishedArtifactMetadata{}, err
	}
	defer clear(secret)
	if err := contextError(ctx); err != nil {
		return PublishedArtifactMetadata{}, err
	}

	// Fail before creating destination-adjacent material unless the source is a
	// canonical, policy-valid container. The encryption primitive verifies the
	// final ciphertext again, so a mutable source cannot carry this result over.
	if _, err := VerifyArtifact(contextReaderAt{ctx: ctx, reader: request.Artifact}, request.ArtifactSize, request.Limits); err != nil {
		return PublishedArtifactMetadata{}, err
	}

	destination, err := normalizeDestination(request.Destination.path)
	if err != nil {
		return PublishedArtifactMetadata{}, err
	}
	parentSnapshot, err := inspectDestination(destination)
	if err != nil {
		return PublishedArtifactMetadata{}, err
	}
	if ops.freeBytes == nil || ops.createStaging == nil || ops.publish == nil || ops.syncDirectory == nil {
		return PublishedArtifactMetadata{}, verificationError(ErrorInputInvalid)
	}
	requiredSpace, ok := publicationSpaceRequirement(request.ArtifactSize)
	if !ok {
		return PublishedArtifactMetadata{}, verificationError(ErrorEncryptedArtifactLimit)
	}
	freeBytes, err := ops.freeBytes(filepath.Dir(destination))
	if err != nil {
		return PublishedArtifactMetadata{}, verificationError(ErrorDestinationUnsafe)
	}
	if freeBytes < requiredSpace {
		return PublishedArtifactMetadata{}, verificationError(ErrorInsufficientSpace)
	}

	stagingPath, err := allocateStagingPath(filepath.Dir(destination))
	if err != nil {
		return PublishedArtifactMetadata{}, verificationError(ErrorStagingFailed)
	}
	staging, err := ops.createStaging(stagingPath)
	if err != nil {
		return PublishedArtifactMetadata{}, verificationError(ErrorStagingFailed)
	}
	stagingClosed := false
	published := false
	defer func() {
		if !stagingClosed {
			_ = staging.Close()
		}
		if !published {
			_ = os.Remove(stagingPath)
		}
	}()

	var encrypted EncryptedArtifactVerification
	switch mode {
	case EncryptionModeDevice:
		encrypted, err = EncryptWithX25519(ctx, staging, request.Artifact, request.ArtifactSize, recipient, secret, request.Limits)
	case EncryptionModePassphrase:
		encrypted, err = EncryptWithPassphrase(ctx, staging, request.Artifact, request.ArtifactSize, secret, request.Limits)
	default:
		err = verificationError(ErrorInputInvalid)
	}
	if err != nil {
		return PublishedArtifactMetadata{}, err
	}
	if err := staging.Sync(); err != nil {
		return PublishedArtifactMetadata{}, verificationError(ErrorStagingFailed)
	}
	stagingInfo, err := staging.Stat()
	if err != nil || !stagingInfo.Mode().IsRegular() || stagingInfo.Size() != encrypted.EncryptedSizeBytes {
		return PublishedArtifactMetadata{}, verificationError(ErrorStagingFailed)
	}
	if err := staging.Close(); err != nil {
		return PublishedArtifactMetadata{}, verificationError(ErrorStagingFailed)
	}
	stagingClosed = true

	// Reopen only after a durable close, then authenticate and verify the exact
	// persisted bytes once more before publication.
	persisted, err := openRegularNoFollow(stagingPath)
	if err != nil {
		return PublishedArtifactMetadata{}, verificationError(ErrorStagingFailed)
	}
	verified, verifyErr := verifyPersistedCiphertext(ctx, persisted, encrypted.EncryptedSizeBytes, mode, recipient, secret, request.Limits)
	checksum, hashErr := hashExact(ctx, persisted, encrypted.EncryptedSizeBytes)
	closeErr := persisted.Close()
	if verifyErr != nil {
		return PublishedArtifactMetadata{}, verifyErr
	}
	if hashErr != nil || closeErr != nil || !sameEncryptedVerification(encrypted, verified) {
		return PublishedArtifactMetadata{}, verificationError(ErrorStagingFailed)
	}

	if ops.beforePublish != nil {
		if err := ops.beforePublish(); err != nil {
			return PublishedArtifactMetadata{}, verificationError(ErrorPublicationFailed)
		}
	}
	if err := revalidateDestination(destination, parentSnapshot); err != nil {
		return PublishedArtifactMetadata{}, err
	}
	if err := ops.syncDirectory(filepath.Dir(destination)); err != nil {
		return PublishedArtifactMetadata{}, verificationError(ErrorPublicationFailed)
	}
	if err := contextError(ctx); err != nil {
		return PublishedArtifactMetadata{}, err
	}
	if err := ops.publish(stagingPath, destination); err != nil {
		if publishedFileMatches(destination, stagingInfo) {
			published = true
		} else if pathExists(destination) {
			return PublishedArtifactMetadata{}, verificationError(ErrorDestinationConflict)
		} else {
			return PublishedArtifactMetadata{}, verificationError(ErrorPublicationFailed)
		}
	} else {
		published = true
	}
	publishedMetadata := PublishedArtifactMetadata{
		State:              PublicationReconcileRequired,
		EncryptedSizeBytes: encrypted.EncryptedSizeBytes,
		ChecksumSHA256:     checksum,
		EncryptionMode:     mode,
		Artifact:           encrypted.Artifact,
	}
	if err := ops.syncDirectory(filepath.Dir(destination)); err != nil {
		return publishedMetadata, verificationError(ErrorPublicationReconcile)
	}

	finalFile, err := openRegularNoFollow(destination)
	if err != nil {
		return publishedMetadata, verificationError(ErrorPublicationReconcile)
	}
	finalInfo, statErr := finalFile.Stat()
	finalChecksum, finalHashErr := hashExact(ctx, finalFile, encrypted.EncryptedSizeBytes)
	finalCloseErr := finalFile.Close()
	if statErr != nil || finalInfo.Size() != encrypted.EncryptedSizeBytes || !os.SameFile(stagingInfo, finalInfo) ||
		finalHashErr != nil || finalCloseErr != nil || finalChecksum != checksum {
		return publishedMetadata, verificationError(ErrorPublicationReconcile)
	}

	publishedMetadata.State = PublicationVerified
	return publishedMetadata, nil
}

func publishedFileMatches(path string, stagingInfo os.FileInfo) bool {
	file, err := openRegularNoFollow(path)
	if err != nil {
		return false
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	return statErr == nil && closeErr == nil && stagingInfo != nil && os.SameFile(stagingInfo, info)
}

func verifyPersistedCiphertext(ctx context.Context, file io.ReaderAt, size int64, mode EncryptionMode, recipient string, secret []byte, limits Limits) (EncryptedArtifactVerification, error) {
	switch mode {
	case EncryptionModeDevice:
		// EncryptWithX25519 already proves recipient/identity equality. The
		// persisted re-read needs only the matching identity.
		_ = recipient
		return DecryptAndVerifyWithX25519(ctx, file, size, secret, limits)
	case EncryptionModePassphrase:
		return DecryptAndVerifyWithPassphrase(ctx, file, size, secret, limits)
	default:
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
}

func sameEncryptedVerification(left, right EncryptedArtifactVerification) bool {
	leftMetadata := left.Artifact.Metadata
	rightMetadata := right.Artifact.Metadata
	if left.EncryptedSizeBytes != right.EncryptedSizeBytes || left.PlaintextSizeBytes != right.PlaintextSizeBytes ||
		left.Artifact.State != right.Artifact.State || left.Artifact.ErrorCode != right.Artifact.ErrorCode ||
		left.Artifact.EncryptionQualified != right.Artifact.EncryptionQualified ||
		left.Artifact.RestoreMutationQualified != right.Artifact.RestoreMutationQualified ||
		leftMetadata.ArtifactSizeBytes != rightMetadata.ArtifactSizeBytes ||
		leftMetadata.FormatVersion != rightMetadata.FormatVersion ||
		leftMetadata.SchemaVersion != rightMetadata.SchemaVersion ||
		leftMetadata.PolicyVersion != rightMetadata.PolicyVersion ||
		leftMetadata.BackupID != rightMetadata.BackupID ||
		leftMetadata.CreatedAt != rightMetadata.CreatedAt ||
		leftMetadata.Scope != rightMetadata.Scope ||
		leftMetadata.RuntimeKind != rightMetadata.RuntimeKind ||
		leftMetadata.RuntimeVersion != rightMetadata.RuntimeVersion ||
		leftMetadata.Installation != rightMetadata.Installation ||
		leftMetadata.InclusionPolicy != rightMetadata.InclusionPolicy ||
		!slices.Equal(leftMetadata.IncludedCategories, rightMetadata.IncludedCategories) ||
		leftMetadata.ExclusionPolicy != rightMetadata.ExclusionPolicy ||
		!slices.Equal(leftMetadata.ExcludedCategories, rightMetadata.ExcludedCategories) ||
		leftMetadata.PayloadSizeBytes != rightMetadata.PayloadSizeBytes ||
		leftMetadata.PayloadSHA256 != rightMetadata.PayloadSHA256 ||
		leftMetadata.PayloadMemberCount != rightMetadata.PayloadMemberCount ||
		leftMetadata.TotalExpandedBytes != rightMetadata.TotalExpandedBytes {
		return false
	}
	return true
}

func hashExact(ctx context.Context, reader io.ReaderAt, size int64) (string, error) {
	if size <= 0 {
		return "", verificationError(ErrorPublicationFailed)
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.NewSectionReader(contextReaderAt{ctx: ctx, reader: reader}, 0, size))
	if err != nil || written != size {
		if err := contextError(ctx); err != nil {
			return "", err
		}
		return "", verificationError(ErrorPublicationFailed)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func publicationSpaceRequirement(plaintextSize int64) (uint64, bool) {
	if plaintextSize <= 0 || plaintextSize > hardMaxContainerBytes {
		return 0, false
	}
	required := plaintextSize + maxEncryptedOverheadBytes + minimumSpaceReserve
	if required < plaintextSize {
		return 0, false
	}
	return uint64(required), true
}

func allocateStagingPath(parent string) (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		random := make([]byte, stagingNameRandomBytes)
		if _, err := rand.Read(random); err != nil {
			return "", err
		}
		candidate := filepath.Join(parent, stagingFilePrefix+hex.EncodeToString(random)+".tmp")
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New("backup staging name unavailable")
}

func normalizeDestination(path string) (string, error) {
	if path == "" || len(path) > maxDestinationBytes || !utf8.ValidString(path) || strings.IndexByte(path, 0) >= 0 ||
		strings.TrimSpace(path) != path || !filepath.IsAbs(path) {
		return "", verificationError(ErrorDestinationInvalid)
	}
	clean := filepath.Clean(path)
	if clean != path || !strings.HasSuffix(filepath.Base(clean), backupFileSuffix) || strings.Contains(filepath.Base(clean), ":") {
		return "", verificationError(ErrorDestinationInvalid)
	}
	if err := validateLocalDestinationPath(clean); err != nil {
		return "", err
	}
	return clean, nil
}

func inspectDestination(path string) (os.FileInfo, error) {
	if err := validateLocalDestinationPath(path); err != nil {
		return nil, err
	}
	parent := filepath.Dir(path)
	directory, err := openDirectoryNoFollow(parent)
	if err != nil {
		return nil, verificationError(ErrorDestinationUnsafe)
	}
	info, statErr := directory.Stat()
	closeErr := directory.Close()
	if statErr != nil || closeErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, verificationError(ErrorDestinationUnsafe)
	}
	if _, err := os.Lstat(path); err == nil {
		return nil, verificationError(ErrorDestinationConflict)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, verificationError(ErrorDestinationUnsafe)
	}
	return info, nil
}

func revalidateDestination(path string, parentSnapshot os.FileInfo) error {
	current, err := inspectDestination(path)
	if err != nil {
		return err
	}
	if parentSnapshot == nil || !os.SameFile(parentSnapshot, current) {
		return verificationError(ErrorDestinationUnsafe)
	}
	return nil
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func bytesClone(value []byte) []byte {
	result := make([]byte, len(value))
	copy(result, value)
	return result
}
