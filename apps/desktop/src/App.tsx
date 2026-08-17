import { useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createDaemonClient } from "./api/client";
import type { Operation } from "./api/types";
import { getDaemonSession, isDaemonNotReady } from "./api/session";
import { DesktopShell } from "./components/DesktopShell";
import { HermesDiscoveryView, type HermesDiscoveryViewState } from "./components/HermesDiscoveryView";
import { HermesInstallPanel } from "./components/HermesInstallPanel";
import { HermesSummaryCard } from "./components/HermesSummaryCard";
import { NodeStatusView } from "./components/NodeStatusView";
import { SettingsView } from "./components/SettingsView";
import { useEventStreamStatus } from "./hooks/useEventStreamStatus";
import { loadLocale, messages, saveLocale, type Locale, type PageId } from "./i18n";

export function App() {
  const queryClient = useQueryClient();
  const [activePage, setActivePage] = useState<PageId>("dashboard");
  const [locale, setLocale] = useState<Locale>(loadLocale);
  const [discoveryCancelled, setDiscoveryCancelled] = useState(false);
  const [confirmInstall, setConfirmInstall] = useState(false);
  const [installBusy, setInstallBusy] = useState(false);
  const [installKey, setInstallKey] = useState<string | null>(null);
  const [activeOperationId, setActiveOperationId] = useState<string | null>(null);
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
  const eventStatus = useEventStreamStatus(client);
  const discoveryKey = ["runtime-discovery", "hermes", sessionQuery.data?.baseUrl] as const;
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
  const operationQuery = useQuery({
    queryKey: ["runtime-install", activeOperationId, sessionQuery.data?.baseUrl],
    queryFn: ({ signal }) => client!.getOperation(activeOperationId!, signal),
    enabled: client !== undefined && activeOperationId !== null,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const startInstall = async () => {
    if (!client || installBusy) return;
    setInstallBusy(true);
    try {
      const key = installKey ?? crypto.randomUUID();
      setInstallKey(key);
      const operation = await client.startHermesInstall(key);
      setActiveOperationId(operation.id);
      setConfirmInstall(false);
    } finally {
      setInstallBusy(false);
    }
  };
  const retryInstall = () => {
    setInstallKey(crypto.randomUUID());
    setActiveOperationId(null);
    setConfirmInstall(true);
  };
  const cancelInstall = async () => {
    if (!client || !activeOperationId || installBusy) return;
    setInstallBusy(true);
    try {
      await client.cancelOperation(activeOperationId);
      await operationQuery.refetch();
    } finally {
      setInstallBusy(false);
    }
  };
  useEffect(() => {
    if (operationQuery.data?.status === "SUCCEEDED" && discoveryQuery.data?.state !== "SUPPORTED") {
      void discoveryQuery.refetch();
    }
  }, [operationQuery.data?.status, discoveryQuery.data?.state, discoveryQuery]);

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
    content = (
      <div>
        <HermesDiscoveryView state={discoveryState} copy={copy} locale={locale} />
        {notInstalled && (
          <HermesInstallPanel
            copy={copy}
            windowsHost={windowsHost}
            confirmOpen={confirmInstall}
            busy={installBusy}
            operation={(operationQuery.data as Operation | undefined) ?? null}
            onOpenConfirm={() => setConfirmInstall(true)}
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
