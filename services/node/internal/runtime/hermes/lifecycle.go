package hermes

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	lifecycleMutationTimeout = 55 * time.Second
	lifecycleStartTimeout    = 35 * time.Second
	lifecyclePollInterval    = 250 * time.Millisecond
	lifecyclePostcondition   = 15 * time.Second
)

type lifecycleObservation struct {
	state            yorvaruntime.LifecycleState
	loginItemPresent bool
}

type LifecycleManager struct {
	run func(context.Context, string, []string, bool) commandResult
}

func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{run: runLifecycleCommand}
}

func (m *LifecycleManager) Status(ctx context.Context, installation yorvaruntime.LifecycleInstallation, nativeID string) (yorvaruntime.LifecycleStatus, error) {
	observation, err := m.observe(ctx, installation, nativeID)
	return yorvaruntime.LifecycleStatus{State: observation.state}, err
}

func (m *LifecycleManager) Start(ctx context.Context, installation yorvaruntime.LifecycleInstallation, nativeID string) error {
	before, err := m.observe(ctx, installation, nativeID)
	if err != nil {
		return err
	}
	if before.state == yorvaruntime.LifecycleRunning {
		return nil
	}
	if err := m.mutate(ctx, installation, nativeID, lifecycleStartArgs(nativeID, before.loginItemPresent), true); err != nil {
		return err
	}
	return m.await(ctx, installation, nativeID, yorvaruntime.LifecycleRunning)
}

func (m *LifecycleManager) Stop(ctx context.Context, installation yorvaruntime.LifecycleInstallation, nativeID string) error {
	before, err := m.observe(ctx, installation, nativeID)
	if err != nil {
		return err
	}
	if before.state == yorvaruntime.LifecycleStopped {
		return nil
	}
	if err := m.mutate(ctx, installation, nativeID, lifecycleStopArgs(nativeID), false); err != nil {
		return err
	}
	return m.await(ctx, installation, nativeID, yorvaruntime.LifecycleStopped)
}

func (m *LifecycleManager) Restart(ctx context.Context, installation yorvaruntime.LifecycleInstallation, nativeID string) error {
	before, err := m.observe(ctx, installation, nativeID)
	if err != nil {
		return err
	}
	if before.state != yorvaruntime.LifecycleRunning {
		return yorvaruntime.ErrInstanceNotRunning
	}
	if !before.loginItemPresent {
		if err := m.mutate(ctx, installation, nativeID, lifecycleStopArgs(nativeID), false); err != nil {
			return err
		}
		if err := m.await(ctx, installation, nativeID, yorvaruntime.LifecycleStopped); err != nil {
			return err
		}
		if err := m.mutate(ctx, installation, nativeID, lifecycleStartArgs(nativeID, false), true); err != nil {
			return err
		}
		return m.await(ctx, installation, nativeID, yorvaruntime.LifecycleRunning)
	}
	if err := m.mutate(ctx, installation, nativeID, lifecycleRestartArgs(nativeID), true); err != nil {
		return err
	}
	return m.await(ctx, installation, nativeID, yorvaruntime.LifecycleRunning)
}

func (m *LifecycleManager) observe(ctx context.Context, installation yorvaruntime.LifecycleInstallation, nativeID string) (lifecycleObservation, error) {
	if err := validateLifecycleTarget(installation, nativeID); err != nil {
		return lifecycleObservation{state: yorvaruntime.LifecycleUnknown}, err
	}
	result := m.run(ctx, installation.Executable, lifecycleStatusArgs(nativeID), false)
	if result.limited {
		return lifecycleObservation{state: yorvaruntime.LifecycleUnknown}, fmt.Errorf("%w: %w", yorvaruntime.ErrLifecycleQueryFailed, errOutputLimit)
	}
	if result.timedOut || errors.Is(result.err, context.DeadlineExceeded) {
		return lifecycleObservation{state: yorvaruntime.LifecycleUnknown}, fmt.Errorf("%w: %w", yorvaruntime.ErrLifecycleQueryFailed, context.DeadlineExceeded)
	}
	if result.err != nil || result.exitCode != 0 || strings.TrimSpace(result.stderr) != "" {
		return lifecycleObservation{state: yorvaruntime.LifecycleUnknown}, yorvaruntime.ErrLifecycleQueryFailed
	}
	return parseLifecycleStatus(result.stdout)
}

