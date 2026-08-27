package hermes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/install"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/upgrademanagement"
)

const upgradeCandidateLabel = "Hermes 0.20.5 packaged snapshot"

type UpgradePlanner struct {
	now func() time.Time
}

func NewUpgradePlanner() *UpgradePlanner {
	return &UpgradePlanner{now: time.Now}
}

func (p *UpgradePlanner) PlanUpgrade(ctx context.Context, installation yorvaruntime.Installation) (yorvaruntime.UpgradePlan, error) {
	if err := ctx.Err(); err != nil {
		return yorvaruntime.UpgradePlan{}, err
	}
	if installation.RuntimeKind != Kind || installation.Path == "" || installation.Version == "" || installation.SupportState != yorvaruntime.DiscoverySupported {
		return yorvaruntime.UpgradePlan{}, yorvaruntime.ErrInvalidManagementContract
	}
	observedAt := time.Now().UTC()
	if p != nil && p.now != nil {
		observedAt = p.now().UTC()
	}
	target := packagedUpgradeTarget()
	active, managed := observeManagedActive(ctx, installation)
	// Version status is useful even for an externally managed or development
	// installation. Matching the packaged version proves that there is no
	// version transition to perform; it does not grant Upgrade/Rollback mutation
	// authority or claim that the current files equal YORVA's sealed snapshot.
	if installation.Version == target.Version {
		return yorvaruntime.UpgradePlan{
			State: yorvaruntime.UpgradeUpToDate, Rollback: yorvaruntime.RollbackUnknown,
			CurrentVersion: installation.Version, TargetVersion: target.Version,
			CandidateLabel: upgradeCandidateLabel, Compatibility: yorvaruntime.UpgradeCompatibilityNotRequired,
			Managed: managed, ObservedAt: observedAt,
		}, nil
	}
	input := upgrademanagement.UpgradePlanInput{
		Managed:         upgrademanagement.ManagedUnknown,
		Target:          upgrademanagement.TargetObservation{State: upgrademanagement.EvidenceVerified, Compiled: target, Observed: target},
		Inventory:       upgrademanagement.InventoryObservation{State: upgrademanagement.EvidenceUnknown},
		ProtectionPoint: upgrademanagement.ProtectionPointObservation{State: upgrademanagement.EvidenceMissing},
		Compatibility:   upgrademanagement.CompatibilityRecord{State: upgrademanagement.EvidenceUnknown},
		PostcheckPolicy: upgrademanagement.PostcheckQualification{State: upgrademanagement.EvidenceUnknown},
	}

	record, ok := active, managed
	if ok {
		input.Managed = upgrademanagement.ManagedProven
		input.Active = upgrademanagement.ActivePointerObservation{
			State:          upgrademanagement.ActivePointerValid,
			GenerationID:   record.GenerationID,
			SealSHA256:     record.SealSHA256,
			ManifestSHA256: record.ManifestSHA256,
		}
		if current, exact := knownSnapshot(record.Version, record.SourcePin); exact {
			input.Current = upgrademanagement.SealedGenerationObservation{
				State:          upgrademanagement.EvidenceVerified,
				GenerationID:   record.GenerationID,
				SealSHA256:     record.SealSHA256,
				ManifestSHA256: record.ManifestSHA256,
				LineageProven:  true,
				Identity:       current,
			}
		} else {
			input.Current = upgrademanagement.SealedGenerationObservation{State: upgrademanagement.EvidenceUnknown}
		}
	} else {
		input.Active = upgrademanagement.ActivePointerObservation{State: upgrademanagement.ActivePointerUnknown}
		input.Current = upgrademanagement.SealedGenerationObservation{State: upgrademanagement.EvidenceUnknown}
	}

	policy := upgrademanagement.BuildUpgradePlan(input)
	state := mapUpgradeAvailability(policy.Availability)
	currentVersion := installation.Version
	if policy.Current.Version != "" {
		currentVersion = policy.Current.Version
	}
	plan := yorvaruntime.UpgradePlan{
		State:                   state,
		Rollback:                yorvaruntime.RollbackUnknown,
		CurrentVersion:          currentVersion,
		TargetVersion:           target.Version,
		CandidateLabel:          upgradeCandidateLabel,
		Compatibility:           yorvaruntime.UpgradeCompatibilityUnknown,
		Reasons:                 publicUpgradeReasons(policy.Reasons),
		Managed:                 ok,
		InventoryComplete:       false,
		ProtectionPointRequired: state != yorvaruntime.UpgradeUpToDate,
		ProtectionPointReady:    false,
		PlanEvidenceComplete:    policy.PlanEvidenceComplete,
		ObservedAt:              observedAt,
	}
	if state == yorvaruntime.UpgradeUpToDate {
		plan.Compatibility = yorvaruntime.UpgradeCompatibilityNotRequired
		plan.Reasons = nil
		plan.ProtectionPointReady = false
	}
	if policy.RollbackEligible {
		plan.Rollback = yorvaruntime.RollbackEligible
	}
	return plan, nil
}

