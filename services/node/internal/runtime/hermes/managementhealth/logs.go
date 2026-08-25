package managementhealth

import (
	"errors"
	"fmt"
	"strings"
)

const (
	MaxLogInputBytes = 256 * 1024
	MaxLogLines      = 200
	MaxLogLineBytes  = 1024
	MaxLogTotalBytes = 64 * 1024
)

var (
	ErrUnsupportedLogCategory = errors.New("unsupported Hermes log category")
	ErrLogSnapshotTooLarge    = errors.New("Hermes log snapshot exceeds the fixed input limit")
)

type LogCategory string

const (
	LogAgent   LogCategory = "agent"
	LogErrors  LogCategory = "errors"
	LogGateway LogCategory = "gateway"
	LogGUI     LogCategory = "gui"
	LogDesktop LogCategory = "desktop"
	LogMCP     LogCategory = "mcp"
)

type LogSnapshot struct {
	Category   LogCategory
	Lines      []string
	TotalBytes int
	Truncated  bool
}

// ProjectLogSnapshot applies a second YORVA-owned safety boundary to bytes
// acquired elsewhere. It never accepts a path and never invokes Hermes.
func ProjectLogSnapshot(category LogCategory, snapshot []byte) (LogSnapshot, error) {
	if !validLogCategory(category) {
		return LogSnapshot{}, ErrUnsupportedLogCategory
	}
	if len(snapshot) > MaxLogInputBytes {
		return LogSnapshot{}, ErrLogSnapshotTooLarge
	}
	if len(snapshot) == 0 {
		return LogSnapshot{Category: category, Lines: []string{}}, nil
	}

	normalized := strings.ReplaceAll(string(snapshot), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	rawLines := strings.Split(normalized, "\n")
	truncated := len(rawLines) > MaxLogLines
	if len(rawLines) > MaxLogLines {
		rawLines = rawLines[:MaxLogLines]
	}

	lines := make([]string, 0, len(rawLines))
	total := 0
	for _, raw := range rawLines {
		line := redactAndSanitize(raw)
		var lineTruncated bool
		line, lineTruncated = truncateUTF8(line, MaxLogLineBytes)
		truncated = truncated || lineTruncated

		separator := 0
		if len(lines) > 0 {
			separator = 1
		}
		remaining := MaxLogTotalBytes - total - separator
		if remaining <= 0 {
			truncated = true
			break
		}
		if len(line) > remaining {
			line, _ = truncateUTF8(line, remaining)
			truncated = true
		}
		if separator != 0 {
			total++
		}
		lines = append(lines, line)
		total += len(line)
		if total == MaxLogTotalBytes && len(lines) < len(rawLines) {
			truncated = true
			break
		}
	}

	if total > MaxLogTotalBytes {
		return LogSnapshot{}, fmt.Errorf("log projection exceeded fixed total limit")
	}
	return LogSnapshot{Category: category, Lines: lines, TotalBytes: total, Truncated: truncated}, nil
}

func validLogCategory(category LogCategory) bool {
	switch category {
	case LogAgent, LogErrors, LogGateway, LogGUI, LogDesktop, LogMCP:
		return true
	default:
		return false
	}
}
