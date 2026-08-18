package hermes

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestOwnershipHandoffRetrySequence(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("installer Apply is Windows user-scope")
	}

	t.Run("does not overwrite previous record before repository", func(t *testing.T) {
		env := newRetryInstallEnv(t)
		if err := os.MkdirAll(env.installDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.installDir, "partial.txt"), []byte("from-a"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeOwnershipRecord(env.installDir, env.identityA()); err != nil {
			t.Fatal(err)
		}
		env.installer.SetInstallIdentity(env.opB)
		if err := env.installer.ValidateTarget(true, env.opA); err != nil {
			t.Fatalf("preflight: %v", err)
		}
		env.failStage = "uv"
		if err := env.installer.Apply(context.Background(), env.opB.ID, nil); err == nil {
			t.Fatal("expected uv failure before repository")
		}
		record, err := readOwnershipRecord(env.installDir)
		if err != nil {
			t.Fatal(err)
		}
		if record.OperationID != env.opA.ID {
			t.Fatalf("previous ownership overwritten before replacement: %s", record.OperationID)
		}
		if got, err := os.ReadFile(filepath.Join(env.installDir, "partial.txt")); err != nil || string(got) != "from-a" {
			t.Fatalf("previous tree disturbed: %q %v", got, err)
		}
	})

	t.Run("operation B retries unchanged failed tree", func(t *testing.T) {
		env := newRetryInstallEnv(t)
		env.installer.SetInstallIdentity(env.opA)
		if err := env.installer.ValidateTarget(false, operation.Operation{}); err != nil {
			t.Fatalf("A preflight: %v", err)
		}
		env.failStage = "dependencies"
		env.mutateOn = "venv"
		if err := env.installer.Apply(context.Background(), env.opA.ID, nil); err == nil {
			t.Fatal("expected operation A to fail after venv")
		}
		if _, err := os.Stat(env.installDir); !os.IsNotExist(err) {
			if _, err := os.Stat(filepath.Join(env.installDir, "venv.ok")); err == nil {
				t.Fatal("failed operation mutated the live tree")
			}
		}

		env.failStage = ""
		env.mutateOn = "venv"
		env.installer.SetInstallIdentity(env.opB)
		if err := env.installer.ValidateTarget(true, env.opA); err != nil {
			t.Fatalf("B preflight: %v", err)
		}
		if err := env.installer.Apply(context.Background(), env.opB.ID, nil); err != nil {
			t.Fatalf("operation B retry: %v", err)
		}
		record, err := readOwnershipRecord(env.installDir)
		if err != nil {
			t.Fatalf("B ownership record: %v", err)
		}
		if err := verifyRecordIdentity(record, env.identityB()); err != nil {
			t.Fatalf("B ownership identity: %v", err)
		}
		if err := requireCurrentOwnedTree(env.installDir, env.identityB()); err != nil {
			t.Fatalf("B inventory after path/bin materialization: %v", err)
		}
		if _, err := os.Stat(filepath.Join(env.installDir, "pyproject.toml")); err != nil {
			t.Fatal("repository replacement missing")
		}
		if _, err := os.Stat(filepath.Join(env.installDir, "bin", "hermes.exe")); err != nil {
			t.Fatal("later approved stages did not continue")
		}
		if env.sawRepository {
			t.Fatal("official repository stage was spawned")
		}
	})

	t.Run("rejects retry after foreign mutation", func(t *testing.T) {
		env := newRetryInstallEnv(t)
		env.installer.SetInstallIdentity(env.opA)
		if err := env.installer.ValidateTarget(false, operation.Operation{}); err != nil {
			t.Fatal(err)
		}
		env.failStage = "dependencies"
		env.mutateOn = "venv"
		if err := env.installer.Apply(context.Background(), env.opA.ID, nil); err == nil {
			t.Fatal("expected operation A failure")
		}
		if err := os.MkdirAll(env.installDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.installDir, "owned.txt"), []byte("a"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeOwnershipRecord(env.installDir, env.identityA()); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.installDir, "stale.exe"), []byte("MZ"), 0o700); err != nil {
			t.Fatal(err)
		}
		env.failStage = ""
		env.installer.SetInstallIdentity(env.opB)
		if installErrorCode(env.installer.ValidateTarget(true, env.opA)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("foreign executable was accepted as a retry target")
		}
		if _, err := os.Stat(filepath.Join(env.installDir, "stale.exe")); err != nil {
			t.Fatal("uncertain extra executable was deleted")
		}
		if err := env.installer.Apply(context.Background(), env.opB.ID, nil); err == nil {
			t.Fatal("Apply replaced a tampered tree")
		}
		if _, err := os.Stat(filepath.Join(env.installDir, "stale.exe")); err != nil {
			t.Fatal("tampered tree was deleted during rejected Apply")
		}
	})

	t.Run("cancel during replacement leaves dest and no temps", func(t *testing.T) {
		env := newRetryInstallEnv(t)
		if err := os.MkdirAll(env.installDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(env.installDir, "keep.bin"), []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeOwnershipRecord(env.installDir, env.identityA()); err != nil {
			t.Fatal(err)
		}
		env.installer.SetInstallIdentity(env.opB)
		if err := env.installer.ValidateTarget(true, env.opA); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		env.cancelOnManifest = cancel
		if err := env.installer.Apply(ctx, env.opB.ID, nil); err == nil {
			t.Fatal("expected cancelled Apply")
		}
		for _, entry := range mustReadDir(t, env.home) {
			if strings.Contains(entry.Name(), "yorva-new-") || strings.Contains(entry.Name(), "yorva-old-") {
				t.Fatalf("temporary directory left behind: %s", entry.Name())
			}
		}
		if got, err := os.ReadFile(filepath.Join(env.installDir, "keep.bin")); err != nil || string(got) != "keep" {
			t.Fatalf("owned dest changed after cancel: %q %v", got, err)
		}
		record, err := readOwnershipRecord(env.installDir)
		if err != nil || record.OperationID != env.opA.ID {
			t.Fatalf("cancel reassigned ownership: %#v %v", record, err)
		}
	})

	t.Run("timeout after mutation refreshes inventory", func(t *testing.T) {
		env := newRetryInstallEnv(t)
		env.installer.SetInstallIdentity(env.opA)
		if err := env.installer.ValidateTarget(false, operation.Operation{}); err != nil {
			t.Fatal(err)
		}
		env.mutateOn = "dependencies"
		env.timeoutStage = "dependencies"
		if err := env.installer.Apply(context.Background(), env.opA.ID, nil); err == nil {
			t.Fatal("expected timed-out dependencies stage")
		}
		if _, err := os.Stat(filepath.Join(env.installDir, "deps.partial")); err == nil {
			t.Fatal("timed-out mutation leaked onto the live tree")
		}
	})
}

