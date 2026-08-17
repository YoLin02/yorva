package app

import (
	"context"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
)

type HermesPrerequisiteHost struct {
	Host *hermes.NodeHost
}

func (a HermesPrerequisiteHost) Inspect() PrerequisiteSnapshot {
	if a.Host == nil {
		return PrerequisiteSnapshot{}
	}
	got := a.Host.Inspect()
	return PrerequisiteSnapshot{
		NodeState:   string(got.Node.State),
		NodeVersion: got.Node.Version,
		NodeCode:    got.Node.ErrorCode,
		NPMState:    string(got.NPM.State),
		NPMVersion:  got.NPM.Version,
		NPMCode:     got.NPM.ErrorCode,
		DepsState:   string(got.NodeDependencies.State),
		DepsCode:    got.NodeDependencies.ErrorCode,
		Retryable:   got.Node.Retryable || got.NPM.Retryable || got.NodeDependencies.Retryable,
		CheckedAt:   got.CheckedAt,
	}
}

func (a HermesPrerequisiteHost) Apply(ctx context.Context, operationID string, report func(operation.Stage, string)) error {
	if a.Host == nil {
		return ErrRuntimeKindNotFound
	}
	return a.Host.Apply(ctx, operationID, report)
}
