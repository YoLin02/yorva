import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createDaemonClient } from "./api/client";
import { getDaemonSession, isDaemonNotReady } from "./api/session";
import { useEventStreamStatus } from "./hooks/useEventStreamStatus";
import { NodeStatusView } from "./components/NodeStatusView";
import { HermesDiscoveryView, type HermesDiscoveryViewState } from "./components/HermesDiscoveryView";

export function App() {
	const queryClient = useQueryClient();
	const [discoveryCancelled, setDiscoveryCancelled] = useState(false);
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

  if (sessionQuery.isError) {
    return <NodeStatusView state={{ kind: "failure", message: "The local daemon could not start." }} />;
  }
  if (!sessionQuery.data || nodeQuery.isPending) {
    return <NodeStatusView state={{ kind: "starting" }} />;
  }
  if (nodeQuery.isError) {
    return <NodeStatusView state={{ kind: "failure", message: "The local Node could not be reached." }} />;
  }
  const cancelDiscovery = () => {
    setDiscoveryCancelled(true);
    void queryClient.cancelQueries({ queryKey: discoveryKey, exact: true });
  };
  const retryDiscovery = () => {
    setDiscoveryCancelled(false);
    void discoveryQuery.refetch({ cancelRefetch: true });
  };
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
  return (
    <NodeStatusView state={{ kind: "connected", node: nodeQuery.data, eventStatus }}>
      <HermesDiscoveryView state={discoveryState} />
    </NodeStatusView>
  );
}