type retryInstallEnv struct {
	t                *testing.T
	home             string
	installDir       string
	installer        *HostInstaller
	opA              operation.Operation
	opB              operation.Operation
	failStage        string
	mutateOn         string
	timeoutStage     string
	cancelOnManifest context.CancelFunc
	sawRepository    bool
}

func newRetryInstallEnv(t *testing.T) *retryInstallEnv {
	t.Helper()
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	archive := writeTestArchive(t, map[string]string{
		officialArchiveRoot + "/LICENSE":             "license",
		officialArchiveRoot + "/pyproject.toml":      `version = "0.20.2"`,
		officialArchiveRoot + "/scripts/install.ps1": "crlf-copy\r\n",
		officialArchiveRoot + "/hermes_cli/main.py":  "pass\n",
	})
	manifest, err := json.Marshal(reviewedManifest())
	if err != nil {
		t.Fatal(err)
	}
	env := &retryInstallEnv{
		t:          t,
		home:       home,
		installDir: installDir,
		opA: operation.Operation{
			ID:             "op_retry_a",
			Type:           operation.TypeRuntimeInstall,
			TargetType:     operation.TargetRuntimeKind,
			TargetID:       "hermes",
			Status:         operation.StatusFailed,
			Retryable:      true,
			SourcePin:      officialCommit,
			OwnershipNonce: "own_retry_a",
		},
		opB: operation.Operation{
			ID:             "op_retry_b",
			Type:           operation.TypeRuntimeInstall,
			TargetType:     operation.TargetRuntimeKind,
			TargetID:       "hermes",
			Status:         operation.StatusPending,
			Retryable:      true,
			SourcePin:      officialCommit,
			OwnershipNonce: "own_retry_b",
		},
	}
	installer := NewHostInstaller(t.TempDir())
	installer.home = func() string { return home }
	installer.installDir = func() string { return installDir }
	installer.shell = func() (string, error) { return `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, nil }
	installer.archive.diskFree = func(string) (uint64, error) { return archiveDiskBudget + archiveDiskMargin, nil }
	installer.acquireArchive = func(context.Context, string) (string, string, error) {
		return archive, sourceOriginBundled, nil
	}
	installer.userPath = func(string) bool { return true }
	installer.applyPathEnv = func(string, string) error { return nil }
	installer.run = func(_ context.Context, invocation installInvocation, _ time.Duration) commandResult {
		joined := strings.Join(invocation.Args, " ")
		if strings.Contains(joined, "-Stage") && strings.Contains(joined, "repository") {
			env.sawRepository = true
		}
		if strings.Contains(joined, "ProtocolVersion") {
			return commandResult{stdout: "1\n"}
		}
		if strings.Contains(joined, "Manifest") {
			if env.cancelOnManifest != nil {
				env.cancelOnManifest()
			}
			return commandResult{stdout: string(manifest) + "\n"}
		}
		stage := stageFromArgs(invocation.Args)
		stageDir := installDirFromArgs(invocation.Args)
		if stageDir == "" {
			stageDir = installDir
		}
		if env.mutateOn != "" && stage == env.mutateOn {
			name := filepath.Join("venv", "created.txt")
			if stage == "dependencies" {
				name = filepath.Join("venv", "deps.partial")
			}
			if err := os.MkdirAll(filepath.Join(stageDir, filepath.Dir(name)), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(stageDir, name), []byte(stage), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(stageDir, "venv", "Scripts"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(stageDir, "venv", "Scripts", "hermes.exe"), []byte("mz-hermes"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if env.timeoutStage != "" && stage == env.timeoutStage {
			return commandResult{timedOut: true, err: context.DeadlineExceeded}
		}
		if env.failStage != "" && stage == env.failStage {
			return commandResult{stdout: `{"stage":"` + stage + `","ok":false,"skipped":false}` + "\n", exitCode: 1, err: errPlatform}
		}
		if stage == "path" {
			launcher := filepath.Join(stageDir, "bin", "hermes.exe")
			if err := os.MkdirAll(filepath.Dir(launcher), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(launcher, []byte("mz"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return commandResult{stdout: `{"stage":"` + stage + `","ok":true,"skipped":false}` + "\n"}
	}
	env.installer = installer
	return env
}

func (env *retryInstallEnv) identityA() ownershipIdentity {
	return ownershipIdentity{
		OperationID: env.opA.ID,
		RuntimeKind: "hermes",
		Target:      env.installDir,
		SourcePin:   officialCommit,
		Nonce:       env.opA.OwnershipNonce,
	}
}

func (env *retryInstallEnv) identityB() ownershipIdentity {
	return ownershipIdentity{
		OperationID: env.opB.ID,
		RuntimeKind: "hermes",
		Target:      env.installDir,
		SourcePin:   officialCommit,
		Nonce:       env.opB.OwnershipNonce,
	}
}
