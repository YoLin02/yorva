package backupmanagement

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestBuildCanonicalHermesRuntimeSnapshotIncludesDefaultAndNamedProfiles(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	root := filepath.Join(localAppData, "hermes")
	writeSnapshotFixture(t, root, ".env", "MODEL_API_KEY=secret\n")
	writeSnapshotFixture(t, root, "config.yaml", "model: test\n")
	writeSnapshotFixture(t, root, "state.db", "default-session-state")
	writeSnapshotFixture(t, root, "auth.json", `{"token":"secret"}`)
	writeSnapshotFixture(t, root, "sessions/default.json", `{"conversation":true}`)
	writeSnapshotFixture(t, root, "profiles/work/.env", "WORK_KEY=secret\n")
	writeSnapshotFixture(t, root, "profiles/work/auth.json", `{"account":"secret"}`)
	writeSnapshotFixture(t, root, "profiles/work/sessions/work.json", `{"session":true}`)
	writeSnapshotFixture(t, root, "profiles/work/skills/example/SKILL.md", "# Example")
	writeSnapshotFixture(t, root, "profiles/work/skills/example/bin/custom.txt", "user skill data")

	writeSnapshotFixture(t, root, "hermes-agent/source.py", "excluded installation")
	writeSnapshotFixture(t, root, "node/node.exe", "excluded installation")
	writeSnapshotFixture(t, root, "bin/hermes.exe", "excluded installation")
	writeSnapshotFixture(t, root, "backups/old.zip", "excluded backup")
	writeSnapshotFixture(t, root, "profiles/work/.venv/key.txt", "excluded dependency")
	writeSnapshotFixture(t, root, "profiles/work/cache/value.txt", "runtime-owned cache is included")

	payload, container := openSnapshotStaging(t)
	result, err := BuildCanonicalHermesRuntimeSnapshot(context.Background(), testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	if err != nil {
		t.Fatalf("BuildCanonicalHermesRuntimeSnapshot() error = %v", err)
	}
	if result.Artifact.State != ArtifactStructureVerified || result.SourceFileCount != 11 {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Equal(result.Artifact.Metadata.IncludedCategories, completeRuntimeSnapshotCategories) {
		t.Fatalf("categories = %#v", result.Artifact.Metadata.IncludedCategories)
	}
	if result.Artifact.Metadata.BackupID != testSnapshotDescriptor().BackupID ||
		result.Artifact.Metadata.CreatedAt != testSnapshotDescriptor().CreatedAt ||
		result.Artifact.Metadata.Installation != testSnapshotDescriptor().Installation ||
		result.Artifact.Metadata.InclusionPolicy != InclusionPolicyID ||
		result.Artifact.Metadata.ExclusionPolicy != ExclusionPolicyID ||
		!slices.Equal(result.Artifact.Metadata.ExcludedCategories, completeRuntimeSnapshotExcludedCategories) {
		t.Fatalf("manifest identity/policy metadata = %#v", result.Artifact.Metadata)
	}

	members := readSnapshotPayloadMembers(t, payload)
	for _, expected := range []string{
		"hermes-runtime/.env",
		"hermes-runtime/state.db",
		"hermes-runtime/auth.json",
		"hermes-runtime/sessions/default.json",
		"hermes-runtime/profiles/work/.env",
		"hermes-runtime/profiles/work/auth.json",
		"hermes-runtime/profiles/work/sessions/work.json",
		"hermes-runtime/profiles/work/skills/example/SKILL.md",
		"hermes-runtime/profiles/work/skills/example/bin/custom.txt",
		"hermes-runtime/profiles/work/cache/value.txt",
	} {
		if _, ok := members[expected]; !ok {
			t.Errorf("missing member %q", expected)
		}
	}
	for _, excluded := range []string{
		"hermes-runtime/hermes-agent/source.py",
		"hermes-runtime/node/node.exe",
		"hermes-runtime/bin/hermes.exe",
		"hermes-runtime/backups/old.zip",
		"hermes-runtime/profiles/work/.venv/key.txt",
	} {
		if _, ok := members[excluded]; ok {
			t.Errorf("excluded member present: %q", excluded)
		}
	}
	if got := string(members["hermes-runtime/profiles/work/.env"]); got != "WORK_KEY=secret\n" {
		t.Fatalf("named Profile credential bytes = %q", got)
	}
	if got := string(members["hermes-runtime/profiles/work/sessions/work.json"]); got != `{"session":true}` {
		t.Fatalf("named Profile session bytes = %q", got)
	}

	containerInfo, err := container.Stat()
	if err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyArtifact(container, containerInfo.Size(), DefaultLimits())
	if err != nil || verified.State != ArtifactStructureVerified {
		t.Fatalf("VerifyArtifact() = %#v, %v", verified, err)
	}
}

func TestBuildHermesRuntimeSnapshotRequiresStoppedExactRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "config.yaml", "model: test\n")
	payload, container := openSnapshotStaging(t)

	_, err := buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), false, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	assertSnapshotErrorCode(t, err, ErrorSourceRuntimeLive)
	if info, statErr := payload.Stat(); statErr != nil || info.Size() != 0 {
		t.Fatalf("payload was written before stopped precondition: size=%d err=%v", info.Size(), statErr)
	}

	invalidDescriptor := testSnapshotDescriptor()
	invalidDescriptor.RuntimeVersion = "0.20.6"
	_, err = buildHermesRuntimeSnapshot(context.Background(), root, invalidDescriptor, true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	assertSnapshotErrorCode(t, err, ErrorInputInvalid)
}

