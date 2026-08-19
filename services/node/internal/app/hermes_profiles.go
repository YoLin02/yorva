package app

import (
	"context"
	"errors"

	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
)

type HermesProfileSource struct{}

func (HermesProfileSource) Create(ctx context.Context, executable, name string) error {
	if err := hermes.CreateProfile(ctx, executable, name); err != nil {
		return classifyHermesProfileError(err)
	}
	return nil
}

func (HermesProfileSource) Delete(ctx context.Context, executable, nativeID string) error {
	if err := hermes.DeleteProfile(ctx, executable, nativeID); err != nil {
		return classifyHermesProfileError(err)
	}
	return nil
}

func (HermesProfileSource) List(ctx context.Context, executable string) ([]ProfileSnapshot, error) {
	natives, err := hermes.ListProfiles(ctx, executable)
	if err != nil {
		return nil, classifyHermesProfileError(err)
	}
	out := make([]ProfileSnapshot, 0, len(natives))
	for _, native := range natives {
		out = append(out, ProfileSnapshot{NativeID: native.NativeID, Default: native.Default})
	}
	return out, nil
}

func classifyHermesProfileError(err error) error {
	if err == nil {
		return nil
	}
	if hermes.IsProfileOutputUnrecognized(err) || errors.Is(err, ErrInstanceOutputUnrecognized) {
		return ErrInstanceOutputUnrecognized
	}
	if errors.Is(err, ErrInstanceOperationTimedOut) || errors.Is(err, context.DeadlineExceeded) {
		return ErrInstanceOperationTimedOut
	}
	if errors.Is(err, ErrInstanceQueryFailed) {
		return ErrInstanceQueryFailed
	}
	return ErrInstanceQueryFailed
}
