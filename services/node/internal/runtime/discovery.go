package runtime

import (
	"context"
	"time"
)

type DiscoveryState string

const (
	DiscoveryNotInstalled     DiscoveryState = "NOT_INSTALLED"
	DiscoverySupported        DiscoveryState = "SUPPORTED"
	DiscoveryUnsupported      DiscoveryState = "UNSUPPORTED"
	DiscoveryBrokenExecutable DiscoveryState = "BROKEN_EXECUTABLE"
	DiscoveryMalformedVersion DiscoveryState = "MALFORMED_VERSION"
	DiscoveryTimedOut         DiscoveryState = "TIMED_OUT"
	DiscoveryAmbiguous        DiscoveryState = "AMBIGUOUS"
)

type ErrorCode string

const (
	ErrorRuntimeNotInstalled       ErrorCode = "RUNTIME_NOT_INSTALLED"
	ErrorRuntimeUnsupported        ErrorCode = "RUNTIME_UNSUPPORTED"
	ErrorRuntimeExecutableBroken   ErrorCode = "RUNTIME_EXECUTABLE_BROKEN"
	ErrorRuntimeVersionMalformed   ErrorCode = "RUNTIME_VERSION_MALFORMED"
	ErrorRuntimeDiscoveryTimeout   ErrorCode = "RUNTIME_DISCOVERY_TIMEOUT"
	ErrorRuntimeDiscoveryCancelled ErrorCode = "RUNTIME_DISCOVERY_CANCELLED"
	ErrorRuntimeCommandOutputLimit ErrorCode = "RUNTIME_COMMAND_OUTPUT_LIMIT"
	ErrorRuntimeDiscoveryAmbiguous ErrorCode = "RUNTIME_DISCOVERY_AMBIGUOUS"
)

type Candidate struct {
	Path      string
	Version   string
	State     DiscoveryState
	ErrorCode ErrorCode
}

type Warning struct {
	Code    string
	Message string
}

type Discovery struct {
	RuntimeKind    Kind
	State          DiscoveryState
	ErrorCode      ErrorCode
	Selected       *Candidate
	Candidates     []Candidate
	Warnings       []Warning
	DetectedAt     time.Time
	SupportedRange string
}

type Discoverer interface {
	Detect(ctx context.Context) (Discovery, error)
}
