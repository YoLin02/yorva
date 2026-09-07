package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type NodeRecoveryService interface {
	CheckNodeRecovery(context.Context) (app.NodeRecovery, error)
}

type NodeRecoveryResponse struct {
	State       string                  `json:"state"`
	NodeVersion string                  `json:"nodeVersion"`
	ErrorCode   *yorvaruntime.ErrorCode `json:"errorCode"`
}

func getNodeRecovery(recovery NodeRecoveryService, nodeVersion string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
		defer cancel()
		if recovery == nil {
			writeNodeRecoveryError(w)
			return
		}
		result, err := recovery.CheckNodeRecovery(ctx)
		if err != nil {
			writeNodeRecoveryError(w)
			return
		}
		state := "RECOVERY_REQUIRED"
		if result.Ready && result.ErrorCode == "" {
			state = "READY"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(NodeRecoveryResponse{
			State: state, NodeVersion: nodeVersion, ErrorCode: nullableErrorCode(result.ErrorCode),
		})
	})
}

func writeNodeRecoveryError(w http.ResponseWriter) {
	writeError(w, http.StatusServiceUnavailable, ErrorBody{
		Code: "NODE_RECOVERY_FAILED", Message: "The local recovery check could not complete.", Retryable: true,
	})
}
