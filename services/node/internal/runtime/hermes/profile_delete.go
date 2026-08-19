package hermes

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

func DeleteProfile(ctx context.Context, executable, nativeID string) error {
	return deleteProfileWith(ctx, executable, nativeID, runProfileDeleteCommand)
}

func deleteProfileWith(ctx context.Context, executable, nativeID string, run func(context.Context, string, string, string) commandResult) error {
	if executable == "" || !filepath.IsAbs(executable) {
		return errProfileExecutableUnsafe
	}
	if nativeID == "default" {
		return errProfileNameInvalid
	}
	if err := officialValidateProfileName(nativeID); err != nil {
		return err
	}
	home := officialHermesHome()
	if home == "" || !filepath.IsAbs(home) {
		return errProfileHomeUnavailable
	}
	result := run(ctx, executable, home, nativeID)
	if result.limited {
		return fmt.Errorf("%w: %w", errProfileQueryFailed, errOutputLimit)
	}
	if result.timedOut || errors.Is(result.err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %w", errProfileQueryFailed, context.DeadlineExceeded)
	}
	if result.err != nil && !errors.Is(result.err, context.Canceled) {
		return errProfileQueryFailed
	}
	if result.err != nil {
		return result.err
	}
	if result.exitCode != 0 {
		return errProfileQueryFailed
	}
	return nil
}

func runProfileDeleteCommand(ctx context.Context, executable, home, nativeID string) commandResult {
	runner := newCommandRunner()
	runner.environment = func() []string { return profileCommandEnvironment(home) }
	return runner.run(ctx, commandInvocation{
		path:       executable,
		executable: executable,
		args:       profileDeleteArgs(nativeID),
	})
}
