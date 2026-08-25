package hermes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/install"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestUpgradePlannerReportsExactManagedCandidateButKeepsUnknownEvidenceClosed(t *testing.T) {
	installation := materializeUpgradeActive(t, "0.20.2", "df4b65147d7ddd74dd449f9067aabbca5aef0ec7")
	planner := NewUpgradePlanner()
	planner.now = func() time.Time { return time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC) }

	plan, err := planner.PlanUpgrade(context.Background(), installation)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("plan validation: %v; plan=%#v", err, plan)
	}
	if plan.State != yorvaruntime.UpgradeUnknown || !plan.Managed || plan.CurrentVersion != "0.20.2" ||
		plan.TargetVersion != "0.20.5" || plan.CandidateLabel != upgradeCandidateLabel ||
		plan.Compatibility != yorvaruntime.UpgradeCompatibilityUnknown || !plan.ProtectionPointRequired ||
		plan.ProtectionPointReady || plan.PlanEvidenceComplete || plan.UpgradeExecutable() || plan.RollbackExecutable() {
		t.Fatalf("unsafe Upgrade plan truth: %#v", plan)
	}
	for _, want := range []yorvaruntime.UpgradePlanReason{
		yorvaruntime.UpgradeReasonInventoryUnknown,
		yorvaruntime.UpgradeReasonProtectionRequired,
		yorvaruntime.UpgradeReasonCompatibilityUnknown,
		yorvaruntime.UpgradeReasonPostchecksUnqualified,
	} {
		if !slices.Contains(plan.Reasons, want) {
			t.Fatalf("Reasons = %v, want %s", plan.Reasons, want)
		}
	}
}

func TestUpgradePlannerTreatsExactActivePackagedSnapshotAsUpToDate(t *testing.T) {
	installation := materializeUpgradeActive(t, officialPackageVersion, officialCommit)
	plan, err := NewUpgradePlanner().PlanUpgrade(context.Background(), installation)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	if plan.State != yorvaruntime.UpgradeUpToDate || plan.Compatibility != yorvaruntime.UpgradeCompatibilityNotRequired ||
		plan.ProtectionPointRequired || len(plan.Reasons) != 0 || plan.UpgradeExecutable() || plan.RollbackExecutable() {
		t.Fatalf("up-to-date plan = %#v", plan)
	}
}

func TestUpgradePlannerDoesNotTrustVersionWithoutMatchingLiveSeal(t *testing.T) {
	installation := materializeUpgradeActive(t, "0.20.2", "df4b65147d7ddd74dd449f9067aabbca5aef0ec7")
	if err := os.WriteFile(installation.Path, []byte("mutated"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := NewUpgradePlanner().PlanUpgrade(context.Background(), installation)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Managed || plan.State != yorvaruntime.UpgradeUnknown || plan.PlanEvidenceComplete ||
		!slices.Contains(plan.Reasons, yorvaruntime.UpgradeReasonManagedEvidenceUnknown) {
		t.Fatalf("mutated active tree was trusted: %#v", plan)
	}
}

func materializeUpgradeActive(t *testing.T, version, sourcePin string) yorvaruntime.Installation {
	t.Helper()
	root := t.TempDir()
	store, err := install.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	const generationID = "gen_aaaaaaaaaaaaaaaaaaaaaa"
	const transactionID = "txn_bbbbbbbbbbbbbbbbbbbbbb"
	generation, err := store.Layout().GenerationPath(generationID)
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(generation, "bin", "hermes.exe")
	if err := os.MkdirAll(filepath.Dir(launcher), 0o700); err != nil {
		t.Fatal(err)
	}
	launcherBytes := []byte("sealed Hermes launcher")
	if err := os.WriteFile(launcher, launcherBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	launcherSum := sha256.Sum256(launcherBytes)
	manifestBytes, err := json.Marshal(install.ManifestFile{Schema: 1, Entries: []install.ManifestEntry{{
		Path: "bin/hermes.exe", Size: int64(len(launcherBytes)), SHA256: hex.EncodeToString(launcherSum[:]),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	manifestSum := sha256.Sum256(manifestBytes)
	manifestSHA := hex.EncodeToString(manifestSum[:])
	if err := os.WriteFile(filepath.Join(generation, "manifest.json"), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 11, 0, 0, 0, time.UTC)
	generationBytes, err := json.Marshal(install.GenerationRecord{
		Schema: 1, LineageID: "11111111111111111111111111111111", TransactionID: transactionID,
		GenerationID: generationID, RuntimeKind: string(Kind), SourcePin: sourcePin, ExpectedVersion: version,
		GenerationRelativePath: install.GenerationRel(generationID), ManifestSHA256: manifestSHA, CreatedAt: now, SealedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	generationBytes = append(generationBytes, '\n')
	sealSum := sha256.Sum256(generationBytes)
	sealSHA := hex.EncodeToString(sealSum[:])
	if err := os.WriteFile(filepath.Join(generation, "generation.json"), generationBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteActive(install.ActiveRecord{
		Schema: 1, RuntimeKind: string(Kind), GenerationID: generationID,
		GenerationRelativePath: install.GenerationRel(generationID), ManifestSHA256: manifestSHA, SealSHA256: sealSHA,
		SourcePin: sourcePin, Version: version, TransactionID: transactionID, ActivatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	return yorvaruntime.Installation{RuntimeKind: Kind, Path: launcher, Version: version, SupportState: yorvaruntime.DiscoverySupported}
}
