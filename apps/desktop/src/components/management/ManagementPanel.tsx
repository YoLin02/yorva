import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";
import type { DaemonClient } from "../../api/client";
import { selectBackupDestination } from "../../api/session";
import type { Instance, MCPPreset, MCPServer, ManagementBackup, ManagementHealth, ManagementLogSnapshot, ManagementUpgradePlan, Skill, SkillSource } from "../../api/types";
import { formatDateTime } from "../../formatDateTime";
import type { AppMessages, Locale } from "../../i18n";
import type { BadgeTone } from "../../types/ui";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { IconClose, IconRefresh } from "../ui/icons";

export function ManagementPanel({ client, instance, copy, locale, onClose }: {
  client: DaemonClient;
  instance: Instance;
  copy: AppMessages;
  locale: Locale;
  onClose: () => void;
}) {
  const [selectedSkillId, setSelectedSkillId] = useState<string | null>(null);
  const [selectedSkillSource, setSelectedSkillSource] = useState("");
  const [backupOperationId, setBackupOperationId] = useState<string | null>(null);
  const [mcpOperationId, setMCPOperationId] = useState<string | null>(null);
  const [upgradeOperationId, setUpgradeOperationId] = useState<string | null>(null);
  const [mcpCredentials, setMCPCredentials] = useState<Record<string, string>>({});
  const [mcpTools, setMCPTools] = useState<Record<string, string[]>>({});
  const [logCategory, setLogCategory] = useState<ManagementLogSnapshot["category"]>("ERRORS");
  const healthRead = instance.capabilities.healthRead;
  const logsRead = instance.capabilities.logsRead;
  const skillRead = instance.capabilities.skillRead;
  const skillMutate = instance.capabilities.skillMutate;
  const mcpRead = instance.capabilities.mcpRead;
  const mcpMutate = instance.capabilities.mcpMutate;
  const upgradePlanRead = instance.capabilities.upgradePlan;
  const upgrade = instance.capabilities.upgrade;
  const rollback = instance.capabilities.rollback;
  const backupRead = instance.capabilities.backupRead;
  const backupMutate = instance.capabilities.backupMutate;
  const restore = instance.capabilities.restore;
  const healthQuery = useQuery({
    queryKey: ["instance-management-health", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.getInstanceHealth(instance.instanceId, signal),
    enabled: healthRead,
    retry: false,
  });
  const logsQuery = useQuery({
    queryKey: ["instance-management-logs", instance.instanceId, logCategory, client.scope],
    queryFn: ({ signal }) => client.getInstanceLogSnapshot(instance.instanceId, logCategory, signal),
    enabled: logsRead,
    retry: false,
  });
  const skillsQuery = useQuery({
    queryKey: ["instance-skills", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceSkills(instance.instanceId, signal),
    enabled: skillRead,
    retry: false,
  });
  const skillQuery = useQuery({
    queryKey: ["instance-skill", instance.instanceId, selectedSkillId, client.scope],
    queryFn: ({ signal }) => client.inspectInstanceSkill(instance.instanceId, selectedSkillId!, signal),
    enabled: skillRead && selectedSkillId !== null,
    retry: false,
  });
  const skillSourcesQuery = useQuery({
    queryKey: ["instance-skill-sources", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceSkillSources(instance.instanceId, signal),
    enabled: skillMutate,
    retry: false,
  });
  const skillMutation = useMutation({
    mutationFn: async ({ action, skillId, sourceId }: { action: "install" | "update" | "enable" | "disable" | "remove"; skillId: string; sourceId?: string }) => {
      const key = crypto.randomUUID();
      if (action === "install") return client.installManagedSkill(instance.instanceId, skillId, sourceId!, key);
      if (action === "update") return client.updateManagedSkill(instance.instanceId, skillId, key);
      if (action === "enable") return client.enableManagedSkill(instance.instanceId, skillId, key);
      if (action === "disable") return client.disableManagedSkill(instance.instanceId, skillId, key);
      return client.removeManagedSkill(instance.instanceId, skillId, key);
    },
    onSuccess: () => {
      void skillsQuery.refetch();
      if (selectedSkillId) void skillQuery.refetch();
    },
  });
  const serversQuery = useQuery({
    queryKey: ["instance-mcp-servers", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceMCPServers(instance.instanceId, signal),
    enabled: mcpRead,
    retry: false,
  });
  const presetsQuery = useQuery({
    queryKey: ["instance-mcp-presets", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceMCPPresets(instance.instanceId, signal),
    enabled: mcpRead,
    retry: false,
  });
  const mcpMutation = useMutation({
    mutationFn: async (input: { action: "install" | "authenticate" | "test" | "configure" | "remove"; id: string }) => {
      const key = crypto.randomUUID();
      if (input.action === "install") return client.installInstanceMCPPreset(instance.instanceId, input.id, key);
      if (input.action === "authenticate") return client.authenticateInstanceMCPServer(instance.instanceId, input.id, mcpCredentials[input.id] ?? "", key);
      if (input.action === "test") return client.testInstanceMCPServer(instance.instanceId, input.id, key);
      if (input.action === "configure") return client.configureInstanceMCPServer(instance.instanceId, input.id, mcpTools[input.id] ?? [], key);
      return client.removeInstanceMCPServer(instance.instanceId, input.id, key);
    },
    onSuccess: (accepted, input) => {
      setMCPOperationId(accepted.id);
      if (input.action === "authenticate") setMCPCredentials((current) => ({ ...current, [input.id]: "" }));
    },
  });
  const mcpOperationQuery = useQuery({
    queryKey: ["management-operation", mcpOperationId, client.scope],
    queryFn: ({ signal }) => client.getOperation(mcpOperationId!, signal),
    enabled: mcpOperationId !== null,
    retry: false,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const refetchMCPServers = serversQuery.refetch;
  const refetchMCPPresets = presetsQuery.refetch;

  useEffect(() => {
    if (mcpOperationQuery.data?.status === "SUCCEEDED") {
      void refetchMCPServers();
      void refetchMCPPresets();
    }
  }, [mcpOperationQuery.data?.status, refetchMCPServers, refetchMCPPresets]);
  const upgradePlanQuery = useQuery({
    queryKey: ["runtime-upgrade-plan", "hermes", client.scope],
    queryFn: ({ signal }) => client.getRuntimeUpgradePlan("hermes", signal),
    enabled: upgradePlanRead,
    retry: false,
  });
  const upgradeMutation = useMutation({
    mutationFn: (action: "upgrade" | "rollback") => action === "upgrade"
      ? client.upgradeManagedRuntime("hermes", crypto.randomUUID())
      : client.rollbackManagedRuntime("hermes", crypto.randomUUID()),
    onSuccess: (accepted) => setUpgradeOperationId(accepted.id),
  });
  const upgradeOperationQuery = useQuery({
    queryKey: ["management-operation", upgradeOperationId, client.scope],
    queryFn: ({ signal }) => client.getOperation(upgradeOperationId!, signal),
    enabled: upgradeOperationId !== null,
    retry: false,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const refetchUpgradePlan = upgradePlanQuery.refetch;

  useEffect(() => {
    if (upgradeOperationQuery.data?.status === "SUCCEEDED") void refetchUpgradePlan();
  }, [upgradeOperationQuery.data?.status, refetchUpgradePlan]);
  const backupsQuery = useQuery({
    queryKey: ["runtime-backups", "hermes", client.scope],
    queryFn: ({ signal }) => client.listRuntimeBackups("hermes", signal),
    enabled: backupRead,
    retry: false,
  });
  const backupMutation = useMutation({
    mutationFn: async () => {
      const destinationRef = await selectBackupDestination();
      if (!destinationRef) return null;
      return client.createRuntimeBackup("hermes", destinationRef, crypto.randomUUID());
    },
    onSuccess: (accepted) => {
      if (accepted) setBackupOperationId(accepted.id);
    },
  });
  const backupDeleteMutation = useMutation({
    mutationFn: (backupId: string) => client.deleteRuntimeBackup(backupId, crypto.randomUUID()),
    onSuccess: (accepted) => setBackupOperationId(accepted.id),
  });
  const backupRestoreMutation = useMutation({
    mutationFn: (backupId: string) => client.restoreRuntimeBackup(backupId, crypto.randomUUID()),
    onSuccess: (accepted) => setBackupOperationId(accepted.id),
  });
  const backupOperationQuery = useQuery({
    queryKey: ["management-operation", backupOperationId, client.scope],
    queryFn: ({ signal }) => client.getOperation(backupOperationId!, signal),
    enabled: backupOperationId !== null,
    retry: false,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const refetchBackups = backupsQuery.refetch;

  useEffect(() => {
    const status = backupOperationQuery.data?.status;
    if (status === "SUCCEEDED") void refetchBackups();
  }, [backupOperationQuery.data?.status, refetchBackups]);

  useEffect(() => {
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [onClose]);

  const refresh = () => {
    if (healthRead) void healthQuery.refetch();
    if (logsRead) void logsQuery.refetch();
    if (skillRead) {
      void skillsQuery.refetch();
      if (selectedSkillId) void skillQuery.refetch();
    }
    if (skillMutate) void skillSourcesQuery.refetch();
    if (mcpRead) {
      void serversQuery.refetch();
      void presetsQuery.refetch();
    }
    if (upgradePlanRead) void upgradePlanQuery.refetch();
    if (backupRead) void backupsQuery.refetch();
  };
  const backupRunning = backupMutation.isPending || backupDeleteMutation.isPending || backupRestoreMutation.isPending || backupOperationQuery.data?.status === "PENDING" || backupOperationQuery.data?.status === "RUNNING";
  const backupFailed = backupMutation.isError || backupDeleteMutation.isError || backupRestoreMutation.isError || backupOperationQuery.data?.status === "FAILED" || backupOperationQuery.data?.status === "CANCELLED";
  const mcpRunning = mcpMutation.isPending || mcpOperationQuery.data?.status === "PENDING" || mcpOperationQuery.data?.status === "RUNNING";
  const mcpFailed = mcpMutation.isError || mcpOperationQuery.data?.status === "FAILED" || mcpOperationQuery.data?.status === "CANCELLED";
  const upgradeRunning = upgradeMutation.isPending || upgradeOperationQuery.data?.status === "PENDING" || upgradeOperationQuery.data?.status === "RUNNING";
  const upgradeFailed = upgradeMutation.isError || upgradeOperationQuery.data?.status === "FAILED" || upgradeOperationQuery.data?.status === "CANCELLED";
  const activeManagementOperationId = backupRunning ? backupOperationId : mcpRunning ? mcpOperationId : null;
  const cancelMutation = useMutation({
    mutationFn: () => client.cancelOperation(activeManagementOperationId!),
    onSuccess: () => {
      void backupOperationQuery.refetch();
      void mcpOperationQuery.refetch();
      void upgradeOperationQuery.refetch();
    },
  });
  const busy = healthQuery.isFetching || logsQuery.isFetching || skillsQuery.isFetching || skillQuery.isFetching || skillSourcesQuery.isFetching || skillMutation.isPending || serversQuery.isFetching || presetsQuery.isFetching || upgradePlanQuery.isFetching || backupsQuery.isFetching || backupRunning || mcpRunning || upgradeRunning;

  return (
    <section className="management-panel" aria-labelledby="management-panel-title">
      <header className="management-panel-header">
        <div>
          <h2 id="management-panel-title">{copy.management.title}: {instance.name}</h2>
          <p className="page-copy">{copy.management.description}</p>
        </div>
        <div className="management-header-actions">
          <Button onClick={refresh} disabled={busy || (!healthRead && !logsRead && !skillRead && !mcpRead && !upgradePlanRead && !backupRead)} className="button-compact button-neutral">
            <IconRefresh className={busy ? "spin" : undefined} />
            {copy.management.refresh}
          </Button>
          {activeManagementOperationId ? <Button onClick={() => cancelMutation.mutate()} disabled={cancelMutation.isPending} className="button-compact button-neutral">{copy.management.cancelOperation}</Button> : null}
          <button type="button" className="modal-close" onClick={onClose} aria-label={copy.management.close}>
            <IconClose />
          </button>
        </div>
      </header>

      <ManagementSection
        title={copy.management.diagnosticsTitle}
        description={copy.management.diagnosticsDescription}
        supported={healthRead || logsRead}
        loading={false}
        error={false}
        copy={copy}
        onRetry={() => undefined}
      >
        <div className="management-diagnostics-grid">
          <DiagnosticBlock title={copy.management.healthTitle} supported={healthRead} copy={copy}>
            {healthQuery.isLoading ? <p className="management-empty" role="status">{copy.management.loading}</p> : null}
            {healthQuery.isError ? <ManagementQueryError copy={copy} onRetry={() => void healthQuery.refetch()} /> : null}
            {healthQuery.data ? <HealthView health={healthQuery.data} copy={copy} locale={locale} /> : null}
          </DiagnosticBlock>
          <DiagnosticBlock title={copy.management.logsTitle} supported={logsRead} copy={copy}>
            <label className="management-log-category">
              <span>{copy.management.logCategory}</span>
              <select value={logCategory} onChange={(event) => setLogCategory(event.target.value as ManagementLogSnapshot["category"])} disabled={!logsRead || logsQuery.isFetching}>
                {(Object.keys(copy.management.logCategories) as ManagementLogSnapshot["category"][]).map((category) => (
                  <option key={category} value={category}>{copy.management.logCategories[category]}</option>
                ))}
              </select>
            </label>
            {logsQuery.isLoading ? <p className="management-empty" role="status">{copy.management.loading}</p> : null}
            {logsQuery.isError ? <ManagementQueryError copy={copy} onRetry={() => void logsQuery.refetch()} /> : null}
            {logsQuery.data ? <LogView snapshot={logsQuery.data} copy={copy} locale={locale} /> : null}
          </DiagnosticBlock>
        </div>
      </ManagementSection>

      <ManagementSection
        title={copy.management.upgradeTitle}
        description={copy.management.upgradeDescription}
        supported={upgradePlanRead}
        loading={upgradePlanQuery.isLoading}
        error={upgradePlanQuery.isError}
        copy={copy}
        onRetry={() => void upgradePlanQuery.refetch()}
      >
        {upgradePlanQuery.data ? <UpgradePlanView plan={upgradePlanQuery.data} copy={copy} locale={locale} /> : null}
        {upgrade || rollback ? <div className="management-item-footer"><span>{upgradeRunning ? copy.management.upgradeRunning : null}</span><div className="management-header-actions">
          {upgrade ? <Button disabled={upgradeRunning} onClick={() => upgradeMutation.mutate("upgrade")}>{copy.management.startUpgrade}</Button> : null}
          {rollback ? <Button disabled={upgradeRunning} onClick={() => upgradeMutation.mutate("rollback")}>{copy.management.startRollback}</Button> : null}
        </div></div> : null}
        {upgradeFailed ? <p className="notice notice-warn" role="alert">{copy.management.upgradeFailed}</p> : null}
      </ManagementSection>

      <ManagementSection
        title={copy.management.backupsTitle}
        description={copy.management.backupsDescription}
        supported={backupRead}
        loading={backupsQuery.isLoading}
        error={backupsQuery.isError}
        copy={copy}
        onRetry={() => void backupsQuery.refetch()}
      >
        {backupMutate ? (
          <div className="management-item-footer">
            <span>{backupRunning ? copy.management.backupCreating : null}</span>
            <Button disabled={backupRunning} onClick={() => backupMutation.mutate()}>{copy.management.createBackup}</Button>
          </div>
        ) : null}
        {backupFailed ? <p className="notice notice-warn" role="alert">{copy.management.backupCreateFailed}</p> : null}
        {backupsQuery.data?.items.length === 0 ? <p className="management-empty">{copy.management.noBackups}</p> : null}
        <div className="management-list">
          {backupsQuery.data?.items.map((backup) => <BackupCard
            key={backup.backupId} backup={backup} copy={copy} locale={locale}
            mutable={backupMutate} restorable={restore} busy={backupRunning}
            onDelete={() => backupDeleteMutation.mutate(backup.backupId)}
            onRestore={() => { if (window.confirm(copy.management.restoreConfirm)) backupRestoreMutation.mutate(backup.backupId); }}
          />)}
        </div>
      </ManagementSection>

      <ManagementSection
        title={copy.management.skillsTitle}
        description={copy.management.skillsDescription}
          supported={skillRead || skillMutate}
          loading={skillsQuery.isLoading || skillSourcesQuery.isLoading}
          error={skillsQuery.isError || skillSourcesQuery.isError}
        copy={copy}
        onRetry={() => void skillsQuery.refetch()}
      >
        {skillsQuery.data?.items.length === 0 ? <p className="management-empty">{copy.management.noSkills}</p> : null}
        <div className="management-list">
          {skillsQuery.data?.items.map((skill) => (
            <SkillCard
              key={skill.id}
              skill={skill}
              inspected={selectedSkillId === skill.id ? skillQuery.data : undefined}
              inspecting={selectedSkillId === skill.id && skillQuery.isLoading}
              inspectFailed={selectedSkillId === skill.id && skillQuery.isError}
              copy={copy}
              onToggle={() => setSelectedSkillId((current) => current === skill.id ? null : skill.id)}
               onRetry={() => void skillQuery.refetch()}
               mutationPending={skillMutation.isPending && skillMutation.variables?.skillId === skill.id}
               onAction={(action) => skillMutation.mutate({ action, skillId: skill.id })}
             />
          ))}
         </div>
        {skillMutate ? (
          <SkillCatalog
            sources={skillSourcesQuery.data?.items ?? []}
            selected={selectedSkillSource}
            pending={skillMutation.isPending}
            copy={copy}
            onSelect={setSelectedSkillSource}
            onInstall={(source) => skillMutation.mutate({ action: "install", skillId: source.skillId, sourceId: source.sourceId })}
          />
        ) : null}
        {skillMutation.isPending ? <p className="management-empty" role="status">{copy.management.skillMutationRunning}</p> : null}
        {skillMutation.isError ? <p className="notice notice-warn" role="alert">{copy.management.skillMutationFailed}</p> : null}
      </ManagementSection>

      <ManagementSection
        title={copy.management.mcpTitle}
        description={copy.management.mcpDescription}
        supported={mcpRead || mcpMutate}
        loading={serversQuery.isLoading || presetsQuery.isLoading}
        error={serversQuery.isError || presetsQuery.isError}
        copy={copy}
        onRetry={() => { void serversQuery.refetch(); void presetsQuery.refetch(); }}
      >
        {serversQuery.data?.items.length === 0 ? <p className="management-empty">{copy.management.noServers}</p> : null}
        <div className="management-list">
          {serversQuery.data?.items.map((server) => <MCPServerCard
            key={server.id} server={server} preset={presetsQuery.data?.items.find((preset) => preset.id === server.presetId)}
            copy={copy} locale={locale} mutable={mcpMutate} busy={mcpRunning}
            credential={mcpCredentials[server.id] ?? ""} selectedTools={mcpTools[server.id] ?? []}
            onCredential={(value) => setMCPCredentials((current) => ({ ...current, [server.id]: value }))}
            onToggleTool={(toolId) => setMCPTools((current) => {
              const selected = current[server.id] ?? [];
              return { ...current, [server.id]: selected.includes(toolId) ? selected.filter((id) => id !== toolId) : [...selected, toolId] };
            })}
            onAction={(action) => mcpMutation.mutate({ action, id: server.id })}
          />)}
        </div>
        <div className="management-catalog">
          <h4>{copy.management.catalogTitle}</h4>
          {presetsQuery.data?.items.length === 0 ? <p className="management-empty">{copy.management.noPresets}</p> : null}
          <div className="management-preset-list">
            {presetsQuery.data?.items.map((preset) => <span key={preset.id} className="management-preset"><strong>{preset.displayName}</strong><code>{preset.id}</code>{mcpMutate ? <Button disabled={mcpRunning} onClick={() => mcpMutation.mutate({ action: "install", id: preset.id })}>{copy.management.installMCP}</Button> : null}</span>)}
          </div>
        </div>
        {mcpRunning ? <p className="management-empty" role="status">{copy.management.mcpMutationRunning}</p> : null}
        {mcpFailed ? <p className="notice notice-warn" role="alert">{copy.management.mcpMutationFailed}</p> : null}
      </ManagementSection>
    </section>
  );
}

function BackupCard({ backup, copy, locale, mutable, restorable, busy, onDelete, onRestore }: { backup: ManagementBackup; copy: AppMessages; locale: Locale; mutable: boolean; restorable: boolean; busy: boolean; onDelete: () => void; onRestore: () => void }) {
  return (
    <article className="management-item">
      <div className="management-item-heading">
        <code>{backup.backupId}</code>
        <Badge tone={backupStateTone(backup.state)}>{copy.management.backupState[backup.state]}</Badge>
      </div>
      <dl className="management-detail-grid">
        <div><dt>{copy.management.backupLastObserved}</dt><dd>{copy.management.backupState[backup.state]}</dd></div>
        <div><dt>{copy.management.backupCreated}</dt><dd>{formatDateTime(backup.createdAt, locale)}</dd></div>
        <div><dt>{copy.management.backupVerified}</dt><dd>{formatDateTime(backup.verifiedAt, locale)}</dd></div>
        <div><dt>{copy.management.backupFormat}</dt><dd>{backup.formatVersion} / {backup.runtimeVersion}</dd></div>
        <div><dt>{copy.management.backupSize}</dt><dd>{new Intl.NumberFormat(locale).format(backup.sizeBytes)} B</dd></div>
        <div><dt>{copy.management.backupChecksum}</dt><dd><code>{backup.checksumSha256}</code></dd></div>
        <div><dt>{copy.management.backupKey}</dt><dd>{copy.management.backupKeyMode[backup.keyMode]}</dd></div>
      </dl>
      {mutable || restorable ? <div className="management-item-footer"><span />
        <div className="management-header-actions">
          {restorable ? <Button disabled={busy || backup.state !== "AVAILABLE"} onClick={onRestore}>{copy.management.restoreBackup}</Button> : null}
          {mutable ? <Button disabled={busy} onClick={onDelete}>{copy.management.deleteBackup}</Button> : null}
        </div>
      </div> : null}
    </article>
  );
}

function backupStateTone(state: ManagementBackup["state"]): BadgeTone {
  if (state === "AVAILABLE") return "ok";
  if (state === "UNKNOWN") return "neutral";
  return "warn";
}

function UpgradePlanView({ plan, copy, locale }: { plan: ManagementUpgradePlan; copy: AppMessages; locale: Locale }) {
  return (
    <article className="management-item">
      <div className="management-item-heading">
        <Badge tone={upgradePlanTone(plan.state)}>{copy.management.upgradeState[plan.state]}</Badge>
        <time dateTime={plan.observedAt}>{formatDateTime(plan.observedAt, locale)}</time>
      </div>
      <dl className="management-detail-grid">
        <div><dt>{copy.management.currentVersion}</dt><dd>{plan.currentVersion}</dd></div>
        <div><dt>{copy.management.candidate}</dt><dd>{plan.candidate.label} ({plan.candidate.version})</dd></div>
        <div><dt>{copy.management.managedStatusLabel}</dt><dd>{copy.management.managedState[plan.managedStatus]}</dd></div>
        <div><dt>{copy.management.compatibilityLabel}</dt><dd>{copy.management.compatibilityState[plan.compatibility]}</dd></div>
        <div><dt>{copy.management.protectionPoint}</dt><dd>{protectionPointCopy(plan, copy)}</dd></div>
      </dl>
      {plan.blockedReasons.length > 0 ? (
        <ul className="management-finding-list">{plan.blockedReasons.map((reason) => <li key={reason}>{copy.management.upgradeReasons[reason]}</li>)}</ul>
      ) : null}
    </article>
  );
}

function protectionPointCopy(plan: ManagementUpgradePlan, copy: AppMessages): string {
  if (!plan.protectionPointRequired) return copy.management.protectionNotRequired;
  return plan.protectionPointReady ? copy.management.protectionRequired : copy.management.protectionNotReady;
}

function upgradePlanTone(state: ManagementUpgradePlan["state"]): BadgeTone {
  if (state === "UP_TO_DATE") return "ok";
  if (state === "BLOCKED") return "error";
  return "warn";
}

function DiagnosticBlock({ title, supported, copy, children }: {
  title: string;
  supported: boolean;
  copy: AppMessages;
  children: ReactNode;
}) {
  return (
    <section className="management-diagnostic-block">
      <div className="management-item-heading"><h4>{title}</h4><Badge tone={supported ? "ok" : "neutral"}>{supported ? copy.instances.capabilityAvailable : copy.instances.capabilityUnavailable}</Badge></div>
      {!supported ? <p className="management-empty">{copy.management.unavailable}</p> : children}
    </section>
  );
}

function ManagementQueryError({ copy, onRetry }: { copy: AppMessages; onRetry: () => void }) {
  return <div className="management-error" role="alert"><span>{copy.management.requestFailed}</span><Button onClick={onRetry}>{copy.management.retry}</Button></div>;
}

function HealthView({ health, copy, locale }: { health: ManagementHealth; copy: AppMessages; locale: Locale }) {
  return (
    <div className="management-diagnostic-content">
      <div className="management-item-heading">
        <Badge tone={healthTone(health.state)}>{copy.management.healthState[health.state]}</Badge>
        <time dateTime={health.observedAt}>{formatDateTime(health.observedAt, locale)}</time>
      </div>
      {health.partial ? <p className="notice notice-warn">{copy.management.partial}</p> : null}
      {health.findings.length === 0 ? <p className="management-empty">{copy.management.noFindings}</p> : (
        <ul className="management-finding-list">{health.findings.map((finding) => <li key={finding.code}><code>{finding.code}</code><span>{copy.management.healthState[finding.state]}</span></li>)}</ul>
      )}
    </div>
  );
}

function LogView({ snapshot, copy, locale }: { snapshot: ManagementLogSnapshot; copy: AppMessages; locale: Locale }) {
  return (
    <div className="management-diagnostic-content">
      {snapshot.truncated ? <p className="notice notice-warn">{copy.management.truncated}</p> : null}
      {snapshot.entries.length === 0 ? <p className="management-empty">{copy.management.noLogs}</p> : (
        <ol className="management-log-list">{snapshot.entries.map((entry, index) => (
          <li key={`${entry.timestamp}-${index}`}><time dateTime={entry.timestamp}>{formatDateTime(entry.timestamp, locale)}</time><pre>{entry.message}</pre></li>
        ))}</ol>
      )}
    </div>
  );
}

function ManagementSection({ title, description, supported, loading, error, copy, onRetry, children }: {
  title: string;
  description: string;
  supported: boolean;
  loading: boolean;
  error: boolean;
  copy: AppMessages;
  onRetry: () => void;
  children: ReactNode;
}) {
  return (
    <section className="management-section">
      <div className="management-section-heading">
        <div><h3>{title}</h3><p className="page-copy">{description}</p></div>
        <Badge tone={supported ? "ok" : "neutral"}>{supported ? copy.instances.capabilityAvailable : copy.instances.capabilityUnavailable}</Badge>
      </div>
      {!supported ? <p className="notice notice-info">{copy.management.unavailable}</p> : null}
      {supported && loading ? <p className="management-empty" role="status">{copy.management.loading}</p> : null}
      {supported && error ? (
        <div className="management-error" role="alert"><span>{copy.management.requestFailed}</span><Button onClick={onRetry}>{copy.management.retry}</Button></div>
      ) : null}
      {supported && !loading && !error ? children : null}
    </section>
  );
}

function SkillCard({ skill, inspected, inspecting, inspectFailed, mutationPending, copy, onToggle, onRetry, onAction }: {
  skill: Skill;
  inspected?: Skill;
  inspecting: boolean;
  inspectFailed: boolean;
  mutationPending: boolean;
  copy: AppMessages;
  onToggle: () => void;
  onRetry: () => void;
  onAction: (action: "update" | "enable" | "disable" | "remove") => void;
}) {
  const open = inspected !== undefined || inspecting || inspectFailed;
  return (
    <article className="management-item">
      <div className="management-item-heading">
        <code>{skill.id}</code>
        <div className="management-badges">
          <Badge tone={skillInstallationTone(skill)}>{copy.management.installationState[skill.installationState]}</Badge>
          <Badge tone={skill.enabledState === "ENABLED" ? "ok" : "neutral"}>{copy.management.enabledState[skill.enabledState]}</Badge>
          <Badge tone={skillScanTone(skill)}>{copy.management.scanState[skill.scanState]}</Badge>
          <Badge tone={skill.ownership === "YORVA_MANAGED" ? "info" : "neutral"}>{copy.management.ownershipState[skill.ownership]}</Badge>
        </div>
      </div>
      <div className="management-item-footer">
        <span>{skill.updateAvailable ? copy.management.updateAvailable : copy.management.noUpdate}</span>
        <div className="management-header-actions">
          {skill.ownership === "YORVA_MANAGED" ? (
            <>
              {skill.updateAvailable ? <Button disabled={mutationPending} onClick={() => onAction("update")}>{copy.management.updateSkill}</Button> : null}
              {skill.projectionState === "PROJECTED"
                ? <Button disabled={mutationPending} onClick={() => onAction("disable")}>{copy.management.disableSkill}</Button>
                : <Button disabled={mutationPending || skill.projectionState === "DRIFT_MODIFIED" || skill.projectionState === "CONFLICT"} onClick={() => onAction("enable")}>{copy.management.enableSkill}</Button>}
              <Button disabled={mutationPending || skill.projectionState === "DRIFT_MODIFIED" || skill.projectionState === "CONFLICT"} onClick={() => onAction("remove")}>{copy.management.removeSkill}</Button>
            </>
          ) : <span className="management-detail">{copy.management.externalReadOnly}</span>}
          <Button variant="ghost" onClick={onToggle}>{open ? copy.management.hideDetails : copy.management.inspect}</Button>
        </div>
      </div>
      {inspecting ? <p className="management-detail">{copy.management.loading}</p> : null}
      {inspectFailed ? <div className="management-error"><span>{copy.management.requestFailed}</span><Button onClick={onRetry}>{copy.management.retry}</Button></div> : null}
      {inspected ? (
        <dl className="management-detail-grid">
          <div><dt>{copy.management.source}</dt><dd>{inspected.sourceId ?? "—"}</dd></div>
          <div><dt>{copy.management.version}</dt><dd>{inspected.version ?? "—"}</dd></div>
          <div><dt>{copy.management.ownership}</dt><dd>{copy.management.ownershipState[inspected.ownership]}</dd></div>
          <div><dt>{copy.management.projection}</dt><dd>{copy.management.projectionState[inspected.projectionState]}</dd></div>
        </dl>
      ) : null}
    </article>
  );
}

function SkillCatalog({ sources, selected, pending, copy, onSelect, onInstall }: {
  sources: SkillSource[];
  selected: string;
  pending: boolean;
  copy: AppMessages;
  onSelect: (sourceId: string) => void;
  onInstall: (source: SkillSource) => void;
}) {
  const current = sources.find((source) => source.sourceId === selected) ?? sources[0];
  return (
    <div className="management-catalog">
      <h4>{copy.management.skillCatalogTitle}</h4>
      {sources.length === 0 ? <p className="management-empty">{copy.management.noSkillSources}</p> : (
        <div className="management-item-footer">
          <select aria-label={copy.management.skillCatalogTitle} value={current?.sourceId ?? ""} onChange={(event) => onSelect(event.target.value)} disabled={pending}>
            {sources.map((source) => <option key={source.sourceId} value={source.sourceId}>{source.displayName} ({source.version})</option>)}
          </select>
          <Button disabled={pending || !current} onClick={() => current && onInstall(current)}>{copy.management.installSkill}</Button>
        </div>
      )}
    </div>
  );
}

function MCPServerCard({ server, preset, copy, locale, mutable, busy, credential, selectedTools, onCredential, onToggleTool, onAction }: {
  server: MCPServer;
  preset?: MCPPreset;
  copy: AppMessages;
  locale: Locale;
  mutable: boolean;
  busy: boolean;
  credential: string;
  selectedTools: string[];
  onCredential: (value: string) => void;
  onToggleTool: (toolId: string) => void;
  onAction: (action: "authenticate" | "test" | "configure" | "remove") => void;
}) {
  return (
    <article className="management-item">
      <div className="management-item-heading">
        <code>{server.id}</code>
        <Badge tone={mcpTone(server)}>{copy.management.mcpState[server.state]}</Badge>
      </div>
      <dl className="management-detail-grid">
        <div><dt>{copy.management.preset}</dt><dd>{server.presetId}</dd></div>
        <div><dt>{copy.management.readyAt}</dt><dd>{server.readyAt ? formatDateTime(server.readyAt, locale) : copy.management.neverReady}</dd></div>
        <div><dt>{copy.management.observedAt}</dt><dd>{formatDateTime(server.observedAt, locale)}</dd></div>
      </dl>
      {mutable ? (
        <div className="management-catalog">
          {preset?.credentialRequired ? <div className="management-item-footer">
            <input type="password" value={credential} onChange={(event) => onCredential(event.target.value)} placeholder={copy.management.mcpCredential} autoComplete="off" disabled={busy} />
            <Button disabled={busy || credential.length === 0} onClick={() => onAction("authenticate")}>{copy.management.authenticateMCP}</Button>
          </div> : null}
          {preset && preset.allowedToolIds.length > 0 ? (
            <div className="management-preset-list">
              {preset.allowedToolIds.map((toolId) => <label key={toolId} className="management-preset"><input type="checkbox" checked={selectedTools.includes(toolId)} onChange={() => onToggleTool(toolId)} disabled={busy} /><code>{toolId}</code></label>)}
            </div>
          ) : null}
          <div className="management-item-footer"><span />
            <div className="management-header-actions">
              {preset && preset.allowedToolIds.length > 0 ? <Button disabled={busy} onClick={() => onAction("configure")}>{copy.management.saveMCPTools}</Button> : null}
              <Button disabled={busy} onClick={() => onAction("test")}>{copy.management.testMCP}</Button>
              <Button disabled={busy} onClick={() => onAction("remove")}>{copy.management.removeMCP}</Button>
            </div>
          </div>
        </div>
      ) : null}
    </article>
  );
}

function skillInstallationTone(skill: Skill): BadgeTone {
  if (skill.installationState === "INSTALLED") return "ok";
  if (skill.installationState === "UNKNOWN") return "warn";
  return "neutral";
}

function skillScanTone(skill: Skill): BadgeTone {
  if (skill.scanState === "CLEAN") return "ok";
  if (skill.scanState === "BLOCKED") return "error";
  if (skill.scanState === "WARNING" || skill.scanState === "UNKNOWN") return "warn";
  return "neutral";
}

function mcpTone(server: MCPServer): BadgeTone {
  if (server.state === "READY") return "ok";
  if (server.state === "FAILED" || server.state === "AUTH_REQUIRED") return "error";
  if (server.state === "UNKNOWN") return "warn";
  return server.state === "CONFIGURED" ? "info" : "neutral";
}

function healthTone(state: ManagementHealth["state"]): BadgeTone {
  if (state === "HEALTHY") return "ok";
  if (state === "UNHEALTHY") return "error";
  if (state === "DEGRADED" || state === "UNKNOWN") return "warn";
  return "neutral";
}
