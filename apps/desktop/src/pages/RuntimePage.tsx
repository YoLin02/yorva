import { HermesDiscoveryView, type HermesDiscoveryViewState } from "../components/HermesDiscoveryView";
import { HermesInstallPanel } from "../components/HermesInstallPanel";
import { HermesPrerequisitePanel } from "../components/HermesPrerequisitePanel";
import type { HermesPrerequisites, Operation } from "../api/types";
import type { InstallRequestError } from "../installDiagnostic";
import type { AppMessages, Locale } from "../i18n";

export function RuntimePage({
  discoveryState,
  discoveryReady,
  copy,
  locale,
  prerequisites,
  prereqOperation,
  prereqLog,
  prereqBusy,
  prereqBlocked,
  prereqRequestError,
  hermesNotInstalled,
  onInstallPrereq,
  onRetryPrereq,
  onCancelPrereq,
  showInstallPanel,
  windowsHost,
  canStartInstall,
  confirmInstall,
  installBusy,
  installOperation,
  installLog,
  installRequestError,
  onOpenConfirm,
  onCloseConfirm,
  onConfirmInstall,
  onCancelInstall,
  onRetryInstall,
  instanceCount,
  onOpenInstances,
}: {
  discoveryState: HermesDiscoveryViewState;
  discoveryReady: boolean;
  copy: AppMessages;
  locale: Locale;
  prerequisites: HermesPrerequisites | null;
  prereqOperation: Operation | null;
  prereqLog: string;
  prereqBusy: boolean;
  prereqBlocked: boolean;
  prereqRequestError: InstallRequestError | null;
  hermesNotInstalled: boolean;
  onInstallPrereq: () => void;
  onRetryPrereq: () => void;
  onCancelPrereq: () => void;
  showInstallPanel: boolean;
  windowsHost: boolean;
  canStartInstall: boolean;
  confirmInstall: boolean;
  installBusy: boolean;
  installOperation: Operation | null;
  installLog: string;
  installRequestError: InstallRequestError | null;
  onOpenConfirm: () => void;
  onCloseConfirm: () => void;
  onConfirmInstall: () => void;
  onCancelInstall: () => void;
  onRetryInstall: () => void;
  instanceCount: number | null;
  onOpenInstances: () => void;
}) {
  return (
    <div className="page-stack runtime-page">
      <HermesDiscoveryView
        state={discoveryState}
        copy={copy}
        locale={locale}
        instanceCount={instanceCount}
        onOpenInstances={onOpenInstances}
      />
      {discoveryReady && (
        <HermesPrerequisitePanel
          copy={copy}
          locale={locale}
          status={prerequisites}
          operation={prereqOperation}
          liveLog={prereqLog}
          busy={prereqBusy}
          blocked={prereqBlocked}
          requestError={prereqRequestError}
          hermesNotInstalled={hermesNotInstalled}
          onInstall={onInstallPrereq}
          onRetryDeps={onRetryPrereq}
          onCancel={onCancelPrereq}
        />
      )}
      {showInstallPanel && (
        <HermesInstallPanel
          copy={copy}
          windowsHost={windowsHost}
          canStart={canStartInstall}
          confirmOpen={confirmInstall}
          busy={installBusy}
          operation={installOperation}
          liveLog={installLog}
          requestError={installRequestError}
          onOpenConfirm={onOpenConfirm}
          onCloseConfirm={onCloseConfirm}
          onConfirm={onConfirmInstall}
          onCancel={onCancelInstall}
          onRetry={onRetryInstall}
        />
      )}
    </div>
  );
}
