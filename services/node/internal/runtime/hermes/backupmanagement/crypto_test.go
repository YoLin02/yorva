package backupmanagement

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"strings"
	"sync"
	"testing"

	"filippo.io/age"
)

const (
	testAgeChunkSize = 64 << 10
	testAgeTagSize   = 16
	testAgeNonceSize = 16
)

func TestAgeX25519RoundTrip(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	recipient, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity() error = %v", err)
	}

	encrypted := &memoryCiphertextStaging{}
	created, err := EncryptWithX25519(context.Background(), encrypted, bytes.NewReader(artifact), int64(len(artifact)), recipient, identity, DefaultLimits())
	if err != nil {
		t.Fatalf("EncryptWithX25519() error = %v", err)
	}
	if created.Artifact.State != ArtifactStructureVerified || created.EncryptedSizeBytes != int64(encrypted.Len()) || bytes.Contains(encrypted.Bytes(), artifact) {
		t.Fatalf("creation result = %#v, encrypted artifact exposed plaintext", created)
	}

	verified, err := DecryptAndVerifyWithX25519(context.Background(), bytes.NewReader(encrypted.Bytes()), int64(encrypted.Len()), identity, DefaultLimits())
	if err != nil {
		t.Fatalf("DecryptAndVerifyWithX25519() error = %v", err)
	}
	if verified.EncryptedSizeBytes != int64(encrypted.Len()) || verified.PlaintextSizeBytes != int64(len(artifact)) ||
		verified.Artifact.State != ArtifactStructureVerified || verified.Artifact.Metadata.RuntimeVersion != "0.20.5" {
		t.Fatalf("verification = %#v", verified)
	}
}

func TestAgePassphraseRoundTrip(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	passphrase := []byte("portable secret phrase")

	encrypted := &memoryCiphertextStaging{}
	if _, err := EncryptWithPassphrase(context.Background(), encrypted, bytes.NewReader(artifact), int64(len(artifact)), passphrase, DefaultLimits()); err != nil {
		t.Fatalf("EncryptWithPassphrase() error = %v", err)
	}
	verified, err := DecryptAndVerifyWithPassphrase(context.Background(), bytes.NewReader(encrypted.Bytes()), int64(encrypted.Len()), passphrase, DefaultLimits())
	if err != nil {
		t.Fatalf("DecryptAndVerifyWithPassphrase() error = %v", err)
	}
	if verified.Artifact.State != ArtifactStructureVerified || verified.PlaintextSizeBytes != int64(len(artifact)) {
		t.Fatalf("verification = %#v", verified)
	}
}