func (m *LifecycleManager) mutate(ctx context.Context, installation yorvaruntime.LifecycleInstallation, nativeID string, args []string, allowBreakaway bool) error {
	if err := validateLifecycleTarget(installation, nativeID); err != nil {
		return err
	}
	result := m.run(ctx, installation.Executable, args, allowBreakaway)
	if result.limited || result.timedOut || result.err != nil || result.exitCode != 0 {
		return yorvaruntime.ErrLifecycleMutationFailed
	}
	return nil
}

func (m *LifecycleManager) await(ctx context.Context, installation yorvaruntime.LifecycleInstallation, nativeID string, expected yorvaruntime.LifecycleState) error {
	deadline := time.NewTimer(lifecyclePostcondition)
	defer deadline.Stop()
	ticker := time.NewTicker(lifecyclePollInterval)
	defer ticker.Stop()
	for {
		status, err := m.Status(ctx, installation, nativeID)
		if err == nil && status.State == expected {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return yorvaruntime.ErrLifecyclePostcondition
		case <-ticker.C:
		}
	}
}

func validateLifecycleTarget(installation yorvaruntime.LifecycleInstallation, nativeID string) error {
	if installation.Version != lifecycleOfficialVersion || installation.Executable == "" || !filepath.IsAbs(installation.Executable) {
		return yorvaruntime.ErrLifecycleQueryFailed
	}
	normalized, err := officialNormalizeProfileName(nativeID)
	if err != nil || normalized != nativeID || officialValidateProfileName(nativeID) != nil {
		return yorvaruntime.ErrLifecycleQueryFailed
	}
	return nil
}

func parseLifecycleStatus(output string) (lifecycleObservation, error) {
	running := strings.Contains(output, lifecycleStatusRunning)
	stopped := strings.Contains(output, lifecycleStatusStopped)
	installed := strings.Contains(output, lifecycleTaskInstalled) || strings.Contains(output, lifecycleStartupInstalled)
	serviceMissing := strings.Contains(output, lifecycleServiceMissing)
	if running != stopped && installed != serviceMissing {
		state := yorvaruntime.LifecycleStopped
		if running {
			state = yorvaruntime.LifecycleRunning
		}
		return lifecycleObservation{state: state, loginItemPresent: installed}, nil
	}

	manualRunning := strings.Contains(output, lifecycleManualRunning)
	manualStopped := strings.Contains(output, lifecycleManualStopped)
	if !running && !stopped && !installed && !serviceMissing && manualRunning != manualStopped {
		state := yorvaruntime.LifecycleStopped
		if manualRunning {
			state = yorvaruntime.LifecycleRunning
		}
		return lifecycleObservation{state: state, loginItemPresent: false}, nil
	}
	return lifecycleObservation{state: yorvaruntime.LifecycleUnknown}, yorvaruntime.ErrLifecycleOutputUnrecognized
}

func runLifecycleCommand(ctx context.Context, executable string, args []string, allowBreakaway bool) commandResult {
	runner := newCommandRunner()
	runner.timeout = lifecycleStartTimeout
	if containsLifecycleVerb(args, "stop") || containsLifecycleVerb(args, "restart") {
		runner.timeout = lifecycleMutationTimeout
	}
	runner.allowBreakaway = allowBreakaway
	runner.environment = func() []string { return profileCommandEnvironment(officialHermesHome()) }
	return runner.run(ctx, commandInvocation{path: executable, executable: executable, args: args})
}

func containsLifecycleVerb(args []string, verb string) bool {
	for index := 0; index+1 < len(args); index++ {
		if args[index] == "gateway" && args[index+1] == verb {
			return true
		}
	}
	return false
}
