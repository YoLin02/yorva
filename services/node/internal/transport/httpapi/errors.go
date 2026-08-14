package httpapi

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func writeError(w http.ResponseWriter, status int, body ErrorBody) {
	if body.Details == nil {
		body.Details = map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: body})
}
