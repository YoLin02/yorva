package backupmanagement

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishVerifiedArtifactAtomicallyPublishesFinalCiphertext(t *testing.T) {
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	destinationPath := filepath.Join(t.TempDir(), "runtime.yorva-backup.age")
	destination, err := InspectLocalDestination(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	recipient, identity, err := GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	credential, err := NewDeviceCreateCredential(recipient, identity)
	if err != nil {
		t.Fatal(err)
	}
	request := PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)),
		Credential: credential, Limits: DefaultLimits(),
	}
	observedBeforePublish := false
	ops := defaultPublicationOps()
	ops.freeBytes = func(string) (uint64, error) { return math.MaxUint64, nil }
	ops.beforePublish = func() error {
		observedBeforePublish = true
		if _, err := os.Lstat(destinationPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("destination became visible before atomic publish: %v", err)
		}
		matches, err := filepath.Glob(filepath.Join(filepath.Dir(destinationPath), stagingFilePrefix+"*"))
		if err != nil || len(matches) != 1 {
			t.Fatalf("private staging at publish boundary = %v, %v", matches, err)
		}
		return nil
	}

	result, err := publishVerifiedArtifact(context.Background(), request, ops)
	if err != nil {
		t.Fatalf("publishVerifiedArtifact() error = %v", err)
	}
	if !observedBeforePublish || result.State != PublicationVerified || result.Artifact.State != ArtifactStructureVerified || result.Artifact.EncryptionQualified ||
		result.EncryptionMode != EncryptionModeDevice || result.EncryptedSizeBytes <= 0 || len(result.ChecksumSHA256) != sha256.Size*2 {
		t.Fatalf("publication result = %#v", result)
	}
	published, err := os.ReadFile(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	wantChecksum := sha256.Sum256(published)
	if result.ChecksumSHA256 != hex.EncodeToString(wantChecksum[:]) || bytes.Contains(published, artifact) {
		t.Fatal("published checksum mismatch or ciphertext exposed plaintext")
	}
	verified, err := DecryptAndVerifyWithX25519(context.Background(), bytes.NewReader(published), int64(len(published)), identity, DefaultLimits())
	if err != nil || verified.Artifact.State != ArtifactStructureVerified {
		t.Fatalf("published ciphertext verification = %#v, %v", verified, err)
	}
	assertNoPublicationStaging(t, filepath.Dir(destinationPath))
	if !credential.consumed || len(credential.secret) != 0 {
		t.Fatal("request-scoped credential was not consumed and cleared")
	}
}

func TestPublishVerifiedArtifactRejectsCollisionWithoutOverwrite(t *testing.T) {
	artifact, destinationPath, destination := publicationFixture(t)
	original := []byte("existing user artifact")
	if err := os.WriteFile(destinationPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	credential := testPassphraseCredential(t)

	result, err := publishVerifiedArtifact(context.Background(), PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)), Credential: credential, Limits: DefaultLimits(),
	}, publicationOpsWithSpace())
	assertEmptyPublicationFailure(t, result, err, ErrorDestinationConflict)
	got, readErr := os.ReadFile(destinationPath)
	if readErr != nil || !bytes.Equal(got, original) {
		t.Fatalf("existing destination changed: %q, %v", got, readErr)
	}
	assertNoPublicationStaging(t, filepath.Dir(destinationPath))
}

func TestPublishVerifiedArtifactRejectsCollisionCreatedAtPublishBoundary(t *testing.T) {
	artifact, destinationPath, destination := publicationFixture(t)
	credential := testPassphraseCredential(t)
	ops := publicationOpsWithSpace()
	ops.beforePublish = func() error { return os.WriteFile(destinationPath, []byte("racer"), 0o600) }

	result, err := publishVerifiedArtifact(context.Background(), PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)), Credential: credential, Limits: DefaultLimits(),
	}, ops)
	assertEmptyPublicationFailure(t, result, err, ErrorDestinationConflict)
	got, readErr := os.ReadFile(destinationPath)
	if readErr != nil || string(got) != "racer" {
		t.Fatalf("racing destination changed: %q, %v", got, readErr)
	}
	assertNoPublicationStaging(t, filepath.Dir(destinationPath))
}