func TestBuildHermesRuntimeSnapshotIncludesStableWALAndExcludesEphemeralSHM(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "state.db", "database")
	writeSnapshotFixture(t, root, "state.db-wal", "committed WAL state")
	writeSnapshotFixture(t, root, "state.db-shm", "ephemeral shared memory")
	payload, container := openSnapshotStaging(t)

	if _, err := buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	members := readSnapshotPayloadMembers(t, payload)
	if got := string(members["hermes-runtime/state.db-wal"]); got != "committed WAL state" {
		t.Fatalf("WAL contents = %q", got)
	}
	if _, present := members["hermes-runtime/state.db-shm"]; present {
		t.Fatal("ephemeral SQLite shared-memory file entered backup")
	}
}

func TestBuildHermesRuntimeSnapshotUsesWindowsCaseInsensitivePolicy(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "config.yaml", "model: test\n")
	writeSnapshotFixture(t, root, "Generations/gen_unsafe/bin/hermes.exe", "must not be archived")
	payload, container := openSnapshotStaging(t)
	_, err := buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if _, present := readSnapshotPayloadMembers(t, payload)["hermes-runtime/Generations/gen_unsafe/bin/hermes.exe"]; present {
		t.Fatal("case-variant managed generation entered Runtime backup")
	}

	root = filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "state.db", "database")
	writeSnapshotFixture(t, root, "STATE.DB-WAL", "committed WAL state")
	writeSnapshotFixture(t, root, "STATE.DB-SHM", "ephemeral shared memory")
	payload, container = openSnapshotStaging(t)
	_, err = buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	members := readSnapshotPayloadMembers(t, payload)
	if _, present := members["hermes-runtime/STATE.DB-WAL"]; !present {
		t.Fatal("case-variant SQLite WAL was not preserved")
	}
	if _, present := members["hermes-runtime/STATE.DB-SHM"]; present {
		t.Fatal("case-variant SQLite SHM entered backup")
	}
}

func TestBuildHermesRuntimeSnapshotExcludesRegenerableLSPTree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "config.yaml", "model: test\n")
	writeSnapshotFixture(t, root, "lsp/bin/pyright-langserver", "regenerable tool")
	writeSnapshotFixture(t, root, "logs/.__agent.lock", "runtime lock")
	payload, container := openSnapshotStaging(t)
	if _, err := buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if _, present := readSnapshotPayloadMembers(t, payload)["hermes-runtime/lsp/bin/pyright-langserver"]; present {
		t.Fatal("regenerable LSP tree entered Runtime backup")
	}
	if _, present := readSnapshotPayloadMembers(t, payload)["hermes-runtime/logs/.__agent.lock"]; present {
		t.Fatal("Runtime log tree entered Runtime backup")
	}
}

func TestBuildHermesRuntimeSnapshotRejectsSourceLink(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "config.yaml", "model: test\n")
	external := filepath.Join(t.TempDir(), "external-secret")
	if err := os.WriteFile(external, []byte("must not be read"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "linked-secret")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("Windows symlink unavailable for this account: %v", err)
		}
		t.Fatal(err)
	}
	payload, container := openSnapshotStaging(t)

	_, err := buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	assertSnapshotErrorCode(t, err, ErrorSourceUnsafe)
}

