package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/managementhealth"
)

const profileHealthStateLimit = 64 * 1024

func (r *ProfileResourceReader) InspectRuntimeHealth(context.Context, yorvaruntime.Installation) (yorvaruntime.HealthObservation, error) {
	return yorvaruntime.HealthObservation{}, errProfileResourcesUnavailable
}

// InspectInstanceHealth is a last-observed Profile fallback. The authenticated
// loopback health API replaces it when available; this fallback remains
// explicitly partial and only projects Hermes' gateway_state field.
func (r *ProfileResourceReader) InspectInstanceHealth(ctx context.Context, installation yorvaruntime.Installation, nativeID string) (yorvaruntime.HealthObservation, error) {
	profileRoot, err := r.resolveProfile(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.HealthObservation{}, err
	}
	observedAt := r.clock()
	statePath := filepath.Join(profileRoot, "gateway_state.json")
	data, _, err := readFixedProfileFile(statePath, profileHealthStateLimit)
	if errors.Is(err, os.ErrNotExist) {
		return yorvaruntime.HealthObservation{State: yorvaruntime.HealthUnknown, Findings: []yorvaruntime.HealthFinding{{Code: "gateway_state", State: yorvaruntime.HealthUnknown}}, Partial: true, ObservedAt: observedAt}, nil
	}
	if err != nil {
		return yorvaruntime.HealthObservation{}, errProfileResourcesUnavailable
	}
	var state struct {
		GatewayState string `json:"gateway_state"`
		UpdatedAt    string `json:"updated_at"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if decoder.Decode(&state) != nil || state.GatewayState == "" {
		return yorvaruntime.HealthObservation{}, errProfileResourcesUnavailable
	}
	if parsed, parseErr := time.Parse(time.RFC3339, state.UpdatedAt); parseErr == nil {
		observedAt = parsed.UTC()
	}
	healthState := yorvaruntime.HealthUnknown
	switch strings.ToLower(state.GatewayState) {
	case "running":
		healthState = yorvaruntime.HealthHealthy
	case "starting", "draining", "stopping":
		healthState = yorvaruntime.HealthDegraded
	case "failed", "crashed", "error":
		healthState = yorvaruntime.HealthUnhealthy
	}
	return yorvaruntime.HealthObservation{State: healthState, Findings: []yorvaruntime.HealthFinding{{Code: "gateway_state", State: healthState}}, Partial: true, ObservedAt: observedAt}, nil
}

func (r *ProfileResourceReader) ReadLogSnapshot(ctx context.Context, installation yorvaruntime.Installation, nativeID string, category yorvaruntime.LogCategory) (yorvaruntime.LogSnapshot, error) {
	profileRoot, err := r.resolveProfile(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.LogSnapshot{}, err
	}
	fileName, sourceCategory, ok := profileLogSource(category)
	if !ok {
		return yorvaruntime.LogSnapshot{}, errProfileResourcesUnavailable
	}
	observedAt := r.clock()
	data, modifiedAt, err := readFixedProfileFile(filepath.Join(profileRoot, "logs", fileName), managementhealth.MaxLogInputBytes)
	if errors.Is(err, os.ErrNotExist) {
		return yorvaruntime.LogSnapshot{Category: category, Entries: []yorvaruntime.LogEntry{}, ObservedAt: observedAt}, nil
	}
	if err != nil {
		return yorvaruntime.LogSnapshot{}, errProfileResourcesUnavailable
	}
	if !modifiedAt.IsZero() {
		observedAt = modifiedAt.UTC()
	}
	data = keepLastLogLines(data, managementhealth.MaxLogLines)
	projected, err := managementhealth.ProjectLogSnapshot(sourceCategory, data)
	if err != nil {
		return yorvaruntime.LogSnapshot{}, errProfileResourcesUnavailable
	}
	entries := make([]yorvaruntime.LogEntry, 0, len(projected.Lines))
	for _, line := range projected.Lines {
		entries = append(entries, yorvaruntime.LogEntry{Timestamp: parseHermesLogTime(line, observedAt), Message: line})
	}
	return yorvaruntime.LogSnapshot{Category: category, Entries: entries, Truncated: projected.Truncated, ObservedAt: observedAt}, nil
}

func profileLogSource(category yorvaruntime.LogCategory) (string, managementhealth.LogCategory, bool) {
	switch category {
	case yorvaruntime.LogCategoryRuntime:
		return "agent.log", managementhealth.LogAgent, true
	case yorvaruntime.LogCategoryErrors:
		return "errors.log", managementhealth.LogErrors, true
	case yorvaruntime.LogCategoryGateway:
		return "gateway.log", managementhealth.LogGateway, true
	case yorvaruntime.LogCategoryMCP:
		return "mcp-stderr.log", managementhealth.LogMCP, true
	default:
		return "", "", false
	}
}

func readFixedProfileFile(path string, maximum int64) ([]byte, time.Time, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	if maximum < 1 || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
		return nil, time.Time{}, errProfileResourcesUnavailable
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer file.Close()
	start := info.Size() - maximum
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, time.Time{}, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum))
	if err != nil {
		return nil, time.Time{}, err
	}
	if start > 0 {
		if newline := bytes.IndexByte(data, '\n'); newline >= 0 {
			data = data[newline+1:]
		}
	}
	return data, info.ModTime().UTC(), nil
}

func keepLastLogLines(data []byte, maximum int) []byte {
	data = bytes.TrimRight(data, "\r\n")
	if len(data) == 0 {
		return data
	}
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines) <= maximum {
		return data
	}
	return bytes.Join(lines[len(lines)-maximum:], []byte{'\n'})
}

func parseHermesLogTime(line string, fallback time.Time) time.Time {
	if len(line) >= 19 {
		if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", line[:19], time.Local); err == nil {
			return parsed.UTC()
		}
	}
	return fallback.UTC()
}
