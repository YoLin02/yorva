import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QRCodeSVG } from "qrcode.react";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { YorvaApiError, type DaemonClient } from "../../api/client";
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
  const [dismissedQrOperationId, setDismissedQrOperationId] = useState<string | null>(null);
  const [pairingCode, setPairingCode] = useState("");
  const [pairingApprovalState, setPairingApprovalState] = useState<"idle" | "submitting" | "approved">("idle");
  const [pairingErrorCode, setPairingErrorCode] = useState<string | null>(null);
  const [selectedKind, setSelectedKind] = useState<ChannelKind>("weixin");

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
  const recoveredKind = followedOperationId
    ? channelsQuery.data?.channels.find((item) => item.activeOperationId === followedOperationId)?.type ?? null
    : null;
  const activeKind = submittedKind ?? recoveredKind;
  const qrQuery = useQuery({
    queryKey: ["channel-qr", followedOperationId, client.scope],
    queryFn: ({ signal }) => client.getChannelQr(followedOperationId!, signal),
    enabled: activeKind === "weixin" && operationActive && operation?.stage === "channel.qr-ready",
    retry: false,
    refetchInterval: 2000,
  });
  const weixinConnected = channelsQuery.data?.channels.some((item) => item.type === "weixin" && item.state === "CONNECTED") ?? false;
  const pairingQuery = useQuery({
    queryKey: ["channel-pairings", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.getWeixinPairingStatus(instance.instanceId, signal),
    enabled: weixinConnected,
    retry: false,
    refetchInterval: weixinConnected ? 5000 : false,
  });

  useEffect(() => {
	if (!operationId || operationActive) return;
	void refetchChannels();
    void queryClient.invalidateQueries({ queryKey: ["hermes-instances"] });
	}, [operationActive, operationId, queryClient, refetchChannels]);

  const startConnect = async (kind: ChannelKind) => {
    setRequestFailed(false);
    setDismissedQrOperationId(null);
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

  const approvePairing = async () => {
    const code = pairingCode.trim().toUpperCase();
    if (!/^[A-HJ-NP-Z2-9]{8}$/.test(code)) {
      setPairingErrorCode("CHANNEL_PAIRING_CODE_INVALID");
      return;
    }
    setPairingApprovalState("submitting");
    setPairingErrorCode(null);
    try {
      await client.approveWeixinPairing(instance.instanceId, code);
      setPairingApprovalState("approved");
      await pairingQuery.refetch();
    } catch (error) {
      setPairingApprovalState("idle");
      setPairingErrorCode(error instanceof YorvaApiError ? error.code : "CHANNEL_PAIRING_APPROVAL_FAILED");
    } finally {
      setPairingCode("");
    }
  };

  const gatewayLabel = lifecycleQuery.data?.state === "RUNNING"
    ? copy.channels.gatewayRunning
    : lifecycleQuery.data?.state === "STOPPED"
      ? copy.channels.gatewayStopped
      : copy.channels.gatewayUnknown;

  const visibleKind = activeKind ?? selectedKind;
  const selectedChannel = channelsQuery.data?.channels.find((item) => item.type === visibleKind) ?? emptyChannel(visibleKind);

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

      {channelsQuery.isLoading ? <p className="notice notice-info">{copy.channels.loading}</p> : null}
      {channelsQuery.isError || requestFailed ? <p className="notice notice-error" role="alert">{copy.channels.requestFailed}</p> : null}

      <div className="channel-switcher" role="tablist" aria-label={copy.channels.title}>
        {(["weixin", "wecom"] as const).map((kind) => (
          <button
            type="button"
            key={kind}
            role="tab"
            aria-selected={visibleKind === kind}
            className={visibleKind === kind ? `channel-switch is-active is-${kind}` : `channel-switch is-${kind}`}
            onClick={() => setSelectedKind(kind)}
          >
            <span aria-hidden="true">{kind === "weixin" ? "微" : "企"}</span>
            {kind === "weixin" ? copy.channels.weixin : copy.channels.wecom}
          </button>
        ))}
      </div>

      <div className="channel-card-list" role="tabpanel" aria-label={visibleKind === "weixin" ? copy.channels.weixin : copy.channels.wecom} aria-live="polite">
        <ChannelCard
          key={visibleKind}
          kind={visibleKind}
          channel={selectedChannel}
          operation={activeKind === visibleKind ? operation : undefined}
          botId={botId}
          secret={secret}
          copy={copy}
          onBotIdChange={setBotId}
          onSecretChange={setSecret}
          onConnect={() => { void startConnect(visibleKind); }}
          onDisconnect={() => setDisconnectTarget(visibleKind)}
          onCancel={() => { void cancel(); }}
          onRefresh={() => { void channelsQuery.refetch(); }}
          pairingPanel={visibleKind === "weixin" && selectedChannel.state === "CONNECTED" ? (
            <WeixinPairing
              code={pairingCode}
              pendingCount={pairingQuery.data?.pendingCount}
              checking={pairingQuery.isLoading || pairingQuery.isFetching}
              submitting={pairingApprovalState === "submitting"}
              approved={pairingApprovalState === "approved"}
              errorCode={pairingQuery.isError ? "CHANNEL_PAIRING_QUERY_FAILED" : pairingErrorCode}
              copy={copy}
              onCodeChange={(value) => {
                setPairingCode(value.toUpperCase().replace(/[^A-HJ-NP-Z2-9]/g, "").slice(0, 8));
                setPairingApprovalState("idle");
                setPairingErrorCode(null);
              }}
              onApprove={() => { void approvePairing(); }}
              onRefresh={() => { void pairingQuery.refetch(); }}
            />
          ) : null}
        />
      </div>

		{activeKind === "weixin" && operation && dismissedQrOperationId !== operation.id && (operationActive || isFailed(operation)) ? (
			<WeixinQrModal
            qr={qrQuery.data}
            operation={operation}
            copy={copy}
            onCancel={() => {
              setDismissedQrOperationId(operation.id);
              void cancel();
            }}
            onDismiss={() => setDismissedQrOperationId(operation.id)}
          />
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
        <p><span>{copy.channels.independenceNote}</span><span>{copy.channels.revocationNote}</span></p>
        <Button onClick={() => { void channelsQuery.refetch(); }}><IconRefresh />{copy.channels.refresh}</Button>
      </footer>
    </section>
  );
}

function ChannelCard({ kind, channel, operation, botId, secret, copy, pairingPanel, onBotIdChange, onSecretChange, onConnect, onDisconnect, onCancel, onRefresh }: {
  kind: ChannelKind;
  channel: Channel;
  operation?: Operation;
  botId: string;
  secret: string;
  copy: AppMessages;
  pairingPanel?: ReactNode;
  onBotIdChange: (value: string) => void;
  onSecretChange: (value: string) => void;
  onConnect: () => void;
  onDisconnect: () => void;
  onCancel: () => void;
  onRefresh: () => void;
}) {
  const busy = isActive(operation);
  const failed = operation?.status === "FAILED";
  const cancelled = operation?.status === "CANCELLED";
  const connected = channel.state === "CONNECTED" && !failed;
  const presentedState = busy ? "CONNECTING" : failed ? "FAILED" : cancelled ? "CANCELLED" : channel.state;
  const tone = presentedState === "CONNECTED" ? "ok" : presentedState === "FAILED" ? "error" : presentedState === "NOT_CONFIGURED" || presentedState === "CANCELLED" ? "neutral" : "warn";
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
      {isFailed(operation) ? <ChannelOperationAlert operation={operation!} copy={copy} compact /> : null}
      {pairingPanel}
      {kind === "wecom" && !connected ? (
        <div className="channel-fields">
          <label>{copy.channels.botId}<input value={botId} onChange={(event) => onBotIdChange(event.target.value)} placeholder={copy.channels.botIdPlaceholder} disabled={busy} autoComplete="off" /></label>
          <label>{copy.channels.secret}<input type="password" value={secret} onChange={(event) => onSecretChange(event.target.value)} placeholder={copy.channels.secretPlaceholder} disabled={busy} autoComplete="new-password" /></label>
        </div>
      ) : null}
      <div className="channel-card-actions">
        {busy ? <Button onClick={onCancel}>{copy.channels.cancel}</Button> : null}
        {connected ? <Button variant="danger" onClick={onDisconnect}>{copy.channels.disconnect}</Button> : null}
        {!busy && !connected ? <Button variant="primary" onClick={onConnect} disabled={kind === "wecom" && !validWeCom}>{failed || cancelled ? copy.channels.retry : copy.channels.connect}</Button> : null}
        {channel.state === "UNKNOWN" && !busy ? <Button onClick={onRefresh}>{copy.channels.refresh}</Button> : null}
      </div>
    </article>
  );
}

function WeixinPairing({ code, pendingCount, checking, submitting, approved, errorCode, copy, onCodeChange, onApprove, onRefresh }: {
  code: string;
  pendingCount: number | undefined;
  checking: boolean;
  submitting: boolean;
  approved: boolean;
  errorCode: string | null;
  copy: AppMessages;
  onCodeChange: (value: string) => void;
  onApprove: () => void;
  onRefresh: () => void;
}) {
  const message = errorCode ? pairingErrorMessage(errorCode, copy) : null;
  return (
    <section className="channel-pairing" aria-labelledby="weixin-pairing-title">
      <div className="channel-pairing-heading">
        <div>
          <h4 id="weixin-pairing-title">{copy.channels.pairingTitle}</h4>
          <p>{copy.channels.pairingDescription}</p>
        </div>
        <button type="button" className="channel-pairing-refresh" onClick={onRefresh} disabled={checking} aria-label={copy.channels.pairingRefresh}><IconRefresh /></button>
      </div>
      <p className="channel-pairing-count" role="status">
        {checking && pendingCount === undefined
          ? copy.channels.pairingChecking
          : pendingCount === 0
            ? copy.channels.noPendingPairings
            : copy.channels.pendingPairings.replace("{count}", String(pendingCount ?? 0))}
      </p>
      <label className="channel-pairing-field">
        <span>{copy.channels.pairingCode}</span>
        <div>
          <input
            value={code}
            onChange={(event) => onCodeChange(event.target.value)}
            placeholder={copy.channels.pairingCodePlaceholder}
            maxLength={8}
            autoComplete="off"
            autoCapitalize="characters"
            spellCheck={false}
            inputMode="text"
            disabled={submitting}
          />
          <Button variant="primary" onClick={onApprove} disabled={submitting || code.length !== 8}>
            {submitting ? copy.channels.approvingPairing : copy.channels.approvePairing}
          </Button>
        </div>
      </label>
      {approved ? <p className="channel-pairing-feedback is-success">{copy.channels.pairingApproved}</p> : null}
      {message ? <p className="channel-pairing-feedback is-error" role="alert">{message}</p> : null}
    </section>
  );
}

function pairingErrorMessage(code: string, copy: AppMessages): string {
  switch (code) {
    case "CHANNEL_PAIRING_CODE_INVALID": return copy.channels.pairingInvalid;
    case "CHANNEL_PAIRING_LOCKED": return copy.channels.pairingLocked;
    case "CHANNEL_PAIRING_QUERY_FAILED": return copy.channels.pairingCheckFailed;
    default: return copy.channels.pairingApprovalFailed;
  }
}

function WeixinQrModal({ qr, operation, copy, onCancel, onDismiss }: {
  qr: { payload: string; expiresAt: string } | undefined;
  operation: Operation;
  copy: AppMessages;
  onCancel: () => void;
  onDismiss: () => void;
}) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, []);
  const seconds = useMemo(() => qr ? Math.max(0, Math.ceil((new Date(qr.expiresAt).getTime() - now) / 1000)) : 0, [now, qr]);
  const terminal = isFailed(operation);
  return (
    <div className="channel-confirm" role="dialog" aria-modal="true" aria-labelledby="weixin-qr-title">
      <div className="instance-modal channel-qr-card">
        <h3 id="weixin-qr-title" className="instance-modal-title">{copy.channels.qrTitle}</h3>
        <p>{copy.channels.qrDescription}</p>
        <div className="channel-qr-code">
          {!terminal && qr && seconds > 0 ? <QRCodeSVG value={qr.payload} size={220} level="M" marginSize={2} /> : <span>{terminal ? (operation.status === "CANCELLED" ? copy.channels.cancelledTitle : copy.channels.failureTitle) : qr ? copy.channels.qrExpired : copy.channels.qrWaiting}</span>}
        </div>
        {!terminal ? <p className="channel-qr-countdown" role="status">{qr && seconds > 0 ? copy.channels.expiresIn.replace("{seconds}", String(seconds)) : copy.channels.qrWaiting}</p> : null}
        {terminal ? <ChannelOperationAlert operation={operation} copy={copy} /> : null}
        <div className="instance-modal-actions"><Button onClick={terminal ? onDismiss : onCancel}>{terminal ? copy.channels.dismiss : copy.channels.cancel}</Button></div>
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

function isFailed(operation: Operation | undefined): operation is Operation {
  return operation?.status === "FAILED" || operation?.status === "CANCELLED";
}

function ChannelOperationAlert({ operation, copy, compact = false }: { operation: Operation; copy: AppMessages; compact?: boolean }) {
  const code = operation.errorCode ?? "UNKNOWN";
  const message = copy.channels.failureMessages[code] ?? copy.channels.unknownFailure;
  return (
    <div className={`channel-operation-alert${compact ? " is-compact" : ""}`} role="alert">
      <strong>{operation.status === "CANCELLED" ? copy.channels.cancelledTitle : copy.channels.failureTitle}</strong>
      <span><b>{copy.channels.failureReason}:</b> {message}</span>
      <span><b>{copy.channels.errorCode}:</b> <code>{code}</code></span>
    </div>
  );
}
