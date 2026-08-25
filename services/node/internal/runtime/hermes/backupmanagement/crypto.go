package backupmanagement

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"filippo.io/age"
)

const (
	maxRecipientBytes          = 256
	maxIdentityBytes           = 256
	maxPassphraseBytes         = 1024
	maxEncryptedOverheadBytes  = 8 << 20
	maximumScryptWorkFactorLog = 18
	maxAgeHeaderBytes          = 64 << 10
	maxAgeHeaderLineBytes      = 4 << 10
	maxAgeHeaderLines          = 128
	maxAgeRecipients           = 1
	minAgeBodyBytes            = 32
	ageX25519Stanza            = "X25519"
	ageScryptStanza            = "scrypt"
)

// CiphertextStaging is caller-owned private staging for a backup ciphertext.
// It must not contain useful data when passed in. Callers remain responsible
// for choosing and protecting the backing store and for discarding it after a
// failed call.
type CiphertextStaging interface {
	io.ReaderAt
	io.WriterAt
	Truncate(size int64) error
}

// EncryptedArtifactVerification is returned only after the final ciphertext
// has been decrypted, fully authenticated and structurally verified.
type EncryptedArtifactVerification struct {
	EncryptedSizeBytes int64
	PlaintextSizeBytes int64
	Artifact           VerificationResult
}

// GenerateX25519Identity returns an age device recipient and its matching
// identity as UTF-8 bytes. The identity must be treated as secret in memory.
func GenerateX25519Identity() (recipient string, identity []byte, err error) {
	parsedIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", nil, verificationError(ErrorEncryptionFailed)
	}
	return parsedIdentity.Recipient().String(), []byte(parsedIdentity.String()), nil
}

