package runtime

import (
	"context"
	"errors"
)

type ModelRegion string

const (
	ModelRegionChina  ModelRegion = "CHINA"
	ModelRegionGlobal ModelRegion = "GLOBAL"
)

type ModelConfigurationState string

const (
	ModelConfigurationUnconfigured ModelConfigurationState = "UNCONFIGURED"
	ModelConfigurationConfigured   ModelConfigurationState = "CONFIGURED"
)

type ModelProviderPreset struct {
	ID                string
	DisplayName       string
	Region            ModelRegion
	RecommendedModels []string
	HelpText          string
}

type ModelInstallation struct {
	Executable string
	Version    string
}

type ModelConfiguration struct {
	ProviderPresetID     string
	ModelID              string
	State                ModelConfigurationState
	CredentialConfigured bool
}

type ModelCredentialStatus struct {
	ProviderPresetID string
	Configured       bool
}

var (
	ErrModelProviderUnsupported    = errors.New("model provider is unsupported")
	ErrModelConfigInvalid          = errors.New("model configuration is invalid")
	ErrModelConfigQueryFailed      = errors.New("model configuration query failed")
	ErrModelConfigApplyFailed      = errors.New("model configuration apply failed")
	ErrModelConfigIncomplete       = errors.New("model configuration is incomplete")
	ErrModelCredentialRequired     = errors.New("model credential is required")
	ErrModelCredentialQueryFailed  = errors.New("model credential query failed")
	ErrModelCredentialWriteFailed  = errors.New("model credential write failed")
	ErrModelCredentialDeleteFailed = errors.New("model credential delete failed")
	ErrInstanceConfigConflict      = errors.New("instance configuration changed concurrently")
)

type ModelConfigurator interface {
	ListProviderPresets() []ModelProviderPreset
	ReadModelConfig(context.Context, ModelInstallation, string) (ModelConfiguration, error)
	ApplyModelConfig(context.Context, ModelInstallation, string, string, string) (ModelConfiguration, error)
	ModelCredentialStatus(context.Context, ModelInstallation, string, string) (ModelCredentialStatus, error)
	SetModelCredential(context.Context, ModelInstallation, string, string, []byte) (ModelCredentialStatus, error)
	DeleteModelCredential(context.Context, ModelInstallation, string, string) (ModelCredentialStatus, error)
}
