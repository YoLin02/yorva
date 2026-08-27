package hermes

import (
	"context"
	"errors"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// Hermes 0.20.5 performs a bounded update-status check as part of --version.
// Leave enough time for that official surface without treating one slow probe as
// permission to wait indefinitely.
const overallDiscoveryTimeout = 32 * time.Second

type Detector struct {
	finder         candidateFinder
	run            func(context.Context, commandInvocation) commandResult
	now            func() time.Time
	overallTimeout time.Duration
}

func NewDetector() *Detector {
	runner := newDiscoveryCommandRunner()
	return &Detector{
		finder:         newCandidateFinder(),
		run:            runner.run,
		now:            time.Now,
		overallTimeout: overallDiscoveryTimeout,
	}
}

func (d *Detector) Detect(ctx context.Context) (yorvaruntime.Discovery, error) {
	discoveryCtx, cancel := context.WithTimeout(ctx, d.overallTimeout)
	defer cancel()

	found := d.finder.find()
	result := yorvaruntime.Discovery{
		RuntimeKind:    Kind,
		Candidates:     make([]yorvaruntime.Candidate, 0, len(found.commands)),
		Warnings:       make([]yorvaruntime.Warning, 0),
		DetectedAt:     d.now().UTC(),
		SupportedRange: supportedRange,
	}
	if len(found.commands) == 0 {
		if found.installationEvidence {
			result.State = yorvaruntime.DiscoveryBrokenExecutable
			result.ErrorCode = yorvaruntime.ErrorRuntimeExecutableBroken
			result.Warnings = append(result.Warnings, yorvaruntime.Warning{
				Code:    "HERMES_CLI_LAUNCHER_MISSING",
				Message: "A Hermes installation was found, but its safe CLI launcher is missing.",
			})
			return result, nil
		}
		result.State = yorvaruntime.DiscoveryNotInstalled
		result.ErrorCode = yorvaruntime.ErrorRuntimeNotInstalled
		return result, nil
	}
	if found.truncated {
		result.Warnings = append(result.Warnings, yorvaruntime.Warning{
			Code:    "CANDIDATE_LIMIT_REACHED",
			Message: "Additional Hermes executable candidates were not evaluated.",
		})
	}

	candidates, err := d.inspectCandidates(discoveryCtx, found.commands)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return yorvaruntime.Discovery{}, ctx.Err()
		}
		return yorvaruntime.Discovery{}, err
	}
	result.Candidates = append(result.Candidates, candidates...)

	return aggregateDiscovery(result, d.finder.installationRoots), nil
}

type candidateInspection struct {
	index     int
	candidate yorvaruntime.Candidate
	err       error
}

// Candidate discovery is bounded to eight fixed --version invocations. Probe
// them together so one stale PATH launcher cannot consume the entire
// application deadline before an official launcher is inspected. Results stay
// in finder order so selection and diagnostics remain deterministic.
func (d *Detector) inspectCandidates(
	ctx context.Context,
	commands []commandInvocation,
) ([]yorvaruntime.Candidate, error) {
	inspections := make(chan candidateInspection, len(commands))
	for index, command := range commands {
		go func() {
			candidate, err := d.inspect(ctx, command)
			inspections <- candidateInspection{index: index, candidate: candidate, err: err}
		}()
	}

	ordered := make([]yorvaruntime.Candidate, len(commands))
	var cancellation error
	for range commands {
		inspection := <-inspections
		ordered[inspection.index] = inspection.candidate
		if errors.Is(inspection.err, context.Canceled) && cancellation == nil {
			cancellation = inspection.err
		}
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil, ctx.Err()
	}
	if cancellation != nil {
		return nil, cancellation
	}
	return ordered, nil
}

