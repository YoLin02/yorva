package openclaw

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func (a *Adapter) lifecycleTarget(ctx context.Context, installation yorvaruntime.LifecycleInstallation, profile string) (string, error) {
	if installation.Version != supportedVersion {
		return "", yorvaruntime.ErrLifecycleMutationFailed
	}
	node, err := a.qualifiedNode(installation.Executable)
	if err != nil {
		return "", yorvaruntime.ErrLifecycleMutationFailed
	}
	if err := a.validateOwnedConfiguration(ctx, node, installation.Executable, profile); err != nil {
		return "", lifecycleError(err)
	}
	return node, nil
}

func (a *Adapter) Start(ctx context.Context, installation yorvaruntime.LifecycleInstallation, profile string) error {
	node, err := a.lifecycleTarget(ctx, installation, profile)
	if err != nil {
		return err
	}
	return a.start(ctx, node, installation.Executable, profile)
}

func (a *Adapter) Stop(ctx context.Context, installation yorvaruntime.LifecycleInstallation, profile string) error {
	node, err := a.lifecycleTarget(ctx, installation, profile)
	if err != nil {
		return err
	}
	return a.stop(ctx, node, installation.Executable, profile)
}

func (a *Adapter) Restart(ctx context.Context, installation yorvaruntime.LifecycleInstallation, profile string) error {
	node, err := a.lifecycleTarget(ctx, installation, profile)
	if err != nil {
		return err
	}
	status, err := a.status(ctx, node, installation.Executable, profile)
	if err != nil {
		return err
	}
	if status.State != yorvaruntime.LifecycleRunning {
		return yorvaruntime.ErrInstanceNotRunning
	}
	if err := a.stop(ctx, node, installation.Executable, profile); err != nil {
		return err
	}
	return a.start(ctx, node, installation.Executable, profile)
}

func (a *Adapter) start(ctx context.Context, node, entry, profile string) error {
	status, err := a.status(ctx, node, entry, profile)
	if err != nil {
		return err
	}
	if status.State == yorvaruntime.LifecycleRunning {
		return nil
	}
	if status.State != yorvaruntime.LifecycleStopped {
		return yorvaruntime.ErrLifecycleQueryFailed
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.Command(node, entry, "--profile", profile, "gateway", "run")
	cmd.Env, cmd.Dir = a.environment(node, profile), a.home
	// No pipes or output-reading goroutines survive a successful handoff.
	// OpenClaw owns its own profile-scoped logs.
	configureProcess(cmd)
	if err := ctx.Err(); err != nil {
		return err
	}
	if cmd.Start() != nil {
		return yorvaruntime.ErrLifecycleMutationFailed
	}
	job, err := ownProcess(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return yorvaruntime.ErrLifecycleMutationFailed
	}
	handedOff := false
	defer func() {
		if !handedOff {
			job.close()
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	for {
		if !processRunning(cmd) {
			return yorvaruntime.ErrLifecycleMutationFailed
		}
		observed, statusErr := a.status(ctx, node, entry, profile)
		if statusErr == nil && observed.State == yorvaruntime.LifecycleRunning {
			if err := ctx.Err(); err != nil {
				return err
			}
			if job.detach() != nil {
				return yorvaruntime.ErrLifecycleMutationFailed
			}
			handedOff = true
			_ = cmd.Process.Release()
			return nil
		}
		if err := waitProbe(ctx); err != nil {
			return err
		}
	}
}

func (a *Adapter) stop(ctx context.Context, node, entry, profile string) error {
	status, err := a.status(ctx, node, entry, profile)
	if err != nil {
		return err
	}
	if status.State == yorvaruntime.LifecycleStopped {
		return nil
	}
	// Require authenticated ownership before asking the official CLI to stop
	// its verified unmanaged listener. A busy unauthenticated port is UNKNOWN.
	if status.State != yorvaruntime.LifecycleRunning {
		return yorvaruntime.ErrLifecycleQueryFailed
	}
	output, err := a.command(ctx, node, entry, profile, 35*time.Second, "gateway", "stop", "--force", "--json")
	defer clear(output)
	if err != nil {
		return yorvaruntime.ErrLifecycleMutationFailed
	}
	var result struct {
		OK     *bool  `json:"ok"`
		Action string `json:"action"`
		Result string `json:"result"`
	}
	if json.Unmarshal(output, &result) != nil || result.OK == nil || !*result.OK || result.Action != "stop" || result.Result != "stopped" {
		return yorvaruntime.ErrLifecycleOutputUnrecognized
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	for {
		observed, statusErr := a.status(ctx, node, entry, profile)
		if statusErr == nil && observed.State == yorvaruntime.LifecycleStopped {
			return nil
		}
		if err := waitProbe(ctx); err != nil {
			return yorvaruntime.ErrLifecyclePostcondition
		}
	}
}

func waitProbe(ctx context.Context) error {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
