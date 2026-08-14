package hermes

import (
	"context"
	"errors"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const overallDiscoveryTimeout = 10 * time.Second

type Detector struct {
	finder         candidateFinder
	run            func(context.Context, string) commandResult
	now            func() time.Time
	overallTimeout time.Duration
}

func NewDetector() *Detector {
	runner := newCommandRunner()
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
		Candidates:     make([]yorvaruntime.Candidate, 0, len(found.paths)),
		Warnings:       make([]yorvaruntime.Warning, 0),
		DetectedAt:     d.now().UTC(),
		SupportedRange: supportedRange,
	}
	if len(found.paths) == 0 {
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

	for _, path := range found.paths {
		if err := discoveryCtx.Err(); err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return yorvaruntime.Discovery{}, ctx.Err()
			}
			break
		}
		candidate, err := d.inspect(discoveryCtx, path)
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return yorvaruntime.Discovery{}, ctx.Err()
			}
			if errors.Is(err, context.Canceled) {
				return yorvaruntime.Discovery{}, err
			}
		}
		result.Candidates = append(result.Candidates, candidate)
	}

	return aggregateDiscovery(result), nil
}

func (d *Detector) inspect(ctx context.Context, path string) (yorvaruntime.Candidate, error) {
	candidate := yorvaruntime.Candidate{Path: path}
	command := d.run(ctx, path)
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

func aggregateDiscovery(result yorvaruntime.Discovery) yorvaruntime.Discovery {
	runnable := make([]int, 0, len(result.Candidates))
	for index, candidate := range result.Candidates {
		if candidate.State == yorvaruntime.DiscoverySupported || candidate.State == yorvaruntime.DiscoveryUnsupported {
			runnable = append(runnable, index)
		}
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
