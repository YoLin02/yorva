import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { createDaemonClient } from "./api/client";
import { getDaemonSession, isDaemonNotReady } from "./api/session";
import { useEventStreamStatus } from "./hooks/useEventStreamStatus";
import { NodeStatusView } from "./components/NodeStatusView";

export function App() {
  const sessionQuery = useQuery({
    queryKey: ["daemon-session"],
    queryFn: getDaemonSession,
    retry: (failures, error) => failures < 20 && isDaemonNotReady(error),
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

  if (sessionQuery.isError) {
    return <NodeStatusView state={{ kind: "failure", message: "The local daemon could not start." }} />;
  }
  if (!sessionQuery.data || nodeQuery.isPending) {
    return <NodeStatusView state={{ kind: "starting" }} />;
  }
  if (nodeQuery.isError) {
    return <NodeStatusView state={{ kind: "failure", message: "The local Node could not be reached." }} />;
  }
  return <NodeStatusView state={{ kind: "connected", node: nodeQuery.data, eventStatus }} />;
}
