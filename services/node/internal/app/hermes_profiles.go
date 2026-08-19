package app

import (
	"context"

	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
)

type HermesProfileSource struct{}

func (HermesProfileSource) Create(ctx context.Context, executable, name string) error {
	if err := hermes.CreateProfile(ctx, executable, name); err != nil {
		if hermes.IsProfileOutputUnrecognized(err) {
			return ErrInstanceOutputUnrecognized
		}
		return ErrInstanceQueryFailed
	}
	return nil
}

func (HermesProfileSource) List(ctx context.Context, executable string) ([]ProfileSnapshot, error) {
	natives, err := hermes.ListProfiles(ctx, executable)
	if hermes.IsProfileOutputUnrecognized(err) {
		return nil, ErrInstanceOutputUnrecognized
	}
	if err != nil {
		return nil, ErrInstanceQueryFailed
	}
	out := make([]ProfileSnapshot, 0, len(natives))
	for _, native := range natives {
		out = append(out, ProfileSnapshot{NativeID: native.NativeID, Default: native.Default})
	}
	return out, nil
}
