import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createDaemonClient } from "./api/client";
import { YorvaApiError } from "./api/client";
import type { InstallRequestError } from "./installDiagnostic";
import { getDaemonSession, isDaemonNotReady } from "./api/session";
import { DesktopShell } from "./components/DesktopShell";
import { HermesDiscoveryView, type HermesDiscoveryViewState } from "./components/HermesDiscoveryView";
import { HermesInstallPanel } from "./components/HermesInstallPanel";
import { HermesPrerequisitePanel } from "./components/HermesPrerequisitePanel";
import { HermesSummaryCard } from "./components/HermesSummaryCard";
import { NodeStatusView } from "./components/NodeStatusView";
import { SettingsView } from "./components/SettingsView";
import { useEventStreamStatus } from "./hooks/useEventStreamStatus";
import { loadLocale, messages, saveLocale, type Locale, type PageId } from "./i18n";
import {
  isHermesPrerequisite,
  isHermesRuntimeInstall,
  newestActiveOperation,
  operationIdFromConflict,
} from "./operationRecovery";

export function App() {
  const queryClient = useQueryClient();
  const [activePage, setActivePage] = useState<PageId>("dashboard");
  const [locale, setLocale] = useState<Locale>(loadLocale);
  const [discoveryCancelled, setDiscoveryCancelled] = useState(false);
  const [confirmInstall, setConfirmInstall] = useState(false);
  const [installBusy, setInstallBusy] = useState(false);
  const [installKey, setInstallKey] = useState<string | null>(null);
  const [activeOperationId, setActiveOperationId] = useState<string | null>(null);
  const [installRequestError, setInstallRequestError] = useState<InstallRequestError | null>(null);
  const [prereqKey, setPrereqKey] = useState<string | null>(null);
  const [prereqOperationId, setPrereqOperationId] = useState<string | null>(null);
  const [prereqBusy, setPrereqBusy] = useState(false);
  const [prereqRequestError, setPrereqRequestError] = useState<InstallRequestError | null>(null);
  const copy = messages[locale];

  const sessionQuery = useQuery({
    queryKey: ["daemon-session"],
    queryFn: getDaemonSession,
    retry: (_failures, error) => isDaemonNotReady(error),
    retryDelay: 200,
  });
  const client = useMemo(
    () => (sessionQuery.data ? createDaemonClient(sessionQuery.data) : undefined),
    [sessionQuery.data],
  );
  const nodeQuery = useQuery({
    queryKey: ["local-node", sessionQuery.data?.baseUrl],
    queryFn: () => client!.getNode(),
    enabled: client !== undefined,
  });
  const refreshFollowedOperations = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["runtime-install"] });
    void queryClient.invalidateQueries({ queryKey: ["runtime-install-log"] });
    void queryClient.invalidateQueries({ queryKey: ["hermes-prereq-operation"] });
    void queryClient.invalidateQueries({ queryKey: ["hermes-prereq-log"] });
    void queryClient.invalidateQueries({ queryKey: ["hermes-operations"] });
    void queryClient.invalidateQueries({ queryKey: ["hermes-prerequisites"] });
  }, [queryClient]);
  const eventStatus = useEventStreamStatus(
    client,
    (event) => {
      if (typeof event.type === "string" && event.type.startsWith("operation.")) {
        refreshFollowedOperations();
      }
    },
    refreshFollowedOperations,
  );
  const discoveryKey = useMemo(
    () => ["runtime-discovery", "hermes", sessionQuery.data?.baseUrl] as const,
    [sessionQuery.data?.baseUrl],
  );
  const discoveryQuery = useQuery({
    queryKey: discoveryKey,
    queryFn: ({ signal }) => client!.detectHermes(signal),
    enabled: client !== undefined && nodeQuery.isSuccess,
    retry: false,
  });

  const changeLocale = (nextLocale: Locale) => {
    setLocale(nextLocale);
    saveLocale(nextLocale);
  };
  const toggleLocale = () => changeLocale(locale === "en-US" ? "zh-CN" : "en-US");
  const cancelDiscovery = () => {
    setDiscoveryCancelled(true);
    void queryClient.cancelQueries({ queryKey: discoveryKey, exact: true });
  };
  const retryDiscovery = () => {
    setDiscoveryCancelled(false);
    void discoveryQuery.refetch({ cancelRefetch: true });
  };
  const hermesOperationsQuery = useQuery({
    queryKey: ["hermes-operations", sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.listOperations("runtime-kind", "hermes", signal),
    enabled: client !== undefined && nodeQuery.isSuccess,
    retry: false,
  });
  const recoveredInstallId = newestActiveOperation(hermesOperationsQuery.data?.operations, isHermesRuntimeInstall)?.id ?? null;
  const recoveredPrereqId = newestActiveOperation(hermesOperationsQuery.data?.operations, isHermesPrerequisite)?.id ?? null;
  const followedInstallId = activeOperationId ?? recoveredInstallId;
  const operationQuery = useQuery({
    queryKey: ["runtime-install", followedInstallId, sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.getOperation(followedInstallId!, signal),
    enabled: client !== undefined && followedInstallId !== null,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const operationLogQuery = useQuery({
    queryKey: ["runtime-install-log", followedInstallId, sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.getOperationLog(followedInstallId!, signal),
    enabled: client !== undefined && followedInstallId !== null,
    refetchInterval: () => {
      const status = operationQuery.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const attachConflict = async (daemon: NonNullable<typeof client>, error: unknown): Promise<"install" | "prereq" | null> => {
    const id = operationIdFromConflict(error);
    if (!id) {
      return null;
    }
    try {
      const operation = await daemon.getOperation(id);
      if (isHermesRuntimeInstall(operation) && (operation.status === "PENDING" || operation.status === "RUNNING")) {
        setActiveOperationId(operation.id);
        return "install";
      }
      if (isHermesPrerequisite(operation) && (operation.status === "PENDING" || operation.status === "RUNNING")) {
        setPrereqOperationId(operation.id);
        return "prereq";
      }
    } catch {
      return null;
    }
    return null;
  };
  const startInstall = async () => {
    if (!client || installBusy) return;
    setInstallBusy(true);
    setInstallRequestError(null);
    try {
      const key = installKey ?? crypto.randomUUID();
      setInstallKey(key);
      const operation = await client.startHermesInstall(key);
      if (!isHermesRuntimeInstall(operation)) {
        throw new YorvaApiError({
          code: "INTERNAL_ERROR",
          message: copy.node.nodeReachFailure,
          retryable: true,
          details: {},
        });
      }
      setActiveOperationId(operation.id);
      setConfirmInstall(false);
    } catch (error) {
      setConfirmInstall(false);
      const attached = await attachConflict(client, error);
      if (attached === "install") {
        setInstallRequestError(null);
      } else if (error instanceof YorvaApiError) {
        setInstallRequestError({
          code: error.code,
          message: error.message,
          retryable: error.retryable,
        });
      } else {
        setInstallRequestError({
          code: "INTERNAL_ERROR",
          message: copy.node.nodeReachFailure,
          retryable: true,
        });
      }
    } finally {
      setInstallBusy(false);
    }
  };
  const retryInstall = () => {
    setInstallKey(crypto.randomUUID());
    setActiveOperationId(null);
    setInstallRequestError(null);
    setConfirmInstall(true);
  };
  const prerequisitesQuery = useQuery({
    queryKey: ["hermes-prerequisites", sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.getHermesPrerequisites(signal),
    enabled: client !== undefined && nodeQuery.isSuccess && discoveryQuery.isSuccess,
    retry: false,
  });
  const followedPrereqOperationId = prereqOperationId ?? recoveredPrereqId ?? prerequisitesQuery.data?.activeOperationId ?? null;
  const prereqOperationQuery = useQuery({
    queryKey: ["hermes-prereq-operation", followedPrereqOperationId, sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.getOperation(followedPrereqOperationId!, signal),
    enabled: client !== undefined && followedPrereqOperationId !== null,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const prereqLogQuery = useQuery({
    queryKey: ["hermes-prereq-log", followedPrereqOperationId, sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.getOperationLog(followedPrereqOperationId!, signal),
    enabled: client !== undefined && followedPrereqOperationId !== null,
    refetchInterval: () => {
      const status = prereqOperationQuery.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const startPrereq = async (nextKey?: string) => {
    if (!client || prereqBusy) return;
    const key = nextKey ?? prereqKey ?? crypto.randomUUID();
    setPrereqBusy(true);
    setPrereqRequestError(null);
    setPrereqKey(key);
    if (nextKey) {
      setPrereqOperationId(null);
    }
    try {
      const operation = await client.startHermesPrerequisites(key);
      if (!isHermesPrerequisite(operation)) {
        throw new YorvaApiError({
          code: "INTERNAL_ERROR",
          message: copy.node.nodeReachFailure,
          retryable: true,
          details: {},
        });
      }
      setPrereqOperationId(operation.id);
    } catch (error) {
      const attached = await attachConflict(client, error);
      if (attached === "prereq" || attached === "install") {
        setPrereqRequestError(null);
      } else if (error instanceof YorvaApiError) {
        setPrereqRequestError({
          code: error.code,
          message: error.message,
          retryable: error.retryable,
        });
      } else {
        setPrereqRequestError({
          code: "INTERNAL_ERROR",
          message: copy.node.nodeReachFailure,
          retryable: true,
        });
      }
      void prerequisitesQuery.refetch();
    } finally {
      setPrereqBusy(false);
    }
  };
  const retryPrereq = () => {
    void startPrereq(crypto.randomUUID());
  };
  const cancelPrereq = async () => {
    if (!client || !followedPrereqOperationId || prereqBusy) return;
    setPrereqBusy(true);
    try {
      await client.cancelOperation(followedPrereqOperationId);
      await prereqOperationQuery.refetch();
    } finally {
      setPrereqBusy(false);
    }
  };
  useEffect(() => {
    if (prereqOperationQuery.data?.status === "SUCCEEDED") {
      void prerequisitesQuery.refetch();
    }
  }, [prereqOperationQuery.data?.status, prerequisitesQuery]);
  const cancelInstall = async () => {
    if (!client || !followedInstallId || installBusy) return;
    setActiveOperationId(followedInstallId);
    setInstallBusy(true);
    try {
      await client.cancelOperation(followedInstallId);
      await operationQuery.refetch();
    } finally {
      setInstallBusy(false);
    }
  };
  useEffect(() => {
    const status = operationQuery.data?.status;
    if (status === "SUCCEEDED" || status === "FAILED" || status === "CANCELLED") {
      void queryClient.invalidateQueries({ queryKey: discoveryKey });
    }
  }, [operationQuery.data?.id, operationQuery.data?.status, queryClient, discoveryKey]);

  let discoveryState: HermesDiscoveryViewState;
  if (discoveryCancelled) {
    discoveryState = { kind: "cancelled", onRetry: retryDiscovery };
  } else if (discoveryQuery.isPending || discoveryQuery.isFetching) {
    discoveryState = { kind: "checking", onCancel: cancelDiscovery };
  } else if (discoveryQuery.isError) {
    discoveryState = { kind: "failure", onRetry: retryDiscovery };
  } else {
    discoveryState = { kind: "complete", discovery: discoveryQuery.data, onRetry: retryDiscovery };
  }

  let content;
  if (activePage === "settings") {
    content = <SettingsView copy={copy} locale={locale} onLocaleChange={changeLocale} />;
  } else if (sessionQuery.isError) {
    content = <NodeStatusView state={{ kind: "failure", message: copy.node.daemonStartFailure }} copy={copy} />;
  } else if (!sessionQuery.data || nodeQuery.isPending) {
    content = <NodeStatusView state={{ kind: "starting" }} copy={copy} />;
  } else if (nodeQuery.isError) {
    content = <NodeStatusView state={{ kind: "failure", message: copy.node.nodeReachFailure }} copy={copy} />;
  } else if (activePage === "runtimes") {
    const windowsHost = nodeQuery.data.platform.toLowerCase() === "windows";
    const notInstalled = discoveryQuery.data?.state === "NOT_INSTALLED";
    const installOperation = operationQuery.data && isHermesRuntimeInstall(operationQuery.data) ? operationQuery.data : null;
    const prereqOperation = prereqOperationQuery.data && isHermesPrerequisite(prereqOperationQuery.data) ? prereqOperationQuery.data : null;
    const installBlocking = Boolean(installOperation && (installOperation.status === "PENDING" || installOperation.status === "RUNNING"));
    const prereqBlocking = Boolean(prereqOperation && (prereqOperation.status === "PENDING" || prereqOperation.status === "RUNNING"));
    const showInstallPanel = notInstalled || followedInstallId !== null || installRequestError !== null;
    content = (
      <div>
        <HermesDiscoveryView state={discoveryState} copy={copy} locale={locale} />
        {discoveryQuery.isSuccess && (
          <HermesPrerequisitePanel
            copy={copy}
            status={prerequisitesQuery.data ?? null}
            operation={prereqOperation}
            liveLog={prereqLogQuery.data?.text ?? ""}
            busy={prereqBusy}
            blocked={installBlocking}
            requestError={prereqRequestError}
            hermesNotInstalled={notInstalled}
            onInstall={() => { void startPrereq(); }}
            onRetryDeps={retryPrereq}
            onCancel={() => { void cancelPrereq(); }}
          />
        )}
        {showInstallPanel && (
          <HermesInstallPanel
            copy={copy}
            windowsHost={windowsHost}
            canStart={notInstalled}
            confirmOpen={confirmInstall && notInstalled && !prereqBlocking}
            busy={installBusy || prereqBlocking}
            operation={installOperation}
            liveLog={operationLogQuery.data?.text ?? ""}
            requestError={installRequestError}
            onOpenConfirm={() => {
              if (!notInstalled || installBlocking) return;
              setInstallRequestError(null);
              setConfirmInstall(true);
            }}
            onCloseConfirm={() => setConfirmInstall(false)}
            onConfirm={() => { void startInstall(); }}
            onCancel={() => { void cancelInstall(); }}
            onRetry={retryInstall}
          />
        )}
      </div>
    );
  } else {
    content = (
      <div className="dashboard-grid">
        <NodeStatusView state={{ kind: "connected", node: nodeQuery.data, eventStatus }} copy={copy} />
        <HermesSummaryCard state={discoveryState} copy={copy} onOpen={() => setActivePage("runtimes")} />
      </div>
    );
  }

  const targetLocale = locale === "en-US" ? "zh-CN" : "en-US";
  return (
    <DesktopShell
      activePage={activePage}
      copy={copy}
      locale={locale}
      nodeVersion={nodeQuery.data?.nodeVersion}
      onNavigate={setActivePage}
      onToggleLocale={toggleLocale}
      targetLocaleLabel={messages[targetLocale].languageName}
    >
      {content}
    </DesktopShell>
  );
}