func TestPublishVerifiedArtifactRejectsChangedDestinationDirectory(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "selected")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	destinationPath := filepath.Join(parent, "runtime.yorva-backup.age")
	destination, err := InspectLocalDestination(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	credential := testPassphraseCredential(t)
	moved := filepath.Join(root, "selected-moved")
	ops := publicationOpsWithSpace()
	var mutationErr error
	ops.beforePublish = func() error {
		if err := os.Rename(parent, moved); err != nil {
			mutationErr = err
			return err
		}
		mutationErr = os.Mkdir(parent, 0o700)
		return mutationErr
	}

	result, err := publishVerifiedArtifact(context.Background(), PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)), Credential: credential, Limits: DefaultLimits(),
	}, ops)
	if mutationErr != nil {
		t.Fatalf("destination directory mutation failed: %v", mutationErr)
	}
	assertEmptyPublicationFailure(t, result, err, ErrorDestinationUnsafe)
	if _, statErr := os.Lstat(destinationPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("destination published through changed directory: %v", statErr)
	}
}

func TestPublishVerifiedArtifactChecksSpaceBeforeStaging(t *testing.T) {
	artifact, _, destination := publicationFixture(t)
	credential := testPassphraseCredential(t)
	created := false
	ops := publicationOpsWithSpace()
	ops.freeBytes = func(string) (uint64, error) { return 1, nil }
	ops.createStaging = func(string) (*os.File, error) {
		created = true
		return nil, errors.New("must not create")
	}

	result, err := publishVerifiedArtifact(context.Background(), PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)), Credential: credential, Limits: DefaultLimits(),
	}, ops)
	assertEmptyPublicationFailure(t, result, err, ErrorInsufficientSpace)
	if created {
		t.Fatal("staging was created before insufficient-space rejection")
	}
}

func TestPublishVerifiedArtifactCancellationCleansStagingAndPublishesNothing(t *testing.T) {
	artifact, destinationPath, destination := publicationFixture(t)
	credential := testPassphraseCredential(t)
	ctx, cancel := context.WithCancel(context.Background())
	ops := publicationOpsWithSpace()
	ops.beforePublish = func() error {
		cancel()
		return nil
	}

	result, err := publishVerifiedArtifact(ctx, PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)), Credential: credential, Limits: DefaultLimits(),
	}, ops)
	assertEmptyPublicationFailure(t, result, err, ErrorOperationCanceled)
	if _, statErr := os.Lstat(destinationPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("canceled publication created destination: %v", statErr)
	}
	assertNoPublicationStaging(t, filepath.Dir(destinationPath))
}

func TestPublishVerifiedArtifactFailureCleansStagingAndPublishesNothing(t *testing.T) {
	artifact, destinationPath, destination := publicationFixture(t)
	credential := testPassphraseCredential(t)
	ops := publicationOpsWithSpace()
	ops.publish = func(string, string) error { return errors.New("injected publish failure") }

	result, err := publishVerifiedArtifact(context.Background(), PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)), Credential: credential, Limits: DefaultLimits(),
	}, ops)
	assertEmptyPublicationFailure(t, result, err, ErrorPublicationFailed)
	if _, statErr := os.Lstat(destinationPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed publication created destination: %v", statErr)
	}
	assertNoPublicationStaging(t, filepath.Dir(destinationPath))
}

func TestPublishVerifiedArtifactConvergesAfterPostMoveError(t *testing.T) {
	artifact, destinationPath, destination := publicationFixture(t)
	ops := publicationOpsWithSpace()
	basePublish := ops.publish
	ops.publish = func(staging, target string) error {
		if err := basePublish(staging, target); err != nil {
			return err
		}
		return errors.New("injected post-move attribute failure")
	}

	result, err := publishVerifiedArtifact(context.Background(), PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)),
		Credential: testPassphraseCredential(t), Limits: DefaultLimits(),
	}, ops)
	if err != nil || result.State != PublicationVerified || result.ChecksumSHA256 == "" {
		t.Fatalf("published artifact remained ambiguous: result=%#v err=%v", result, err)
	}
	if _, err := os.Stat(destinationPath); err != nil {
		t.Fatalf("published destination missing: %v", err)
	}
	assertNoPublicationStaging(t, filepath.Dir(destinationPath))
}

