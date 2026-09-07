package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/YoLin02/yorva/services/node/internal/diagnostics"
)

type DiagnosticBundleService interface {
	Build(context.Context) (diagnostics.Bundle, error)
}

func exportDiagnosticBundle(service DiagnosticBundleService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: "DIAGNOSTICS_UNAVAILABLE", Message: "The diagnostic export service is unavailable.", Retryable: true})
			return
		}
		bundle, err := service.Build(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "DIAGNOSTICS_EXPORT_FAILED", Message: "The sanitized diagnostic bundle could not be created.", Retryable: true})
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+bundle.FileName+`"`)
		w.Header().Set("Content-Length", strconv.Itoa(len(bundle.Bytes)))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(bundle.Bytes)
	})
}
