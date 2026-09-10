package runtime

import (
	"context"
	"errors"
)

// NativeInstance is an authoritative adapter observation, not a transport DTO.
// Protection and default status are independent: an external instance can be
// protected without being the Runtime default.
type NativeInstance struct {
	NativeID  string
	Default   bool
	Protected bool
}

type InstanceManager interface {
	List(context.Context, string) ([]NativeInstance, error)
	ValidateName(string) error
	Create(context.Context, string, string) error
	Delete(context.Context, string, string) error
}

var (
	ErrInstanceNameInvalid        = errors.New("instance name is invalid")
	ErrInstanceInventoryFailed    = errors.New("instance query failed")
	ErrInstanceOutputUnrecognized = errors.New("instance output unrecognized")
	ErrInstanceOperationTimedOut  = errors.New("instance operation timed out")
)
