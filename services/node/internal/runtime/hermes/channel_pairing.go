package hermes

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const channelPairingTimeout = 5 * time.Second

var (
	pairingCountPattern = regexp.MustCompile(`(?m)^\s*Pending Pairing Requests \(([0-9]+)\):\s*$`)
	pairingCodePattern  = regexp.MustCompile(`^[A-HJ-NP-Z2-9]{8}$`)
	pairingPlatform     = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
)

func (m *ChannelManager) PairingStatus(ctx context.Context, installation yorvaruntime.ChannelInstallation, nativeID string, kind channel.Type) (yorvaruntime.ChannelPairingStatus, error) {
	if err := validateChannelTarget(ctx, installation, nativeID); err != nil {
		return yorvaruntime.ChannelPairingStatus{}, err
	}
	if kind != channel.Weixin {
		return yorvaruntime.ChannelPairingStatus{}, yorvaruntime.ErrChannelNotSupported
	}
	result := m.runPairing(ctx, installation.Executable, lifecycleProfileArgs(nativeID, "pairing", "list"))
	defer clearCommandResult(&result)
	if result.err != nil || result.exitCode != 0 || result.limited || result.timedOut || strings.TrimSpace(result.stderr) != "" {
		return yorvaruntime.ChannelPairingStatus{}, yorvaruntime.ErrChannelPairingQuery
	}
	count, err := parsePendingPairingCount(result.stdout, kind)
	if err != nil {
		return yorvaruntime.ChannelPairingStatus{}, yorvaruntime.ErrChannelPairingQuery
	}
	return yorvaruntime.ChannelPairingStatus{PendingCount: count}, nil
}

func (m *ChannelManager) ApprovePairing(ctx context.Context, installation yorvaruntime.ChannelInstallation, nativeID string, kind channel.Type, code []byte) error {
	defer clearCredentialBytes(code)
	if err := validateChannelTarget(ctx, installation, nativeID); err != nil {
		return err
	}
	if kind != channel.Weixin {
		return yorvaruntime.ErrChannelNotSupported
	}
	normalized := strings.ToUpper(strings.TrimSpace(string(code)))
	if !pairingCodePattern.MatchString(normalized) {
		return yorvaruntime.ErrChannelPairingCode
	}
	result := m.runPairing(ctx, installation.Executable, lifecycleProfileArgs(nativeID, "pairing", "approve", string(kind), normalized))
	defer clearCommandResult(&result)
	combined := result.stdout + "\n" + result.stderr
	switch {
	case strings.Contains(combined, "locked out after too many failed approval attempts"):
		return yorvaruntime.ErrChannelPairingLocked
	case strings.Contains(combined, "not found or expired"):
		return yorvaruntime.ErrChannelPairingCode
	case result.err == nil && result.exitCode == 0 && !result.limited && !result.timedOut && strings.Contains(result.stdout, "Approved! User"):
		return nil
	default:
		return yorvaruntime.ErrChannelPairingApproval
	}
}

func parsePendingPairingCount(output string, kind channel.Type) (int, error) {
	if strings.Contains(output, "No pending pairing requests.") || strings.Contains(output, "No pairing data found.") {
		return 0, nil
	}
	matches := pairingCountPattern.FindStringSubmatchIndex(output)
	if len(matches) != 4 {
		return 0, yorvaruntime.ErrChannelPairingQuery
	}
	declared, err := strconv.Atoi(output[matches[2]:matches[3]])
	if err != nil || declared <= 0 {
		return 0, yorvaruntime.ErrChannelPairingQuery
	}
	section := output[matches[1]:]
	parsed := 0
	matching := 0
	headerSeen := false
	separatorSeen := false
	for _, line := range strings.Split(section, "\n") {
		fields := strings.Fields(line)
		if !headerSeen {
			if len(fields) >= 2 && fields[0] == "Platform" && fields[1] == "Request" {
				headerSeen = true
			}
			continue
		}
		if !separatorSeen {
			if len(fields) > 0 && allDashes(fields[0]) {
				separatorSeen = true
			}
			continue
		}
		if len(fields) < 2 || !pairingPlatform.MatchString(fields[0]) {
			return 0, yorvaruntime.ErrChannelPairingQuery
		}
		parsed++
		if fields[0] == string(kind) {
			matching++
		}
		if parsed == declared {
			return matching, nil
		}
	}
	return 0, yorvaruntime.ErrChannelPairingQuery
}

func allDashes(value string) bool {
	return value != "" && strings.Trim(value, "-") == ""
}

func clearCommandResult(result *commandResult) {
	result.stdout = ""
	result.stderr = ""
}

func runChannelPairingCommand(ctx context.Context, executable string, args []string) commandResult {
	runner := newCommandRunner()
	runner.timeout = channelPairingTimeout
	runner.environment = func() []string { return profileCommandEnvironment(officialHermesHome()) }
	return runner.run(ctx, commandInvocation{path: executable, executable: executable, args: args})
}