// EncryptWithX25519 writes a single-recipient age v1 ciphertext to staging.
// The matching identity is required so the exact staged ciphertext can be
// authenticated and verified before success is reported.
func EncryptWithX25519(
	ctx context.Context,
	dst CiphertextStaging,
	artifact io.ReaderAt,
	artifactSize int64,
	recipient string,
	identity []byte,
	limits Limits,
) (EncryptedArtifactVerification, error) {
	if len(recipient) == 0 || len(recipient) > maxRecipientBytes || len(identity) == 0 || len(identity) > maxIdentityBytes {
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
	parsedRecipient, err := age.ParseX25519Recipient(recipient)
	if err != nil || parsedRecipient.String() != recipient {
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
	parsedIdentity, err := age.ParseX25519Identity(string(identity))
	if err != nil || parsedIdentity.String() != string(identity) {
		return EncryptedArtifactVerification{}, verificationError(ErrorCredentialInvalid)
	}
	if parsedIdentity.Recipient().String() != parsedRecipient.String() {
		return EncryptedArtifactVerification{}, verificationError(ErrorCredentialInvalid)
	}
	return encryptArtifact(ctx, dst, artifact, artifactSize, parsedRecipient, parsedIdentity, ageX25519Stanza, limits)
}

// EncryptWithPassphrase writes an age v1 scrypt ciphertext to staging. The
// passphrase is consumed in memory and is never persisted by this package.
func EncryptWithPassphrase(
	ctx context.Context,
	dst CiphertextStaging,
	artifact io.ReaderAt,
	artifactSize int64,
	passphrase []byte,
	limits Limits,
) (EncryptedArtifactVerification, error) {
	if len(passphrase) == 0 || len(passphrase) > maxPassphraseBytes {
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
	recipient, err := age.NewScryptRecipient(string(passphrase))
	if err != nil {
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
	identity.SetMaxWorkFactor(maximumScryptWorkFactorLog)
	return encryptArtifact(ctx, dst, artifact, artifactSize, recipient, identity, ageScryptStanza, limits)
}

func encryptArtifact(
	ctx context.Context,
	dst CiphertextStaging,
	artifact io.ReaderAt,
	artifactSize int64,
	recipient age.Recipient,
	identity age.Identity,
	expectedStanza string,
	limits Limits,
) (result EncryptedArtifactVerification, err error) {
	if ctx == nil || dst == nil || artifact == nil || recipient == nil || identity == nil || artifactSize < 0 || !limits.valid() {
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
	if artifactSize > limits.MaxContainerBytes {
		return EncryptedArtifactVerification{}, verificationError(ErrorArchiveTotalSizeLimit)
	}
	if err := contextError(ctx); err != nil {
		return EncryptedArtifactVerification{}, err
	}
	if err := dst.Truncate(0); err != nil {
		return EncryptedArtifactVerification{}, verificationError(ErrorEncryptionFailed)
	}

	keepStaging := false
	defer func() {
		if !keepStaging {
			_ = dst.Truncate(0)
		}
	}()

	positioned := &offsetWriter{writer: dst}
	encryptedWriter, err := age.Encrypt(contextWriter{ctx: ctx, writer: positioned}, recipient)
	if err != nil {
		return EncryptedArtifactVerification{}, mapEncryptError(ctx)
	}
	plaintext := io.NewSectionReader(contextReaderAt{ctx: ctx, reader: artifact}, 0, artifactSize)
	written, copyErr := io.Copy(encryptedWriter, plaintext)
	closeErr := encryptedWriter.Close()
	if copyErr != nil || written != artifactSize || closeErr != nil {
		return EncryptedArtifactVerification{}, mapEncryptError(ctx)
	}
	if err := contextError(ctx); err != nil {
		return EncryptedArtifactVerification{}, err
	}
	if positioned.offset <= 0 {
		return EncryptedArtifactVerification{}, verificationError(ErrorEncryptionFailed)
	}

	result, err = decryptAndVerify(ctx, dst, positioned.offset, identity, expectedStanza, limits)
	if err != nil {
		return EncryptedArtifactVerification{}, err
	}
	keepStaging = true
	return result, nil
}

// DecryptAndVerifyWithX25519 authenticates every ciphertext chunk before it
// returns metadata. Plaintext is never returned by this boundary.
func DecryptAndVerifyWithX25519(
	ctx context.Context,
	encrypted io.ReaderAt,
	encryptedSize int64,
	identity []byte,
	limits Limits,
) (EncryptedArtifactVerification, error) {
	if len(identity) == 0 || len(identity) > maxIdentityBytes {
		return EncryptedArtifactVerification{}, verificationError(ErrorCredentialInvalid)
	}
	parsedIdentity, err := age.ParseX25519Identity(string(identity))
	if err != nil || parsedIdentity.String() != string(identity) {
		return EncryptedArtifactVerification{}, verificationError(ErrorCredentialInvalid)
	}
	return decryptAndVerify(ctx, encrypted, encryptedSize, parsedIdentity, ageX25519Stanza, limits)
}

// DecryptAndVerifyWithPassphrase authenticates every ciphertext chunk before
// it returns metadata. The passphrase is consumed in memory only.
func DecryptAndVerifyWithPassphrase(
	ctx context.Context,
	encrypted io.ReaderAt,
	encryptedSize int64,
	passphrase []byte,
	limits Limits,
) (EncryptedArtifactVerification, error) {
	if len(passphrase) == 0 || len(passphrase) > maxPassphraseBytes {
		return EncryptedArtifactVerification{}, verificationError(ErrorCredentialInvalid)
	}
	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return EncryptedArtifactVerification{}, verificationError(ErrorCredentialInvalid)
	}
	identity.SetMaxWorkFactor(maximumScryptWorkFactorLog)
	return decryptAndVerify(ctx, encrypted, encryptedSize, identity, ageScryptStanza, limits)
}

func decryptAndVerify(
	ctx context.Context,
	encrypted io.ReaderAt,
	encryptedSize int64,
	identity age.Identity,
	expectedStanza string,
	limits Limits,
) (EncryptedArtifactVerification, error) {
	if ctx == nil || encrypted == nil || identity == nil || encryptedSize <= 0 || !limits.valid() {
		return EncryptedArtifactVerification{}, verificationError(ErrorInputInvalid)
	}
	if encryptedSize > limits.MaxContainerBytes+maxEncryptedOverheadBytes {
		return EncryptedArtifactVerification{}, verificationError(ErrorEncryptedArtifactLimit)
	}
	if err := contextError(ctx); err != nil {
		return EncryptedArtifactVerification{}, err
	}

	frozen, err := snapshotAgeHeader(ctx, encrypted, encryptedSize, expectedStanza)
	if err != nil {
		return EncryptedArtifactVerification{}, err
	}
	plaintext, plaintextSize, err := age.DecryptReaderAt(frozen, encryptedSize, identity)
	if err != nil {
		if err := contextError(ctx); err != nil {
			return EncryptedArtifactVerification{}, err
		}
		var noMatch *age.NoIdentityMatchError
		if errors.As(err, &noMatch) {
			return EncryptedArtifactVerification{}, verificationError(ErrorCredentialInvalid)
		}
		return EncryptedArtifactVerification{}, verificationError(ErrorDecryptionFailed)
	}
	if plaintextSize < 0 || plaintextSize > limits.MaxContainerBytes {
		return EncryptedArtifactVerification{}, verificationError(ErrorEncryptedArtifactLimit)
	}
	recordedPlaintext := &recordingReaderAt{reader: plaintext}
	if err := scanAuthenticatedPlaintext(ctx, recordedPlaintext, plaintextSize); err != nil {
		return EncryptedArtifactVerification{}, err
	}
	verified, err := VerifyArtifact(contextReaderAt{ctx: ctx, reader: recordedPlaintext}, plaintextSize, limits)
	if recordedPlaintext.recordedError() != nil {
		return EncryptedArtifactVerification{}, verificationError(ErrorDecryptionFailed)
	}
	if err != nil {
		return EncryptedArtifactVerification{}, err
	}
	return EncryptedArtifactVerification{
		EncryptedSizeBytes: encryptedSize,
		PlaintextSizeBytes: plaintextSize,
		Artifact:           verified,
	}, nil
}

// snapshotAgeHeader applies independent, strict resource limits before age is
// allowed to parse the file. The returned ReaderAt serves the exact header
// bytes that were checked, preventing a mutable backing store from changing
// the header between preflight and authentication.
func snapshotAgeHeader(ctx context.Context, encrypted io.ReaderAt, encryptedSize int64, expectedStanza string) (io.ReaderAt, error) {
	reader := bufio.NewReaderSize(
		io.NewSectionReader(contextReaderAt{ctx: ctx, reader: encrypted}, 0, encryptedSize),
		maxAgeHeaderLineBytes+1,
	)
	header := make([]byte, 0, 512)
	recipientCount := 0
	lineCount := 0
	sawStanza := false

	for {
		line, err := reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			return nil, verificationError(ErrorEncryptedArtifactLimit)
		}
		if err != nil {
			if err := contextError(ctx); err != nil {
				return nil, err
			}
			return nil, verificationError(ErrorDecryptionFailed)
		}
		lineCount++
		if lineCount > maxAgeHeaderLines || len(line) > maxAgeHeaderLineBytes || len(header)+len(line) > maxAgeHeaderBytes {
			return nil, verificationError(ErrorEncryptedArtifactLimit)
		}
		if bytes.IndexByte(line, '\r') >= 0 {
			return nil, verificationError(ErrorDecryptionFailed)
		}
		header = append(header, line...)

		if lineCount == 1 {
			if !bytes.Equal(line, []byte("age-encryption.org/v1\n")) {
				return nil, verificationError(ErrorDecryptionFailed)
			}
			continue
		}
		if bytes.HasPrefix(line, []byte("-> ")) {
			recipientCount++
			if recipientCount > maxAgeRecipients {
				return nil, verificationError(ErrorEncryptedArtifactLimit)
			}
			fields := bytes.Fields(bytes.TrimSuffix(line, []byte{'\n'}))
			if len(fields) < 2 || string(fields[1]) != expectedStanza {
				return nil, verificationError(ErrorDecryptionFailed)
			}
			sawStanza = true
			continue
		}
		if bytes.HasPrefix(line, []byte("--- ")) {
			if recipientCount != 1 || !sawStanza || encryptedSize-int64(len(header)) < minAgeBodyBytes {
				return nil, verificationError(ErrorDecryptionFailed)
			}
			return &frozenHeaderReaderAt{
				header:     bytes.Clone(header),
				source:     encrypted,
				bodyOffset: int64(len(header)),
				size:       encryptedSize,
			}, nil
		}
		if !sawStanza || len(line) == 1 {
			return nil, verificationError(ErrorDecryptionFailed)
		}
	}
}

func scanAuthenticatedPlaintext(ctx context.Context, plaintext io.ReaderAt, plaintextSize int64) error {
	reader := io.NewSectionReader(contextReaderAt{ctx: ctx, reader: plaintext}, 0, plaintextSize)
	written, err := io.Copy(io.Discard, reader)
	if err != nil {
		if err := contextError(ctx); err != nil {
			return err
		}
		return verificationError(ErrorDecryptionFailed)
	}
	if written != plaintextSize {
		return verificationError(ErrorDecryptionFailed)
	}
	return nil
}

type offsetWriter struct {
	writer io.WriterAt
	offset int64
}

func (w *offsetWriter) Write(p []byte) (int, error) {
	n, err := w.writer.WriteAt(p, w.offset)
	w.offset += int64(n)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

type frozenHeaderReaderAt struct {
	header     []byte
	source     io.ReaderAt
	bodyOffset int64
	size       int64
}

func (r *frozenHeaderReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("negative offset")
	}
	if len(p) == 0 {
		return 0, nil
	}
	if off >= r.size {
		return 0, io.EOF
	}
	want := len(p)
	available := r.size - off
	if int64(len(p)) > available {
		p = p[:available]
	}

	total := 0
	if off < r.bodyOffset {
		n := copy(p, r.header[off:])
		total += n
		off += int64(n)
		p = p[n:]
	}
	if len(p) > 0 && off < r.size {
		n, err := r.source.ReadAt(p, off)
		total += n
		if err != nil && !(err == io.EOF && n == len(p)) {
			return total, err
		}
		if n != len(p) {
			return total, io.EOF
		}
	}
	if total < want {
		return total, io.EOF
	}
	return total, nil
}

type contextReaderAt struct {
	ctx    context.Context
	reader io.ReaderAt
}

func (r contextReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.ReadAt(p, off)
}

type contextWriter struct {
	ctx    context.Context
	writer io.Writer
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.writer.Write(p)
}

type recordingReaderAt struct {
	mu     sync.Mutex
	reader io.ReaderAt
	err    error
}

func (r *recordingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.reader.ReadAt(p, off)
	if err == io.EOF && n == len(p) {
		err = nil
	}
	if err != nil && err != io.EOF {
		r.mu.Lock()
		if r.err == nil {
			r.err = err
		}
		r.mu.Unlock()
	}
	return n, err
}

func (r *recordingReaderAt) recordedError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func mapEncryptError(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	return verificationError(ErrorEncryptionFailed)
}

func contextError(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return verificationError(ErrorOperationCanceled)
	}
	return nil
}