func (d *Detector) inspect(ctx context.Context, invocation commandInvocation) (yorvaruntime.Candidate, error) {
	candidate := yorvaruntime.Candidate{Path: invocation.path}
	command := d.run(ctx, invocation)
	if command.limited {
		candidate.State = yorvaruntime.DiscoveryBrokenExecutable
		candidate.ErrorCode = yorvaruntime.ErrorRuntimeCommandOutputLimit
		return candidate, nil
	}
	if command.timedOut || errors.Is(command.err, context.DeadlineExceeded) {
		candidate.State = yorvaruntime.DiscoveryTimedOut
		candidate.ErrorCode = yorvaruntime.ErrorRuntimeDiscoveryTimeout
		return candidate, command.err
	}
	if errors.Is(command.err, context.Canceled) {
		return candidate, command.err
	}
	if command.err != nil || command.exitCode != 0 {
		candidate.State = yorvaruntime.DiscoveryBrokenExecutable
		candidate.ErrorCode = yorvaruntime.ErrorRuntimeExecutableBroken
		return candidate, nil
	}

	parsed, err := parseVersionBanner(command.stdout)
	if err != nil {
		candidate.State = yorvaruntime.DiscoveryMalformedVersion
		candidate.ErrorCode = yorvaruntime.ErrorRuntimeVersionMalformed
		return candidate, nil
	}
	candidate.Version = parsed.String()
	if parsed.supported() {
		candidate.State = yorvaruntime.DiscoverySupported
	} else {
		candidate.State = yorvaruntime.DiscoveryUnsupported
		candidate.ErrorCode = yorvaruntime.ErrorRuntimeUnsupported
	}
	return candidate, nil
}

func aggregateDiscovery(result yorvaruntime.Discovery, officialRoots []string) yorvaruntime.Discovery {
	runnable := make([]int, 0, len(result.Candidates))
	for index, candidate := range result.Candidates {
		if candidate.State == yorvaruntime.DiscoverySupported || candidate.State == yorvaruntime.DiscoveryUnsupported {
			runnable = append(runnable, index)
		}
	}
	if selected, ok := officialLauncherAliasSelection(result.Candidates, runnable, officialRoots); ok {
		result.Selected = &selected
		result.State = selected.State
		result.ErrorCode = selected.ErrorCode
		result.Warnings = append(result.Warnings, yorvaruntime.Warning{
			Code:    "HERMES_LAUNCHER_ALIAS",
			Message: "Official bin and venv Hermes launchers are aliases of one installation.",
		})
		return result
	}
	if len(runnable) > 1 {
		result.State = yorvaruntime.DiscoveryAmbiguous
		result.ErrorCode = yorvaruntime.ErrorRuntimeDiscoveryAmbiguous
		result.Warnings = append(result.Warnings, yorvaruntime.Warning{
			Code:    "MULTIPLE_RUNNABLE_CANDIDATES",
			Message: "Multiple Hermes executables were found; none was selected.",
		})
		return result
	}
	if len(runnable) == 1 {
		selected := result.Candidates[runnable[0]]
		result.Selected = &selected
		result.State = selected.State
		result.ErrorCode = selected.ErrorCode
		if len(result.Candidates) > 1 {
			result.Warnings = append(result.Warnings, yorvaruntime.Warning{
				Code:    "OTHER_CANDIDATES_UNUSABLE",
				Message: "Other Hermes executable candidates could not be used.",
			})
		}
		if selected.State == yorvaruntime.DiscoveryUnsupported && strings.Contains(selected.Version, "-") {
			result.Warnings = append(result.Warnings, yorvaruntime.Warning{
				Code:    "PRERELEASE_UNTESTED",
				Message: "The detected Hermes prerelease is not in the tested compatibility range.",
			})
		}
		return result
	}

	priority := []yorvaruntime.DiscoveryState{
		yorvaruntime.DiscoveryMalformedVersion,
		yorvaruntime.DiscoveryTimedOut,
		yorvaruntime.DiscoveryBrokenExecutable,
	}
	for _, state := range priority {
		for _, candidate := range result.Candidates {
			if candidate.State == state {
				result.State = state
				result.ErrorCode = candidate.ErrorCode
				return result
			}
		}
	}
	result.State = yorvaruntime.DiscoveryTimedOut
	result.ErrorCode = yorvaruntime.ErrorRuntimeDiscoveryTimeout
	return result
}
