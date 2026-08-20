package hermes

import (
	"reflect"
	"testing"
)

func TestQualifiedModelProviderPresetsMatchPinnedHermes(t *testing.T) {
	want := []struct {
		id, nativeProvider, env string
		region                  ModelRegion
	}{
		{"deepseek", "deepseek", "DEEPSEEK_API_KEY", ModelRegionChina},
		{"qwen", "alibaba", "DASHSCOPE_API_KEY", ModelRegionChina},
		{"kimi", "kimi-coding-cn", "KIMI_CN_API_KEY", ModelRegionChina},
		{"minimax", "minimax-cn", "MINIMAX_CN_API_KEY", ModelRegionChina},
		{"glm", "zai", "GLM_API_KEY", ModelRegionChina},
		{"openrouter", "openrouter", "OPENROUTER_API_KEY", ModelRegionGlobal},
		{"openai", "openai-api", "OPENAI_API_KEY", ModelRegionGlobal},
		{"anthropic", "anthropic", "ANTHROPIC_API_KEY", ModelRegionGlobal},
	}
	if modelSurfaceVersion != profileOfficialVersion {
		t.Fatalf("model qualification version = %q, profile version = %q", modelSurfaceVersion, profileOfficialVersion)
	}
	if len(qualifiedModelProviderPresets) != len(want) {
		t.Fatalf("preset count = %d, want %d", len(qualifiedModelProviderPresets), len(want))
	}
	for index, expected := range want {
		got := qualifiedModelProviderPresets[index]
		if got.safe.ID != expected.id || got.hermesProviderID != expected.nativeProvider ||
			got.credentialEnvName != expected.env || got.safe.Region != expected.region {
			t.Fatalf("preset[%d] = %#v, want %#v", index, got, expected)
		}
		if len(got.safe.RecommendedModels) == 0 {
			t.Fatalf("preset %q has no reviewed models", got.safe.ID)
		}
	}
}

func TestListModelProviderPresetsReturnsSafeIndependentCopies(t *testing.T) {
	first := ListModelProviderPresets()
	second := ListModelProviderPresets()
	if len(first) == 0 || !reflect.DeepEqual(first, second) {
		t.Fatalf("safe preset lists differ: %#v %#v", first, second)
	}
	first[0].RecommendedModels[0] = "modified"
	if reflect.DeepEqual(first, second) {
		t.Fatal("caller mutation changed the shared preset list")
	}
	if _, err := lookupModelProviderPreset("DEEPSEEK_API_KEY"); !IsModelProviderUnsupported(err) {
		t.Fatalf("private env name lookup error = %v", err)
	}
}
