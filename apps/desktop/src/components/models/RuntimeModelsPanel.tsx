import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { DaemonClient } from "../../api/client";
import type { Instance } from "../../api/types";
import type { AppMessages } from "../../i18n";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { IconChevronRight, IconPlus } from "../ui/icons";

type RuntimeModelPage = "overview" | "provider-connection" | "model-profile" | "instance-bindings";

export function RuntimeModelsPanel({ client, runtimeId, instances, copy }: { client: DaemonClient; runtimeId: string; instances: Instance[]; copy: AppMessages }) {
  const queryClient = useQueryClient();
  const credentialRef = useRef<HTMLInputElement>(null);
  const [page, setPage] = useState<RuntimeModelPage>("overview");
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

  const presets = useQuery({ queryKey: ["model-provider-presets", runtimeId, client.scope], queryFn: ({ signal }) => client.listRuntimeModelProviderPresets(runtimeId, signal), staleTime: Infinity });
  const connections = useQuery({ queryKey: ["runtime-model-connections", runtimeId, client.scope], queryFn: ({ signal }) => client.listRuntimeModelProviderConnections(runtimeId, signal), retry: false });
  const profiles = useQuery({ queryKey: ["runtime-model-profiles", runtimeId, client.scope], queryFn: ({ signal }) => client.listRuntimeModelProfiles(runtimeId, signal), retry: false });
  const runtimeDefault = useQuery({ queryKey: ["runtime-model-default", runtimeId, client.scope], queryFn: ({ signal }) => client.getRuntimeModelDefault(runtimeId, signal), retry: false });
  const bindings = useQuery({ queryKey: ["runtime-model-bindings", runtimeId, client.scope], queryFn: ({ signal }) => client.listRuntimeModelBindings(runtimeId, signal), retry: false });
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

  const createConnection = useMutation({
    mutationFn: () => client.createRuntimeModelProviderConnection(runtimeId, providerPresetId, connectionName, credential),
    onSuccess: async (created) => {
      setCredential("");
      if (credentialRef.current) credentialRef.current.value = "";
      setConnectionName("");
      setProfileConnectionId(created.id);
      await invalidate();
      setPage("overview");
    },
  });
  const deleteConnection = useMutation({ mutationFn: (id: string) => client.deleteRuntimeModelProviderConnection(runtimeId, id), onSuccess: invalidate });
  const createProfile = useMutation({
    mutationFn: () => client.createRuntimeModelProfile(runtimeId, profileConnectionId, profileName, selectedModelIds, defaultModelId),
    onSuccess: async (created) => {
      setProfileName(""); setSelectedModelIds([]); setDefaultModelId(""); setApplyProfileId(created.id);
      await invalidate();
      setPage("overview");
    },
  });
  const deleteProfile = useMutation({ mutationFn: (id: string) => client.deleteRuntimeModelProfile(runtimeId, id), onSuccess: invalidate });
  const setDefault = useMutation({ mutationFn: (id: string) => client.setRuntimeModelDefault(runtimeId, id), onSuccess: invalidate });
  const clearDefault = useMutation({ mutationFn: () => client.clearRuntimeModelDefault(runtimeId), onSuccess: invalidate });
  const applyProfile = useMutation({
    mutationFn: () => client.applyRuntimeModelProfile(runtimeId, effectiveApplyProfileId, applyInstanceIds, applyMode, crypto.randomUUID()),
    onSuccess: (accepted) => setOperationId(accepted.id),
  });

  useEffect(() => {
    if (operation.data?.status === "SUCCEEDED") {
      void invalidate().then(() => setPage("overview"));
    } else if (operation.data?.status === "FAILED" || operation.data?.status === "CANCELLED") {
      void invalidate();
    }
  }, [invalidate, operation.data?.status]);

  const selectedConnection = connections.data?.items.find((item) => item.id === profileConnectionId);
  const selectedPreset = presets.data?.items.find((item) => item.id === selectedConnection?.providerPresetId);
  const modelOptions = selectedPreset?.recommendedModels ?? [];
  const busy = createConnection.isPending || deleteConnection.isPending || createProfile.isPending || deleteProfile.isPending || setDefault.isPending || clearDefault.isPending || applyProfile.isPending || operation.data?.status === "PENDING" || operation.data?.status === "RUNNING";
  const failed = createConnection.isError || deleteConnection.isError || createProfile.isError || deleteProfile.isError || setDefault.isError || clearDefault.isError || applyProfile.isError || operation.data?.status === "FAILED" || operation.data?.status === "CANCELLED";
  const loading = presets.isLoading || connections.isLoading || profiles.isLoading || runtimeDefault.isLoading || bindings.isLoading;
  const queryFailed = presets.isError || connections.isError || profiles.isError || runtimeDefault.isError || bindings.isError;
  const effectiveApplyProfileId = applyMode === "INHERIT" ? runtimeDefault.data?.modelProfileId ?? "" : applyProfileId;
  const profileNames = useMemo(() => new Map(profiles.data?.items.map((item) => [item.id, item.displayName]) ?? []), [profiles.data?.items]);

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
  const openPage = (nextPage: RuntimeModelPage) => {
    createConnection.reset();
    createProfile.reset();
    applyProfile.reset();
    setOperationId(null);
    setPage(nextPage);
  };
  const submit = (event: FormEvent, action: () => void) => {
    event.preventDefault();
    action();
  };

  return (
    <div className="runtime-models-page">
      {page === "overview" ? <div className="runtime-models-intro"><div><h3>{copy.management.sharedModelsTitle}</h3><p>{copy.management.sharedModelsDescription}</p></div></div> : null}
      {loading ? <p className="management-empty">{copy.management.loading}</p> : null}
      {queryFailed ? <p className="notice notice-warn">{copy.management.requestFailed}</p> : null}
      {failed ? <p className="notice notice-warn">{copy.management.sharedModelsMutationFailed}</p> : null}

      {page === "overview" ? (
        <>
          <ModelResourceGroup
            title={copy.management.providerConnectionsTitle}
            description={copy.management.providerConnectionsDescription}
            count={connections.data?.items.length ?? 0}
            action={<Button className="button-compact" disabled={loading || queryFailed} onClick={() => openPage("provider-connection")}><IconPlus />{copy.management.addProviderConnection}</Button>}
          >
            {connections.data?.items.length === 0 ? <p className="management-empty">{copy.management.noProviderConnections}</p> : <div className="runtime-model-resource-list">{connections.data?.items.map((item) => <article key={item.id}><div><strong>{item.displayName}</strong><span>{presets.data?.items.find((preset) => preset.id === item.providerPresetId)?.displayName ?? item.providerPresetId}</span></div><div><Badge tone={item.credentialConfigured ? "ok" : "warn"}>{item.credentialConfigured ? copy.models.credentialConfigured : copy.models.credentialMissing}</Badge><Button disabled={busy} onClick={() => deleteConnection.mutate(item.id)}>{copy.management.removeResource}</Button></div></article>)}</div>}
          </ModelResourceGroup>

          <ModelResourceGroup
            title={copy.management.modelProfilesTitle}
            description={copy.management.modelProfilesDescription}
            count={profiles.data?.items.length ?? 0}
            action={<Button className="button-compact" disabled={busy || (connections.data?.items.length ?? 0) === 0} onClick={() => openPage("model-profile")}><IconPlus />{copy.management.addModelProfile}</Button>}
          >
            {profiles.data?.items.length === 0 ? <p className="management-empty">{copy.management.noModelProfiles}</p> : <div className="runtime-model-resource-list">{profiles.data?.items.map((item) => <article key={item.id}><div><strong>{item.displayName}</strong><span>{item.defaultModelId} · {item.selectedModelIds.length} {copy.management.profileModelCount}</span></div><div>{runtimeDefault.data?.modelProfileId === item.id ? <Badge tone="info">{copy.management.runtimeDefaultBadge}</Badge> : null}<Button disabled={busy} onClick={() => deleteProfile.mutate(item.id)}>{copy.management.removeResource}</Button></div></article>)}</div>}
          </ModelResourceGroup>

          <section className="runtime-model-group runtime-model-setting-group">
            <header><div><h3>{copy.management.runtimeModelDefaultTitle}</h3><p>{copy.management.runtimeModelDefaultDescription}</p></div></header>
            <div className="runtime-model-inline"><select aria-label={copy.management.runtimeModelDefaultTitle} value={runtimeDefault.data?.modelProfileId ?? ""} onChange={(event) => event.target.value ? setDefault.mutate(event.target.value) : clearDefault.mutate()} disabled={busy}><option value="">{copy.management.noRuntimeModelDefault}</option>{profiles.data?.items.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}</select>{runtimeDefault.data?.modelProfileId ? <Button disabled={busy} onClick={() => clearDefault.mutate()}>{copy.management.clearRuntimeDefault}</Button> : null}</div>
          </section>

          <ModelResourceGroup
            title={copy.management.modelBindingsTitle}
            description={copy.management.modelBindingsDescription}
            action={<Button className="button-compact" disabled={busy || (profiles.data?.items.length ?? 0) === 0} onClick={() => openPage("instance-bindings")}>{copy.management.configureModelBindings}<IconChevronRight /></Button>}
          >
            {bindings.data?.items.length === 0 ? <p className="management-empty">{copy.management.noModelBindings}</p> : <div className="runtime-model-binding-list">{bindings.data?.items.map((item) => <article key={item.instanceId}><div><strong>{item.instanceName}</strong><span>{item.mode === "EXTERNAL_CONFIGURATION" ? copy.management.modelBindingMode.EXTERNAL_CONFIGURATION : profileNames.get(item.modelProfileId) ?? item.modelProfileId}</span></div><div><Badge tone={item.state === "SUCCEEDED" ? "ok" : item.state === "PENDING" ? "neutral" : "warn"}>{copy.management.modelBindingState[item.state]}</Badge><span>{copy.management.modelBindingMode[item.mode]}</span></div></article>)}</div>}
          </ModelResourceGroup>
        </>
      ) : (
        <section className="runtime-model-config-page" aria-labelledby="runtime-model-config-title">
          <header className="runtime-model-config-header">
            <Button variant="ghost" className="runtime-model-config-back" onClick={() => openPage("overview")}>← {copy.management.backToSharedModels}</Button>
            <div>
              <h3 id="runtime-model-config-title">{page === "provider-connection" ? copy.management.addProviderConnection : page === "model-profile" ? copy.management.addModelProfile : copy.management.configureModelBindings}</h3>
              <p>{page === "provider-connection" ? copy.management.providerConnectionConfigurationDescription : page === "model-profile" ? copy.management.modelProfileConfigurationDescription : copy.management.modelBindingConfigurationDescription}</p>
            </div>
          </header>

          {page === "provider-connection" ? (
            <form className="runtime-model-config-form" onSubmit={(event) => submit(event, () => createConnection.mutate())}>
              <div className="runtime-model-form-grid">
                <label><span>{copy.models.provider}</span><select value={providerPresetId} onChange={(event) => setProviderPresetId(event.target.value)}><option value="">—</option>{presets.data?.items.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}</select></label>
                <label><span>{copy.management.resourceDisplayName}</span><input value={connectionName} maxLength={80} onChange={(event) => setConnectionName(event.target.value)} /></label>
                <label className="runtime-model-config-wide"><span>{copy.models.apiKey}</span><input ref={credentialRef} type="password" value={credential} onChange={(event) => setCredential(event.target.value)} autoComplete="new-password" /></label>
              </div>
              <footer><Button onClick={() => openPage("overview")} disabled={busy}>{copy.management.cancelModelConfiguration}</Button><Button type="submit" className="button-primary" disabled={busy || !providerPresetId || !connectionName.trim() || !credential}>{copy.management.addProviderConnection}</Button></footer>
            </form>
          ) : page === "model-profile" ? (
            <form className="runtime-model-config-form" onSubmit={(event) => submit(event, () => createProfile.mutate())}>
              <div className="runtime-model-form-grid">
                <label><span>{copy.management.providerConnection}</span><select value={profileConnectionId} onChange={(event) => { setProfileConnectionId(event.target.value); setSelectedModelIds([]); setDefaultModelId(""); }}><option value="">—</option>{connections.data?.items.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}</select></label>
                <label><span>{copy.management.resourceDisplayName}</span><input value={profileName} maxLength={80} onChange={(event) => setProfileName(event.target.value)} /></label>
                <fieldset className="runtime-model-options runtime-model-config-wide"><legend>{copy.management.profileModels}</legend>{modelOptions.map((modelId) => <label key={modelId}><input type="checkbox" checked={selectedModelIds.includes(modelId)} onChange={() => toggleModel(modelId)} /><span>{modelId}</span><input type="radio" name="runtime-default-model" aria-label={`${modelId}: ${copy.models.defaultModel}`} disabled={!selectedModelIds.includes(modelId)} checked={defaultModelId === modelId} onChange={() => setDefaultModelId(modelId)} /></label>)}</fieldset>
              </div>
              <footer><Button onClick={() => openPage("overview")} disabled={busy}>{copy.management.cancelModelConfiguration}</Button><Button type="submit" className="button-primary" disabled={busy || !profileConnectionId || !profileName.trim() || selectedModelIds.length === 0 || !defaultModelId}>{copy.management.addModelProfile}</Button></footer>
            </form>
          ) : (
            <form className="runtime-model-config-form" onSubmit={(event) => submit(event, () => applyProfile.mutate())}>
              <div className="runtime-model-form-grid">
                <label className="runtime-model-config-wide"><span>{copy.management.modelProfile}</span><select value={effectiveApplyProfileId} disabled={busy || applyMode === "INHERIT"} onChange={(event) => setApplyProfileId(event.target.value)}><option value="">—</option>{profiles.data?.items.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}</select></label>
                <fieldset className="runtime-model-config-choice runtime-model-config-wide"><legend>{copy.management.assignmentInstances}</legend><div>{instances.filter((item) => item.availability === "AVAILABLE").map((item) => <label key={item.instanceId}><input type="checkbox" checked={applyInstanceIds.includes(item.instanceId)} onChange={() => toggleInstance(item.instanceId)} /><span>{item.name}</span></label>)}</div></fieldset>
                <fieldset className="runtime-model-config-choice runtime-model-config-wide"><legend>{copy.management.bindingMode}</legend><div><label><input type="radio" name="model-binding-mode" checked={applyMode === "INHERIT"} onChange={() => setApplyMode("INHERIT")} />{copy.management.modelBindingMode.INHERIT}</label><label><input type="radio" name="model-binding-mode" checked={applyMode === "OVERRIDE"} onChange={() => setApplyMode("OVERRIDE")} />{copy.management.modelBindingMode.OVERRIDE}</label></div></fieldset>
              </div>
              <footer><Button onClick={() => openPage("overview")} disabled={busy}>{copy.management.cancelModelConfiguration}</Button><Button type="submit" className="button-primary" disabled={busy || !effectiveApplyProfileId || applyInstanceIds.length === 0}>{busy ? copy.management.modelApplicationRunning : copy.management.applyModelProfile}</Button></footer>
            </form>
          )}
        </section>
      )}
    </div>
  );
}

function ModelResourceGroup({ title, description, count, action, children }: { title: string; description: string; count?: number; action: ReactNode; children: ReactNode }) {
  return (
    <section className="runtime-model-group">
      <header>
        <div><h3>{title}</h3><p>{description}</p></div>
        <div className="runtime-model-group-actions">{count !== undefined ? <Badge tone="neutral">{count}</Badge> : null}{action}</div>
      </header>
      {children}
    </section>
  );
}
