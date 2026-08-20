import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QRCodeSVG } from "qrcode.react";
import { useEffect, useMemo, useState } from "react";
import type { DaemonClient } from "../../api/client";
import type { Channel, Instance, Operation } from "../../api/types";
import type { AppMessages } from "../../i18n";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { IconClose, IconRefresh } from "../ui/icons";

type ChannelKind = Channel["type"];

export function ChannelPanel({ client, instance, copy, onClose }: {
  client: DaemonClient;
  instance: Instance;
  copy: AppMessages;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [submittedOperationId, setSubmittedOperationId] = useState<string | null>(null);
  const [submittedKind, setSubmittedKind] = useState<ChannelKind | null>(null);
  const [botId, setBotId] = useState("");
  const [secret, setSecret] = useState("");
  const [requestFailed, setRequestFailed] = useState(false);
  const [disconnectTarget, setDisconnectTarget] = useState<ChannelKind | null>(null);

  const channelsQuery = useQuery({
    queryKey: ["instance-channels", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceChannels(instance.instanceId, signal),
    retry: false,
    refetchInterval: 5000,
  });
  const lifecycleQuery = useQuery({
    queryKey: ["instance-lifecycle", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.getInstanceLifecycle(instance.instanceId, signal),
    retry: false,
    refetchInterval: 5000,
  });
  const recovered = channelsQuery.data?.channels.find((item) => item.activeOperationId)?.activeOperationId ?? null;
  const followedOperationId = submittedOperationId ?? recovered;
  const operationQuery = useQuery({
    queryKey: ["channel-operation", followedOperationId, client.scope],
    queryFn: ({ signal }) => client.getOperation(followedOperationId!, signal),
    enabled: followedOperationId !== null,
    retry: false,
    refetchInterval: (query) => isActive(query.state.data) ? 750 : false,
  });
  const operation = operationQuery.data;
	const operationId = operation?.id;
	const operationActive = isActive(operation);
	const refetchChannels = channelsQuery.refetch;
  const recoveredKind = channelsQuery.data?.channels.find((item) => item.activeOperationId === followedOperationId)?.type ?? null;
  const activeKind = submittedKind ?? recoveredKind;
  const qrQuery = useQuery({
    queryKey: ["channel-qr", followedOperationId, client.scope],
    queryFn: ({ signal }) => client.getChannelQr(followedOperationId!, signal),
    enabled: activeKind === "weixin" && operationActive && operation?.stage === "qr_ready",
    retry: false,
    refetchInterval: 2000,
  });

  useEffect(() => {
	if (!operationId || operationActive) return;
	void refetchChannels();
    void queryClient.invalidateQueries({ queryKey: ["hermes-instances"] });
	}, [operationActive, operationId, queryClient, refetchChannels]);

  const startConnect = async (kind: ChannelKind) => {
    setRequestFailed(false);
    try {
      const next = kind === "weixin"
        ? await client.connectWeixin(instance.instanceId, crypto.randomUUID())
        : await client.connectWeCom(instance.instanceId, botId.trim(), secret, crypto.randomUUID());
      setSecret("");
      setSubmittedKind(kind);
      setSubmittedOperationId(next.id);
    } catch {
      setSecret("");
      setRequestFailed(true);
      void channelsQuery.refetch();
    }
  };

  const disconnect = async (kind: ChannelKind) => {
    setDisconnectTarget(null);
    setRequestFailed(false);
    try {
      const next = await client.disconnectChannel(instance.instanceId, kind, crypto.randomUUID());
      setSubmittedKind(kind);
      setSubmittedOperationId(next.id);
    } catch {
      setRequestFailed(true);
      void channelsQuery.refetch();
    }
  };

  const cancel = async () => {
    if (!followedOperationId) return;
    try {
      await client.cancelOperation(followedOperationId);
      setSecret("");
      await operationQuery.refetch();
      await channelsQuery.refetch();
    } catch {
      setRequestFailed(true);
    }
  };

  const gatewayLabel = lifecycleQuery.data?.state === "RUNNING"
    ? copy.channels.gatewayRunning
    : lifecycleQuery.data?.state === "STOPPED"
      ? copy.channels.gatewayStopped
      : copy.channels.gatewayUnknown;

  return (
    <section className="channel-panel">
      <header className="channel-panel-header">
        <div>
          <h2 className="instance-modal-title">{copy.channels.title}</h2>
          <p className="page-copy">{copy.channels.description}</p>
        </div>
        <div className="channel-header-actions">
          <span className="channel-gateway-state">{copy.channels.gatewayState}: <b>{gatewayLabel}</b></span>
          <button type="button" className="modal-close" onClick={onClose} aria-label={copy.channels.close}><IconClose /></button>
        </div>
      </header>

      <p className="channel-independence-note">{copy.channels.independenceNote}</p>
      {channelsQuery.isLoading ? <p className="notice notice-info">{copy.channels.loading}</p> : null}
      {channelsQuery.isError || requestFailed ? <p className="notice notice-error" role="alert">{copy.channels.requestFailed}</p> : null}

      <div className="channel-card-grid" aria-live="polite">
        {(["weixin", "wecom"] as const).map((kind) => {
          const state = channelsQuery.data?.channels.find((item) => item.type === kind) ?? emptyChannel(kind);
          return (
            <ChannelCard
              key={kind}
              kind={kind}
              channel={state}
              operation={activeKind === kind ? operation : undefined}
              botId={botId}
              secret={secret}
              copy={copy}
              onBotIdChange={setBotId}
              onSecretChange={setSecret}
              onConnect={() => { void startConnect(kind); }}
              onDisconnect={() => setDisconnectTarget(kind)}
              onCancel={() => { void cancel(); }}
              onRefresh={() => { void channelsQuery.refetch(); }}
            />
          );
        })}
      </div>

		{activeKind === "weixin" && operationActive && operation ? (
			<WeixinQrModal qr={qrQuery.data} operation={operation} copy={copy} onCancel={() => { void cancel(); }} />
      ) : null}

      {disconnectTarget ? (
        <div className="channel-confirm" role="dialog" aria-modal="true" aria-labelledby="channel-disconnect-title">
          <div className="instance-modal channel-confirm-card">
            <h3 id="channel-disconnect-title" className="instance-modal-title">{copy.channels.disconnect}</h3>
            <p>{copy.channels.disconnectConfirm}</p>
            <div className="instance-modal-actions">
              <Button onClick={() => setDisconnectTarget(null)}>{copy.channels.cancel}</Button>
              <Button variant="danger" onClick={() => { void disconnect(disconnectTarget); }}>{copy.channels.disconnect}</Button>
            </div>
          </div>
        </div>
      ) : null}

      <footer className="channel-panel-footer">
        <p>{copy.channels.revocationNote}</p>
        <Button onClick={() => { void channelsQuery.refetch(); }}><IconRefresh />{copy.channels.refresh}</Button>
      </footer>
    </section>
  );
}

function ChannelCard({ kind, channel, operation, botId, secret, copy, onBotIdChange, onSecretChange, onConnect, onDisconnect, onCancel, onRefresh }: {
  kind: ChannelKind;
  channel: Channel;
  operation?: Operation;
  botId: string;
  secret: string;
  copy: AppMessages;
  onBotIdChange: (value: string) => void;
  onSecretChange: (value: string) => void;
  onConnect: () => void;
  onDisconnect: () => void;
  onCancel: () => void;
  onRefresh: () => void;
}) {
  const busy = isActive(operation);
  const failed = operation?.status === "FAILED" || operation?.status === "CANCELLED";
  const connected = channel.state === "CONNECTED" && !failed;
  const presentedState = busy ? "CONNECTING" : failed ? "FAILED" : channel.state;
  const tone = presentedState === "CONNECTED" ? "ok" : presentedState === "FAILED" ? "error" : presentedState === "NOT_CONFIGURED" ? "neutral" : "warn";
  const validWeCom = botId.trim() !== "" && secret !== "";
  return (
    <article className="channel-card">
      <div className="channel-card-heading">
        <div className={`channel-mark is-${kind}`}>{kind === "weixin" ? "微" : "企"}</div>
        <div>
          <h3>{kind === "weixin" ? copy.channels.weixin : copy.channels.wecom}</h3>
          <p>{kind === "weixin" ? copy.channels.weixinDescription : copy.channels.wecomDescription}</p>
        </div>
		<Badge tone={tone}>{busy && operation?.type === "channel.disconnect" ? copy.channels.disconnecting : copy.channels.state[presentedState]}</Badge>
      </div>
      <div className="channel-safe-identity">
        <span>{copy.channels.safeIdentity}</span>
        <strong>{channel.accountLabel || channel.externalId || copy.channels.noIdentity}</strong>
      </div>
      {kind === "wecom" && !connected ? (
        <div className="channel-fields">
          <label>{copy.channels.botId}<input value={botId} onChange={(event) => onBotIdChange(event.target.value)} placeholder={copy.channels.botIdPlaceholder} disabled={busy} autoComplete="off" /></label>
          <label>{copy.channels.secret}<input type="password" value={secret} onChange={(event) => onSecretChange(event.target.value)} placeholder={copy.channels.secretPlaceholder} disabled={busy} autoComplete="new-password" /></label>
        </div>
      ) : null}
      <div className="channel-card-actions">
        {busy ? <Button onClick={onCancel}>{copy.channels.cancel}</Button> : null}
        {connected ? <Button variant="danger" onClick={onDisconnect}>{copy.channels.disconnect}</Button> : null}
        {!busy && !connected ? <Button variant="primary" onClick={onConnect} disabled={kind === "wecom" && !validWeCom}>{failed ? copy.channels.retry : copy.channels.connect}</Button> : null}
        {channel.state === "UNKNOWN" && !busy ? <Button onClick={onRefresh}>{copy.channels.refresh}</Button> : null}
      </div>
    </article>
  );
}

function WeixinQrModal({ qr, operation, copy, onCancel }: {
  qr: { payload: string; expiresAt: string } | undefined;
  operation: Operation;
  copy: AppMessages;
  onCancel: () => void;
}) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, []);
  const seconds = useMemo(() => qr ? Math.max(0, Math.ceil((new Date(qr.expiresAt).getTime() - now) / 1000)) : 0, [now, qr]);
  return (
    <div className="channel-confirm" role="dialog" aria-modal="true" aria-labelledby="weixin-qr-title">
      <div className="instance-modal channel-qr-card">
        <h3 id="weixin-qr-title" className="instance-modal-title">{copy.channels.qrTitle}</h3>
        <p>{copy.channels.qrDescription}</p>
        <div className="channel-qr-code">
          {qr && seconds > 0 ? <QRCodeSVG value={qr.payload} size={220} level="M" marginSize={2} /> : <span>{qr ? copy.channels.qrExpired : copy.channels.qrWaiting}</span>}
        </div>
        <p className="channel-qr-countdown" role="status">{qr && seconds > 0 ? copy.channels.expiresIn.replace("{seconds}", String(seconds)) : operation.stage}</p>
        <div className="instance-modal-actions"><Button onClick={onCancel}>{copy.channels.cancel}</Button></div>
      </div>
    </div>
  );
}

function emptyChannel(kind: ChannelKind): Channel {
  return { type: kind, state: "UNKNOWN", accountLabel: "", externalId: "", lastCheckedAt: null, activeOperationId: null };
}

function isActive(operation: Operation | undefined): boolean {
  return operation?.status === "PENDING" || operation?.status === "RUNNING";
}
