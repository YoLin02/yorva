import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";
import type { DaemonClient } from "../../api/client";
import { discardSkillImport, selectSkillImport, type SkillImportSelection } from "../../api/session";
import type { Channel, Instance, Lifecycle, MCPPreset, MCPServer, ManagementBackup, ManagementHealth, ManagementLogSnapshot, ManagementUpgradePlan, ModelConfiguration, Skill, SkillSource } from "../../api/types";
import { formatDateTime } from "../../formatDateTime";
import type { AppMessages, Locale } from "../../i18n";
import type { BadgeTone } from "../../types/ui";
import { ModelConfigurationPanel } from "../models/ModelConfigurationPanel";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { IconActivity, IconArchiveRestore, IconChevronDown, IconClose, IconFileArchive, IconFolderInput, IconPlus, IconRefresh, IconSearch } from "../ui/icons";

type ManagementScope = "instance" | "runtime";
type RuntimeManagementTab = "overview" | "instances" | "models" | "skills" | "mcp" | "maintenance" | "diagnostics" | "operations";

export function ManagementPanel({ client, instance, instances = [instance], runtimeCapabilities, scope = "instance", copy, locale, onClose, onOpenModels, onOpenChannels }: {
  client: DaemonClient;
  instance: Instance;
  instances?: Instance[];
  runtimeCapabilities?: Instance["capabilities"];
  scope?: ManagementScope;
  copy: AppMessages;
  locale: Locale;
  onClose?: () => void;
  onOpenModels?: () => void;
  onOpenChannels?: () => void;
}) {
  const queryClient = useQueryClient();
  const runtimeMode = scope === "runtime";
  const [runtimeTab, setRuntimeTab] = useState<RuntimeManagementTab>("overview");
  const [selectedInstanceId, setSelectedInstanceId] = useState(instance.instanceId);
  const [assignmentInstanceIds, setAssignmentInstanceIds] = useState<string[]>([instance.instanceId]);
  const targetInstance = instances.find((item) => item.instanceId === selectedInstanceId) ?? instance;
  const [selectedSkillId, setSelectedSkillId] = useState<string | null>(null);
  const [selectedSkillSource, setSelectedSkillSource] = useState("");
  const [skillImport, setSkillImport] = useState<SkillImportSelection | null>(null);
  const [skillImportId, setSkillImportId] = useState("");
  const [skillImportError, setSkillImportError] = useState(false);
  const [externalSkillsOpen, setExternalSkillsOpen] = useState(false);
  const [skillOperationIds, setSkillOperationIds] = useState<string[]>([]);
  const [backupOperationId, setBackupOperationId] = useState<string | null>(null);
  const [mcpOperationIds, setMCPOperationIds] = useState<string[]>([]);
  const [mcpOperationAction, setMCPOperationAction] = useState<"install" | "authenticate" | "test" | "configure" | "remove" | null>(null);
  const [upgradeOperationId, setUpgradeOperationId] = useState<string | null>(null);
  const [mcpCredentials, setMCPCredentials] = useState<Record<string, string>>({});
  const [mcpTools, setMCPTools] = useState<Record<string, string[]>>({});
  const [mcpComposerOpen, setMCPComposerOpen] = useState(false);
  const [selectedMCPPresetId, setSelectedMCPPresetId] = useState("");
  const [mcpImportedCount, setMCPImportedCount] = useState<number | null>(null);
  const [mcpImportFailed, setMCPImportFailed] = useState(false);
  const [logCategory, setLogCategory] = useState<ManagementLogSnapshot["category"]>("ERRORS");
  const capabilities = targetInstance.capabilities;
  const runtimeCaps = runtimeCapabilities ?? instance.capabilities;
  const healthRead = capabilities.healthRead;
  const logsRead = capabilities.logsRead;
  const skillRead = capabilities.skillRead;
  const skillMutate = capabilities.skillMutate;
  const mcpRead = capabilities.mcpRead;
  const mcpMutate = capabilities.mcpMutate;
  const mcpTest = capabilities.mcpTest;
  const upgradePlanRead = runtimeMode ? runtimeCaps.upgradePlan : capabilities.upgradePlan;
  const upgrade = runtimeMode ? runtimeCaps.upgrade : capabilities.upgrade;
  const rollback = runtimeMode ? runtimeCaps.rollback : capabilities.rollback;
  const backupRead = runtimeMode ? runtimeCaps.backupRead : capabilities.backupRead;
  const backupMutate = runtimeMode ? runtimeCaps.backupMutate : capabilities.backupMutate;
  const restore = runtimeMode ? runtimeCaps.restore : capabilities.restore;
  const healthQuery = useQuery({
    queryKey: ["instance-management-health", targetInstance.instanceId, client.scope],
    queryFn: ({ signal }) => client.getInstanceHealth(targetInstance.instanceId, signal),
    enabled: healthRead && (!runtimeMode || runtimeTab === "overview" || runtimeTab === "diagnostics"),
    retry: false,
  });
  const logsQuery = useQuery({
    queryKey: ["instance-management-logs", targetInstance.instanceId, logCategory, client.scope],
    queryFn: ({ signal }) => client.getInstanceLogSnapshot(targetInstance.instanceId, logCategory, signal),
    enabled: logsRead && (!runtimeMode || runtimeTab === "diagnostics"),
    retry: false,
  });
  const skillsQuery = useQuery({
    queryKey: ["instance-skills", targetInstance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceSkills(targetInstance.instanceId, signal),
    enabled: skillRead && (!runtimeMode || runtimeTab === "skills"),
    retry: false,
  });
  const skillQuery = useQuery({
    queryKey: ["instance-skill", targetInstance.instanceId, selectedSkillId, client.scope],
    queryFn: ({ signal }) => client.inspectInstanceSkill(targetInstance.instanceId, selectedSkillId!, signal),
    enabled: skillRead && selectedSkillId !== null && (!runtimeMode || runtimeTab === "skills"),
    retry: false,
  });
  const skillSourcesQuery = useQuery({
    queryKey: ["instance-skill-sources", targetInstance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceSkillSources(targetInstance.instanceId, signal),
    enabled: runtimeMode && runtimeTab === "skills" && skillMutate,
    retry: false,
  });
  const skillAssignmentQueries = useQueries({
    queries: instances.map((item) => ({
      queryKey: ["instance-skills", item.instanceId, client.scope] as const,
      queryFn: ({ signal }: { signal: AbortSignal }) => client.listInstanceSkills(item.instanceId, signal),
      enabled: runtimeMode && runtimeTab === "skills" && assignmentInstanceIds.includes(item.instanceId) && item.availability === "AVAILABLE" && item.capabilities.skillRead,
      retry: false,
    })),
  });
  const skillMutation = useMutation({
    mutationFn: async ({ action, skillId, sourceId, sourceRef }: { action: "install" | "import" | "update" | "enable" | "disable" | "remove"; skillId: string; sourceId?: string; sourceRef?: string }) => {
      const key = crypto.randomUUID();
      if (action === "import") return [await client.importManagedSkill(targetInstance.instanceId, skillId, sourceRef!, key)];
      if (action === "install") {
        const targetIds = runtimeMode && assignmentInstanceIds.length > 0 ? assignmentInstanceIds : [targetInstance.instanceId];
        return Promise.all(targetIds.map((instanceId) => client.installManagedSkill(instanceId, skillId, sourceId!, crypto.randomUUID())));
      }
      if (action === "update") return [await client.updateManagedSkill(targetInstance.instanceId, skillId, key)];
      if (action === "enable") return [await client.enableManagedSkill(targetInstance.instanceId, skillId, key)];
      if (action === "disable") return [await client.disableManagedSkill(targetInstance.instanceId, skillId, key)];
      return [await client.removeManagedSkill(targetInstance.instanceId, skillId, key)];
    },
    onMutate: () => setSkillOperationIds([]),
    onSuccess: (accepted) => {
      setSkillOperationIds(accepted.map((operation) => operation.id));
      setSkillImport(null);
      setSkillImportId("");
    },
  });
  const skillOperationQueries = useQueries({
    queries: skillOperationIds.map((operationId) => ({
      queryKey: ["management-operation", operationId, client.scope] as const,
      queryFn: ({ signal }: { signal: AbortSignal }) => client.getOperation(operationId, signal),
      retry: false,
      refetchInterval: (query: { state: { data?: { status?: string } } }) => {
        const status = query.state.data?.status;
        return status === "PENDING" || status === "RUNNING" ? 1000 : false;
      },
    })),
  });
  const skillAllSucceeded = skillOperationQueries.length > 0 && skillOperationQueries.every((query) => query.data?.status === "SUCCEEDED");

  useEffect(() => {
    if (!skillAllSucceeded) return;
    void queryClient.invalidateQueries({ queryKey: ["instance-skills"] });
    void queryClient.invalidateQueries({ queryKey: ["instance-skill"] });
    void queryClient.invalidateQueries({ queryKey: ["instance-skill-sources"] });
  }, [queryClient, skillAllSucceeded]);
  const serversQuery = useQuery({
    queryKey: ["instance-mcp-servers", targetInstance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceMCPServers(targetInstance.instanceId, signal),
    enabled: mcpRead && (!runtimeMode || runtimeTab === "mcp"),
    retry: false,
  });
  const presetsQuery = useQuery({
    queryKey: ["runtime-mcp-definitions", "hermes", client.scope],
    queryFn: ({ signal }) => client.listRuntimeMCPDefinitions("hermes", signal),
    enabled: runtimeMode && runtimeTab === "mcp" && mcpRead,
    retry: false,
  });
  const mcpAssignmentQueries = useQueries({
    queries: instances.map((item) => ({
      queryKey: ["instance-mcp-servers", item.instanceId, client.scope] as const,
      queryFn: ({ signal }: { signal: AbortSignal }) => client.listInstanceMCPServers(item.instanceId, signal),
      enabled: runtimeMode && runtimeTab === "mcp" && assignmentInstanceIds.includes(item.instanceId) && item.availability === "AVAILABLE" && item.capabilities.mcpRead,
      retry: false,
    })),
  });
  const mcpMutation = useMutation({
    mutationFn: async (input: { action: "install" | "authenticate" | "test" | "configure" | "remove"; id: string }) => {
      const key = crypto.randomUUID();
      if (input.action === "install") {
        const targetIds = runtimeMode && assignmentInstanceIds.length > 0 ? assignmentInstanceIds : [targetInstance.instanceId];
        const preset = presetsQuery.data?.items.find((item) => item.id === input.id);
        const credential = mcpCredentials[input.id] ?? "";
        const enabledToolIds = mcpTools[input.id] ?? preset?.allowedToolIds ?? [];
        const accepted = await Promise.all(targetIds.map((instanceId) => client.installInstanceMCPPreset(instanceId, input.id, credential, enabledToolIds, crypto.randomUUID())));
        return accepted;
      }
      if (input.action === "authenticate") return [await client.authenticateInstanceMCPServer(targetInstance.instanceId, input.id, mcpCredentials[input.id] ?? "", key)];
      if (input.action === "test") return [await client.testInstanceMCPServer(targetInstance.instanceId, input.id, key)];
      if (input.action === "configure") return [await client.configureInstanceMCPServer(targetInstance.instanceId, input.id, mcpTools[input.id] ?? [], key)];
      return [await client.removeInstanceMCPServer(targetInstance.instanceId, input.id, key)];
    },
    onSuccess: (accepted, input) => {
      setMCPOperationIds(accepted.map((operation) => operation.id));
      setMCPOperationAction(input.action);
      if (input.action === "authenticate") setMCPCredentials((current) => ({ ...current, [input.id]: "" }));
    },
  });
  const mcpOperationQueries = useQueries({
    queries: mcpOperationIds.map((operationId) => ({
      queryKey: ["management-operation", operationId, client.scope] as const,
      queryFn: ({ signal }: { signal: AbortSignal }) => client.getOperation(operationId, signal),
      retry: false,
      refetchInterval: (query: { state: { data?: { status?: string } } }) => {
        const status = query.state.data?.status;
        return status === "PENDING" || status === "RUNNING" ? 1000 : false;
      },
    })),
  });
  const mcpAllSucceeded = mcpOperationQueries.length > 0 && mcpOperationQueries.every((query) => query.data?.status === "SUCCEEDED");
  const refetchMCPServers = serversQuery.refetch;
  const refetchMCPPresets = presetsQuery.refetch;

  useEffect(() => {
    if (mcpAllSucceeded) {
      void refetchMCPServers();
      void refetchMCPPresets();
    }
  }, [mcpAllSucceeded, refetchMCPServers, refetchMCPPresets]);
  const upgradePlanQuery = useQuery({
    queryKey: ["runtime-upgrade-plan", "hermes", client.scope],
    queryFn: ({ signal }) => client.getRuntimeUpgradePlan("hermes", signal),
    enabled: runtimeMode && runtimeTab === "maintenance" && upgradePlanRead,
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
    enabled: runtimeMode && runtimeTab === "maintenance" && backupRead,
    retry: false,
  });
  const backupMutation = useMutation({
    mutationFn: () => client.createRuntimeBackup("hermes", crypto.randomUUID()),
    onSuccess: (accepted) => setBackupOperationId(accepted.id),
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

  const instanceDetailsEnabled = !runtimeMode && targetInstance.availability === "AVAILABLE" && Boolean(onOpenModels || onOpenChannels);
  const lifecycleQuery = useQuery({
    queryKey: ["instance-management-lifecycle", targetInstance.instanceId, client.scope],
    queryFn: ({ signal }) => client.getInstanceLifecycle(targetInstance.instanceId, signal),
    enabled: instanceDetailsEnabled && capabilities.lifecycle,
    retry: false,
  });
  const modelQuery = useQuery({
    queryKey: ["instance-management-model", targetInstance.instanceId, client.scope],
    queryFn: ({ signal }) => client.getModelConfiguration(targetInstance.instanceId, signal),
    enabled: instanceDetailsEnabled && onOpenModels !== undefined,
    retry: false,
  });
  const channelsQuery = useQuery({
    queryKey: ["instance-management-channels", targetInstance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listInstanceChannels(targetInstance.instanceId, signal),
    enabled: instanceDetailsEnabled && onOpenChannels !== undefined,
    retry: false,
  });
  const [lifecycleOperationId, setLifecycleOperationId] = useState<string | null>(null);
  const lifecycleMutation = useMutation({
    mutationFn: (action: "start" | "stop" | "restart") => client.startInstanceLifecycle(targetInstance.instanceId, action, crypto.randomUUID()),
    onSuccess: (accepted) => setLifecycleOperationId(accepted.id),
  });
  const lifecycleOperationQuery = useQuery({
    queryKey: ["management-operation", lifecycleOperationId, client.scope],
    queryFn: ({ signal }) => client.getOperation(lifecycleOperationId!, signal),
    enabled: lifecycleOperationId !== null,
    retry: false,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 1000 : false;
    },
  });
  const refetchLifecycle = lifecycleQuery.refetch;
  useEffect(() => {
    if (lifecycleOperationQuery.data?.status === "SUCCEEDED") void refetchLifecycle();
  }, [lifecycleOperationQuery.data?.status, refetchLifecycle]);

  const runtimeOperationsQuery = useQuery({
    queryKey: ["runtime-management-operations", targetInstance.runtimeInstallationId, client.scope],
    queryFn: ({ signal }) => client.listOperations("runtime-installation", targetInstance.runtimeInstallationId, signal),
    enabled: runtimeMode && runtimeTab === "operations",
    retry: false,
  });

  useEffect(() => {
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose?.();
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [onClose]);

  const refresh = () => {
    if (healthRead && (!runtimeMode || runtimeTab === "overview" || runtimeTab === "diagnostics")) void healthQuery.refetch();
    if (logsRead && (!runtimeMode || runtimeTab === "diagnostics")) void logsQuery.refetch();
    if (skillRead && (!runtimeMode || runtimeTab === "skills")) {
      void skillsQuery.refetch();
      if (selectedSkillId) void skillQuery.refetch();
    }
    if (runtimeMode && runtimeTab === "skills" && skillMutate) void skillSourcesQuery.refetch();
    if (mcpRead && (!runtimeMode || runtimeTab === "mcp")) {
      void serversQuery.refetch();
      if (runtimeMode) void presetsQuery.refetch();
    }
    if (runtimeMode && runtimeTab === "maintenance" && upgradePlanRead) void upgradePlanQuery.refetch();
    if (runtimeMode && runtimeTab === "maintenance" && backupRead) void backupsQuery.refetch();
    if (runtimeMode && runtimeTab === "operations") void runtimeOperationsQuery.refetch();
    if (runtimeMode && runtimeTab === "models") {
      void queryClient.invalidateQueries({ queryKey: ["model-configuration", targetInstance.instanceId] });
      void queryClient.invalidateQueries({ queryKey: ["model-credential", targetInstance.instanceId] });
    }
    if (!runtimeMode) {
      if (capabilities.lifecycle) void lifecycleQuery.refetch();
      if (onOpenModels) void modelQuery.refetch();
      if (onOpenChannels) void channelsQuery.refetch();
    }
  };
  const selectTargetInstance = (instanceId: string) => {
    setSelectedInstanceId(instanceId);
    setSelectedSkillId(null);
    setMCPCredentials({});
    setMCPTools({});
  };
  const toggleManagedInstance = (instanceId: string) => {
    const next = assignmentInstanceIds.includes(instanceId)
      ? assignmentInstanceIds.filter((id) => id !== instanceId)
      : [...assignmentInstanceIds, instanceId];
    if (next.length === 0) return;
    setAssignmentInstanceIds(next);
    if (!next.includes(selectedInstanceId)) selectTargetInstance(next[0]);
  };
  const backupRunning = backupMutation.isPending || backupDeleteMutation.isPending || backupRestoreMutation.isPending || backupOperationQuery.data?.status === "PENDING" || backupOperationQuery.data?.status === "RUNNING";
  const backupFailed = backupMutation.isError || backupDeleteMutation.isError || backupRestoreMutation.isError || backupOperationQuery.data?.status === "FAILED" || backupOperationQuery.data?.status === "CANCELLED";
  const backupFailureMessage = ({
    BACKUP_SOURCE_RUNTIME_NOT_STOPPED: copy.management.backupRuntimeNotStopped,
    BACKUP_SOURCE_CHANGED: copy.management.backupSourceChanged,
    BACKUP_SOURCE_UNSAFE: copy.management.backupSourceUnsafe,
    BACKUP_SOURCE_INCOMPLETE: copy.management.backupSourceIncomplete,
    BACKUP_DESTINATION_INSUFFICIENT_SPACE: copy.management.backupInsufficientSpace,
    BACKUP_STAGING_FAILED: copy.management.backupStagingFailed,
    BACKUP_ENCRYPTION_FAILED: copy.management.backupEncryptionFailed,
  } as Record<string, string>)[backupOperationQuery.data?.errorCode ?? ""] ?? copy.management.backupCreateFailed;
  const skillRunning = skillMutation.isPending || skillOperationQueries.some((query) => query.data?.status === "PENDING" || query.data?.status === "RUNNING");
  const skillFailed = skillMutation.isError || skillOperationQueries.some((query) => query.isError || query.data?.status === "FAILED" || query.data?.status === "CANCELLED");
  const mcpRunning = mcpMutation.isPending || mcpOperationQueries.some((query) => query.data?.status === "PENDING" || query.data?.status === "RUNNING");
  const mcpFailed = mcpMutation.isError || mcpOperationQueries.some((query) => query.isError || query.data?.status === "FAILED" || query.data?.status === "CANCELLED");
  const upgradeRunning = upgradeMutation.isPending || upgradeOperationQuery.data?.status === "PENDING" || upgradeOperationQuery.data?.status === "RUNNING";
  const upgradeFailed = upgradeMutation.isError || upgradeOperationQuery.data?.status === "FAILED" || upgradeOperationQuery.data?.status === "CANCELLED";
  const activeMCPIndex = mcpOperationQueries.findIndex((query) => query.data?.status === "PENDING" || query.data?.status === "RUNNING");
  const activeManagementOperationId = backupRunning ? backupOperationId : mcpRunning && activeMCPIndex >= 0 ? mcpOperationIds[activeMCPIndex] : null;
  const cancelMutation = useMutation({
    mutationFn: () => client.cancelOperation(activeManagementOperationId!),
    onSuccess: () => {
      void backupOperationQuery.refetch();
      for (const query of mcpOperationQueries) void query.refetch();
      void upgradeOperationQuery.refetch();
    },
  });
  const busy = healthQuery.isFetching || logsQuery.isFetching || skillsQuery.isFetching || skillQuery.isFetching || skillSourcesQuery.isFetching || skillAssignmentQueries.some((query) => query.isFetching) || mcpAssignmentQueries.some((query) => query.isFetching) || skillRunning || serversQuery.isFetching || presetsQuery.isFetching || upgradePlanQuery.isFetching || backupsQuery.isFetching || runtimeOperationsQuery.isFetching || lifecycleQuery.isFetching || modelQuery.isFetching || channelsQuery.isFetching || lifecycleMutation.isPending || backupRunning || mcpRunning || upgradeRunning;
  const displayedSkills = runtimeMode
    ? Array.from(new Map(skillAssignmentQueries.flatMap((query, index) => assignmentInstanceIds.includes(instances[index].instanceId) ? (query.data?.items ?? []) : []).map((skill) => [skill.id, skill])).values())
    : (skillsQuery.data?.items ?? []);
  const managedSkills = displayedSkills.filter((skill) => skill.ownership === "YORVA_MANAGED");
  const externalSkills = displayedSkills.filter((skill) => skill.ownership !== "YORVA_MANAGED");
  const displayedServers = runtimeMode
    ? Array.from(new Map(mcpAssignmentQueries.flatMap((query, index) => assignmentInstanceIds.includes(instances[index].instanceId) ? (query.data?.items ?? []) : []).map((server) => [server.id, server])).values())
    : (serversQuery.data?.items ?? []);
  const skillAssignments = (skillId: string) => instances.filter((item, index) => assignmentInstanceIds.includes(item.instanceId) && skillAssignmentQueries[index]?.data?.items.some((skill) => skill.id === skillId && skill.installationState === "INSTALLED")).map((item) => item.name);
  const mcpAssignments = (serverId: string) => instances.filter((item, index) => assignmentInstanceIds.includes(item.instanceId) && mcpAssignmentQueries[index]?.data?.items.some((server) => server.id === serverId && server.state !== "NOT_CONFIGURED")).map((item) => item.name);
  const installableSkillSources = skillSourcesQuery.data?.items ?? [];
  const openSkillImport = async (kind: "ZIP" | "DIRECTORY") => {
    setSkillImportError(false);
    try {
      const selected = await selectSkillImport(kind);
      if (selected) {
        setSkillImport(selected);
        setSkillImportId(selected.suggestedSkillId);
      }
    } catch {
      setSkillImportError(true);
    }
  };
  const closeSkillImport = () => {
    if (skillImport) void discardSkillImport(skillImport.sourceRef);
    setSkillImport(null);
    setSkillImportId("");
  };
  const openMCPComposer = () => {
    const first = presetsQuery.data?.items[0];
    setMCPOperationIds([]);
    setMCPOperationAction(null);
    setSelectedMCPPresetId((current) => current || first?.id || "");
    if (first && mcpTools[first.id] === undefined) setMCPTools((current) => ({ ...current, [first.id]: first.allowedToolIds }));
    setMCPComposerOpen(true);
  };
  const importExistingMCP = async () => {
    setMCPImportedCount(null);
    setMCPImportFailed(false);
    const result = await serversQuery.refetch();
    if (result.isError) {
      setMCPImportFailed(true);
      return;
    }
    setMCPImportedCount(result.data?.items.length ?? 0);
    await queryClient.invalidateQueries({ queryKey: ["instance-mcp-servers"] });
  };
  const inspectSkill = (skillId: string) => {
    if (selectedSkillId === skillId) {
      setSelectedSkillId(null);
      return;
    }
    const sourceIndex = skillAssignmentQueries.findIndex((query, index) => assignmentInstanceIds.includes(instances[index].instanceId) && query.data?.items.some((skill) => skill.id === skillId));
    if (runtimeMode && sourceIndex >= 0 && instances[sourceIndex].instanceId !== selectedInstanceId) {
      selectTargetInstance(instances[sourceIndex].instanceId);
    }
    setSelectedSkillId(skillId);
  };
  const openSkillPreview = (skillId: string) => {
    const sourceIndex = skillAssignmentQueries.findIndex((query, index) => assignmentInstanceIds.includes(instances[index].instanceId) && query.data?.items.some((skill) => skill.id === skillId));
    if (sourceIndex >= 0 && instances[sourceIndex].instanceId !== selectedInstanceId) {
      selectTargetInstance(instances[sourceIndex].instanceId);
    }
    setSelectedSkillId(skillId);
  };
  const runtimeSkillRow = (skill: Skill) => {
    const targetSkill = skillsQuery.data?.items.find((item) => item.id === skill.id);
    return <RuntimeSkillRow
      key={skill.id}
      skill={targetSkill ?? skill}
      assignedInstances={skillAssignments(skill.id)}
      copy={copy}
      mutable={skillMutate && targetSkill?.ownership === "YORVA_MANAGED"}
      mutationPending={skillRunning && skillMutation.variables?.skillId === skill.id}
      onOpen={() => openSkillPreview(skill.id)}
      onToggle={() => skillMutation.mutate({ action: skill.enabledState === "ENABLED" ? "disable" : "enable", skillId: skill.id })}
    />;
  };

  return (
    <section className={runtimeMode ? "management-panel runtime-management-panel" : "management-panel"} aria-labelledby="management-panel-title">
      <header className="management-panel-header">
        <div>
          <h2 id="management-panel-title">{runtimeMode ? copy.management.runtimeTitle : `${copy.management.title}: ${targetInstance.name}`}</h2>
          <p className="page-copy">{runtimeMode ? copy.management.runtimeDescription : copy.management.description}</p>
        </div>
        <div className="management-header-actions">
          <Button onClick={refresh} disabled={busy || (runtimeMode && !healthRead && !logsRead && !skillRead && !mcpRead && !upgradePlanRead && !backupRead)} className="button-compact button-neutral">
            <IconRefresh className={busy ? "spin" : undefined} />
            {copy.management.refresh}
          </Button>
          {activeManagementOperationId ? <Button onClick={() => cancelMutation.mutate()} disabled={cancelMutation.isPending} className="button-compact button-neutral">{copy.management.cancelOperation}</Button> : null}
          {onClose ? <button type="button" className="modal-close" onClick={onClose} aria-label={copy.management.close}><IconClose /></button> : null}
        </div>
      </header>

      {runtimeMode ? (
        <nav className="runtime-management-tabs" aria-label={copy.management.runtimeNavigation}>
          {(["overview", "instances", "models", "skills", "mcp", "maintenance", "operations"] as RuntimeManagementTab[]).map((tab) => (
            <button key={tab} type="button" className={runtimeTab === tab ? "is-active" : undefined} aria-current={runtimeTab === tab ? "page" : undefined} onClick={() => { setRuntimeTab(tab); setSelectedSkillId(null); }}>
              {copy.management.runtimeTabs[tab]}
            </button>
          ))}
        </nav>
      ) : null}

      {runtimeMode && runtimeTab === "overview" ? (
        <div className="runtime-overview-grid">
          <RuntimeSummaryCard label={copy.management.runtimeSummaryInstances} value={String(instances.filter((item) => item.availability === "AVAILABLE").length)} detail={copy.management.runtimeSummaryInstancesDetail.replace("{count}", String(instances.length))} />
          <RuntimeSummaryCard label={copy.management.runtimeSummaryResources} value={skillRead || mcpRead ? copy.instances.capabilityAvailable : copy.instances.capabilityUnavailable} detail={copy.management.runtimeSummaryResourcesDetail} />
          <RuntimeSummaryCard label={copy.management.runtimeSummaryMaintenance} value={backupRead || upgradePlanRead ? copy.instances.capabilityAvailable : copy.instances.capabilityUnavailable} detail={copy.management.runtimeSummaryMaintenanceDetail} />
          <RuntimeSummaryCard label={copy.management.runtimeSummaryDiagnostics} value={healthQuery.data ? copy.management.healthState[healthQuery.data.state] : healthRead ? copy.management.loading : copy.instances.capabilityUnavailable} detail={copy.management.runtimeSummaryDiagnosticsDetail} />
        </div>
      ) : null}

      {runtimeMode && runtimeTab === "instances" ? (
        <ManagementSection title={copy.management.runtimeInstancesTitle} description={copy.management.runtimeInstancesDescription} supported loading={false} error={false} copy={copy} onRetry={() => undefined}>
          <div className="runtime-instance-list">
            {instances.map((item) => (
              <article key={item.instanceId} className={selectedInstanceId === item.instanceId ? "runtime-instance-row is-selected" : "runtime-instance-row"}>
                <button type="button" className="runtime-instance-row-main" onClick={() => selectTargetInstance(item.instanceId)}>
                  <span><strong>{item.name}</strong>{item.default ? <Badge tone="neutral">{copy.instances.defaultLabel}</Badge> : null}</span>
                  <span>{copy.instances.availability[item.availability]}</span>
                </button>
                <Button
                  className="button-compact button-neutral runtime-instance-diagnostics"
                  disabled={item.availability !== "AVAILABLE" || (!item.capabilities.healthRead && !item.capabilities.logsRead)}
                  onClick={() => {
                    selectTargetInstance(item.instanceId);
                    setRuntimeTab("diagnostics");
                  }}
                >
                  <IconActivity />
                  {copy.management.openDiagnostics}
                </Button>
              </article>
            ))}
          </div>
        </ManagementSection>
      ) : null}

      {runtimeMode && runtimeTab === "diagnostics" ? (
        <DiagnosticInstanceSwitcher instances={instances} value={targetInstance.instanceId} copy={copy} onChange={selectTargetInstance} />
      ) : null}

      {runtimeMode && runtimeTab === "models" ? (
        <div className="runtime-model-workspace">
          <RuntimeInstanceSelector
            instances={instances}
            value={targetInstance.instanceId}
            label={copy.management.configuringInstance}
            description={copy.management.modelConfiguringInstanceDescription}
            copy={copy}
            onChange={selectTargetInstance}
          />
          <ModelConfigurationPanel client={client} instance={targetInstance} copy={copy} locale={locale} embedded />
        </div>
      ) : null}

      {runtimeMode && runtimeTab === "skills" && selectedSkillId === null ? (
        <RuntimeInstanceSelector
          instances={instances}
          value={targetInstance.instanceId}
          label={copy.management.configuringInstance}
          description={copy.management.configuringInstanceDescription}
          copy={copy}
          onChange={(instanceId) => {
            setAssignmentInstanceIds([instanceId]);
            selectTargetInstance(instanceId);
          }}
        />
      ) : null}

      {!runtimeMode ? (
        <InstanceOverview
          instance={targetInstance}
          lifecycle={lifecycleQuery.data?.state}
          lifecycleBusy={lifecycleMutation.isPending || lifecycleOperationQuery.data?.status === "PENDING" || lifecycleOperationQuery.data?.status === "RUNNING"}
          model={modelQuery.data}
          channels={channelsQuery.data?.channels ?? []}
          skills={skillsQuery.data?.items ?? []}
          servers={serversQuery.data?.items ?? []}
          health={healthQuery.data}
          copy={copy}
          onLifecycle={(action) => {
            if ((action === "stop" || action === "restart") && !window.confirm(action === "stop" ? copy.instances.lifecycleStopWarning : copy.instances.lifecycleRestartWarning)) return;
            lifecycleMutation.mutate(action);
          }}
          onOpenModels={onOpenModels}
          onOpenChannels={onOpenChannels}
        />
      ) : null}

      {(!runtimeMode || runtimeTab === "diagnostics") ? <ManagementSection
        title={copy.management.diagnosticsTitle}
        description={copy.management.diagnosticsDescription}
        supported={healthRead || logsRead || (runtimeMode && capabilities.securityAudit)}
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
          {runtimeMode ? <DiagnosticBlock title={copy.management.securityTitle} supported={capabilities.securityAudit} copy={copy}>
            <p className="management-empty">{copy.management.securityDescription}</p>
          </DiagnosticBlock> : null}
        </div>
      </ManagementSection> : null}

      {runtimeMode && runtimeTab === "maintenance" ? <ManagementSection
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
      </ManagementSection> : null}

      {runtimeMode && runtimeTab === "maintenance" ? <ManagementSection
        title={copy.management.backupsTitle}
        description={copy.management.backupsDescription}
        supported={backupRead}
        loading={backupsQuery.isLoading}
        error={backupsQuery.isError}
        copy={copy}
        onRetry={() => void backupsQuery.refetch()}
      >
        <p className="management-empty">{copy.management.backupRequiresStopped}</p>
        {backupMutate ? (
          <div className="management-item-footer">
            <span>{backupRunning ? copy.management.backupCreating : null}</span>
            <Button disabled={backupRunning} onClick={() => backupMutation.mutate()}>{copy.management.createBackup}</Button>
          </div>
        ) : null}
        {backupFailed ? <p className="notice notice-warn" role="alert">{backupFailureMessage}</p> : null}
        {backupsQuery.data?.items.length === 0 ? <p className="management-empty">{copy.management.noBackups}</p> : null}
        <div className="management-list">
          {backupsQuery.data?.items.map((backup) => <BackupCard
            key={backup.backupId} backup={backup} copy={copy} locale={locale}
            mutable={backupMutate} restorable={restore} busy={backupRunning}
            onDelete={() => backupDeleteMutation.mutate(backup.backupId)}
            onRestore={() => { if (window.confirm(copy.management.restoreConfirm)) backupRestoreMutation.mutate(backup.backupId); }}
          />)}
        </div>
      </ManagementSection> : null}

      {runtimeMode && runtimeTab === "skills" && selectedSkillId !== null ? (
        <SkillPreviewPage
          skill={skillQuery.data ?? displayedSkills.find((skill) => skill.id === selectedSkillId)}
          instanceName={targetInstance.name}
          loading={skillQuery.isLoading}
          error={skillQuery.isError}
          copy={copy}
          onBack={() => setSelectedSkillId(null)}
          onRetry={() => void skillQuery.refetch()}
        />
      ) : null}

      {(!runtimeMode || (runtimeTab === "skills" && selectedSkillId === null)) ? <ManagementSection
        title={runtimeMode ? copy.management.skillsTitle : copy.management.skillBindingsTitle}
        description={runtimeMode ? copy.management.skillsDescription : copy.management.skillBindingsDescription}
          supported={skillRead || skillMutate}
          loading={skillsQuery.isLoading || skillSourcesQuery.isLoading}
          error={skillsQuery.isError || skillSourcesQuery.isError}
        copy={copy}
        onRetry={() => void skillsQuery.refetch()}
        className={runtimeMode ? "runtime-skills-section" : undefined}
        showCapabilityBadge={!runtimeMode}
        hideHeading={runtimeMode}
      >
        {runtimeMode ? <div className="runtime-skill-toolbar" aria-label={copy.management.skillTools}>
          <Button disabled={busy} onClick={() => { void skillSourcesQuery.refetch(); void skillsQuery.refetch(); }}><IconRefresh />{copy.management.checkSkillUpdates}</Button>
          <Button disabled={busy} onClick={() => setRuntimeTab("maintenance")}><IconArchiveRestore />{copy.management.restoreSkills}</Button>
          <Button disabled={busy || !skillMutate} onClick={() => void openSkillImport("ZIP")}><IconFileArchive />{copy.management.installSkillZip}</Button>
          <Button disabled={busy || !skillMutate} onClick={() => void openSkillImport("DIRECTORY")}><IconFolderInput />{copy.management.importExistingSkill}</Button>
          <Button disabled={busy} onClick={() => { setExternalSkillsOpen(true); void skillsQuery.refetch(); }}><IconSearch />{copy.management.discoverSkills}</Button>
        </div> : null}
        {skillImportError ? <p className="notice notice-warn" role="alert">{copy.management.skillImportSourceFailed}</p> : null}
        {runtimeMode ? (
          <div className="runtime-skill-groups">
            <section className="runtime-skill-group runtime-skill-group-managed">
              <header><div><h4>{copy.management.managedSkillsGroup}</h4><p>{copy.management.managedSkillsGroupDescription}</p></div><span className="runtime-skill-group-count">{managedSkills.length}</span></header>
              {managedSkills.length === 0 ? <p className="management-empty">{copy.management.noManagedSkills}</p> : <div className="runtime-skill-group-list">{managedSkills.map(runtimeSkillRow)}</div>}
            </section>
            <details className="runtime-skill-group runtime-skill-group-external" open={externalSkillsOpen} onToggle={(event) => setExternalSkillsOpen(event.currentTarget.open)}>
              <summary><div><h4>{copy.management.externalSkillsGroup}</h4><p>{copy.management.externalSkillsGroupDescription}</p></div><span className="runtime-skill-group-summary-meta"><span className="runtime-skill-group-count">{externalSkills.length}</span><span className="runtime-skill-group-chevron"><IconChevronDown /></span></span></summary>
              {externalSkills.length === 0 ? <p className="management-empty">{copy.management.noSkills}</p> : <div className="runtime-skill-group-list">{externalSkills.map(runtimeSkillRow)}</div>}
            </details>
          </div>
        ) : (
          <>
          {displayedSkills.length === 0 ? <p className="management-empty">{copy.management.noSkills}</p> : null}
          <div className="management-list">
            {displayedSkills.map((skill) => {
            const targetSkill = skillsQuery.data?.items.find((item) => item.id === skill.id);
            return <SkillCard
              key={skill.id}
              skill={targetSkill ?? skill}
              assignedInstances={[]}
              inspected={selectedSkillId === skill.id ? skillQuery.data : undefined}
              inspecting={selectedSkillId === skill.id && skillQuery.isLoading}
              inspectFailed={selectedSkillId === skill.id && skillQuery.isError}
              copy={copy}
              mutable={false}
              inspectable
              onToggle={() => inspectSkill(skill.id)}
              onRetry={() => void skillQuery.refetch()}
              mutationPending={false}
              onAction={() => undefined}
            />;
          })}
          </div>
          </>
        )}
        {runtimeMode && skillMutate && installableSkillSources.length > 0 ? (
          <SkillCatalog
            sources={installableSkillSources}
            selected={selectedSkillSource}
            installedSkillIds={skillsQuery.data?.items.filter((skill) => skill.installationState === "INSTALLED").map((skill) => skill.id) ?? []}
            assignmentReady={assignmentInstanceIds.length > 0}
            appliesBeyondObserved={assignmentInstanceIds.some((instanceId) => instanceId !== targetInstance.instanceId)}
            pending={skillRunning}
            copy={copy}
            onSelect={setSelectedSkillSource}
            onInstall={(source) => skillMutation.mutate({ action: "install", skillId: source.skillId, sourceId: source.sourceId })}
          />
        ) : null}
        {skillRunning ? <p className="management-empty" role="status">{copy.management.skillMutationRunning}</p> : null}
        {skillFailed ? <p className="notice notice-warn" role="alert">{copy.management.skillMutationFailed}</p> : null}
        {skillImport ? <div className="skill-import-backdrop" role="presentation" onMouseDown={closeSkillImport}>
          <div className="skill-import-dialog" role="dialog" aria-modal="true" aria-labelledby="skill-import-title" onMouseDown={(event) => event.stopPropagation()}>
            <h3 id="skill-import-title">{copy.management.skillImportTitle}</h3>
            <p>{copy.management.skillImportDescription}</p>
            <label><span>{copy.management.skillUniqueId}</span><input value={skillImportId} onChange={(event) => setSkillImportId(event.target.value.toLowerCase())} autoFocus /></label>
            <small>{copy.management.skillImportTarget.replace("{instance}", targetInstance.name)}</small>
            <div className="management-header-actions"><Button onClick={closeSkillImport}>{copy.management.cancelSkillImport}</Button><Button className="button-primary" disabled={!/^[a-z][a-z0-9_-]{0,63}$/.test(skillImportId) || skillRunning} onClick={() => skillMutation.mutate({ action: "import", skillId: skillImportId, sourceRef: skillImport.sourceRef })}>{copy.management.installSkill}</Button></div>
          </div>
        </div> : null}
      </ManagementSection> : null}

      {(!runtimeMode || runtimeTab === "mcp") ? <ManagementSection
        title={runtimeMode ? copy.management.mcpTitle : copy.management.mcpBindingsTitle}
        description={runtimeMode ? copy.management.mcpDescription : copy.management.mcpBindingsDescription}
        supported={mcpRead || mcpMutate}
        loading={serversQuery.isLoading || presetsQuery.isLoading}
        error={serversQuery.isError || presetsQuery.isError}
        copy={copy}
        onRetry={() => { void serversQuery.refetch(); void presetsQuery.refetch(); }}
        className={runtimeMode ? "runtime-mcp-section" : undefined}
        hideHeading={runtimeMode}
      >
        {runtimeMode ? <div className="runtime-mcp-page">
          <div className="runtime-mcp-toolbar" aria-label={copy.management.mcpTools}>
            <Button disabled={busy || !mcpRead} onClick={() => void importExistingMCP()}><IconFolderInput />{copy.management.importExistingMCP}</Button>
            <Button className="button-primary" disabled={mcpRunning || !mcpMutate} onClick={openMCPComposer}><IconPlus />{copy.management.addMCP}</Button>
          </div>
          {mcpImportedCount !== null ? <p className="notice notice-info" role="status">{copy.management.mcpImportSucceeded.replace("{count}", String(mcpImportedCount))}</p> : null}
          {mcpImportFailed ? <p className="notice notice-warn" role="alert">{copy.management.mcpImportFailed}</p> : null}
          <section className="runtime-mcp-group">
            <header className="runtime-mcp-group-heading">
              <div><h3>{copy.management.mcpDefinitionsTitle}</h3><p>{copy.management.mcpDefinitionsDescription}</p></div>
            </header>
            {mcpRead && !mcpMutate ? <p className="runtime-mcp-capability-note">{copy.management.mcpReadOnlyCapability}</p> : null}
            {presetsQuery.data?.items.length === 0 ? <div className="runtime-mcp-empty"><strong>{copy.management.noPresets}</strong><span>{copy.management.noPresetsDescription}</span></div> : (
              <div className="runtime-mcp-definition-list">{presetsQuery.data?.items.map((preset) => <article key={preset.id} className="runtime-mcp-definition-row"><div><strong>{preset.displayName}</strong><code>{preset.id}</code><p>{preset.description}</p></div><Badge tone="info">{copy.management.reviewedPreset}</Badge></article>)}</div>
            )}
          </section>
          {mcpComposerOpen && !(mcpOperationAction === "install" && mcpAllSucceeded) ? <MCPComposer
            presets={presetsQuery.data?.items ?? []}
            selectedPresetId={selectedMCPPresetId}
            instances={instances} selectedInstances={assignmentInstanceIds}
            credentials={mcpCredentials} tools={mcpTools} busy={mcpRunning} copy={copy}
            onSelectPreset={(preset) => { setSelectedMCPPresetId(preset.id); if (mcpTools[preset.id] === undefined) setMCPTools((current) => ({ ...current, [preset.id]: preset.allowedToolIds })); }}
            onToggleInstance={toggleManagedInstance}
            onCredential={(presetId, value) => setMCPCredentials((current) => ({ ...current, [presetId]: value }))}
            onToggleTool={(presetId, toolId) => setMCPTools((current) => { const selected = current[presetId] ?? []; return { ...current, [presetId]: selected.includes(toolId) ? selected.filter((id) => id !== toolId) : [...selected, toolId] }; })}
            onCancel={() => setMCPComposerOpen(false)}
            onSubmit={(presetId) => mcpMutation.mutate({ action: "install", id: presetId })}
          /> : null}
          <section className="runtime-mcp-group">
            <header className="runtime-mcp-group-heading"><div><h3>{copy.management.mcpBindingsTitle}</h3><p>{copy.management.mcpBindingsDescription}</p></div></header>
            <InstanceAssignmentSelector instances={instances} selected={assignmentInstanceIds} copy={copy} onToggle={toggleManagedInstance} />
            {displayedServers.length === 0 ? <div className="runtime-mcp-empty"><strong>{copy.management.noServersTitle}</strong><span>{copy.management.noServersDescription}</span></div> : null}
            <div className="management-list">
              {displayedServers.map((server) => {
                const targetServer = serversQuery.data?.items.find((item) => item.id === server.id);
                const preset = presetsQuery.data?.items.find((item) => item.id === server.presetId);
                return <MCPServerCard key={server.id} server={targetServer ?? server} preset={preset}
                  assignedInstances={mcpAssignments(server.id)} copy={copy} locale={locale}
                  mutable={mcpMutate && preset !== undefined && targetServer?.ownership === "YORVA_MANAGED"} testable={mcpTest && targetServer?.ownership === "YORVA_MANAGED"}
                  busy={mcpRunning} credential={mcpCredentials[server.id] ?? ""} selectedTools={mcpTools[server.id] ?? targetServer?.enabledToolIds ?? preset?.allowedToolIds ?? []}
                  onCredential={(value) => setMCPCredentials((current) => ({ ...current, [server.id]: value }))}
                  onToggleTool={(toolId) => setMCPTools((current) => { const selected = current[server.id] ?? preset?.allowedToolIds ?? []; return { ...current, [server.id]: selected.includes(toolId) ? selected.filter((id) => id !== toolId) : [...selected, toolId] }; })}
                  onAction={(action) => mcpMutation.mutate({ action, id: server.id })} />;
              })}
            </div>
          </section>
        </div> : <>
          {displayedServers.length === 0 ? <div className="runtime-mcp-empty"><strong>{copy.management.noServersTitle}</strong><span>{copy.management.noServersDescription}</span></div> : null}
          <div className="management-list">{displayedServers.map((server) => <MCPServerCard key={server.id} server={server} assignedInstances={[]} copy={copy} locale={locale} mutable={false} testable={false} busy={false} credential="" selectedTools={[]} onCredential={() => undefined} onToggleTool={() => undefined} onAction={() => undefined} />)}</div>
        </>}
        {mcpRunning ? <p className="management-empty" role="status">{copy.management.mcpMutationRunning}</p> : null}
        {mcpFailed ? <p className="notice notice-warn" role="alert">{copy.management.mcpMutationFailed}</p> : null}
      </ManagementSection> : null}

      {runtimeMode && runtimeTab === "operations" ? (
        <ManagementSection title={copy.management.operationsTitle} description={copy.management.operationsDescription} supported loading={runtimeOperationsQuery.isLoading} error={runtimeOperationsQuery.isError} copy={copy} onRetry={() => void runtimeOperationsQuery.refetch()}>
          {runtimeOperationsQuery.data?.operations.length === 0 ? <p className="management-empty">{copy.management.noOperations}</p> : null}
          <div className="management-list">
            {runtimeOperationsQuery.data?.operations.map((operation) => (
              <article key={operation.id} className="management-item">
                <div className="management-item-heading"><strong>{operation.type}</strong><Badge tone={operation.status === "SUCCEEDED" ? "ok" : operation.status === "FAILED" ? "error" : "neutral"}>{operation.status}</Badge></div>
                <div className="management-meta"><span>{operation.stage}</span><time>{formatDateTime(operation.updatedAt, locale)}</time></div>
              </article>
            ))}
          </div>
        </ManagementSection>
      ) : null}
    </section>
  );
}

function RuntimeSummaryCard({ label, value, detail }: { label: string; value: string; detail: string }) {
  return <article className="runtime-summary-card"><span>{label}</span><strong>{value}</strong><p>{detail}</p></article>;
}

function DiagnosticInstanceSwitcher({ instances, value, copy, onChange }: { instances: Instance[]; value: string; copy: AppMessages; onChange: (value: string) => void }) {
  return (
    <section className="diagnostic-instance-switcher" aria-label={copy.management.diagnosticInstance}>
      <div><strong>{copy.management.diagnosticInstance}</strong><small>{copy.management.diagnosticInstanceDescription}</small></div>
      <div className="diagnostic-instance-options">
        {instances.filter((item) => item.availability === "AVAILABLE").map((item) => (
          <button key={item.instanceId} type="button" className={value === item.instanceId ? "is-active" : undefined} aria-pressed={value === item.instanceId} onClick={() => onChange(item.instanceId)}>{item.name}</button>
        ))}
      </div>
    </section>
  );
}

function InstanceAssignmentSelector({ instances, selected, copy, onToggle }: { instances: Instance[]; selected: string[]; copy: AppMessages; onToggle: (instanceId: string) => void }) {
  return (
    <fieldset className="runtime-assignment-selector">
      <legend>{copy.management.assignmentInstances}</legend>
      <p>{copy.management.assignmentInstancesDescription}</p>
      <div>
        {instances.filter((item) => item.availability === "AVAILABLE").map((item) => (
          <label key={item.instanceId}><input type="checkbox" checked={selected.includes(item.instanceId)} onChange={() => onToggle(item.instanceId)} /><span>{item.name}</span></label>
        ))}
      </div>
    </fieldset>
  );
}

function RuntimeInstanceSelector({ instances, value, label, description, copy, onChange }: { instances: Instance[]; value: string; label: string; description: string; copy: AppMessages; onChange: (instanceId: string) => void }) {
  return (
    <label className="runtime-skill-instance-selector">
      <span className="runtime-skill-instance-copy"><strong>{label}</strong><span>{description}</span></span>
      <select value={value} onChange={(event) => onChange(event.target.value)} aria-label={label}>
        {instances.filter((item) => item.availability === "AVAILABLE").map((item) => (
          <option key={item.instanceId} value={item.instanceId}>{item.name}{item.default ? ` (${copy.instances.defaultLabel})` : ""}</option>
        ))}
      </select>
    </label>
  );
}

function InstanceOverview({ instance, lifecycle, lifecycleBusy, model, channels, skills, servers, health, copy, onLifecycle, onOpenModels, onOpenChannels }: {
  instance: Instance;
  lifecycle?: Lifecycle["state"];
  lifecycleBusy: boolean;
  model?: ModelConfiguration;
  channels: Channel[];
  skills: Skill[];
  servers: MCPServer[];
  health?: ManagementHealth;
  copy: AppMessages;
  onLifecycle: (action: "start" | "stop" | "restart") => void;
  onOpenModels?: () => void;
  onOpenChannels?: () => void;
}) {
  const enabledSkills = skills.filter((skill) => skill.enabledState === "ENABLED").length;
  const boundServers = servers.filter((server) => server.state !== "NOT_CONFIGURED").length;
  const connectedChannels = channels.filter((channel) => channel.state === "CONNECTED").length;
  const lifecycleLabel = lifecycle === "RUNNING" ? copy.instances.lifecycleRunning : lifecycle === "STOPPED" ? copy.instances.lifecycleStopped : lifecycle === "UNKNOWN" ? copy.instances.lifecycleUnknown : copy.instances.availability[instance.availability];
  return (
    <section className="instance-management-overview">
      <div className="instance-management-status">
        <span>{copy.management.currentState}</span>
        <Badge tone={lifecycle === "RUNNING" ? "ok" : lifecycle === "STOPPED" ? "neutral" : "warn"}>{lifecycleLabel}</Badge>
        {instance.capabilities.lifecycle ? <div className="management-header-actions">
          {lifecycle !== "RUNNING" ? <Button disabled={lifecycleBusy} onClick={() => onLifecycle("start")}>{copy.instances.lifecycleStart}</Button> : <Button disabled={lifecycleBusy} onClick={() => onLifecycle("stop")}>{copy.instances.lifecycleStop}</Button>}
          <Button disabled={lifecycleBusy || lifecycle !== "RUNNING"} onClick={() => onLifecycle("restart")}>{copy.instances.lifecycleRestart}</Button>
        </div> : null}
      </div>
      <div className="instance-management-summary-grid">
        <RuntimeSummaryCard label={copy.models.model} value={model?.state === "CONFIGURED" ? model.modelId : copy.models.configState.UNCONFIGURED} detail={model?.providerPresetId || copy.management.openDetailedConfiguration} />
        <RuntimeSummaryCard label={copy.channels.title} value={String(connectedChannels)} detail={copy.management.connectedChannelsDetail.replace("{count}", String(channels.length))} />
        <RuntimeSummaryCard label={copy.management.skillBindingsTitle} value={String(enabledSkills)} detail={copy.management.enabledBindingsDetail} />
        <RuntimeSummaryCard label={copy.management.mcpBindingsTitle} value={String(boundServers)} detail={copy.management.boundBindingsDetail} />
        <RuntimeSummaryCard label={copy.management.healthTitle} value={health ? copy.management.healthState[health.state] : copy.instances.capabilityUnavailable} detail={copy.management.instanceHealthDetail} />
      </div>
      <div className="management-header-actions instance-detail-actions">
        {onOpenModels ? <Button onClick={onOpenModels}>{copy.management.configureModel}</Button> : null}
        {onOpenChannels ? <Button onClick={onOpenChannels}>{copy.management.configureChannels}</Button> : null}
      </div>
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
        <div className="management-upgrade-evidence">
          <strong>{copy.management.upgradeEvidenceRequired}</strong>
          <ul className="management-finding-list">{plan.blockedReasons.map((reason) => <li key={reason}>{copy.management.upgradeReasons[reason]}</li>)}</ul>
        </div>
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

function ManagementSection({ title, description, supported, loading, error, copy, onRetry, className, showCapabilityBadge = true, hideHeading = false, children }: {
  title: string;
  description: string;
  supported: boolean;
  loading: boolean;
  error: boolean;
  copy: AppMessages;
  onRetry: () => void;
  className?: string;
  showCapabilityBadge?: boolean;
  hideHeading?: boolean;
  children: ReactNode;
}) {
  return (
    <section className={`management-section${className ? ` ${className}` : ""}`} aria-label={hideHeading ? title : undefined}>
      {!hideHeading ? <div className="management-section-heading">
        <div><h3>{title}</h3><p className="page-copy">{description}</p></div>
        {showCapabilityBadge ? <Badge tone={supported ? "ok" : "neutral"}>{supported ? copy.instances.capabilityAvailable : copy.instances.capabilityUnavailable}</Badge> : null}
      </div> : null}
      {!supported ? <p className="notice notice-info">{copy.management.unavailable}</p> : null}
      {supported && loading ? <p className="management-empty" role="status">{copy.management.loading}</p> : null}
      {supported && error ? (
        <div className="management-error" role="alert"><span>{copy.management.requestFailed}</span><Button onClick={onRetry}>{copy.management.retry}</Button></div>
      ) : null}
      {supported && !loading && !error ? children : null}
    </section>
  );
}

function RuntimeSkillRow({ skill, assignedInstances, copy, mutable, mutationPending, onOpen, onToggle }: {
  skill: Skill;
  assignedInstances: string[];
  copy: AppMessages;
  mutable: boolean;
  mutationPending: boolean;
  onOpen: () => void;
  onToggle: () => void;
}) {
  const enabled = skill.enabledState === "ENABLED";
  const toggleReady = mutable && (skill.enabledState === "ENABLED" || skill.enabledState === "DISABLED") && skill.projectionState !== "DRIFT_MODIFIED" && skill.projectionState !== "CONFLICT";
  return (
    <article className="runtime-skill-row">
      <button type="button" className="runtime-skill-row-main" aria-label={`${skill.id}: ${skill.description || copy.management.skillDescriptionFallback}`} onClick={onOpen}>
        <span className="runtime-skill-row-copy">
          <code>{skill.id}</code>
          <span className="runtime-skill-row-description">{skill.description || copy.management.skillDescriptionFallback}</span>
          <span className="runtime-skill-row-assignment">{assignedInstances.length > 0 ? `${copy.management.appliedInstances}: ${assignedInstances.join(", ")}` : copy.management.noUpdate}</span>
        </span>
      </button>
      <div className="runtime-skill-row-actions">
        <label className="runtime-skill-switch" title={!toggleReady ? copy.management.skillToggleReadOnly : undefined}>
          <input
            type="checkbox"
            role="switch"
            aria-label={`${skill.id}: ${enabled ? copy.management.disableSkill : copy.management.enableSkill}`}
            aria-checked={enabled}
            checked={enabled}
            disabled={!toggleReady || mutationPending}
            onChange={onToggle}
          />
          <span aria-hidden="true" />
        </label>
      </div>
    </article>
  );
}

function SkillPreviewPage({ skill, instanceName, loading, error, copy, onBack, onRetry }: {
  skill?: Skill;
  instanceName: string;
  loading: boolean;
  error: boolean;
  copy: AppMessages;
  onBack: () => void;
  onRetry: () => void;
}) {
  return (
    <section className="skill-preview-page">
      <button type="button" className="skill-preview-back" onClick={onBack}>← {copy.management.skillPreviewBack}</button>
      {loading ? <p className="management-empty" role="status">{copy.management.loading}</p> : null}
      {error ? <ManagementQueryError copy={copy} onRetry={onRetry} /> : null}
      {skill ? (
        <>
          <header className="skill-preview-heading">
            <div><h3>{skill.id}</h3><p>{skill.description || copy.management.skillDescriptionFallback}</p></div>
            <div className="management-badges">
              <Badge tone={skillInstallationTone(skill)}>{copy.management.installationState[skill.installationState]}</Badge>
              <Badge tone={skill.enabledState === "ENABLED" ? "ok" : "neutral"}>{copy.management.enabledState[skill.enabledState]}</Badge>
              <Badge tone={skill.ownership === "YORVA_MANAGED" ? "info" : "neutral"}>{copy.management.ownershipState[skill.ownership]}</Badge>
            </div>
          </header>
          <dl className="management-detail-grid skill-preview-meta">
            <div><dt>{copy.management.version}</dt><dd>{skill.version || "—"}</dd></div>
            <div><dt>{copy.management.source}</dt><dd>{skill.sourceId || "—"}</dd></div>
            <div><dt>{copy.management.appliedInstances}</dt><dd>{instanceName}</dd></div>
            <div><dt>{copy.management.projection}</dt><dd>{copy.management.projectionState[skill.projectionState]}</dd></div>
          </dl>
          <div className="skill-preview-content">
            <div><h4>{copy.management.skillPreviewContent}</h4></div>
            {skill.preview ? <pre>{skill.preview}</pre> : <p className="management-empty">{copy.management.skillPreviewUnavailable}</p>}
          </div>
        </>
      ) : null}
    </section>
  );
}

function SkillCard({ skill, assignedInstances, inspected, inspecting, inspectFailed, mutationPending, mutable, inspectable, copy, onToggle, onRetry, onAction }: {
  skill: Skill;
  assignedInstances: string[];
  inspected?: Skill;
  inspecting: boolean;
  inspectFailed: boolean;
  mutationPending: boolean;
  mutable: boolean;
  inspectable: boolean;
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
        <span>{assignedInstances.length > 0 ? `${copy.management.appliedInstances}: ${assignedInstances.join(", ")}` : skill.updateAvailable ? copy.management.updateAvailable : copy.management.noUpdate}</span>
        <div className="management-header-actions">
          {mutable && skill.ownership === "YORVA_MANAGED" ? (
            <>
              {skill.updateAvailable ? <Button disabled={mutationPending} onClick={() => onAction("update")}>{copy.management.updateSkill}</Button> : null}
              {skill.projectionState === "PROJECTED"
                ? <Button disabled={mutationPending} onClick={() => onAction("disable")}>{copy.management.disableSkill}</Button>
                : <Button disabled={mutationPending || skill.projectionState === "DRIFT_MODIFIED" || skill.projectionState === "CONFLICT"} onClick={() => onAction("enable")}>{copy.management.enableSkill}</Button>}
              <Button disabled={mutationPending || skill.projectionState === "DRIFT_MODIFIED" || skill.projectionState === "CONFLICT"} onClick={() => onAction("remove")}>{copy.management.removeSkill}</Button>
            </>
          ) : <span className="management-detail">{skill.ownership === "YORVA_MANAGED" ? copy.management.bindingManagedAtRuntime : copy.management.externalReadOnly}</span>}
          <Button variant="ghost" disabled={!inspectable} onClick={onToggle}>{open ? copy.management.hideDetails : copy.management.inspect}</Button>
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

function SkillCatalog({ sources, selected, installedSkillIds, assignmentReady, appliesBeyondObserved, pending, copy, onSelect, onInstall }: {
  sources: SkillSource[];
  selected: string;
  installedSkillIds: string[];
  assignmentReady: boolean;
  appliesBeyondObserved: boolean;
  pending: boolean;
  copy: AppMessages;
  onSelect: (sourceId: string) => void;
  onInstall: (source: SkillSource) => void;
}) {
  const current = sources.find((source) => source.sourceId === selected) ?? sources[0];
  const installedOnObservedTarget = Boolean(current && installedSkillIds.includes(current.skillId));
  return (
    <div className="management-catalog">
      <h4>{copy.management.skillCatalogTitle}</h4>
      {sources.length === 0 ? <p className="management-empty">{copy.management.noSkillSources}</p> : (
        <div className="management-item-footer">
          <select aria-label={copy.management.skillCatalogTitle} value={current?.sourceId ?? ""} onChange={(event) => onSelect(event.target.value)} disabled={pending}>
            {sources.map((source) => <option key={source.sourceId} value={source.sourceId}>{source.displayName} ({source.version})</option>)}
          </select>
          <Button disabled={pending || !current || !assignmentReady || (installedOnObservedTarget && !appliesBeyondObserved)} onClick={() => current && onInstall(current)}>{installedOnObservedTarget ? appliesBeyondObserved ? copy.management.applyToInstances : copy.management.alreadyInstalled : copy.management.installSkill}</Button>
        </div>
      )}
    </div>
  );
}

function MCPComposer({ presets, selectedPresetId, instances, selectedInstances, credentials, tools, busy, copy, onSelectPreset, onToggleInstance, onCredential, onToggleTool, onCancel, onSubmit }: {
  presets: MCPPreset[];
  selectedPresetId: string;
  instances: Instance[];
  selectedInstances: string[];
  credentials: Record<string, string>;
  tools: Record<string, string[]>;
  busy: boolean;
  copy: AppMessages;
  onSelectPreset: (preset: MCPPreset) => void;
  onToggleInstance: (instanceId: string) => void;
  onCredential: (presetId: string, value: string) => void;
  onToggleTool: (presetId: string, toolId: string) => void;
  onCancel: () => void;
  onSubmit: (presetId: string) => void;
}) {
  const preset = presets.find((item) => item.id === selectedPresetId) ?? presets[0];
  if (!preset) return null;
  const selectedTools = tools[preset.id] ?? preset.allowedToolIds;
  const credential = credentials[preset.id] ?? "";
  const preview = JSON.stringify({
    id: preset.id,
    transport: "HTTPS",
    endpoint: "<reviewed-preset-endpoint>",
    authentication: preset.credentialRequired ? "Bearer <credential-redacted>" : "None",
    enabledToolIds: selectedTools,
  }, null, 2);
  const canSubmit = selectedInstances.length > 0 && selectedTools.length > 0 && (!preset.credentialRequired || credential.length > 0);
  return <section className="runtime-mcp-composer" aria-label={copy.management.addMCP}>
    <header><div><h3>{copy.management.addMCP}</h3><p>{copy.management.addMCPDescription}</p></div><button type="button" onClick={onCancel} aria-label={copy.management.cancelAddMCP}>×</button></header>
    <div className="runtime-mcp-preset-types" role="list" aria-label={copy.management.mcpPresetType}>
      {presets.map((item) => <button key={item.id} type="button" className={item.id === preset.id ? "is-selected" : undefined} onClick={() => onSelectPreset(item)}>{item.displayName}</button>)}
    </div>
    <div className="runtime-mcp-fields">
      <label><span>{copy.management.mcpUniqueId}</span><input value={preset.id} readOnly /></label>
      <label><span>{copy.management.mcpDisplayName}</span><input value={preset.displayName} readOnly /></label>
      <label className="runtime-mcp-field-wide"><span>{copy.management.mcpDefinitionDescription}</span><textarea value={preset.description} readOnly rows={2} /></label>
      <label><span>{copy.management.mcpHomepage}</span><input value={preset.homepageUrl} readOnly /></label>
      <label><span>{copy.management.mcpDocumentation}</span><input value={preset.documentationUrl} readOnly /></label>
    </div>
    <fieldset className="runtime-assignment-selector runtime-mcp-bindings-field"><legend>{copy.management.mcpBindingInstances}</legend><p>{copy.management.mcpBindingInstancesDescription}</p><div>{instances.map((instance) => <label key={instance.instanceId}><input type="checkbox" checked={selectedInstances.includes(instance.instanceId)} onChange={() => onToggleInstance(instance.instanceId)} disabled={instance.availability !== "AVAILABLE" || !instance.capabilities.mcpMutate || busy} />{instance.name}</label>)}</div></fieldset>
    {preset.credentialRequired ? <label className="runtime-mcp-secret"><span>{copy.management.mcpCredential}</span><input type="password" value={credential} onChange={(event) => onCredential(preset.id, event.target.value)} autoComplete="off" disabled={busy} /></label> : null}
    <fieldset className="runtime-mcp-tools"><legend>{copy.management.mcpToolScope}</legend><div>{preset.allowedToolIds.map((toolId) => <label key={toolId}><input type="checkbox" checked={selectedTools.includes(toolId)} onChange={() => onToggleTool(preset.id, toolId)} disabled={busy} /><code>{toolId}</code></label>)}</div></fieldset>
    <div className="runtime-mcp-preview"><div><strong>{copy.management.mcpConfigPreview}</strong><span>{copy.management.mcpConfigPreviewDescription}</span></div><pre>{preview}</pre></div>
    <footer><Button onClick={onCancel} disabled={busy}>{copy.management.cancelAddMCP}</Button><Button className="button-primary" onClick={() => onSubmit(preset.id)} disabled={busy || !canSubmit}>{copy.management.testAndAddMCP}</Button></footer>
  </section>;
}

function MCPServerCard({ server, preset, assignedInstances, copy, locale, mutable, testable, busy, credential, selectedTools, onCredential, onToggleTool, onAction }: {
  server: MCPServer;
  preset?: MCPPreset;
  assignedInstances: string[];
  copy: AppMessages;
  locale: Locale;
  mutable: boolean;
  testable: boolean;
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
        <div className="management-badges">
          <Badge tone={server.ownership === "YORVA_MANAGED" ? "info" : "neutral"}>{copy.management.ownershipState[server.ownership]}</Badge>
          <Badge tone={mcpTone(server)}>{copy.management.mcpState[server.state]}</Badge>
        </div>
      </div>
      <dl className="management-detail-grid">
        <div><dt>{copy.management.preset}</dt><dd>{server.presetId}</dd></div>
        <div><dt>{copy.management.readyAt}</dt><dd>{server.readyAt ? formatDateTime(server.readyAt, locale) : copy.management.neverReady}</dd></div>
        <div><dt>{copy.management.observedAt}</dt><dd>{formatDateTime(server.observedAt, locale)}</dd></div>
      </dl>
      {assignedInstances.length > 0 ? <p className="management-detail">{copy.management.appliedInstances}: {assignedInstances.join(", ")}</p> : null}
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
              {testable ? <Button disabled={busy} onClick={() => onAction("test")}>{copy.management.testMCP}</Button> : null}
              <Button disabled={busy} onClick={() => onAction("remove")}>{copy.management.removeMCP}</Button>
            </div>
          </div>
        </div>
      ) : <p className="management-detail">{copy.management.externalReadOnly}</p>}
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