func observeManagedActive(ctx context.Context, installation yorvaruntime.Installation) (install.ActiveRecord, bool) {
	if installation.RuntimeKind != Kind || installation.Path == "" || installation.Version == "" {
		return install.ActiveRecord{}, false
	}
	abs, err := filepath.Abs(installation.Path)
	if err != nil {
		return install.ActiveRecord{}, false
	}
	ancestor := filepath.Dir(abs)
	for depth := 0; depth < 7; depth++ {
		if ctx.Err() != nil {
			return install.ActiveRecord{}, false
		}
		store, err := install.NewStore(ancestor)
		if err == nil {
			record, loadErr := store.LoadActive()
			if loadErr == nil && record.RuntimeKind == string(Kind) && record.Version == installation.Version {
				generation, pathErr := store.Layout().GenerationPath(record.GenerationID)
				if pathErr == nil && activeLauncherMatches(abs, generation) &&
					generationLineageMatches(generation, record) &&
					install.VerifySealedTree(generation, record.GenerationID, record.ManifestSHA256, record.SealSHA256) == nil {
					return record, true
				}
			}
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			break
		}
		ancestor = parent
	}
	return install.ActiveRecord{}, false
}

func generationLineageMatches(generation string, active install.ActiveRecord) bool {
	path := filepath.Join(generation, "generation.json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<10 {
		return false
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(payload)
	if hex.EncodeToString(sum[:]) != active.SealSHA256 {
		return false
	}
	var record install.GenerationRecord
	if json.Unmarshal(payload, &record) != nil {
		return false
	}
	lineage, err := hex.DecodeString(record.LineageID)
	return err == nil && len(lineage) == 16 &&
		record.GenerationID == active.GenerationID && record.TransactionID == active.TransactionID &&
		record.RuntimeKind == active.RuntimeKind && record.SourcePin == active.SourcePin &&
		record.ExpectedVersion == active.Version && record.GenerationRelativePath == active.GenerationRelativePath &&
		record.ManifestSHA256 == active.ManifestSHA256
}

func activeLauncherMatches(observed, generation string) bool {
	for _, launcher := range []string{
		filepath.Join(generation, "bin", "hermes.exe"),
		filepath.Join(generation, "venv", "Scripts", "hermes.exe"),
	} {
		if strings.EqualFold(filepath.Clean(observed), filepath.Clean(launcher)) {
			return true
		}
	}
	return false
}

func packagedUpgradeTarget() upgrademanagement.SnapshotIdentity {
	return snapshotIdentity(
		officialPackageVersion, officialCommit, officialArchiveSize, officialArchiveSHA256,
		officialArchiveRoot, officialScriptSize, officialScriptSHA256,
	)
}

func knownSnapshot(version, sourcePin string) (upgrademanagement.SnapshotIdentity, bool) {
	switch {
	case version == officialPackageVersion && sourcePin == officialCommit:
		return packagedUpgradeTarget(), true
	case version == "0.20.2" && sourcePin == "df4b65147d7ddd74dd449f9067aabbca5aef0ec7":
		return snapshotIdentity(
			"0.20.2", "df4b65147d7ddd74dd449f9067aabbca5aef0ec7", 71869305,
			"2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba",
			"hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7", 233712,
			"2e1de1867299ce34d5fc73ce63022934acb8966f69f3f53306a37afc3dac29a3",
		), true
	default:
		return upgrademanagement.SnapshotIdentity{}, false
	}
}

func snapshotIdentity(version, commit string, archiveSize int64, archiveSHA, archiveRoot string, scriptSize int64, scriptSHA string) upgrademanagement.SnapshotIdentity {
	return upgrademanagement.SnapshotIdentity{
		Version:     version,
		Commit:      commit,
		Archive:     upgrademanagement.ArtifactIdentity{SizeBytes: archiveSize, SHA256: archiveSHA},
		LicensePath: "LICENSE",
		License:     upgrademanagement.ArtifactIdentity{SizeBytes: officialLicenseSize, SHA256: officialLicenseSHA256},
		Source: upgrademanagement.SourceIdentity{
			Repository: officialRepository, ArchiveRoot: archiveRoot, InstallerPath: officialScriptPath,
			Installer: upgrademanagement.ArtifactIdentity{SizeBytes: scriptSize, SHA256: scriptSHA},
		},
	}
}

func mapUpgradeAvailability(state upgrademanagement.UpgradeAvailability) yorvaruntime.UpgradeAvailabilityState {
	switch state {
	case upgrademanagement.UpgradeUpToDate:
		return yorvaruntime.UpgradeUpToDate
	case upgrademanagement.UpgradeAvailable:
		return yorvaruntime.UpgradeAvailable
	case upgrademanagement.UpgradeBlocked:
		return yorvaruntime.UpgradeBlocked
	default:
		return yorvaruntime.UpgradeUnknown
	}
}

func publicUpgradeReasons(reasons []upgrademanagement.ReasonCode) []yorvaruntime.UpgradePlanReason {
	result := make([]yorvaruntime.UpgradePlanReason, 0, 6)
	seen := make(map[yorvaruntime.UpgradePlanReason]struct{}, 6)
	add := func(reason yorvaruntime.UpgradePlanReason) {
		if _, exists := seen[reason]; !exists {
			seen[reason] = struct{}{}
			result = append(result, reason)
		}
	}
	for _, reason := range reasons {
		switch reason {
		case upgrademanagement.ReasonManagedUnknown, upgrademanagement.ReasonUnmanaged,
			upgrademanagement.ReasonActivePointerUnknown, upgrademanagement.ReasonActivePointerNotValid:
			add(yorvaruntime.UpgradeReasonManagedEvidenceUnknown)
		case upgrademanagement.ReasonCurrentSealUnknown, upgrademanagement.ReasonCurrentSealInvalid,
			upgrademanagement.ReasonCurrentIdentityInvalid, upgrademanagement.ReasonActiveGenerationMismatch:
			add(yorvaruntime.UpgradeReasonCurrentIdentityUnknown)
		case upgrademanagement.ReasonInventoryUnknown, upgrademanagement.ReasonInventoryIncomplete:
			add(yorvaruntime.UpgradeReasonInventoryUnknown)
		case upgrademanagement.ReasonProtectionPointUnknown, upgrademanagement.ReasonProtectionPointUnavailable:
			add(yorvaruntime.UpgradeReasonProtectionRequired)
		case upgrademanagement.ReasonCompatibilityUnknown, upgrademanagement.ReasonCompatibilityMissing,
			upgrademanagement.ReasonCompatibilityInvalid, upgrademanagement.ReasonCompatibilityPairMismatch,
			upgrademanagement.ReasonUpgradeCompatibilityUnknown, upgrademanagement.ReasonUpgradeCompatibilityUnsafe,
			upgrademanagement.ReasonRollbackCompatibilityUnknown, upgrademanagement.ReasonRollbackCompatibilityUnsafe:
			add(yorvaruntime.UpgradeReasonCompatibilityUnknown)
		case upgrademanagement.ReasonPostchecksUnknown, upgrademanagement.ReasonPostchecksUnqualified:
			add(yorvaruntime.UpgradeReasonPostchecksUnqualified)
		}
	}
	return result
}
