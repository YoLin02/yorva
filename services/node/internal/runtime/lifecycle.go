package runtime

import (
	"context"
	"errors"
)

type LifecycleState string

const (
	LifecycleRunning LifecycleState = "RUNNING"
	LifecycleStopped LifecycleState = "STOPPED"
	LifecycleUnknown LifecycleState = "UNKNOWN"
)

type LifecycleInstallation struct {
	Executable string
	Version    string
}

type LifecycleStatus struct {
	State LifecycleState
}

var (
	ErrLifecycleQueryFailed        = errors.New("lifecycle status query failed")
	ErrLifecycleOutputUnrecognized = errors.New("lifecycle status output unrecognized")
	ErrLifecycleMutationFailed     = errors.New("lifecycle mutation failed")
	ErrLifecyclePostcondition      = errors.New("lifecycle postcondition failed")
	ErrInstanceNotRunning          = errors.New("instance is not running")
)

type LifecycleManager interface {
	Status(context.Context, LifecycleInstallation, string) (LifecycleStatus, error)
	Start(context.Context, LifecycleInstallation, string) error
	Stop(context.Context, LifecycleInstallation, string) error
	Restart(context.Context, LifecycleInstallation, string) error
}
