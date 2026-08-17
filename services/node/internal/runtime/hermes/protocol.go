package hermes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const stageResultLimit = 64 * 1024

type manifestStage struct {
	Name           string `json:"name"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	NeedsUserInput bool   `json:"needs_user_input"`
}

type installerManifest struct {
	ProtocolVersion int             `json:"protocol_version"`
	Stages          []manifestStage `json:"stages"`
}

type stageResult struct {
	Stage      string `json:"stage"`
	OK         bool   `json:"ok"`
	Skipped    bool   `json:"skipped"`
	Reason     string `json:"reason"`
	DurationMS int64  `json:"duration_ms"`
}

func reviewedManifest() installerManifest {
	return installerManifest{
		ProtocolVersion: officialProtocol,
		Stages: []manifestStage{
			{Name: "uv", Title: "Installing uv package manager", Category: "prereqs", NeedsUserInput: false},
			{Name: "python", Title: "Verifying Python", Category: "prereqs", NeedsUserInput: false},
			{Name: "git", Title: "Installing Git", Category: "prereqs", NeedsUserInput: false},
			{Name: "node", Title: "Detecting Node.js", Category: "prereqs", NeedsUserInput: false},
			{Name: "system-packages", Title: "Installing ripgrep and ffmpeg", Category: "prereqs", NeedsUserInput: false},
			{Name: "repository", Title: "Cloning Hermes repository", Category: "install", NeedsUserInput: false},
			{Name: "venv", Title: "Creating Python virtual environment", Category: "install", NeedsUserInput: false},
			{Name: "dependencies", Title: "Installing Python dependencies", Category: "install", NeedsUserInput: false},
			{Name: "node-deps", Title: "Installing Node.js dependencies", Category: "install", NeedsUserInput: false},
			{Name: "path", Title: "Adding Hermes to PATH", Category: "finalize", NeedsUserInput: false},
			{Name: "config-templates", Title: "Writing configuration templates", Category: "finalize", NeedsUserInput: false},
			{Name: "platform-sdks", Title: "Installing messaging platform SDKs", Category: "finalize", NeedsUserInput: false},
			{Name: "bootstrap-marker", Title: "Marking install complete", Category: "finalize", NeedsUserInput: false},
			{Name: "configure", Title: "Configuring API keys and models", Category: "post-install", NeedsUserInput: true},
			{Name: "gateway", Title: "Starting messaging gateway", Category: "post-install", NeedsUserInput: true},
		},
	}
}

func approvedInstallStages() []string {
	return []string{
		"uv", "python", "git", "node", "system-packages", "repository",
		"venv", "dependencies", "node-deps", "path", "config-templates", "bootstrap-marker",
	}
}

func excludedInstallStages() []string {
	return []string{"desktop", "platform-sdks", "configure", "gateway"}
}

func yorvaOwnedOfficialStages() []string {
	return []string{"node", "node-deps"}
}

func parseProtocolVersion(output string) error {
	trimmed := strings.TrimSpace(output)
	if trimmed != "1" {
		return installError(yorvaruntime.ErrorRuntimeInstallProtocolUnsupported, fmt.Errorf("protocol %q", trimmed))
	}
	return nil
}

func parseAndValidateManifest(output string) error {
	frame, err := lastJSONObject(output)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallManifestMismatch, err)
	}
	var got installerManifest
	decoder := json.NewDecoder(bytes.NewReader(frame))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&got); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallManifestMismatch, err)
	}
	want := reviewedManifest()
	if got.ProtocolVersion != want.ProtocolVersion || len(got.Stages) != len(want.Stages) {
		return installError(yorvaruntime.ErrorRuntimeInstallManifestMismatch, errors.New("manifest protocol or length changed"))
	}
	seen := make(map[string]struct{}, len(got.Stages))
	for index, stage := range got.Stages {
		if _, exists := seen[stage.Name]; exists || stage.Name == "" {
			return installError(yorvaruntime.ErrorRuntimeInstallManifestMismatch, errors.New("duplicate or empty stage"))
		}
		seen[stage.Name] = struct{}{}
		expected := want.Stages[index]
		if stage.Name != expected.Name || stage.Category != expected.Category || stage.NeedsUserInput != expected.NeedsUserInput {
			return installError(yorvaruntime.ErrorRuntimeInstallManifestMismatch, fmt.Errorf("stage %d changed", index))
		}
	}
	if _, found := seen["desktop"]; found {
		return installError(yorvaruntime.ErrorRuntimeInstallManifestMismatch, errors.New("desktop stage must be absent"))
	}
	return nil
}

func parseStageResult(requested string, output string) (stageResult, error) {
	frame, err := lastJSONObject(output)
	if err != nil {
		return stageResult{}, installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	var result stageResult
	if err := json.Unmarshal(frame, &result); err != nil {
		return stageResult{}, installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if result.Stage != requested {
		return stageResult{}, installError(yorvaruntime.ErrorRuntimeInstallStageFailed, fmt.Errorf("result stage %q", result.Stage))
	}
	return result, nil
}

func lastJSONObject(output string) ([]byte, error) {
	if len(output) > stageResultLimit {
		return nil, errors.New("installer output exceeded parse bound")
	}
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, errors.New("installer produced no JSON frame")
	}
	lines := strings.Split(trimmed, "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		candidate := strings.TrimSpace(lines[index])
		if candidate == "" {
			continue
		}
		if !json.Valid([]byte(candidate)) || !strings.HasPrefix(candidate, "{") {
			return nil, errors.New("final installer frame is not a JSON object")
		}
		return []byte(candidate), nil
	}
	return nil, errors.New("installer produced no JSON frame")
}

func isApprovedStage(name string) bool {
	for _, stage := range approvedInstallStages() {
		if stage == name {
			return true
		}
	}
	return false
}

func isExcludedStage(name string) bool {
	for _, stage := range excludedInstallStages() {
		if stage == name {
			return true
		}
	}
	return false
}

func isYorvaOwnedOfficialStage(name string) bool {
	for _, stage := range yorvaOwnedOfficialStages() {
		if stage == name {
			return true
		}
	}
	return false
}
