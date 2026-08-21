package hermes

import (
	"context"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/install"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func (h *HostInstaller) applyGeneration(ctx context.Context, operationID string, report func(operation.Stage, string)) error {
	if h.currentOp.ID == "" {
		h.SetInstallIdentity(operation.Operation{
			ID:        operationID,
			TargetID:  "hermes",
			SourcePin: officialCommit,
		})
	}
	if !h.PlatformSupported() {
		return installError(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, errPlatform)
	}
	root := h.home()
	lock, err := install.AcquireLock(root)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	defer lock.Release()
	store, err := install.NewStore(root)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if report != nil {
		h.afterStage = func(stage, _ string) {
			report(installStageName(stage), "")
		}
	}
	mgr := install.NewManager(store, func(ctx context.Context, generation, home string) error {
		return h.BuildGeneration(ctx, operationID, generation, home)
	}, func(ctx context.Context, generation, version string) error {
		return h.ValidateGeneration(ctx, generation, version)
	})
	if h.env.Read != nil {
		mgr = mgrWithEnv(mgr, h.env)
	}
	var txn install.InstallTransaction
	if h.currentOp.TransactionID != "" {
		txn, err = store.LoadTransaction(h.currentOp.TransactionID)
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
	} else {
		txn, err = install.NewCreatedTransaction(string(Kind), operationID, firstNonEmpty(h.currentOp.SourcePin, officialCommit), officialPackageVersion)
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
	}
	if report != nil {
		report(operation.StageSourceDownload, "")
	}
	txn, err = mgr.SealNew(ctx, txn)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if report != nil {
		report(operation.StageInstallPath, "")
	}
	txn, err = mgr.PublishAndActivate(ctx, txn)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	txn, err = mgr.ReconcileEnvironment(ctx, txn)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if txn.State != install.StateCommitted {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, errPlatform)
	}
	if report != nil {
		report(operation.StageCleanup, "")
	}
	return nil
}

func mgrWithEnv(mgr *install.Manager, env install.EnvironmentStore) *install.Manager {
	return install.WithEnvironment(mgr, env)
}