func TestAgeRejectsWrongIdentityAndPassphraseWithoutResult(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	recipient, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	_, wrongIdentity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	x25519Encrypted := &memoryCiphertextStaging{}
	if _, err := EncryptWithX25519(context.Background(), x25519Encrypted, bytes.NewReader(artifact), int64(len(artifact)), recipient, identity, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	result, err := DecryptAndVerifyWithX25519(context.Background(), bytes.NewReader(x25519Encrypted.Bytes()), int64(x25519Encrypted.Len()), wrongIdentity, DefaultLimits())
	assertEmptyCryptoFailure(t, result, err, ErrorCredentialInvalid)

	passphraseEncrypted := &memoryCiphertextStaging{}
	if _, err := EncryptWithPassphrase(context.Background(), passphraseEncrypted, bytes.NewReader(artifact), int64(len(artifact)), []byte("right passphrase"), DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	result, err = DecryptAndVerifyWithPassphrase(context.Background(), bytes.NewReader(passphraseEncrypted.Bytes()), int64(passphraseEncrypted.Len()), []byte("wrong passphrase"), DefaultLimits())
	assertEmptyCryptoFailure(t, result, err, ErrorCredentialInvalid)
}

func TestAgeRejectsTamperAndTruncationWithoutResult(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	recipient, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	encrypted := &memoryCiphertextStaging{}
	if _, err := EncryptWithX25519(context.Background(), encrypted, bytes.NewReader(artifact), int64(len(artifact)), recipient, identity, DefaultLimits()); err != nil {
		t.Fatal(err)
	}

	tampered := append([]byte(nil), encrypted.Bytes()...)
	tampered[len(tampered)-24] ^= 0x80
	result, err := DecryptAndVerifyWithX25519(context.Background(), bytes.NewReader(tampered), int64(len(tampered)), identity, DefaultLimits())
	assertEmptyCryptoFailure(t, result, err, ErrorDecryptionFailed)

	truncated := encrypted.Bytes()[:encrypted.Len()-1]
	result, err = DecryptAndVerifyWithX25519(context.Background(), bytes.NewReader(truncated), int64(len(truncated)), identity, DefaultLimits())
	assertEmptyCryptoFailure(t, result, err, ErrorDecryptionFailed)

	largeArtifact := buildTestArtifact(t, []testZipMember{{
		name: PayloadArchiveRoot + "/large.db", body: bytes.Repeat([]byte("authenticated chunk data"), 12_000), method: zip.Store, mode: 0o600,
	}}, "")
	largeEncrypted := &memoryCiphertextStaging{}
	if _, err := EncryptWithX25519(context.Background(), largeEncrypted, bytes.NewReader(largeArtifact), int64(len(largeArtifact)), recipient, identity, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	middleTamper := append([]byte(nil), largeEncrypted.Bytes()...)
	middleTamper[len(middleTamper)/2] ^= 0x40
	result, err = DecryptAndVerifyWithX25519(context.Background(), bytes.NewReader(middleTamper), int64(len(middleTamper)), identity, DefaultLimits())
	assertEmptyCryptoFailure(t, result, err, ErrorDecryptionFailed)
}

func TestAgeRejectsTamperInOuterZIPGapNotVisitedByStructureParser(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	gappedArtifact, gapStart := addOuterZIPGap(t, artifact, 3*testAgeChunkSize)
	if _, err := VerifyArtifactBytes(gappedArtifact, DefaultLimits()); err != nil {
		t.Fatalf("gapped fixture must remain a structurally valid ZIP: %v", err)
	}

	recipient, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	encrypted := &memoryCiphertextStaging{}
	if _, err := EncryptWithX25519(context.Background(), encrypted, bytes.NewReader(gappedArtifact), int64(len(gappedArtifact)), recipient, identity, DefaultLimits()); err != nil {
		t.Fatal(err)
	}

	header, err := age.ExtractHeader(bytes.NewReader(encrypted.Bytes()))
	if err != nil {
		t.Fatalf("ExtractHeader() error = %v", err)
	}
	gapEnd := gapStart + 3*testAgeChunkSize
	chunkIndex := (gapStart + testAgeChunkSize - 1) / testAgeChunkSize
	chunkStart := chunkIndex * testAgeChunkSize
	if chunkStart+testAgeChunkSize > gapEnd {
		t.Fatal("fixture has no complete age chunk inside the ignored ZIP gap")
	}

	tampered := append([]byte(nil), encrypted.Bytes()...)
	encryptedChunkStart := len(header) + testAgeNonceSize + chunkIndex*(testAgeChunkSize+testAgeTagSize)
	tampered[encryptedChunkStart+32] ^= 0x20
	parsedIdentity, err := age.ParseX25519Identity(string(identity))
	if err != nil {
		t.Fatal(err)
	}
	partiallyAuthenticated, plaintextSize, err := age.DecryptReaderAt(bytes.NewReader(tampered), int64(len(tampered)), parsedIdentity)
	if err != nil {
		t.Fatalf("tamper must avoid the final chunk: %v", err)
	}
	if _, err := VerifyArtifact(contextReaderAt{ctx: context.Background(), reader: &recordingReaderAt{reader: partiallyAuthenticated}}, plaintextSize, DefaultLimits()); err != nil {
		t.Fatalf("fixture must prove structural parsing alone misses the gap chunk: %v", err)
	}

	result, err := DecryptAndVerifyWithX25519(context.Background(), bytes.NewReader(tampered), int64(len(tampered)), identity, DefaultLimits())
	assertEmptyCryptoFailure(t, result, err, ErrorDecryptionFailed)
}

func TestAgeCancellationAndLimitsFailClosed(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	recipient, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	output := &memoryCiphertextStaging{}
	created, err := EncryptWithX25519(canceled, output, bytes.NewReader(artifact), int64(len(artifact)), recipient, identity, DefaultLimits())
	assertEmptyCryptoFailure(t, created, err, ErrorOperationCanceled)
	if output.Len() != 0 {
		t.Fatalf("canceled creation = %#v, output bytes = %d", created, output.Len())
	}

	encrypted := &memoryCiphertextStaging{}
	if _, err := EncryptWithX25519(context.Background(), encrypted, bytes.NewReader(artifact), int64(len(artifact)), recipient, identity, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	result, err := DecryptAndVerifyWithX25519(canceled, bytes.NewReader(encrypted.Bytes()), int64(encrypted.Len()), identity, DefaultLimits())
	assertEmptyCryptoFailure(t, result, err, ErrorOperationCanceled)

	limits := DefaultLimits()
	limits.MaxContainerBytes = int64(len(artifact) - 1)
	created, err = EncryptWithX25519(context.Background(), output, bytes.NewReader(artifact), int64(len(artifact)), recipient, identity, limits)
	assertEmptyCryptoFailure(t, created, err, ErrorArchiveTotalSizeLimit)
}

func TestAgeRejectsUnboundedOrMalformedCredentialInput(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	output := &memoryCiphertextStaging{}
	_, identity, identityErr := GenerateX25519Identity()
	if identityErr != nil {
		t.Fatal(identityErr)
	}
	_, err := EncryptWithX25519(context.Background(), output, bytes.NewReader(artifact), int64(len(artifact)), "not-an-age-recipient", identity, DefaultLimits())
	assertErrorCode(t, err, ErrorInputInvalid)

	_, err = EncryptWithPassphrase(context.Background(), output, bytes.NewReader(artifact), int64(len(artifact)), []byte(strings.Repeat("x", maxPassphraseBytes+1)), DefaultLimits())
	assertErrorCode(t, err, ErrorInputInvalid)
}

func TestAgeCreateFailsClosedWhenFinalCiphertextChangesBeforeVerification(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	recipient, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	staging := &memoryCiphertextStaging{tamperOnFirstRead: true}

	result, err := EncryptWithX25519(
		context.Background(), staging, bytes.NewReader(artifact), int64(len(artifact)), recipient, identity, DefaultLimits(),
	)
	assertEmptyCryptoFailure(t, result, err, ErrorDecryptionFailed)
	if staging.Len() != 0 {
		t.Fatalf("failed final ciphertext verification left %d staged bytes", staging.Len())
	}
}

func TestAgeHeaderPreflightRejectsResourceAbuse(t *testing.T) {
	_, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "unbounded first line",
			data: bytes.Repeat([]byte("x"), maxAgeHeaderLineBytes+2),
		},
		{
			name: "too many recipients",
			data: []byte("age-encryption.org/v1\n-> X25519 first\na\n-> X25519 second\nb\n--- mac\n" + strings.Repeat("x", minAgeBodyBytes)),
		},
		{
			name: "oversized header",
			data: []byte("age-encryption.org/v1\n-> X25519 first\n" + strings.Repeat(strings.Repeat("a", 1023)+"\n", maxAgeHeaderBytes/1024+1) + "--- mac\n" + strings.Repeat("x", minAgeBodyBytes)),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := DecryptAndVerifyWithX25519(context.Background(), bytes.NewReader(test.data), int64(len(test.data)), identity, DefaultLimits())
			assertEmptyCryptoFailure(t, result, err, ErrorEncryptedArtifactLimit)
		})
	}
}

func TestAgeDeclaredCiphertextLimitRejectsBeforeRead(t *testing.T) {
	_, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	reader := &countingReaderAt{}
	limits := DefaultLimits()
	result, err := DecryptAndVerifyWithX25519(
		context.Background(), reader, limits.MaxContainerBytes+maxEncryptedOverheadBytes+1, identity, limits,
	)
	assertEmptyCryptoFailure(t, result, err, ErrorEncryptedArtifactLimit)
	if reader.ReadCount() != 0 {
		t.Fatalf("oversized declared ciphertext caused %d reads", reader.ReadCount())
	}
}

func TestAgeHeaderPreflightFreezesBytesForAgeParser(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	recipient, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	staging := &memoryCiphertextStaging{}
	if _, err := EncryptWithX25519(context.Background(), staging, bytes.NewReader(artifact), int64(len(artifact)), recipient, identity, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	header, err := age.ExtractHeader(bytes.NewReader(staging.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	mutable := &changingHeaderReaderAt{data: staging.Bytes(), headerSize: int64(len(header))}

	result, err := DecryptAndVerifyWithX25519(context.Background(), mutable, int64(staging.Len()), identity, DefaultLimits())
	if err != nil {
		t.Fatalf("DecryptAndVerifyWithX25519() error = %v", err)
	}
	if result.Artifact.State != ArtifactStructureVerified {
		t.Fatalf("verification = %#v", result)
	}
	if mutable.HeaderReadCount() != 1 {
		t.Fatalf("mutable source header reads = %d, want 1 frozen snapshot read", mutable.HeaderReadCount())
	}
}

func assertEmptyCryptoFailure(t *testing.T, result EncryptedArtifactVerification, err error, code ErrorCode) {
	t.Helper()
	assertErrorCode(t, err, code)
	if result.EncryptedSizeBytes != 0 || result.PlaintextSizeBytes != 0 || result.Artifact.State != "" ||
		result.Artifact.ErrorCode != "" || len(result.Artifact.Metadata.IncludedCategories) != 0 {
		t.Fatalf("failure returned verification metadata: %#v", result)
	}
}

type memoryCiphertextStaging struct {
	mu                sync.Mutex
	data              []byte
	tamperOnFirstRead bool
}

func (m *memoryCiphertextStaging) ReadAt(p []byte, off int64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if off < 0 {
		return 0, io.EOF
	}
	if m.tamperOnFirstRead {
		m.tamperOnFirstRead = false
		if len(m.data) >= 24 {
			m.data[len(m.data)-24] ^= 0x80
		}
	}
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n := copy(p, m.data[off:])
	if n != len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (m *memoryCiphertextStaging) WriteAt(p []byte, off int64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if off < 0 {
		return 0, io.ErrShortWrite
	}
	end := off + int64(len(p))
	if end > int64(len(m.data)) {
		m.data = append(m.data, make([]byte, end-int64(len(m.data)))...)
	}
	return copy(m.data[off:end], p), nil
}

func (m *memoryCiphertextStaging) Truncate(size int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if size < 0 {
		return io.ErrUnexpectedEOF
	}
	if size <= int64(len(m.data)) {
		m.data = m.data[:size]
		return nil
	}
	m.data = append(m.data, make([]byte, size-int64(len(m.data)))...)
	return nil
}

func (m *memoryCiphertextStaging) Bytes() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return bytes.Clone(m.data)
}

func (m *memoryCiphertextStaging) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.data)
}

type countingReaderAt struct {
	mu    sync.Mutex
	reads int
}

func (r *countingReaderAt) ReadAt([]byte, int64) (int, error) {
	r.mu.Lock()
	r.reads++
	r.mu.Unlock()
	return 0, io.EOF
}

func (r *countingReaderAt) ReadCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reads
}

type changingHeaderReaderAt struct {
	mu          sync.Mutex
	data        []byte
	headerSize  int64
	headerReads int
}

func (r *changingHeaderReaderAt) ReadAt(p []byte, off int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if off < r.headerSize {
		r.headerReads++
		if r.headerReads > 1 {
			for index := range p {
				p[index] = 'x'
			}
			return len(p), nil
		}
	}
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[off:])
	if n != len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (r *changingHeaderReaderAt) HeaderReadCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.headerReads
}

func addOuterZIPGap(t *testing.T, artifact []byte, gapSize int) ([]byte, int) {
	t.Helper()
	if len(artifact) < 22 || binary.LittleEndian.Uint32(artifact[len(artifact)-22:]) != 0x06054b50 {
		t.Fatal("fixture has no terminal ZIP end record")
	}
	eocdOffset := len(artifact) - 22
	centralDirectoryOffset := int(binary.LittleEndian.Uint32(artifact[eocdOffset+16:]))
	if centralDirectoryOffset <= 0 || centralDirectoryOffset >= eocdOffset {
		t.Fatal("fixture central-directory offset is invalid")
	}

	result := make([]byte, len(artifact)+gapSize)
	copy(result, artifact[:centralDirectoryOffset])
	for index := centralDirectoryOffset; index < centralDirectoryOffset+gapSize; index++ {
		result[index] = 0xa5
	}
	copy(result[centralDirectoryOffset+gapSize:], artifact[centralDirectoryOffset:])
	newEOCDOffset := eocdOffset + gapSize
	binary.LittleEndian.PutUint32(result[newEOCDOffset+16:], uint32(centralDirectoryOffset+gapSize))
	return result, centralDirectoryOffset
}
