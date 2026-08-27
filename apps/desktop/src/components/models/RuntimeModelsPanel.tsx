import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { DaemonClient } from "../../api/client";
import type { Instance } from "../../api/types";
import type { AppMessages } from "../../i18n";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { IconPlus } from "../ui/icons";

export function RuntimeModelsPanel({ client, instances, copy }: { client: DaemonClient; instances: Instance[]; copy: AppMessages }) {
  const queryClient = useQueryClient();
  const credentialRef = useRef<HTMLInputElement>(null);
  const [providerPresetId, setProviderPresetId] = useState("");
  const [connectionName, setConnectionName] = useState("");
  const [credential, setCredential] = useState("");
  const [profileConnectionId, setProfileConnectionId] = useState("");
  const [profileName, setProfileName] = useState("");
  const [selectedModelIds, setSelectedModelIds] = useState<string[]>([]);
  const [defaultModelId, setDefaultModelId] = useState("");
  const [applyProfileId, setApplyProfileId] = useState("");
  const [applyInstanceIds, setApplyInstanceIds] = useState<string[]>(instances.filter((item) => item.availability === "AVAILABLE").map((item) => item.instanceId));
  const [applyMode, setApplyMode] = useState<"INHERIT" | "OVERRIDE">("INHERIT");
  const [operationId, setOperationId] = useState<string | null>(null);

  const presets = useQuery({ queryKey: ["model-provider-presets", client.scope], queryFn: ({ signal }) => client.listModelProviderPresets(signal), staleTime: Infinity });
  const connections = useQuery({ queryKey: ["runtime-model-connections", "hermes", client.scope], queryFn: ({ signal }) => client.listRuntimeModelProviderConnections("hermes", signal), retry: false });
  const profiles = useQuery({ queryKey: ["runtime-model-profiles", "hermes", client.scope], queryFn: ({ signal }) => client.listRuntimeModelProfiles("hermes", signal), retry: false });
  const runtimeDefault = useQuery({ queryKey: ["runtime-model-default", "hermes", client.scope], queryFn: ({ signal }) => client.getRuntimeModelDefault("hermes", signal), retry: false });
  const bindings = useQuery({ queryKey: ["runtime-model-bindings", "hermes", client.scope], queryFn: ({ signal }) => client.listRuntimeModelBindings("hermes", signal), retry: false });
  const operation = useQuery({
    queryKey: ["runtime-model-application", operationId, client.scope],
    queryFn: ({ signal }) => client.getOperation(operationId!, signal), enabled: operationId !== null, retry: false,
    refetchInterval: (query) => query.state.data?.status === "PENDING" || query.state.data?.status === "RUNNING" ? 700 : false,
  });

  const invalidate = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["runtime-model-connections"] }),
      queryClient.invalidateQueries({ queryKey: ["runtime-model-profiles"] }),
      queryClient.invalidateQueries({ queryKey: ["runtime-model-default"] }),
      queryClient.invalidateQueries({ queryKey: ["runtime-model-bindings"] }),
      queryClient.invalidateQueries({ queryKey: ["model-configuration"] }),
      queryClient.invalidateQueries({ queryKey: ["model-credential"] }),
    ]);
  }, [queryClient]);

  useEffect(() => {
    if (operation.data?.status === "SUCCEEDED" || operation.data?.status === "FAILED" || operation.data?.status === "CANCELLED") void invalidate();
  }, [invalidate, operation.data?.status]);

  const createConnection = useMutation({
    mutationFn: () => client.createRuntimeModelProviderConnection("hermes", providerPresetId, connectionName, credential),
    onSuccess: async (created) => {
      setCredential("");
      if (credentialRef.current) credentialRef.current.value = "";
      setConnectionName("");
      setProfileConnectionId(created.id);
      await invalidate();
    },
  });
  const deleteConnection = useMutation({ mutationFn: (id: string) => client.deleteRuntimeModelProviderConnection("hermes", id), onSuccess: invalidate });
  const createProfile = useMutation({
    mutationFn: () => client.createRuntimeModelProfile("hermes", profileConnectionId, profileName, selectedModelIds, defaultModelId),
    onSuccess: async (created) => {
      setProfileName(""); setSelectedModelIds([]); setDefaultModelId(""); setApplyProfileId(created.id);
      await invalidate();
    },
  });
  const deleteProfile = useMutation({ mutationFn: (id: string) => client.deleteRuntimeModelProfile("hermes", id), onSuccess: invalidate });
  const setDefault = useMutation({ mutationFn: (id: string) => client.setRuntimeModelDefault("hermes", id), onSuccess: invalidate });
  const clearDefault = useMutation({ mutationFn: () => client.clearRuntimeModelDefault("hermes"), onSuccess: invalidate });
  const applyProfile = useMutation({
    mutationFn: () => client.applyRuntimeModelProfile("hermes", effectiveApplyProfileId, applyInstanceIds, applyMode, crypto.randomUUID()),
    onSuccess: (accepted) => setOperationId(accepted.id),
  });

  const selectedConnection = connections.data?.items.find((item) => item.id === profileConnectionId);
  const selectedPreset = presets.data?.items.find((item) => item.id === selectedConnection?.providerPresetId);
  const modelOptions = selectedPreset?.recommendedModels ?? [];
  const busy = createConnection.isPending || deleteConnection.isPending || createProfile.isPending || deleteProfile.isPending || setDefault.isPending || clearDefault.isPending || applyProfile.isPending || operation.data?.status === "PENDING" || operation.data?.status === "RUNNING";
  const failed = createConnection.isError || deleteConnection.isError || createProfile.isError || deleteProfile.isError || setDefault.isError || clearDefault.isError || applyProfile.isError || operation.data?.status === "FAILED" || operation.data?.status === "CANCELLED";
  const loading = presets.isLoading || connections.isLoading || profiles.isLoading || runtimeDefault.isLoading || bindings.isLoading;
  const queryFailed = presets.isError || connections.isError || profiles.isError || runtimeDefault.isError || bindings.isError;
  const effectiveApplyProfileId = applyMode === "INHERIT" ? runtimeDefault.data?.modelProfileId ?? "" : applyProfileId;

  const toggleModel = (modelId: string) => {
    setSelectedModelIds((current) => {
      if (current.includes(modelId)) {
        const next = current.filter((item) => item !== modelId);
        if (defaultModelId === modelId) setDefaultModelId(next[0] ?? "");
        return next;
      }
      if (!defaultModelId) setDefaultModelId(modelId);
      return [...current, modelId];
    });
  };
  const toggleInstance = (instanceId: string) => setApplyInstanceIds((current) => current.includes(instanceId) ? current.filter((id) => id !== instanceId) : [...current, instanceId]);
  const profileNames = useMemo(() => new Map(profiles.data?.items.map((item) => [item.id, item.displayName]) ?? []), [profiles.data?.items]);

  return (
    <div className="runtime-models-page">
      <div className="runtime-models-intro"><div><h3>{copy.management.sharedModelsTitle}</h3><p>{copy.management.sharedModelsDescription}</p></div></div>
      {loading ? <p className="management-empty">{copy.management.loading}</p> : null}
      {queryFailed ? <p className="notice notice-warn">{copy.management.requestFailed}</p> : null}
      {failed ? <p className="notice notice-warn">{copy.management.sharedModelsMutationFailed}</p> : null}

      <section className="runtime-model-group">
        <header><div><h3>{copy.management.providerConnectionsTitle}</h3><p>{copy.management.providerConnectionsDescription}</p></div><Badge tone="neutral">{connections.data?.items.length ?? 0}</Badge></header>
        <div className="runtime-model-form runtime-model-connection-form">
          <label><span>{copy.models.provider}</span><select value={providerPresetId} onChange={(event) => setProviderPresetId(event.target.value)}><option value="">—</option>{presets.data?.items.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}</select></label>
          <label><span>{copy.management.resourceDisplayName}</span><input value={connectionName} maxLength={80} onChange={(event) => setConnectionName(event.target.value)} /></label>
          <label><span>{copy.models.apiKey}</span><input ref={credentialRef} type="password" value={credential} onChange={(event) => setCredential(event.target.value)} autoComplete="new-password" /></label>
          <Button className="button-primary" disabled={busy || !providerPresetId || !connectionName.trim() || !credential} onClick={() => createConnection.mutate()}><IconPlus />{copy.management.addProviderConnection}</Button>
        </div>
        {connections.data?.items.length === 0 ? <p className="management-empty">{copy.management.noProviderConnections}</p> : <div className="runtime-model-resource-list">{connections.data?.items.map((item) => <article key={item.id}><div><strong>{item.displayName}</strong><span>{presets.data?.items.find((preset) => preset.id === item.providerPresetId)?.displayName ?? item.providerPresetId}</span></div><div><Badge tone={item.credentialConfigured ? "ok" : "warn"}>{item.credentialConfigured ? copy.models.credentialConfigured : copy.models.credentialMissing}</Badge><Button disabled={busy} onClick={() => deleteConnection.mutate(item.id)}>{copy.management.removeResource}</Button></div></article>)}</div>}
      </section>

      <section className="runtime-model-group">
        <header><div><h3>{copy.management.modelProfilesTitle}</h3><p>{copy.management.modelProfilesDescription}</p></div><Badge tone="neutral">{profiles.data?.items.length ?? 0}</Badge></header>
        <div className="runtime-model-form">
          <label><span>{copy.management.providerConnection}</span><select value={profileConnectionId} onChange={(event) => { setProfileConnectionId(event.target.value); setSelectedModelIds([]); setDefaultModelId(""); }}><option value="">—</option>{connections.data?.items.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}</select></label>
          <label><span>{copy.management.resourceDisplayName}</span><input value={profileName} maxLength={80} onChange={(event) => setProfileName(event.target.value)} /></label>
          <fieldset className="runtime-model-options"><legend>{copy.management.profileModels}</legend>{modelOptions.map((modelId) => <label key={modelId}><input type="checkbox" checked={selectedModelIds.includes(modelId)} onChange={() => toggleModel(modelId)} /><span>{modelId}</span><input type="radio" name="runtime-default-model" aria-label={`${modelId}: ${copy.models.defaultModel}`} disabled={!selectedModelIds.includes(modelId)} checked={defaultModelId === modelId} onChange={() => setDefaultModelId(modelId)} /></label>)}</fieldset>
          <Button className="button-primary" disabled={busy || !profileConnectionId || !profileName.trim() || selectedModelIds.length === 0 || !defaultModelId} onClick={() => createProfile.mutate()}><IconPlus />{copy.management.addModelProfile}</Button>
        </div>
        {profiles.data?.items.length === 0 ? <p className="management-empty">{copy.management.noModelProfiles}</p> : <div className="runtime-model-resource-list">{profiles.data?.items.map((item) => <article key={item.id}><div><strong>{item.displayName}</strong><span>{item.defaultModelId} · {item.selectedModelIds.length} {copy.management.profileModelCount}</span></div><div>{runtimeDefault.data?.modelProfileId === item.id ? <Badge tone="info">{copy.management.runtimeDefaultBadge}</Badge> : null}<Button disabled={busy} onClick={() => deleteProfile.mutate(item.id)}>{copy.management.removeResource}</Button></div></article>)}</div>}
      </section>

      <section className="runtime-model-group">
        <header><div><h3>{copy.management.runtimeModelDefaultTitle}</h3><p>{copy.management.runtimeModelDefaultDescription}</p></div></header>
        <div className="runtime-model-inline"><select aria-label={copy.management.runtimeModelDefaultTitle} value={runtimeDefault.data?.modelProfileId ?? ""} onChange={(event) => event.target.value ? setDefault.mutate(event.target.value) : clearDefault.mutate()} disabled={busy}><option value="">{copy.management.noRuntimeModelDefault}</option>{profiles.data?.items.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}</select>{runtimeDefault.data?.modelProfileId ? <Button disabled={busy} onClick={() => clearDefault.mutate()}>{copy.management.clearRuntimeDefault}</Button> : null}</div>
      </section>

      <section className="runtime-model-group">
        <header><div><h3>{copy.management.modelBindingsTitle}</h3><p>{copy.management.modelBindingsDescription}</p></div></header>
        <div className="runtime-model-apply">
          <label><span>{copy.management.modelProfile}</span><select value={effectiveApplyProfileId} disabled={busy || applyMode === "INHERIT"} onChange={(event) => setApplyProfileId(event.target.value)}><option value="">—</option>{profiles.data?.items.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}</select></label>
          <fieldset><legend>{copy.management.assignmentInstances}</legend><div>{instances.filter((item) => item.availability === "AVAILABLE").map((item) => <label key={item.instanceId}><input type="checkbox" checked={applyInstanceIds.includes(item.instanceId)} onChange={() => toggleInstance(item.instanceId)} /><span>{item.name}</span></label>)}</div></fieldset>
          <fieldset><legend>{copy.management.bindingMode}</legend><div><label><input type="radio" name="model-binding-mode" checked={applyMode === "INHERIT"} onChange={() => setApplyMode("INHERIT")} />{copy.management.modelBindingMode.INHERIT}</label><label><input type="radio" name="model-binding-mode" checked={applyMode === "OVERRIDE"} onChange={() => setApplyMode("OVERRIDE")} />{copy.management.modelBindingMode.OVERRIDE}</label></div></fieldset>
          <Button className="button-primary" disabled={busy || !effectiveApplyProfileId || applyInstanceIds.length === 0} onClick={() => applyProfile.mutate()}>{busy ? copy.management.modelApplicationRunning : copy.management.applyModelProfile}</Button>
        </div>
        {bindings.data?.items.length === 0 ? <p className="management-empty">{copy.management.noModelBindings}</p> : <div className="runtime-model-binding-list">{bindings.data?.items.map((item) => <article key={item.instanceId}><div><strong>{item.instanceName}</strong><span>{item.mode === "EXTERNAL_CONFIGURATION" ? copy.management.modelBindingMode.EXTERNAL_CONFIGURATION : profileNames.get(item.modelProfileId) ?? item.modelProfileId}</span></div><div><Badge tone={item.state === "SUCCEEDED" ? "ok" : item.state === "PENDING" ? "neutral" : "warn"}>{copy.management.modelBindingState[item.state]}</Badge><span>{copy.management.modelBindingMode[item.mode]}</span></div></article>)}</div>}
      </section>
    </div>
  );
}
