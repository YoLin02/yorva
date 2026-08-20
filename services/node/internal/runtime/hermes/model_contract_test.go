package hermes

import (
	"reflect"
	"strings"
	"testing"
)

func TestModelConfigCommandContract(t *testing.T) {
	if got, want := modelConfigGetArgs("coder", modelProviderConfigKey), []string{"--profile", "coder", "config", "get", "model.provider", "--json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("get args = %#v, want %#v", got, want)
	}
	if got, want := modelConfigSetArgs("coder", modelDefaultConfigKey, "deepseek-v4-pro"), []string{"--profile", "coder", "config", "set", "model.default", "deepseek-v4-pro"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("set args = %#v, want %#v", got, want)
	}
}

func TestModelValidationContractIsExplicitAndSecretFree(t *testing.T) {
	preset, err := lookupModelProviderPreset("deepseek")
	if err != nil {
		t.Fatal(err)
	}
	args := modelValidationArgs("coder", preset, "deepseek-v4-pro")
	want := []string{
		"--profile", "coder",
		"--provider", "deepseek",
		"--model", "deepseek-v4-pro",
		"--toolsets", "context_engine",
		"--oneshot", modelValidationPrompt,
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("validation args = %#v, want %#v", args, want)
	}
	joined := strings.Join(args, "\x00")
	for _, forbidden := range []string{"API_KEY", credentialSentinel, ".env", "config.yaml", "--api-key"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("validation argv contains forbidden material %q", forbidden)
		}
	}
	if got := modelConfigGetArgs("coder", contextEngineConfigKey); !reflect.DeepEqual(got, []string{"--profile", "coder", "config", "get", "context.engine", "--json"}) {
		t.Fatalf("context-engine guard args = %#v", got)
	}
}

func TestProfileCommandEnvironmentRejectsProviderCredentials(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", credentialSentinel)
	t.Setenv("DASHSCOPE_API_KEY", credentialSentinel)
	t.Setenv("OPENAI_API_KEY", credentialSentinel)
	env := profileCommandEnvironment(officialHermesHome())
	if profileEnvHasSecret(env, credentialSentinel) {
		t.Fatal("ambient provider credential reached child environment")
	}
}
