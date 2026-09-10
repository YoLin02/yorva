// Package openclaw owns the official OpenClaw CLI and Gateway integration.
package openclaw

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const Kind yorvaruntime.Kind = "openclaw"
const supportedVersion = "2026.9.3"

// The qualified 2026.9.3 package.json is 135,311 bytes because it publishes
// a large export map. Keep metadata bounded above the real release's size.
const packageMetadataLimit = 256 * 1024

var errUnsafeTarget = errors.New("OpenClaw target is not qualified")
var versionPattern = regexp.MustCompile(`^OpenClaw (\d{4}\.\d+\.\d+) \([a-f0-9]+\)$`)

// Adapter uses the current OS user's canonical profile root. Tests replace only
// these process-local dependencies, never the user's Runtime configuration.
type Adapter struct {
	createMu sync.Mutex
	home     string
	path     string
	appData  string
}

func New() (*Adapter, error) {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return nil, errUnsafeTarget
	}
	return &Adapter{home: filepath.Clean(home), path: os.Getenv("PATH"), appData: os.Getenv("APPDATA")}, nil
}

func Register(registry *yorvaruntime.Registry) error {
	a, err := New()
	if err != nil {
		return err
	}
	bundle := yorvaruntime.Bundle{Descriptor: yorvaruntime.Descriptor{Kind: Kind, Name: "OpenClaw", Description: "OpenClaw AI Gateway"}, Discoverer: a}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		bundle.Instances = a
		bundle.Lifecycle = a
		bundle.Health = a
		bundle.InstanceManagement = a
	}
	return registry.Register(Kind, bundle)
}

func (a *Adapter) Detect(ctx context.Context) (yorvaruntime.Discovery, error) {
	result := yorvaruntime.Discovery{RuntimeKind: Kind, State: yorvaruntime.DiscoveryNotInstalled, ErrorCode: yorvaruntime.ErrorRuntimeNotInstalled, DetectedAt: time.Now().UTC(), SupportedRange: supportedVersion, Candidates: []yorvaruntime.Candidate{}, Warnings: []yorvaruntime.Warning{}}
	for _, entry := range a.candidateEntries() {
		candidate := yorvaruntime.Candidate{Path: entry, State: yorvaruntime.DiscoveryBrokenExecutable, ErrorCode: yorvaruntime.ErrorRuntimeExecutableBroken}
		node, err := a.resolveNode(entry)
		if err == nil {
			output, runErr := a.command(ctx, node, entry, "", 15*time.Second, "--version")
			match := versionPattern.FindStringSubmatch(strings.TrimSpace(string(output)))
			switch {
			case errors.Is(runErr, context.Canceled):
				return result, runErr
			case errors.Is(runErr, context.DeadlineExceeded):
				candidate.State, candidate.ErrorCode = yorvaruntime.DiscoveryTimedOut, yorvaruntime.ErrorRuntimeDiscoveryTimeout
			case runErr != nil:
			case len(match) != 2:
				candidate.State, candidate.ErrorCode = yorvaruntime.DiscoveryMalformedVersion, yorvaruntime.ErrorRuntimeVersionMalformed
			default:
				candidate.Version = match[1]
				candidate.State, candidate.ErrorCode = yorvaruntime.DiscoveryUnsupported, yorvaruntime.ErrorRuntimeUnsupported
				if candidate.Version == supportedVersion && runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
					candidate.State, candidate.ErrorCode = yorvaruntime.DiscoverySupported, ""
				}
			}
		}
		result.Candidates = append(result.Candidates, candidate)
	}
	if len(result.Candidates) == 1 {
		selected := result.Candidates[0]
		result.State, result.ErrorCode = selected.State, selected.ErrorCode
		result.Selected = &selected
	} else if len(result.Candidates) > 1 {
		result.State, result.ErrorCode = yorvaruntime.DiscoveryAmbiguous, yorvaruntime.ErrorRuntimeDiscoveryAmbiguous
	}
	return result, nil
}

func (a *Adapter) candidateEntries() []string {
	dirs := filepath.SplitList(a.path)
	if filepath.IsAbs(a.appData) {
		dirs = append(dirs, filepath.Join(a.appData, "npm"))
	}
	seen := map[string]bool{}
	entries := []string{}
	for _, dir := range dirs {
		if !filepath.IsAbs(dir) {
			continue
		}
		// npm's documented global installation layout. Never execute a .cmd
		// wrapper or infer arbitrary commands from its contents.
		entry := filepath.Join(dir, "node_modules", "openclaw", "openclaw.mjs")
		resolved, err := filepath.EvalSymlinks(entry)
		if err != nil || !regularFile(resolved) {
			continue
		}
		key := strings.ToLower(resolved)
		if !seen[key] {
			seen[key] = true
			entries = append(entries, resolved)
		}
	}
	return entries
}

func (a *Adapter) resolveNode(entry string) (string, error) {
	if !filepath.IsAbs(entry) || filepath.Base(entry) != "openclaw.mjs" || !regularFile(entry) {
		return "", errUnsafeTarget
	}
	data, err := readBoundedFile(filepath.Join(filepath.Dir(entry), "package.json"), packageMetadataLimit)
	if err != nil {
		return "", errUnsafeTarget
	}
	var pkg struct {
		Name string            `json:"name"`
		Bin  map[string]string `json:"bin"`
	}
	if json.Unmarshal(data, &pkg) != nil || pkg.Name != "openclaw" || pkg.Bin["openclaw"] != "openclaw.mjs" {
		return "", errUnsafeTarget
	}
	dirs := append([]string{filepath.Dir(filepath.Dir(filepath.Dir(entry)))}, filepath.SplitList(a.path)...)
	for _, dir := range dirs {
		if !filepath.IsAbs(dir) {
			continue
		}
		node := filepath.Join(dir, "node.exe")
		if resolved, err := filepath.EvalSymlinks(node); err == nil && regularFile(resolved) {
			return resolved, nil
		}
	}
	return "", errUnsafeTarget
}

func regularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func readBoundedFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errUnsafeTarget
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errUnsafeTarget
	}
	defer file.Close()
	return readBounded(file, limit)
}
