package hermes

const (
	modelProviderConfigKey = "model.provider"
	modelDefaultConfigKey  = "model.default"
	contextEngineConfigKey = "context.engine"
	toolsDisabledToolset   = "context_engine"
	defaultContextEngine   = "compressor"
	modelValidationPrompt  = "Reply with exactly YORVA_CONNECTION_OK and do not perform any other action."
)

func modelConfigGetArgs(nativeID, key string) []string {
	return []string{"--profile", nativeID, "config", "get", key, "--json"}
}

func modelConfigSetArgs(nativeID, key, value string) []string {
	return []string{"--profile", nativeID, "config", "set", key, value}
}

func modelValidationArgs(nativeID string, preset modelProviderPreset, modelID string) []string {
	return []string{
		"--profile", nativeID,
		"--provider", preset.hermesProviderID,
		"--model", modelID,
		"--toolsets", toolsDisabledToolset,
		"--oneshot", modelValidationPrompt,
	}
}
