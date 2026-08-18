package app

import (
	"context"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
)

func (s *RuntimeInstall) persistCreate(ctx context.Context, created operation.Operation) error {
	if err := s.store.CreateOperation(ctx, created); err != nil {
		return err
	}
	s.emitOperation(operation.Operation{}, created, true)
	return nil
}

func (s *RuntimeInstall) persistUpdate(ctx context.Context, current, next operation.Operation) error {
	if err := s.store.UpdateOperation(ctx, current, next); err != nil {
		return err
	}
	s.emitOperation(current, next, false)
	return nil
}

func (s *RuntimeInstall) emitOperation(previous, next operation.Operation, created bool) {
	if s == nil || s.events == nil {
		return
	}
	eventType := events.TypeForCommittedOperation(created, string(previous.Status), string(next.Status))
	if eventType == "" {
		return
	}
	payload := events.OperationPayload{
		OperationID:   next.ID,
		Type:          string(next.Type),
		Status:        string(next.Status),
		Stage:         string(next.Stage),
		CorrelationID: next.CorrelationID,
	}
	if next.ErrorCode != "" {
		payload.ErrorCode = string(next.ErrorCode)
	}
	s.events.Publish(events.NewOperationEvent(eventType, payload, s.now()))
}
