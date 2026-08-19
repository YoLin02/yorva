package hermes

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type NativeProfile struct {
	NativeID string
	Default  bool
}

var (
	errProfileQueryFailed      = errors.New("hermes profile query failed")
	errProfileHomeUnavailable  = errors.New("hermes home is unavailable")
	errProfileExecutableUnsafe = errors.New("hermes profile executable is unsafe")
)

func ListProfiles(ctx context.Context, executable string) ([]NativeProfile, error) {
	return listProfilesWith(ctx, executable, runProfileListCommand)
}

func listProfilesWith(ctx context.Context, executable string, run func(context.Context, string, string) commandResult) ([]NativeProfile, error) {
	if executable == "" || !filepath.IsAbs(executable) {
		return nil, errProfileExecutableUnsafe
	}
	home := officialHermesHome()
	if home == "" || !filepath.IsAbs(home) {
		return nil, errProfileHomeUnavailable
	}
	result := run(ctx, executable, home)
	if result.limited {
		return nil, fmt.Errorf("%w: %w", errProfileQueryFailed, errOutputLimit)
	}
	if result.timedOut || errors.Is(result.err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("%w: %w", errProfileQueryFailed, context.DeadlineExceeded)
	}
	if result.err != nil && !errors.Is(result.err, context.Canceled) {
		return nil, errProfileQueryFailed
	}
	if result.err != nil {
		return nil, result.err
	}
	if result.exitCode != 0 {
		return nil, errProfileQueryFailed
	}
	parsed, err := parseOfficialProfileList(result.stdout)
	if err != nil {
		return nil, err
	}
	out := make([]NativeProfile, 0, len(parsed.profiles))
	for _, row := range parsed.profiles {
		out = append(out, NativeProfile{NativeID: row.name, Default: row.isDefault})
	}
	return out, nil
}

func runProfileListCommand(ctx context.Context, executable, home string) commandResult {
	runner := newCommandRunner()
	runner.environment = func() []string { return profileCommandEnvironment(home) }
	return runner.run(ctx, commandInvocation{
		path:       executable,
		executable: executable,
		args:       profileListArgs(),
	})
}

func IsProfileQueryFailure(err error) bool {
	return errors.Is(err, errProfileQueryFailed) ||
		errors.Is(err, errProfileHomeUnavailable) ||
		errors.Is(err, errProfileExecutableUnsafe) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

func IsProfileOutputUnrecognized(err error) bool {
	return errors.Is(err, errProfileListUnrecognized) ||
		errors.Is(err, errProfileListOversized) ||
		errors.Is(err, errProfileListDuplicateName) ||
		errors.Is(err, errProfileListEmptyTable)
}

func profileEnvHasSecret(env []string, secret string) bool {
	for _, entry := range env {
		if strings.Contains(entry, secret) {
			return true
		}
	}
	return false
}
