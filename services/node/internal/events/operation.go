package events

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	TypeOperationStarted   = "operation.started"
	TypeOperationProgress  = "operation.progress"
	TypeOperationCompleted = "operation.completed"
	TypeOperationFailed    = "operation.failed"
	TypeOperationCancelled = "operation.cancelled"
)

// OperationPayload is the only SSE body published for Operation transitions.
// It is intentionally closed: no message, stdout, stderr, URL, path or raw error.
type OperationPayload struct {
	OperationID   string `json:"operationId"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	Stage         string `json:"stage"`
	ErrorCode     string `json:"errorCode,omitempty"`
	CorrelationID string `json:"correlationId"`
}

func NewOperationEvent(eventType string, payload OperationPayload, now time.Time) Event {
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte("{}")
	}
	occurred := now.UTC()
	if occurred.IsZero() {
		occurred = time.Now().UTC()
	}
	return Event{
		Type:       eventType,
		OccurredAt: occurred,
		Data:       data,
	}
}

func TypeForCommittedOperation(created bool, previousStatus, nextStatus string) string {
	switch nextStatus {
	case "SUCCEEDED":
		return TypeOperationCompleted
	case "FAILED":
		return TypeOperationFailed
	case "CANCELLED":
		return TypeOperationCancelled
	}
	if created {
		return TypeOperationStarted
	}
	return TypeOperationProgress
}

func formatEventID(seq uint64) string {
	return fmt.Sprintf("evt_%d", seq)
}
