package hermes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	nodeProbeTimeout  = 5 * time.Second
	nodeDepsStampName = ".yorva-node-deps"
)

type PrerequisiteState string

const (
	PrereqReady        PrerequisiteState = "READY"
	PrereqMissing      PrerequisiteState = "MISSING"
	PrereqUnsupported  PrerequisiteState = "UNSUPPORTED"
	PrereqBroken       PrerequisiteState = "BROKEN"
	PrereqNotInstalled PrerequisiteState = "NOT_INSTALLED"
	PrereqFailed       PrerequisiteState = "FAILED"
	PrereqTimedOut     PrerequisiteState = "TIMED_OUT"
)

type ComponentStatus struct {
	State     PrerequisiteState
	Version   string
	ErrorCode yorvaruntime.ErrorCode
	Retryable bool
}

type Prerequisites struct {
	Node             ComponentStatus
	NPM              ComponentStatus
	NodeDependencies ComponentStatus
	CheckedAt        time.Time
	ActiveOperation  string
}

func (h *NodeHost) Inspect() Prerequisites {
	now := time.Now().UTC()
	node := h.inspectNode()
	npm := ComponentStatus{State: PrereqMissing, ErrorCode: yorvaruntime.ErrorHermesNPMMissing, Retryable: true}
	deps := ComponentStatus{State: PrereqNotInstalled, Retryable: true}
	if node.State == PrereqReady {
		npm = h.inspectNPM()
		if npm.State == PrereqReady {
			deps = h.inspectDeps()
		}
	}
	return Prerequisites{Node: node, NPM: npm, NodeDependencies: deps, CheckedAt: now}
}

func (h *NodeHost) inspectNode() ComponentStatus {
	exe := filepath.Join(h.nodeDir(), "node.exe")
	if !isRegularFile(exe) {
		return ComponentStatus{State: PrereqMissing, ErrorCode: yorvaruntime.ErrorHermesNodeMissing, Retryable: true}
	}
	version, err := h.probeVersion(exe, nil)
	if err != nil {
		return ComponentStatus{State: PrereqBroken, ErrorCode: yorvaruntime.ErrorHermesNodeMissing, Retryable: true}
	}
	if !nodeVersionSupported(version) {
		return ComponentStatus{State: PrereqUnsupported, Version: version, ErrorCode: yorvaruntime.ErrorHermesNodeUnsupported, Retryable: true}
	}
	return ComponentStatus{State: PrereqReady, Version: version}
}

func (h *NodeHost) inspectNPM() ComponentStatus {
	cli := managedNpmCLI(h.nodeDir())
	if !isRegularFile(cli) {
		return ComponentStatus{State: PrereqMissing, ErrorCode: yorvaruntime.ErrorHermesNPMMissing, Retryable: true}
	}
	version, err := h.probeVersion(filepath.Join(h.nodeDir(), "node.exe"), []string{cli, "--version"})
	if err != nil {
		return ComponentStatus{State: PrereqBroken, ErrorCode: yorvaruntime.ErrorHermesNPMMissing, Retryable: true}
	}
	if !npmVersionSupported(version) {
		return ComponentStatus{State: PrereqUnsupported, Version: version, ErrorCode: yorvaruntime.ErrorHermesNPMUnsupported, Retryable: true}
	}
	return ComponentStatus{State: PrereqReady, Version: version}
}

func (h *NodeHost) inspectDeps() ComponentStatus {
	lock := filepath.Join(h.installDir(), "package-lock.json")
	modules := filepath.Join(h.installDir(), "node_modules")
	if !isRegularFile(lock) {
		return ComponentStatus{State: PrereqNotInstalled, Retryable: true}
	}
	want, err := fileSHA256(lock)
	if err != nil {
		return ComponentStatus{State: PrereqNotInstalled, Retryable: true}
	}
	stamp, err := os.ReadFile(filepath.Join(h.installDir(), nodeDepsStampName))
	if err != nil || strings.TrimSpace(string(stamp)) != want {
		if isDirectory(modules) {
			return ComponentStatus{State: PrereqFailed, ErrorCode: yorvaruntime.ErrorHermesNodeDepsFailed, Retryable: true}
		}
		return ComponentStatus{State: PrereqNotInstalled, Retryable: true}
	}
	if !isDirectory(modules) {
		return ComponentStatus{State: PrereqNotInstalled, Retryable: true}
	}
	return ComponentStatus{State: PrereqReady}
}

func (h *NodeHost) probeVersion(executable string, args []string) (string, error) {
	if len(args) == 0 {
		args = []string{"--version"}
	}
	result := h.run(context.Background(), installInvocation{Executable: executable, Args: args, Dir: h.nodeDir()}, nodeProbeTimeout)
	if result.err != nil || result.timedOut || result.limited || result.exitCode != 0 {
		return "", errors.New("version probe failed")
	}
	return strings.TrimSpace(strings.TrimPrefix(result.stdout, "v")), nil
}

func nodeVersionSupported(version string) bool {
	return normalizeManagedVersion(version) == officialNodeVersion
}

func npmVersionSupported(version string) bool {
	return normalizeManagedVersion(version) == officialNpmVersion
}

func normalizeManagedVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func compareLooseVersion(got, min string) int {
	g := parseLoose(got)
	m := parseLoose(min)
	for i := 0; i < 3; i++ {
		if g[i] < m[i] {
			return -1
		}
		if g[i] > m[i] {
			return 1
		}
	}
	return 0
}

func parseLoose(value string) [3]int {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.Split(value, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		fmt.Sscanf(parts[i], "%d", &out[i])
	}
	return out
}
