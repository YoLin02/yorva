package hermes

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

func CreateProfile(ctx context.Context, executable, name string) error {
	return createProfileWith(ctx, executable, name, runProfileCreateCommand)
}

func createProfileWith(ctx context.Context, executable, name string, run func(context.Context, string, string, string) commandResult) error {
	if executable == "" || !filepath.IsAbs(executable) {
		return errProfileExecutableUnsafe
	}
	if err := validateYORVACreateProfileName(name); err != nil {
		return err
	}
	home := officialHermesHome()
	if home == "" || !filepath.IsAbs(home) {
		return errProfileHomeUnavailable
	}
	result := run(ctx, executable, home, name)
	if result.limited {
		return fmt.Errorf("%w: %w", errProfileQueryFailed, errOutputLimit)
	}
	if result.timedOut || errors.Is(result.err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %w", errProfileQueryFailed, context.DeadlineExceeded)
	}
	if errors.Is(result.err, context.Canceled) {
		return result.err
	}
	if result.err != nil || result.exitCode != 0 {
		return errProfileQueryFailed
	}
	return nil
}

func runProfileCreateCommand(ctx context.Context, executable, home, name string) commandResult {
	runner := newCommandRunner()
	runner.environment = func() []string { return profileCommandEnvironment(home) }
	return runner.run(ctx, commandInvocation{
		path:       executable,
		executable: executable,
		args:       profileCreateArgs(name),
	})
}