func TestPublishVerifiedArtifactReturnsReconcileStateAfterPublishedSyncFailure(t *testing.T) {
	artifact, destinationPath, destination := publicationFixture(t)
	ops := publicationOpsWithSpace()
	baseSync := ops.syncDirectory
	syncCalls := 0
	ops.syncDirectory = func(path string) error {
		syncCalls++
		if syncCalls == 2 {
			return errors.New("injected post-publish sync failure")
		}
		return baseSync(path)
	}

	result, err := publishVerifiedArtifact(context.Background(), PublicationRequest{
		Destination: destination, Artifact: bytes.NewReader(artifact), ArtifactSize: int64(len(artifact)),
		Credential: testPassphraseCredential(t), Limits: DefaultLimits(),
	}, ops)
	assertErrorCode(t, err, ErrorPublicationReconcile)
	if result.State != PublicationReconcileRequired || result.ChecksumSHA256 == "" || result.EncryptedSizeBytes <= 0 {
		t.Fatalf("published reconciliation truth missing: %#v", result)
	}
	if _, err := os.Stat(destinationPath); err != nil {
		t.Fatalf("published destination missing: %v", err)
	}
	assertNoPublicationStaging(t, filepath.Dir(destinationPath))
}

func TestInspectLocalDestinationRejectsADSAndSymlink(t *testing.T) {
	root := t.TempDir()
	ads := filepath.Join(root, "runtime:stream.yorva-backup.age")
	_, err := InspectLocalDestination(ads)
	assertErrorCode(t, err, ErrorDestinationInvalid)

	realDirectory := filepath.Join(root, "real")
	if err := os.Mkdir(realDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(realDirectory, link); err != nil {
		t.Skipf("symlink unavailable on this host: %v", err)
	}
	_, err = InspectLocalDestination(filepath.Join(link, "runtime.yorva-backup.age"))
	assertErrorCode(t, err, ErrorDestinationUnsafe)
}

func TestPublicationTypesDoNotFormatPathOrSecret(t *testing.T) {
	destinationPath := filepath.Join(t.TempDir(), "private-name.yorva-backup.age")
	destination, err := InspectLocalDestination(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("portable-secret-canary")
	credential, err := NewPassphraseCreateCredential(secret)
	if err != nil {
		t.Fatal(err)
	}
	formatted := fmt.Sprintf("%v %#v %v %#v", destination, destination, credential, credential)
	if strings.Contains(formatted, destinationPath) || strings.Contains(formatted, string(secret)) {
		t.Fatalf("formatted publication boundary leaked path or secret: %s", formatted)
	}
}

func publicationFixture(t *testing.T) ([]byte, string, LocalDestination) {
	t.Helper()
	artifact := buildTestArtifact(t, normalPayloadMembers(), "")
	destinationPath := filepath.Join(t.TempDir(), "runtime.yorva-backup.age")
	destination, err := InspectLocalDestination(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	return artifact, destinationPath, destination
}

func testPassphraseCredential(t *testing.T) *CreateCredential {
	t.Helper()
	credential, err := NewPassphraseCreateCredential([]byte("one-shot portable passphrase"))
	if err != nil {
		t.Fatal(err)
	}
	return credential
}

func publicationOpsWithSpace() publicationOps {
	ops := defaultPublicationOps()
	ops.freeBytes = func(string) (uint64, error) { return math.MaxUint64, nil }
	return ops
}

func assertEmptyPublicationFailure(t *testing.T, result PublishedArtifactMetadata, err error, code ErrorCode) {
	t.Helper()
	assertErrorCode(t, err, code)
	if result.State != "" || result.EncryptedSizeBytes != 0 || result.ChecksumSHA256 != "" || result.EncryptionMode != "" || result.Artifact.State != "" {
		t.Fatalf("failed publication returned metadata: %#v", result)
	}
}

func assertNoPublicationStaging(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, stagingFilePrefix+"*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("publication staging remained: %v", matches)
	}
}
