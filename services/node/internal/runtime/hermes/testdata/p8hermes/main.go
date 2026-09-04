package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const version = "0.20.5"

var profileNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

type modelState struct {
	Provider string `json:"provider,omitempty"`
	Default  string `json:"default,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(9)
	}
}

func run(args []string) error {
	root, err := hermesRoot()
	if err != nil {
		return err
	}
	switch {
	case equal(args, "--version"):
		fmt.Printf("Hermes Agent v%s (P8 qualification fixture)\n", version)
		return nil
	case equal(args, "profile", "list"):
		return listProfiles(root)
	case len(args) == 5 && args[0] == "profile" && args[1] == "create" && args[3] == "--no-alias" && args[4] == "--no-skills":
		return initializeProfile(root, args[2])
	case len(args) == 4 && args[0] == "profile" && args[1] == "delete" && args[3] == "--yes":
		if !validProfileName(args[2]) {
			return fmt.Errorf("invalid profile")
		}
		if args[2] == "default" {
			return fmt.Errorf("default profile is protected")
		}
		return os.RemoveAll(filepath.Join(root, "profiles", args[2]))
	}

	profile, tail, err := splitProfile(args)
	if err != nil {
		return err
	}
	profileRoot := root
	if profile != "default" {
		profileRoot = filepath.Join(root, "profiles", profile)
	}
	if info, statErr := os.Stat(profileRoot); statErr != nil || !info.IsDir() {
		return fmt.Errorf("profile is missing")
	}

	switch {
	case equal(tail, "gateway", "status"):
		if _, statErr := os.Stat(filepath.Join(profileRoot, ".p8-gateway-running")); statErr == nil {
			fmt.Println("✓ Gateway is running (PID: 4242)")
		} else {
			fmt.Println("✗ Gateway is not running")
		}
		return nil
	case equal(tail, "gateway", "install", "--no-start-on-login", "--start-now"), equal(tail, "gateway", "start"):
		return os.WriteFile(filepath.Join(profileRoot, ".p8-gateway-running"), []byte("running\n"), 0o600)
	case equal(tail, "gateway", "stop"):
		err := os.Remove(filepath.Join(profileRoot, ".p8-gateway-running"))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	case len(tail) == 4 && tail[0] == "config" && tail[1] == "get" && tail[3] == "--json":
		return getConfig(profileRoot, tail[2])
	case len(tail) == 4 && tail[0] == "config" && tail[1] == "set":
		return setConfig(profileRoot, tail[2], tail[3])
	case contains(tail, "--oneshot"):
		return nil
	default:
		return fmt.Errorf("unsupported P8 Hermes fixture invocation: %s", strings.Join(args, " "))
	}
}

func hermesRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv("HERMES_HOME"))
	if root == "" {
		local := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if local == "" {
			return "", fmt.Errorf("LOCALAPPDATA is required")
		}
		root = filepath.Join(local, "hermes")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func initializeProfile(root, profile string) error {
	if !validProfileName(profile) || profile == "default" {
		return fmt.Errorf("invalid profile")
	}
	directory := filepath.Join(root, "profiles", profile)
	if _, err := os.Stat(directory); err == nil {
		return fmt.Errorf("profile already exists")
	}
	return initializeProfileRoot(directory)
}

func initializeProfileRoot(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0o700); err != nil {
		return err
	}
	config := filepath.Join(root, "config.yaml")
	if _, err := os.Stat(config); os.IsNotExist(err) {
		return os.WriteFile(config, []byte("{}\n"), 0o600)
	}
	return nil
}

func listProfiles(root string) error {
	if err := initializeProfileRoot(root); err != nil {
		return err
	}
	names := []string{"default"}
	entries, err := os.ReadDir(filepath.Join(root, "profiles"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names[1:])
	fmt.Print("\n Profile          Model                        Gateway      Alias        Distribution\n")
	fmt.Print(" ───────────────    ───────────────────────────    ───────────    ───────────    ────────────────────\n")
	for _, name := range names {
		profileRoot := root
		if name != "default" {
			profileRoot = filepath.Join(root, "profiles", name)
		}
		gateway := "stopped"
		if _, statErr := os.Stat(filepath.Join(profileRoot, ".p8-gateway-running")); statErr == nil {
			gateway = "running"
		}
		fmt.Printf("  %-15s  %-27s  %-11s  %-11s  %s\n", name, "—", gateway, "—", "—")
	}
	fmt.Println()
	return nil
}

func splitProfile(args []string) (string, []string, error) {
	if len(args) >= 2 && args[0] == "--profile" {
		if !validProfileName(args[1]) {
			return "", nil, fmt.Errorf("empty profile")
		}
		return args[1], args[2:], nil
	}
	return "default", args, nil
}

func validProfileName(profile string) bool {
	return profile == "default" || profileNamePattern.MatchString(profile)
}

func getConfig(root, key string) error {
	state, err := readModelState(root)
	if err != nil {
		return err
	}
	var value any
	switch key {
	case "model":
		if state.Provider == "" && state.Default == "" {
			value = ""
		} else {
			value = state
		}
	case "model.provider":
		value = state.Provider
	case "model.default":
		value = state.Default
	case "context.engine":
		value = "compressor"
	default:
		value = ""
	}
	return json.NewEncoder(os.Stdout).Encode(value)
}

func setConfig(root, key, value string) error {
	state, err := readModelState(root)
	if err != nil {
		return err
	}
	switch key {
	case "model.provider":
		state.Provider = value
	case "model.default":
		state.Default = value
	default:
		return fmt.Errorf("unsupported config key")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, ".p8-model.json"), append(data, '\n'), 0o600)
}

func readModelState(root string) (modelState, error) {
	data, err := os.ReadFile(filepath.Join(root, ".p8-model.json"))
	if os.IsNotExist(err) {
		return modelState{}, nil
	}
	if err != nil {
		return modelState{}, err
	}
	var state modelState
	if err := json.Unmarshal(data, &state); err != nil {
		return modelState{}, err
	}
	return state, nil
}

func equal(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