func TestBuildHermesRuntimeSnapshotRejectsStagingInsideSource(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "config.yaml", "model: test\n")
	payload, err := os.Create(filepath.Join(root, "payload.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = payload.Close() })
	container, err := os.Create(filepath.Join(root, "container.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = container.Close() })

	_, err = buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	assertSnapshotErrorCode(t, err, ErrorSourceUnsafe)
}

func TestSnapshotInventoryAndCopyRejectChangedSource(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	path := writeSnapshotFixture(t, root, "state.db", "before")
	inventory, err := inventorySnapshotSource(context.Background(), root, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after-longer"), 0o600); err != nil {
		t.Fatal(err)
	}
	var source snapshotSourceEntry
	for _, entry := range inventory.entries {
		if !entry.isDir {
			source = entry
		}
	}
	err = copyStableSnapshotFile(context.Background(), io.Discard, source)
	assertSnapshotErrorCode(t, err, ErrorSourceChanged)
}

func TestBuildHermesRuntimeSnapshotCancellationIsStable(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "config.yaml", "model: test\n")
	payload, container := openSnapshotStaging(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := buildHermesRuntimeSnapshot(ctx, root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	assertSnapshotErrorCode(t, err, ErrorOperationCanceled)
}

func TestBuildHermesRuntimeSnapshotHonorsSourceLimits(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	writeSnapshotFixture(t, root, "large.bin", "0123456789")
	payload, container := openSnapshotStaging(t)
	limits := DefaultLimits()
	limits.MaxMemberExpandedBytes = 4

	_, err := buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, limits)
	assertSnapshotErrorCode(t, err, ErrorArchiveMemberSizeLimit)
}

func TestRealWindowsHermesRuntimeSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "windows" || os.Getenv("YORVA_REAL_HERMES_BACKUP_SMOKE") != "1" {
		t.Skip("set YORVA_REAL_HERMES_BACKUP_SMOKE=1 to snapshot the stopped local Hermes Runtime")
	}
	payload, container := openSnapshotStaging(t)
	result, err := BuildCanonicalHermesRuntimeSnapshot(context.Background(), testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	if err != nil {
		t.Fatalf("real Hermes Runtime snapshot failed: %v", err)
	}
	if result.Artifact.State != ArtifactStructureVerified || result.SourceFileCount == 0 || result.SourceBytes == 0 {
		t.Fatalf("real Hermes Runtime snapshot = %#v", result)
	}
	t.Logf("real Hermes Runtime snapshot verified: files=%d sourceBytes=%d payloadBytes=%d", result.SourceFileCount, result.SourceBytes, result.Artifact.Metadata.PayloadSizeBytes)
}

func testSnapshotDescriptor() SnapshotDescriptor {
	return SnapshotDescriptor{
		BackupID:       "backup_bbbbbbbbbbbbbbbbbbbbbb",
		CreatedAt:      "2026-08-25T02:03:04Z",
		RuntimeVersion: QualifiedSnapshotRuntimeVersion,
		Installation: InstallationIdentity{
			State:        InstallationManaged,
			GenerationID: "gen_bbbbbbbbbbbbbbbbbbbbbb",
			SourcePin:    "a0ca7c19204e514f9590ce3b812e029b315ab9e9",
		},
	}
}

func openSnapshotStaging(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	directory := t.TempDir()
	payload, err := os.OpenFile(filepath.Join(directory, "payload.zip"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = payload.Close() })
	container, err := os.OpenFile(filepath.Join(directory, "container.zip"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = container.Close() })
	return payload, container
}

func writeSnapshotFixture(t *testing.T, root, relative, body string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readSnapshotPayloadMembers(t *testing.T, payload *os.File) map[string][]byte {
	t.Helper()
	info, err := payload.Stat()
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(payload, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string][]byte, len(reader.File))
	for _, member := range reader.File {
		if member.FileInfo().IsDir() {
			result[member.Name] = nil
			continue
		}
		stream, err := member.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read %q: read=%v close=%v", member.Name, readErr, closeErr)
		}
		result[member.Name] = bytes.Clone(body)
	}
	return result
}

func assertSnapshotErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got nil", want)
	}
	got, ok := ErrorCodeOf(err)
	if !ok || got != want {
		t.Fatalf("ErrorCodeOf(%v) = %q, %v; want %q", err, got, ok, want)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("raw context error escaped: %v", err)
	}
}
