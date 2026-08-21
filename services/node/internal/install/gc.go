package install

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	FailGCBeforeDelete = "gc-before-delete"
	FailGCDuringDelete = "gc-during-delete"
)

type GCHooks struct {
	BeforeDelete func(path string) error
	DuringDelete func(path string) error
}

type GCReport struct {
	Deleted []string
	Kept    []string
}

func Collect(store *Store, hooks GCHooks) (GCReport, error) {
	return collectWithFail(store, hooks, "")
}

func collectWithFail(store *Store, hooks GCHooks, failAt string) (GCReport, error) {
	var report GCReport
	protected, err := retentionSet(store)
	if err != nil {
		return report, err
	}
	candidates, err := gcCandidates(store)
	if err != nil {
		return report, err
	}
	for _, path := range candidates {
		rel, _ := filepath.Rel(store.layout.Root, path)
		slash := filepath.ToSlash(rel)
		if _, ok := protected[filepath.Clean(path)]; ok {
			report.Kept = append(report.Kept, slash)
			continue
		}
		if failAt == FailGCBeforeDelete {
			return report, errInjectedGC
		}
		if hooks.BeforeDelete != nil {
			if err := hooks.BeforeDelete(path); err != nil {
				return report, err
			}
		}
		active := store.ReadActive()
		if active.Valid {
			if gen, err := store.layout.GenerationPath(active.GenerationID); err == nil && filepath.Clean(gen) == filepath.Clean(path) {
				report.Kept = append(report.Kept, slash)
				continue
			}
		}
		if !gcSafePath(store.layout, path) {
			report.Kept = append(report.Kept, slash)
			continue
		}
		if failAt == FailGCDuringDelete {
			return report, errInjectedGC
		}
		if hooks.DuringDelete != nil {
			if err := hooks.DuringDelete(path); err != nil {
				return report, err
			}
		}
		if err := os.RemoveAll(path); err != nil {
			return report, err
		}
		report.Deleted = append(report.Deleted, slash)
	}
	return report, nil
}

var errInjectedGC = errors.New("injected gc failure")

func retentionSet(store *Store) (map[string]struct{}, error) {
	protected := map[string]struct{}{}
	txns, err := loadValidTransactions(store)
	if err != nil {
		return nil, err
	}
	active := store.ReadActive()
	if active.Valid {
		if path, err := store.layout.GenerationPath(active.GenerationID); err == nil {
			protected[filepath.Clean(path)] = struct{}{}
		}
	}
	if path, ok := latestPreviousCommitted(store, txns, active.GenerationID); ok {
		protected[filepath.Clean(path)] = struct{}{}
	}
	if path, ok := latestFailedProven(store, txns); ok {
		protected[filepath.Clean(path)] = struct{}{}
	}
	for _, txn := range txns {
		if !txn.State.Nonterminal() {
			continue
		}
		if path, err := store.layout.StagingPath(txn.ID); err == nil {
			protected[filepath.Clean(path)] = struct{}{}
		}
		if path, err := store.layout.GenerationPath(txn.GenerationID); err == nil {
			protected[filepath.Clean(path)] = struct{}{}
		}
	}
	return protected, nil
}

func latestPreviousCommitted(store *Store, txns []InstallTransaction, activeGen string) (string, bool) {
	var committed []InstallTransaction
	for _, txn := range txns {
		if txn.State == StateCommitted && txn.GenerationID != activeGen {
			committed = append(committed, txn)
		}
	}
	if len(committed) == 0 {
		return "", false
	}
	sort.Slice(committed, func(i, j int) bool {
		return txnTime(committed[i]).After(txnTime(committed[j]))
	})
	path, err := store.layout.GenerationPath(committed[0].GenerationID)
	if err != nil {
		return "", false
	}
	if _, err := os.Lstat(path); err != nil {
		return "", false
	}
	return path, true
}

