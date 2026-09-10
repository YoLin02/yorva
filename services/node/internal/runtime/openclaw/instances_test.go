package openclaw

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestNamesPathsAndExternalOwnership(t *testing.T) {
	a := &Adapter{home: t.TempDir()}
	for _, name := range []string{"default", "Default", "../other", `..\other`, "work;whoami", "work/name", "-flag", "UPPER", strings.Repeat("x", 65)} {
		if !errors.Is(a.ValidateName(name), yorvaruntime.ErrInstanceNameInvalid) {
			t.Fatalf("accepted create name %q", name)
		}
	}
	for _, name := range []string{"work", "a_1", "a-b"} {
		if err := a.ValidateName(name); err != nil {
			t.Fatalf("valid name %q: %v", name, err)
		}
	}
	if _, err := a.profileRoot(`..\other`); err == nil {
		t.Fatal("accepted traversal")
	}
	root, _ := a.profileRoot("work")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.ownedProfile("work"); err == nil {
		t.Fatal("external directory treated as owned")
	}
	features, err := a.ResolveInstanceManagement(context.Background(), yorvaruntime.Installation{}, "work")
	if err != nil || !features.DisableLifecycle || features.Health == nil {
		t.Fatalf("external capabilities = %#v, %v", features, err)
	}
	data, _ := json.Marshal(ownership{Schema: 1, Profile: "work", Port: 29120})
	marker := filepath.Join(root, markerName)
	if err := os.WriteFile(marker, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.ownedProfile("work"); err != nil {
		t.Fatal(err)
	}
	features, _ = a.ResolveInstanceManagement(context.Background(), yorvaruntime.Installation{}, "work")
	if features.DisableLifecycle {
		t.Fatal("owned target not available")
	}
	if _, _, err := a.ownedProfile("default"); err == nil {
		t.Fatal("default became owned")
	}
	data, _ = json.Marshal(ownership{Schema: 1, Profile: "another", Port: 29120})
	if err := os.WriteFile(marker, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.ownedProfile("work"); err == nil {
		t.Fatal("foreign marker accepted")
	}
}

func TestEnvironmentExcludesInheritedCredentialsAndPathOverrides(t *testing.T) {
	for key, value := range map[string]string{"OPENAI_API_KEY": "provider-secret", "OPENCLAW_GATEWAY_TOKEN": "gateway-secret", "OPENCLAW_STATE_DIR": "foreign-state", "OPENCLAW_HOME": "foreign-home", "NODE_OPTIONS": "--import=foreign", "YORVA_TOKEN": "daemon-secret"} {
		t.Setenv(key, value)
	}
	a := &Adapter{home: t.TempDir(), path: `C:\Windows\System32`}
	env := strings.Join(a.environment(filepath.Join(a.home, "node.exe"), "default"), "\n")
	for _, forbidden := range []string{"provider-secret", "gateway-secret", "foreign-state", "foreign-home", "--import", "daemon-secret"} {
		if strings.Contains(env, forbidden) {
			t.Fatalf("inherited forbidden value %q", forbidden)
		}
	}
	if !strings.Contains(env, "OPENCLAW_LOG_DIR="+filepath.Join(a.home, ".openclaw", "logs")) {
		t.Fatal("default logs use wrong profile")
	}
}

func TestBoundedOutputDoesNotExposeSecretOnFailure(t *testing.T) {
	data, err := readBounded(strings.NewReader(strings.Repeat("secret", 100)), 100)
	if !errors.Is(err, errOutputLimit) || len(data) != 0 || strings.Contains(err.Error(), "secret") {
		t.Fatalf("output = %q, error = %v", data, err)
	}
}
