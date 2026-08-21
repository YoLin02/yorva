package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
)

const maxDownloadSourcesRequestBytes = 16 * 1024

type HermesDownloadSourceSettingsService interface {
	Get(context.Context) (downloadsources.Config, error)
	Save(context.Context, downloadsources.Config) (downloadsources.Config, error)
	Reset(context.Context) (downloadsources.Config, error)
}

func getHermesDownloadSources(service HermesDownloadSourceSettingsService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeSettingsUnavailable(w)
			return
		}
		config, err := service.Get(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{
				Code: "SETTINGS_READ_FAILED", Message: "Yorva could not read the Hermes download sources.", Retryable: true,
			})
			return
		}
		writeDownloadSources(w, config)
	})
}

func putHermesDownloadSources(service HermesDownloadSourceSettingsService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeSettingsUnavailable(w)
			return
		}
		config, err := decodeDownloadSources(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{
				Code: "SETTINGS_INVALID", Message: "Every download source must be a credential-free HTTPS URL.", Retryable: false,
			})
			return
		}
		config, err = service.Save(r.Context(), config)
		if errors.Is(err, downloadsources.ErrInvalid) {
			writeError(w, http.StatusBadRequest, ErrorBody{
				Code: "SETTINGS_INVALID", Message: "Every download source must be a credential-free HTTPS URL.", Retryable: false,
			})
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{
				Code: "SETTINGS_WRITE_FAILED", Message: "Yorva could not save the Hermes download sources.", Retryable: true,
			})
			return
		}
		writeDownloadSources(w, config)
	})
}

func deleteHermesDownloadSources(service HermesDownloadSourceSettingsService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeSettingsUnavailable(w)
			return
		}
		config, err := service.Reset(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{
				Code: "SETTINGS_WRITE_FAILED", Message: "Yorva could not restore the default Hermes download sources.", Retryable: true,
			})
			return
		}
		writeDownloadSources(w, config)
	})
}

func decodeDownloadSources(r *http.Request) (downloadsources.Config, error) {
	if r.Body == nil {
		return downloadsources.Config{}, io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxDownloadSourcesRequestBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > maxDownloadSourcesRequestBytes {
		return downloadsources.Config{}, io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var config downloadsources.Config
	if err := decoder.Decode(&config); err != nil {
		return downloadsources.Config{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return downloadsources.Config{}, errors.New("trailing json")
	}
	return downloadsources.Normalize(config)
}

func writeDownloadSources(w http.ResponseWriter, config downloadsources.Config) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

func writeSettingsUnavailable(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, ErrorBody{
		Code: "SETTINGS_UNAVAILABLE", Message: "Hermes download source settings are unavailable.", Retryable: true,
	})
}
