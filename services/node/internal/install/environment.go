package install

import (
	"os"
	"path/filepath"
	"strings"
)

// ObservedEnvironment is HKCU\Environment as seen by reconcile. It is not an activation source.
type ObservedEnvironment struct {
	HermesHome  string
	PathEntries []string
}

// EnvironmentPolicy is the fixed desired state derived from valid active.json.
type EnvironmentPolicy struct {
	HermesHome       string
	DesiredBin       string
	RemovableBins    []string
	AllowRemoveStale bool
}

// EnvironmentPlan is the pure result of ComputeEnvironmentPlan.
type EnvironmentPlan struct {
	HermesHome    string
	SetHermesHome bool
	PathEntries   []string
	PathChanged   bool
	DesiredBin    string
}

// EnvironmentStore is injectable registry IO. Default writes HKCU on Windows.
type EnvironmentStore struct {
	Read      func() (ObservedEnvironment, error)
	WriteHome func(string) error
	WritePath func([]string) error
	Broadcast func() error
}

func ComputeEnvironmentPlan(policy EnvironmentPolicy, observed ObservedEnvironment) EnvironmentPlan {
	plan := EnvironmentPlan{
		HermesHome: policy.HermesHome,
		DesiredBin: policy.DesiredBin,
	}
	if policy.HermesHome != "" && !sameEnvPath(observed.HermesHome, policy.HermesHome) {
		plan.SetHermesHome = true
	}
	var kept []string
	for _, entry := range observed.PathEntries {
		if policy.AllowRemoveStale && isRemovableBin(entry, policy.RemovableBins) {
			plan.PathChanged = true
			continue
		}
		kept = append(kept, entry)
	}
	if policy.DesiredBin == "" {
		plan.PathEntries = kept
		return plan
	}
	if len(kept) > 0 && sameEnvPath(kept[0], policy.DesiredBin) {
		plan.PathEntries = kept
		return plan
	}
	without := make([]string, 0, len(kept))
	for _, entry := range kept {
		if sameEnvPath(entry, policy.DesiredBin) {
			plan.PathChanged = true
			continue
		}
		without = append(without, entry)
	}
	plan.PathEntries = append([]string{policy.DesiredBin}, without...)
	plan.PathChanged = true
	return plan
}

func (p EnvironmentPlan) Observation(observed ObservedEnvironment) EnvironmentObservation {
	return EnvironmentObservation{
		HermesHomeOK: p.HermesHome == "" || sameEnvPath(observed.HermesHome, p.HermesHome),
		PathOK:       p.DesiredBin == "" || pathHasDesiredPrefix(observed.PathEntries, p.DesiredBin),
	}
}

func pathHasDesiredPrefix(entries []string, desired string) bool {
	return len(entries) > 0 && sameEnvPath(entries[0], desired)
}

func isRemovableBin(entry string, removable []string) bool {
	for _, bin := range removable {
		if sameEnvPath(entry, bin) {
			return true
		}
	}
	return false
}

func sameEnvPath(a, b string) bool {
	return filepath.Clean(expandEnvPath(a)) == filepath.Clean(expandEnvPath(b))
}

func expandEnvPath(value string) string {
	local := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	replaced := strings.ReplaceAll(value, "%LOCALAPPDATA%", local)
	return strings.ReplaceAll(replaced, "%LocalAppData%", local)
}

func (s *Store) HasCommitted() bool {
	views, _, err := s.ListTransactionViews()
	if err != nil {
		return false
	}
	for _, view := range views {
		if view.Valid && view.State == StateCommitted {
			return true
		}
	}
	return false
}

func (s *Store) RemovableBins(activeGen string) []string {
	var bins []string
	entries, err := os.ReadDir(s.layout.GenerationsRoot())
	if err != nil {
		return bins
	}
	for _, entry := range entries {
		if !entry.IsDir() || ParseGenerationID(entry.Name()) != nil || entry.Name() == activeGen {
			continue
		}
		bins = append(bins, filepath.Join(s.layout.GenerationsRoot(), entry.Name(), "bin"))
	}
	return bins
}

func PolicyFromActive(store *Store, rec ActiveRecord, allowStale bool) (EnvironmentPolicy, error) {
	genAbs, err := store.layout.GenerationPath(rec.GenerationID)
	if err != nil {
		return EnvironmentPolicy{}, err
	}
	if err := VerifySealedTree(genAbs, rec.GenerationID, rec.ManifestSHA256, rec.SealSHA256); err != nil {
		return EnvironmentPolicy{}, err
	}
	return EnvironmentPolicy{
		HermesHome:       store.layout.Root,
		DesiredBin:       filepath.Join(genAbs, "bin"),
		RemovableBins:    store.RemovableBins(rec.GenerationID),
		AllowRemoveStale: allowStale,
	}, nil
}
