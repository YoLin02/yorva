package openclaw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// The test executable supplies a deterministic external CLI, exercising argv,
// environment, bounded streams and process ownership without mocking those
// layers. Release compatibility is separately exercised with the real CLI.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--openclaw-contract-child" {
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	if len(os.Args) > 1 && filepath.Base(os.Args[1]) == "openclaw.mjs" {
		data, err := os.ReadFile(os.Args[1])
		var fixture struct{ Mode, Version string }
		if err != nil || json.Unmarshal(data, &fixture) != nil {
			os.Exit(8)
		}
		switch fixture.Mode {
		case "failed":
			fmt.Fprint(os.Stderr, "secret-from-cli")
			os.Exit(1)
		case "limit":
			fmt.Print(strings.Repeat("secret-from-cli", 30000))
			os.Exit(0)
		case "tree", "orphan":
			executable, _ := os.Executable()
			child := exec.Command(executable, "--openclaw-contract-child")
			if fixture.Mode == "tree" {
				child.Stdout, child.Stderr = os.Stdout, os.Stderr
			}
			if child.Start() != nil {
				os.Exit(9)
			}
			_ = os.WriteFile(filepath.Join(os.Getenv("USERPROFILE"), "child.pid"), []byte(fmt.Sprint(child.Process.Pid)), 0600)
			if fixture.Mode == "orphan" {
				os.Exit(0)
			}
			time.Sleep(time.Minute)
		case "sleep":
			time.Sleep(time.Minute)
		default:
			if len(os.Args) == 3 && os.Args[2] == "--version" {
				fmt.Print(fixture.Version)
				os.Exit(0)
			}
			if len(os.Args) == 7 && os.Args[2] == "--profile" && strings.Join(os.Args[4:], " ") == "config validate --json" {
				profile := os.Args[3]
				root := ".openclaw-" + profile
				if profile == "default" {
					root = ".openclaw"
				}
				_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"valid": true, "path": filepath.Join(os.Getenv("USERPROFILE"), root, "openclaw.json")})
				os.Exit(0)
			}
			os.Exit(7)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func contractCLI(t *testing.T, mode, version string) (*Adapter, string, string) {
	t.Helper()
	root := t.TempDir()
	prefix := filepath.Join(root, "npm")
	packageRoot := filepath.Join(prefix, "node_modules", "openclaw")
	if err := os.MkdirAll(packageRoot, 0700); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(packageRoot, "openclaw.mjs")
	data, _ := json.Marshal(map[string]string{"Mode": mode, "Version": version})
	if err := os.WriteFile(entry, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "package.json"), []byte(`{"name":"openclaw","version":"2026.9.3","bin":{"openclaw":"openclaw.mjs"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	node := filepath.Join(prefix, "node.exe")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	{
		source, err := os.Open(executable)
		if err != nil {
			t.Fatal(err)
		}
		defer source.Close()
		target, err := os.OpenFile(node, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0700)
		if err != nil {
			t.Fatal(err)
		}
		_, copyErr := io.Copy(target, source)
		closeErr := target.Close()
		if copyErr != nil || closeErr != nil {
			t.Fatalf("copy helper: %v, %v", copyErr, closeErr)
		}
	}
	return &Adapter{home: root, path: prefix, appData: filepath.Join(root, "appdata")}, node, entry
}

func TestDiscoveryStatesAndCanonicalCandidateDeduplication(t *testing.T) {
	for _, tc := range []struct {
		name, mode, version string
		expected            yorvaruntime.DiscoveryState
	}{
		{"supported", "", "OpenClaw 2026.9.3 (1391f7c)", yorvaruntime.DiscoverySupported},
		{"unsupported", "", "OpenClaw 2026.9.4 (abc123)", yorvaruntime.DiscoveryUnsupported},
		{"malformed", "", "OpenClaw unknown", yorvaruntime.DiscoveryMalformedVersion},
		{"broken", "failed", "", yorvaruntime.DiscoveryBrokenExecutable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _, entry := contractCLI(t, tc.mode, tc.version)
			a.path += string(os.PathListSeparator) + a.path
			result, err := a.Detect(context.Background())
			expected := tc.expected
			if expected == yorvaruntime.DiscoverySupported && (runtime.GOOS != "windows" || runtime.GOARCH != "amd64") {
				expected = yorvaruntime.DiscoveryUnsupported
			}
			canonical, canonicalErr := filepath.EvalSymlinks(entry)
			if err != nil || canonicalErr != nil || result.State != expected || len(result.Candidates) != 1 || result.Selected.Path != canonical {
				t.Fatalf("discovery = %#v, %v", result, err)
			}
		})
	}
	a := &Adapter{home: t.TempDir()}
	if result, err := a.Detect(context.Background()); err != nil || result.State != yorvaruntime.DiscoveryNotInstalled {
		t.Fatalf("missing = %#v, %v", result, err)
	}
}

func TestOfficialInventoryValidationAndOwnershipProjection(t *testing.T) {
	a, _, entry := contractCLI(t, "", "")
	for _, profile := range []string{"default", "external", "owned"} {
		root, _ := a.profileRoot(profile)
		if err := os.Mkdir(root, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "openclaw.json"), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
		if profile == "owned" {
			data, _ := json.Marshal(ownership{Schema: 1, Profile: profile, Port: 29120})
			if err := os.WriteFile(filepath.Join(root, markerName), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	items, err := a.List(context.Background(), entry)
	if err != nil || len(items) != 3 {
		t.Fatalf("list = %#v, %v", items, err)
	}
	for _, item := range items {
		if item.Protected != (item.NativeID != "owned") || item.Default != (item.NativeID == "default") {
			t.Fatalf("ownership = %#v", item)
		}
	}
	if err := os.Remove(filepath.Join(a.home, ".openclaw-external", "openclaw.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := a.List(context.Background(), entry); !errors.Is(err, yorvaruntime.ErrInstanceInventoryFailed) {
		t.Fatalf("broken profile = %v", err)
	}
}

func TestDiscoveryAmbiguityTimeoutAndInvalidPackageFailClosed(t *testing.T) {
	a, _, entry := contractCLI(t, "", "OpenClaw 2026.9.3 (1391f7c)")
	b, _, _ := contractCLI(t, "", "OpenClaw 2026.9.3 (1391f7c)")
	a.path += string(os.PathListSeparator) + b.path
	result, err := a.Detect(context.Background())
	if err != nil || result.State != yorvaruntime.DiscoveryAmbiguous || result.Selected != nil {
		t.Fatalf("ambiguous = %#v, %v", result, err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(entry), "package.json"), []byte(`{"name":"foreign","bin":{"openclaw":"openclaw.mjs"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.resolveNode(entry); !errors.Is(err, errUnsafeTarget) {
		t.Fatalf("invalid package = %v", err)
	}
	a, _, _ = contractCLI(t, "sleep", "")
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	result, err = a.Detect(ctx)
	if err != nil || result.State != yorvaruntime.DiscoveryTimedOut {
		t.Fatalf("timeout = %#v, %v", result, err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if _, err := a.Detect(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
}

func TestRealReleaseSizedMetadataAndOversizeRejection(t *testing.T) {
	a, node, entry := contractCLI(t, "", "OpenClaw 2026.9.3 (1391f7c)")
	metadata := filepath.Join(filepath.Dir(entry), "package.json")
	base := `{"name":"openclaw","version":"2026.9.3","bin":{"openclaw":"openclaw.mjs"}}`
	// Size observed in the actual npm 2026.9.3 payload, beyond the initial
	// 128 KiB assumption. JSON padding represents its unrelated export map.
	data := []byte(base + strings.Repeat(" ", 135311-len(base)))
	if err := os.WriteFile(metadata, data, 0600); err != nil {
		t.Fatal(err)
	}
	canonicalNode, _ := filepath.EvalSymlinks(node)
	if got, err := a.qualifiedNode(entry); err != nil || got != canonicalNode {
		t.Fatalf("qualified release metadata = %q, %v", got, err)
	}
	if err := os.WriteFile(metadata, []byte(base+strings.Repeat(" ", packageMetadataLimit)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.qualifiedNode(entry); !errors.Is(err, errUnsafeTarget) {
		t.Fatalf("oversized package metadata = %v", err)
	}
}

func TestCommandBoundsAndCancellationOwnDescendants(t *testing.T) {
	a, node, entry := contractCLI(t, "limit", "")
	if data, err := a.command(context.Background(), node, entry, "", 5*time.Second, "--version"); !errors.Is(err, errOutputLimit) || len(data) != 0 {
		t.Fatalf("limited output bytes=%d, %v", len(data), err)
	}
	a, node, entry = contractCLI(t, "tree", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	if _, err := a.command(ctx, node, entry, "", 30*time.Second, "--version"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancel = %v", err)
	}
	if time.Since(started) > 6*time.Second {
		t.Fatal("descendant kept command pipes open after cancellation")
	}
	data, err := os.ReadFile(filepath.Join(a.home, "child.pid"))
	if err != nil {
		t.Fatal("test did not launch its descendant")
	}
	var pid int
	if _, err := fmt.Sscan(string(data), &pid); err != nil {
		t.Fatal(err)
	}
	child, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	defer child.Release()
	if runtime.GOOS == "windows" && processRunning(&exec.Cmd{Process: child}) {
		t.Fatal("descendant survived command cancellation")
	}
}

func TestSuccessfulCommandAlsoJoinsOwnedDescendants(t *testing.T) {
	a, node, entry := contractCLI(t, "orphan", "")
	if _, err := a.command(context.Background(), node, entry, "", 5*time.Second, "--version"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(a.home, "child.pid"))
	if err != nil {
		t.Fatal("test did not launch its descendant")
	}
	var pid int
	if _, err := fmt.Sscan(string(data), &pid); err != nil {
		t.Fatal(err)
	}
	child, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	defer child.Release()
	if runtime.GOOS == "windows" && processRunning(&exec.Cmd{Process: child}) {
		t.Fatal("descendant survived successful command cleanup")
	}
}
