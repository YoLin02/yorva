import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createDaemonClient } from "./api/client";
import { YorvaApiError } from "./api/client";
import type { InstallRequestError } from "./installDiagnostic";
import { getDaemonSession, isDaemonNotReady } from "./api/session";
import { DesktopShell } from "./components/layout/DesktopShell";
import type { HermesDiscoveryViewState } from "./components/HermesDiscoveryView";
import { useEventStreamStatus } from "./hooks/useEventStreamStatus";
import { loadLocale, messages, saveLocale, type Locale, type PageId } from "./i18n";
import {
  isHermesPrerequisite,
  isHermesRuntimeInstall,
  isInstanceCreate,
  isInstanceDelete,
  newestActiveOperation,
  operationIdFromConflict,
} from "./operationRecovery";
import { DashboardPage } from "./pages/DashboardPage";
import { InstancesPage } from "./pages/InstancesPage";
import { RuntimePage } from "./pages/RuntimePage";
import { SettingsPage } from "./pages/SettingsPage";

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
  const [createName, setCreateName] = useState("");
  const [createKey, setCreateKey] = useState<string | null>(null);
  const [createOperationId, setCreateOperationId] = useState<string | null>(null);
  const [createBusy, setCreateBusy] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<import("./api/types").Instance | null>(null);
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const [deleteKey, setDeleteKey] = useState<string | null>(null);
  const [deleteOperationId, setDeleteOperationId] = useState<string | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);
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
    void queryClient.invalidateQueries({ queryKey: ["hermes-instances"] });
    void queryClient.invalidateQueries({ queryKey: ["instance-operations"] });
    void queryClient.invalidateQueries({ queryKey: ["instance-create"] });
    void queryClient.invalidateQueries({ queryKey: ["instance-delete"] });
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

  const hermesSupported = discoveryQuery.data?.state === "SUPPORTED";
  const instancesQuery = useQuery({
    queryKey: ["hermes-instances", sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.listHermesInstances(signal),
    enabled: client !== undefined && nodeQuery.isSuccess && hermesSupported && activePage === "instances",
    retry: false,
  });
  const instanceOperationsQuery = useQuery({
    queryKey: ["instance-operations", instancesQuery.data?.runtimeInstallationId, sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) =>
      client!.listOperations("runtime-installation", instancesQuery.data!.runtimeInstallationId, signal),
    enabled: client !== undefined && Boolean(instancesQuery.data?.runtimeInstallationId),
    retry: false,
  });
  const recoveredCreate = newestActiveOperation(instanceOperationsQuery.data?.operations, isInstanceCreate);
  const recoveredDelete = newestActiveOperation(instanceOperationsQuery.data?.operations, isInstanceDelete);
  const followedCreateId = createOperationId ?? recoveredCreate?.id ?? null;
  const followedDeleteId = deleteOperationId ?? recoveredDelete?.id ?? null;
  const recoveredDeleteTarget =
    instancesQuery.data?.instances.find((item) => item.name === recoveredDelete?.message) ?? null;
  const resolvedDeleteTarget = deleteTarget ?? recoveredDeleteTarget;
  const resolvedDeleteConfirmation = deleteTarget
    ? deleteConfirmation
    : (recoveredDelete?.message ?? deleteConfirmation);
  const createOperationQuery = useQuery({
    queryKey: ["instance-create", followedCreateId, sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.getOperation(followedCreateId!, signal),
    enabled: client !== undefined && followedCreateId !== null,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const startCreate = async () => {
    if (!client || createBusy) return;
    setCreateBusy(true);
    try {
      const key = createKey ?? crypto.randomUUID();
      setCreateKey(key);
      const operation = await client.createHermesInstance(createName, key);
      setCreateOperationId(operation.id);
    } finally {
      setCreateBusy(false);
    }
  };
  const cancelCreate = async () => {
    if (!client || !followedCreateId || createBusy) return;
    setCreateBusy(true);
    try {
      await client.cancelOperation(followedCreateId);
      await createOperationQuery.refetch();
    } finally {
      setCreateBusy(false);
    }
  };
  useEffect(() => {
    if (createOperationQuery.data?.status === "SUCCEEDED") {
      void queryClient.invalidateQueries({ queryKey: ["hermes-instances"] });
    }
  }, [createOperationQuery.data?.status, queryClient]);
  const deleteOperationQuery = useQuery({
    queryKey: ["instance-delete", followedDeleteId, sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.getOperation(followedDeleteId!, signal),
    enabled: client !== undefined && followedDeleteId !== null,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const startDelete = async () => {
    if (!client || !resolvedDeleteTarget || deleteBusy) return;
    setDeleteBusy(true);
    try {
      const key = deleteKey ?? crypto.randomUUID();
      setDeleteKey(key);
      const operation = await client.deleteInstance(
        resolvedDeleteTarget.instanceId,
        resolvedDeleteConfirmation,
        key,
      );
      setDeleteOperationId(operation.id);
    } finally {
      setDeleteBusy(false);
    }
  };
  const cancelDelete = async () => {
    if (!client || !followedDeleteId || deleteBusy) return;
    setDeleteBusy(true);
    try {
      await client.cancelOperation(followedDeleteId);
      await deleteOperationQuery.refetch();
    } finally {
      setDeleteBusy(false);
    }
  };
  useEffect(() => {
    if (deleteOperationQuery.data?.status === "SUCCEEDED") {
      void queryClient.invalidateQueries({ queryKey: ["hermes-instances"] });
    }
  }, [deleteOperationQuery.data?.status, queryClient]);

  let content;
  if (activePage === "settings") {
    content = <SettingsPage copy={copy} locale={locale} onLocaleChange={changeLocale} />;
  } else if (sessionQuery.isError) {
    content = (
      <DashboardPage
        nodeState={{ kind: "failure", message: copy.node.daemonStartFailure }}
        discoveryState={discoveryState}
        copy={copy}
        locale={locale}
        onOpenRuntimes={() => setActivePage("runtimes")}
        onOpenSettings={() => setActivePage("settings")}
      />
    );
  } else if (!sessionQuery.data || nodeQuery.isPending) {
    content = (
      <DashboardPage
        nodeState={{ kind: "starting" }}
        discoveryState={discoveryState}
        copy={copy}
        locale={locale}
        onOpenRuntimes={() => setActivePage("runtimes")}
        onOpenSettings={() => setActivePage("settings")}
      />
    );
  } else if (nodeQuery.isError) {
    content = (
      <DashboardPage
        nodeState={{ kind: "failure", message: copy.node.nodeReachFailure }}
        discoveryState={discoveryState}
        copy={copy}
        locale={locale}
        onOpenRuntimes={() => setActivePage("runtimes")}
        onOpenSettings={() => setActivePage("settings")}
      />
    );
  } else if (activePage === "runtimes") {
    const windowsHost = nodeQuery.data.platform.toLowerCase() === "windows";
    const notInstalled = discoveryQuery.data?.state === "NOT_INSTALLED";
    const installOperation = operationQuery.data && isHermesRuntimeInstall(operationQuery.data) ? operationQuery.data : null;
    const prereqOperation = prereqOperationQuery.data && isHermesPrerequisite(prereqOperationQuery.data) ? prereqOperationQuery.data : null;
    const installBlocking = Boolean(installOperation && (installOperation.status === "PENDING" || installOperation.status === "RUNNING"));
    const prereqBlocking = Boolean(prereqOperation && (prereqOperation.status === "PENDING" || prereqOperation.status === "RUNNING"));
    const showInstallPanel = notInstalled || followedInstallId !== null || installRequestError !== null;
    content = (
      <RuntimePage
        discoveryState={discoveryState}
        discoveryReady={discoveryQuery.isSuccess}
        copy={copy}
        locale={locale}
        prerequisites={prerequisitesQuery.data ?? null}
        prereqOperation={prereqOperation}
        prereqLog={prereqLogQuery.data?.text ?? ""}
        prereqBusy={prereqBusy}
        prereqBlocked={installBlocking}
        prereqRequestError={prereqRequestError}
        hermesNotInstalled={notInstalled}
        onInstallPrereq={() => { void startPrereq(); }}
        onRetryPrereq={retryPrereq}
        onCancelPrereq={() => { void cancelPrereq(); }}
        showInstallPanel={showInstallPanel}
        windowsHost={windowsHost}
        canStartInstall={notInstalled}
        confirmInstall={confirmInstall && notInstalled && !prereqBlocking}
        installBusy={installBusy || prereqBlocking}
        installOperation={installOperation}
        installLog={operationLogQuery.data?.text ?? ""}
        installRequestError={installRequestError}
        onOpenConfirm={() => {
          if (!notInstalled || installBlocking) return;
          setInstallRequestError(null);
          setConfirmInstall(true);
        }}
        onCloseConfirm={() => setConfirmInstall(false)}
        onConfirmInstall={() => { void startInstall(); }}
        onCancelInstall={() => { void cancelInstall(); }}
        onRetryInstall={retryInstall}
      />
    );
  } else if (activePage === "instances") {
    content = (
      <InstancesPage
        supported={hermesSupported}
        loading={instancesQuery.isPending || instancesQuery.isFetching}
        error={instancesQuery.isError}
        inventory={instancesQuery.data ?? null}
        createName={createName}
        createBusy={createBusy}
        createOperation={createOperationQuery.data ?? null}
        copy={copy}
        locale={locale}
        onRefresh={() => {
          void instancesQuery.refetch();
        }}
        onCreateNameChange={setCreateName}
        onCreate={() => {
          void startCreate();
        }}
        onCancelCreate={() => {
          void cancelCreate();
        }}
        deleteTarget={resolvedDeleteTarget}
        deleteConfirmation={resolvedDeleteConfirmation}
        deleteBusy={deleteBusy}
        deleteOperation={deleteOperationQuery.data ?? null}
        onDeleteTargetChange={(item) => {
          setDeleteTarget(item);
          setDeleteConfirmation("");
          setDeleteOperationId(null);
          setDeleteKey(null);
        }}
        onDeleteConfirmationChange={setDeleteConfirmation}
        onDelete={() => {
          void startDelete();
        }}
        onCancelDelete={() => {
          void cancelDelete();
        }}
      />
    );
  } else {
    content = (
      <DashboardPage
        nodeState={{ kind: "connected", node: nodeQuery.data, eventStatus }}
        discoveryState={discoveryState}
        copy={copy}
        locale={locale}
        onOpenRuntimes={() => setActivePage("runtimes")}
        onOpenSettings={() => setActivePage("settings")}
      />
    );
  }

  return (
    <DesktopShell
      activePage={activePage}
      copy={copy}
      locale={locale}
      onNavigate={setActivePage}
      onLocaleChange={changeLocale}
    >
      {content}
    </DesktopShell>
  );
}
