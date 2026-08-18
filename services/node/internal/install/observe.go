package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

var officialRootNames = map[string]struct{}{
	dirControl: {}, dirGenerations: {}, dirStaging: {}, dirFailed: {}, "cache": {}, "node": {},
	".env": {}, "config.yaml": {}, "SOUL.md": {}, "cron": {}, "sessions": {}, "logs": {},
	"pairing": {}, "hooks": {}, "image_cache": {}, "audio_cache": {}, "memories": {}, "skills": {},
}

func Observe(store *Store, env EnvironmentStore) (Observation, error) {
	views, collision, err := store.inspectTransactions()
	if err != nil {
		return Observation{}, err
	}
	active := store.ReadActive()
	rec, _ := store.LoadActive()
	primary := primaryTxn(views, active)
	var staging, generation TreeObservation
	if primary.Valid {
		full, loadErr := store.LoadTransaction(primary.ID)
		if loadErr == nil {
			if path, pathErr := store.layout.StagingPath(full.ID); pathErr == nil {
				staging = observeTree(path, full)
			}
			if path, pathErr := store.layout.GenerationPath(full.GenerationID); pathErr == nil {
				generation = observeTree(path, full)
			}
		}
	}
	unknown, err := scanUnknownDirectories(store.layout, views)
	if err != nil {
		return Observation{}, err
	}
	return Observation{
		Transactions:        views,
		Staging:             staging,
		Generation:          generation,
		UnknownDirectories:  unknown,
		ReservedIDCollision: collision,
		Active:              active,
		Environment:         observeEnvironment(store, rec, env),
	}, nil
}

func primaryTxn(views []TransactionView, active ActivePointer) TransactionView {
	var nonterminal []TransactionView
	for _, view := range views {
		if view.Valid && view.State.Nonterminal() {
			nonterminal = append(nonterminal, view)
		}
	}
	if len(nonterminal) == 1 {
		return nonterminal[0]
	}
	if active.Valid {
		for _, view := range views {
			if view.Valid && view.GenerationID == active.GenerationID {
				return view
			}
		}
	}
	return TransactionView{}
}

func observeTree(abs string, txn InstallTransaction) TreeObservation {
	info, err := os.Lstat(abs)
	if err != nil {
		return TreeObservation{}
	}
	obs := TreeObservation{Present: true}
	if err := rejectReparse(abs); err != nil || !info.IsDir() {
		return obs
	}
	empty, err := dirEmpty(abs)
	if err == nil {
		obs.Empty = empty
	}
	if rec, ok := readGenerationFile(abs); ok {
		obs.LineageID = rec.GenerationID
		obs.LineageMatch = rec.GenerationID == txn.GenerationID
	}
	if txn.ManifestSHA256 != "" && txn.SealSHA256 != "" &&
		VerifyPublishedGeneration(abs, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256) == nil {
		obs.Sealed = true
		obs.LineageMatch = true
		obs.LineageID = txn.GenerationID
	}
	return obs
}

func readGenerationFile(abs string) (GenerationRecord, bool) {
	payload, err := os.ReadFile(filepath.Join(abs, fileGeneration))
	if err != nil {
		return GenerationRecord{}, false
	}
	var rec GenerationRecord
	if json.Unmarshal(payload, &rec) != nil || rec.GenerationID == "" {
		return GenerationRecord{}, false
	}
	return rec, true
}

func observeEnvironment(store *Store, rec ActiveRecord, env EnvironmentStore) EnvironmentObservation {
	if rec.GenerationID == "" || env.Read == nil {
		return EnvironmentObservation{}
	}
	policy, err := PolicyFromActive(store, rec, store.HasCommitted())
	if err != nil {
		return EnvironmentObservation{}
	}
	observed, err := env.Read()
	if err != nil {
		return EnvironmentObservation{}
	}
	return ComputeEnvironmentPlan(policy, observed).Observation(observed)
}

func scanUnknownDirectories(layout Layout, views []TransactionView) ([]string, error) {
	knownTxn := map[string]struct{}{}
	knownGen := map[string]struct{}{}
	for _, view := range views {
		if view.Valid {
			knownTxn[view.ID] = struct{}{}
			knownGen[view.GenerationID] = struct{}{}
		}
	}
	var unknown []string
	entries, err := os.ReadDir(layout.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if _, official := officialRootNames[name]; official {
			continue
		}
		if entry.IsDir() {
			unknown = append(unknown, name)
		}
	}
	if gens, err := os.ReadDir(layout.GenerationsRoot()); err == nil {
		for _, entry := range gens {
			if !entry.IsDir() {
				continue
			}
			if _, known := knownGen[entry.Name()]; !known {
				unknown = append(unknown, dirGenerations+"/"+entry.Name())
			}
		}
	}
	if stages, err := os.ReadDir(layout.StagingRoot()); err == nil {
		for _, entry := range stages {
			if !entry.IsDir() {
				continue
			}
			if _, known := knownTxn[entry.Name()]; !known {
				unknown = append(unknown, dirStaging+"/"+entry.Name())
			}
		}
	}
	sort.Strings(unknown)
	return unknown, nil
}
