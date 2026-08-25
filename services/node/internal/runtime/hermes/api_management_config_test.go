package hermes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const (
	defaultAPIKey = "default-api-key-0123456789"
	namedAPIKey   = "named-api-key-012345678901"
)

func TestResolveAPIManagementTargetDefaultAndNamedIndependent(t *testing.T) {
	root := t.TempDir()
	writeAPIProfileFixture(t, root, "default", "API_SERVER_KEY="+defaultAPIKey+"\n", "")
	writeAPIProfileFixture(t, root, "work", "API_SERVER_KEY="+namedAPIKey+"\nAPI_SERVER_HOST=127.0.0.1\nAPI_SERVER_PORT=9753\n", `gateway:
  api_server:
    enabled: true
`)

	defaultTarget, err := resolveAPIManagementTargetAt(root, apiManagementVersion, "default")
	if err != nil {
		t.Fatalf("resolve default: %v", err)
	}
	defer defaultTarget.clear()
	if defaultTarget.host != apiManagementDefaultHost || defaultTarget.port != apiManagementDefaultPort || defaultTarget.pathPrefix != "" || string(defaultTarget.key) != defaultAPIKey {
		t.Fatal("default target projection mismatch")
	}

	namedTarget, err := resolveAPIManagementTargetAt(root, apiManagementVersion, "work")
	if err != nil {
		t.Fatalf("resolve named: %v", err)
	}
	defer namedTarget.clear()
	if namedTarget.host != "127.0.0.1" || namedTarget.port != 9753 || namedTarget.pathPrefix != "" || string(namedTarget.key) != namedAPIKey {
		t.Fatal("named target projection mismatch")
	}
}

func TestResolveAPIManagementTargetMultiplexUsesDefaultListenerAndNamedSecret(t *testing.T) {
	root := t.TempDir()
	writeAPIProfileFixture(t, root, "default", "API_SERVER_KEY="+defaultAPIKey+"\nAPI_SERVER_PORT=9864\n", `gateway:
  multiplex_profiles: true
  multiplex_profile_allowlist: [work]
  api_server:
    enabled: true
`)
	writeAPIProfileFixture(t, root, "work", "API_SERVER_KEY="+namedAPIKey+"\nAPI_SERVER_PORT=9999\n", `gateway:
  api_server:
    enabled: false
`)

	target, err := resolveAPIManagementTargetAt(root, apiManagementVersion, "work")
	if err != nil {
		t.Fatalf("resolve multiplex target: %v", err)
	}
	defer target.clear()
	if target.port != 9864 || target.pathPrefix != "/p/work" || string(target.key) != namedAPIKey {
		t.Fatal("multiplex target projection mismatch")
	}

	defaultTarget, err := resolveAPIManagementTargetAt(root, apiManagementVersion, "default")
	if err != nil {
		t.Fatalf("resolve multiplex default: %v", err)
	}
	defer defaultTarget.clear()
	if defaultTarget.port != 9864 || defaultTarget.pathPrefix != "" || string(defaultTarget.key) != defaultAPIKey {
		t.Fatal("multiplex default target projection mismatch")
	}
}

func TestResolveAPIManagementTargetFailsClosed(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		profile       string
		defaultEnv    string
		defaultConfig string
		namedEnv      string
		namedConfig   string
	}{
		{name: "different patch", version: "0.20.2", profile: "default", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n"},
		{name: "remote host", version: apiManagementVersion, profile: "default", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\nAPI_SERVER_HOST=0.0.0.0\n"},
		{name: "duplicate key", version: apiManagementVersion, profile: "default", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\nAPI_SERVER_KEY=" + namedAPIKey + "\n"},
		{name: "yaml key authority", version: apiManagementVersion, profile: "default", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n", defaultConfig: "gateway:\n  api_server:\n    key: shadow-secret-0123456789\n"},
		{name: "multiplex profile omitted", version: apiManagementVersion, profile: "work", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n", defaultConfig: "gateway:\n  multiplex_profiles: true\n  multiplex_profile_allowlist: [other]\n", namedEnv: "API_SERVER_KEY=" + namedAPIKey + "\n", namedConfig: "gateway:\n  api_server:\n    enabled: false\n"},
		{name: "multiplex secondary not disabled", version: apiManagementVersion, profile: "work", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n", defaultConfig: "gateway:\n  multiplex_profiles: true\n", namedEnv: "API_SERVER_KEY=" + namedAPIKey + "\n", namedConfig: ""},
		{name: "multiplex secondary yaml key", version: apiManagementVersion, profile: "work", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n", defaultConfig: "gateway:\n  multiplex_profiles: true\n", namedEnv: "API_SERVER_KEY=" + namedAPIKey + "\n", namedConfig: "gateway:\n  api_server:\n    enabled: false\n    key: shadow-secret-0123456789\n"},
		{name: "duplicate yaml field", version: apiManagementVersion, profile: "default", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n", defaultConfig: "gateway:\n  multiplex_profiles: false\n  multiplex_profiles: true\n"},
		{name: "ambiguous multiplex locations", version: apiManagementVersion, profile: "default", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n", defaultConfig: "multiplex_profiles: false\ngateway:\n  multiplex_profiles: true\n"},
		{name: "ambiguous listener locations", version: apiManagementVersion, profile: "default", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n", defaultConfig: "platforms:\n  api_server:\n    host: 127.0.0.1\ngateway:\n  api_server:\n    host: 127.0.0.1\n"},
		{name: "ambiguous listener field", version: apiManagementVersion, profile: "default", defaultEnv: "API_SERVER_KEY=" + defaultAPIKey + "\n", defaultConfig: "gateway:\n  api_server:\n    host: 127.0.0.1\n    extra:\n      host: 127.0.0.1\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeAPIProfileFixture(t, root, "default", test.defaultEnv, test.defaultConfig)
			if test.profile != "default" {
				writeAPIProfileFixture(t, root, test.profile, test.namedEnv, test.namedConfig)
			}
			if target, err := resolveAPIManagementTargetAt(root, test.version, test.profile); err == nil {
				target.clear()
				t.Fatal("unsafe or unqualified target was accepted")
			}
		})
	}
}

func TestResolveAPIManagementTargetRejectsUnsafeConfigFile(t *testing.T) {
	root := t.TempDir()
	writeAPIProfileFixture(t, root, "default", "API_SERVER_KEY="+defaultAPIKey+"\n", "gateway: &shared\n  multiplex_profiles: false\ncopy: *shared\n")
	_, err := resolveAPIManagementTargetAt(root, apiManagementVersion, "default")
	if !errors.Is(err, errAPIManagementUnsafe) {
		t.Fatalf("alias config error = %v", err)
	}
}

func writeAPIProfileFixture(t *testing.T, root, profile, env, config string) {
	t.Helper()
	directory := root
	if profile != "default" {
		directory = filepath.Join(root, "profiles", profile)
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if env != "" {
		if err := os.WriteFile(filepath.Join(directory, ".env"), []byte(env), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if config != "" {
		if err := os.WriteFile(filepath.Join(directory, "config.yaml"), []byte(config), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
