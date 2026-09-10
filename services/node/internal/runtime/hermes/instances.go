package hermes

import (
	"context"
	"errors"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// InstanceManager translates Hermes Profile semantics at the Runtime boundary.
type InstanceManager struct{}

func (InstanceManager) ValidateName(name string) error {
	if err := ValidateCreateProfileName(name); err != nil {
		return yorvaruntime.ErrInstanceNameInvalid
	}
	return nil
}

func (InstanceManager) List(ctx context.Context, executable string) ([]yorvaruntime.NativeInstance, error) {
	natives, err := ListProfiles(ctx, executable)
	if err != nil {
		return nil, normalizeInstanceError(err)
	}
	out := make([]yorvaruntime.NativeInstance, 0, len(natives))
	for _, native := range natives {
		out = append(out, yorvaruntime.NativeInstance{NativeID: native.NativeID, Default: native.Default, Protected: native.Default})
	}
	return out, nil
}

func (InstanceManager) Create(ctx context.Context, executable, name string) error {
	return normalizeInstanceError(CreateProfile(ctx, executable, name))
}

func (InstanceManager) Delete(ctx context.Context, executable, nativeID string) error {
	return normalizeInstanceError(DeleteProfile(ctx, executable, nativeID))
}

func normalizeInstanceError(err error) error {
	switch {
	case err == nil:
		return nil
	case IsProfileOutputUnrecognized(err), errors.Is(err, yorvaruntime.ErrInstanceOutputUnrecognized):
		return yorvaruntime.ErrInstanceOutputUnrecognized
	case errors.Is(err, context.DeadlineExceeded):
		return yorvaruntime.ErrInstanceOperationTimedOut
	case errors.Is(err, context.Canceled):
		return context.Canceled
	default:
		return yorvaruntime.ErrInstanceInventoryFailed
	}
}
