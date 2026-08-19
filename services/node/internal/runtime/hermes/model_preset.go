package hermes

import "errors"

// Model provider mappings are qualified against Hermes 0.20.2 at
// df4b65147d7ddd74dd449f9067aabbca5aef0ec7. The native provider and
// credential names stay adapter-private.
const modelSurfaceVersion = "0.20.2"

type ModelRegion string

const (
	ModelRegionChina  ModelRegion = "CHINA"
	ModelRegionGlobal ModelRegion = "GLOBAL"
)

type ModelProviderPreset struct {
	ID                string
	DisplayName       string
	Region            ModelRegion
	RecommendedModels []string
	HelpText          string
}

type modelProviderPreset struct {
	safe              ModelProviderPreset
	hermesProviderID  string
	credentialEnvName string
}

var (
	errModelProviderUnsupported = errors.New("hermes model provider is unsupported")
	errModelVersionUnsupported  = errors.New("hermes model surface version is unsupported")
)

var qualifiedModelProviderPresets = []modelProviderPreset{
	{
		safe: ModelProviderPreset{
			ID: "deepseek", DisplayName: "DeepSeek", Region: ModelRegionChina,
			RecommendedModels: []string{"deepseek-v4-pro", "deepseek-v4-flash"},
		},
		hermesProviderID: "deepseek", credentialEnvName: "DEEPSEEK_API_KEY",
	},
	{
		safe: ModelProviderPreset{
			ID: "qwen", DisplayName: "Qwen / Alibaba DashScope", Region: ModelRegionChina,
			RecommendedModels: []string{"qwen3.7-max", "qwen3.7-plus", "qwen3.6-plus"},
			HelpText:          "Hermes 0.20.2 uses the DashScope international compatible endpoint.",
		},
		hermesProviderID: "alibaba", credentialEnvName: "DASHSCOPE_API_KEY",
	},
	{
		safe: ModelProviderPreset{
			ID: "kimi", DisplayName: "Kimi / Moonshot (China)", Region: ModelRegionChina,
			RecommendedModels: []string{"kimi-k3", "kimi-k2.7-code", "kimi-k2.6"},
		},
		hermesProviderID: "kimi-coding-cn", credentialEnvName: "KIMI_CN_API_KEY",
	},
	{
		safe: ModelProviderPreset{
			ID: "minimax", DisplayName: "MiniMax (China)", Region: ModelRegionChina,
			RecommendedModels: []string{"MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.5"},
		},
		hermesProviderID: "minimax-cn", credentialEnvName: "MINIMAX_CN_API_KEY",
	},
	{
		safe: ModelProviderPreset{
			ID: "glm", DisplayName: "GLM / Zhipu", Region: ModelRegionChina,
			RecommendedModels: []string{"glm-5.2", "glm-5.1", "glm-5"},
			HelpText:          "Hermes selects the qualified Z.AI endpoint for this provider.",
		},
		hermesProviderID: "zai", credentialEnvName: "GLM_API_KEY",
	},
	{
		safe: ModelProviderPreset{
			ID: "openrouter", DisplayName: "OpenRouter", Region: ModelRegionGlobal,
			RecommendedModels: []string{"anthropic/claude-sonnet-4.6", "openai/gpt-5.4"},
		},
		hermesProviderID: "openrouter", credentialEnvName: "OPENROUTER_API_KEY",
	},
	{
		safe: ModelProviderPreset{
			ID: "openai", DisplayName: "OpenAI", Region: ModelRegionGlobal,
			RecommendedModels: []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.4"},
		},
		hermesProviderID: "openai-api", credentialEnvName: "OPENAI_API_KEY",
	},
	{
		safe: ModelProviderPreset{
			ID: "anthropic", DisplayName: "Anthropic", Region: ModelRegionGlobal,
			RecommendedModels: []string{"claude-fable-5", "claude-sonnet-5", "claude-opus-4-8"},
		},
		hermesProviderID: "anthropic", credentialEnvName: "ANTHROPIC_API_KEY",
	},
}

func ListModelProviderPresets() []ModelProviderPreset {
	result := make([]ModelProviderPreset, 0, len(qualifiedModelProviderPresets))
	for _, preset := range qualifiedModelProviderPresets {
		copyPreset := preset.safe
		copyPreset.RecommendedModels = append([]string(nil), preset.safe.RecommendedModels...)
		result = append(result, copyPreset)
	}
	return result
}

func lookupModelProviderPreset(id string) (modelProviderPreset, error) {
	for _, preset := range qualifiedModelProviderPresets {
		if preset.safe.ID == id {
			return preset, nil
		}
	}
	return modelProviderPreset{}, errModelProviderUnsupported
}

func IsModelProviderUnsupported(err error) bool {
	return errors.Is(err, errModelProviderUnsupported)
}

func IsModelVersionUnsupported(err error) bool {
	return errors.Is(err, errModelVersionUnsupported)
}