func latestFailedProven(store *Store, txns []InstallTransaction) (string, bool) {
	type hit struct {
		path string
		at   time.Time
	}
	var hits []hit
	for _, txn := range txns {
		if txn.State != StateFailed {
			continue
		}
		for _, path := range failedTreePaths(store, txn) {
			if lineageProven(path, txn) {
				hits = append(hits, hit{path: path, at: txnTime(txn)})
			}
		}
	}
	if len(hits) == 0 {
		return "", false
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].at.After(hits[j].at) })
	return hits[0].path, true
}

func failedTreePaths(store *Store, txn InstallTransaction) []string {
	var paths []string
	if path, err := store.layout.StagingPath(txn.ID); err == nil {
		paths = append(paths, path)
	}
	if path, err := store.layout.FailedPath(txn.ID); err == nil {
		paths = append(paths, path)
	}
	if path, err := store.layout.GenerationPath(txn.GenerationID); err == nil {
		paths = append(paths, path)
	}
	return paths
}

func lineageProven(path string, txn InstallTransaction) bool {
	if _, err := os.Lstat(path); err != nil {
		return false
	}
	if txn.ManifestSHA256 != "" && txn.SealSHA256 != "" {
		return VerifyPublishedGeneration(path, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256) == nil
	}
	rec, ok := readGenerationFile(path)
	if ok {
		return rec.GenerationID == txn.GenerationID && rec.TransactionID == txn.ID
	}
	candidate, ok := readCandidateRecord(path)
	return ok && candidate.GenerationID == txn.GenerationID && candidate.TransactionID == txn.ID
}

func txnTime(txn InstallTransaction) time.Time {
	if txn.CommittedAt != nil {
		return *txn.CommittedAt
	}
	if !txn.UpdatedAt.IsZero() {
		return txn.UpdatedAt
	}
	return txn.CreatedAt
}

func gcCandidates(store *Store) ([]string, error) {
	txns, err := loadValidTransactions(store)
	if err != nil {
		return nil, err
	}
	knownTxn := map[string]InstallTransaction{}
	knownGen := map[string]InstallTransaction{}
	for _, txn := range txns {
		knownTxn[txn.ID] = txn
		knownGen[txn.GenerationID] = txn
	}
	var out []string
	out = append(out, listedKnownDirs(store.layout.StagingRoot(), knownTxn)...)
	out = append(out, listedKnownDirs(store.layout.FailedRoot(), knownTxn)...)
	if gens, err := os.ReadDir(store.layout.GenerationsRoot()); err == nil {
		for _, entry := range gens {
			if !entry.IsDir() {
				continue
			}
			txn, ok := knownGen[entry.Name()]
			if !ok {
				continue
			}
			path := filepath.Join(store.layout.GenerationsRoot(), entry.Name())
			if lineageProven(path, txn) {
				out = append(out, path)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

func listedKnownDirs(root string, known map[string]InstallTransaction) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		txn, ok := known[entry.Name()]
		if !ok {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if lineageProven(path, txn) {
			out = append(out, path)
		}
	}
	return out
}

func gcSafePath(layout Layout, path string) bool {
	clean := filepath.Clean(path)
	if !pathContained(layout.Root, clean) {
		return false
	}
	info, err := os.Lstat(clean)
	if err != nil || !info.IsDir() || isReparsePoint(info) {
		return false
	}
	if err := rejectReparse(clean); err != nil {
		return false
	}
	rel, err := filepath.Rel(layout.Root, clean)
	if err != nil {
		return false
	}
	slash := filepath.ToSlash(rel)
	switch {
	case hasDirPrefix(slash, dirStaging), hasDirPrefix(slash, dirFailed), hasDirPrefix(slash, dirGenerations), hasDirPrefix(slash, "cache"):
		return true
	default:
		return false
	}
}

func hasDirPrefix(rel, dir string) bool {
	return rel == dir || len(rel) > len(dir) && rel[:len(dir)+1] == dir+"/"
}

func loadValidTransactions(store *Store) ([]InstallTransaction, error) {
	views, _, err := store.inspectTransactions()
	if err != nil {
		return nil, err
	}
	var txns []InstallTransaction
	for _, view := range views {
		if !view.Valid {
			continue
		}
		txn, err := store.LoadTransaction(view.ID)
		if err != nil {
			continue
		}
		txns = append(txns, txn)
	}
	return txns, nil
}
