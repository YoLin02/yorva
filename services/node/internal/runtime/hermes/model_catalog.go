package hermes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	modelCatalogMaxBytes = 2 * 1024 * 1024
	modelCatalogMaxItems = 1000
)

func (m *ModelManager) FetchProviderModels(ctx context.Context, presetID string, credential []byte) ([]string, error) {
	preset, err := lookupModelProviderPreset(presetID)
	if err != nil {
		return nil, err
	}
	if err := validateModelCredential(credential); err != nil {
		return nil, yorvaruntime.ErrModelConfigInvalid
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, preset.catalogURL, nil)
	if err != nil {
		return nil, yorvaruntime.ErrModelCatalogFetchFailed
	}
	req.Header.Set("Accept", "application/json")
	switch preset.catalogAuth {
	case "anthropic":
		req.Header.Set("x-api-key", string(credential))
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+string(credential))
	}
	client := m.catalogClient
	if client == nil {
		client = &http.Client{}
	}
	// The credential is qualified for exactly the compiled Provider endpoint.
	// Never forward it to a redirect target, including custom authentication
	// headers that net/http does not classify as sensitive automatically.
	qualifiedClient := *client
	qualifiedClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	response, err := qualifiedClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, yorvaruntime.ErrModelCatalogFetchFailed
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, yorvaruntime.ErrModelCatalogFetchFailed
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, modelCatalogMaxBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > modelCatalogMaxBytes {
		return nil, yorvaruntime.ErrModelCatalogFetchFailed
	}
	models, err := parseProviderModelCatalog(payload, preset.catalogShape)
	if err != nil || len(models) == 0 {
		return nil, yorvaruntime.ErrModelCatalogFetchFailed
	}
	return models, nil
}

func parseProviderModelCatalog(payload []byte, shape string) ([]string, error) {
	var ids []string
	if shape == "dashscope" {
		var response struct {
			Output struct {
				Models []struct {
					Model string `json:"model"`
				} `json:"models"`
			} `json:"output"`
		}
		if err := json.Unmarshal(payload, &response); err != nil {
			return nil, err
		}
		for _, model := range response.Output.Models {
			ids = append(ids, model.Model)
		}
	} else {
		var response struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(payload, &response); err != nil {
			return nil, err
		}
		for _, model := range response.Data {
			ids = append(ids, model.ID)
		}
	}
	result := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) != id || validateModelID(id) != nil {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
		if len(result) == modelCatalogMaxItems {
			break
		}
	}
	return result, nil
}
