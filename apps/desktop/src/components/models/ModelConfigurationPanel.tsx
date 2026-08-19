import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { DaemonClient } from "../../api/client";
import { YorvaApiError } from "../../api/client";
import type { Instance, ModelProviderPreset, Operation } from "../../api/types";
import { formatDateTime } from "../../formatDateTime";
import type { AppMessages, Locale } from "../../i18n";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";

type ModelConfigurationPanelProps = {
  client: DaemonClient;
  instance: Instance;
  copy: AppMessages;
  locale: Locale;
  onClose: () => void;
};

export function ModelConfigurationPanel({ client, instance, copy, locale, onClose }: ModelConfigurationPanelProps) {
  const queryClient = useQueryClient();
  const [providerPresetId, setProviderPresetId] = useState("");
  const [modelId, setModelId] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const [errorCode, setErrorCode] = useState("");
  const [validationOperationId, setValidationOperationId] = useState<string | null>(null);
  const initializedInstance = useRef("");
  const passwordRef = useRef<HTMLInputElement>(null);
  const available = instance.availability === "AVAILABLE";

  const presetsQuery = useQuery({
    queryKey: ["model-provider-presets", client.scope],
    queryFn: ({ signal }) => client.listModelProviderPresets(signal),
    staleTime: Infinity,
  });
  const configurationQuery = useQuery({
    queryKey: ["model-configuration", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.getModelConfiguration(instance.instanceId, signal),
    enabled: available,
    retry: false,
  });
  const credentialQuery = useQuery({
    queryKey: ["model-credential", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.getModelCredential(instance.instanceId, signal),
    enabled: available,
    retry: false,
  });
  const operationsQuery = useQuery({
    queryKey: ["model-validation-operations", instance.instanceId, client.scope],
    queryFn: ({ signal }) => client.listOperations("instance", instance.instanceId, signal),
    enabled: available,
    retry: false,
  });
  const recoveredValidation = operationsQuery.data?.operations.find(
    (operation) => isActiveModelValidation(operation),
  );
  const followedValidationId = validationOperationId ?? recoveredValidation?.id ?? null;
  const validationOperationQuery = useQuery({
    queryKey: ["model-validation", instance.instanceId, followedValidationId, client.scope],
    queryFn: ({ signal }) => client.getOperation(followedValidationId!, signal),
    enabled: followedValidationId !== null,
    refetchInterval: (query) => isActiveOperation(query.state.data) ? 500 : false,
  });

  useEffect(() => {
    if (!configurationQuery.data || initializedInstance.current === instance.instanceId) return;
    initializedInstance.current = instance.instanceId;
    setProviderPresetId(configurationQuery.data.providerPresetId);
    setModelId(configurationQuery.data.modelId);
  }, [configurationQuery.data, instance.instanceId]);

  useEffect(() => {
    const status = validationOperationQuery.data?.status;
    if (status !== "SUCCEEDED" && status !== "FAILED" && status !== "CANCELLED") return;
    void queryClient.invalidateQueries({ queryKey: ["model-configuration", instance.instanceId] });
    void queryClient.invalidateQueries({ queryKey: ["model-validation-operations", instance.instanceId] });
  }, [instance.instanceId, queryClient, validationOperationQuery.data?.status]);

  useEffect(() => () => {
    if (passwordRef.current) passwordRef.current.value = "";
  }, []);

  const selectedPreset = useMemo(
    () => presetsQuery.data?.items.find((preset) => preset.id === providerPresetId),
    [presetsQuery.data?.items, providerPresetId],
  );
  const configured = credentialQuery.data?.configured ?? configurationQuery.data?.credentialConfigured ?? false;
  const validationBusy = isActiveOperation(validationOperationQuery.data) || recoveredValidation !== undefined;
  const disabled = !available || busy;

  const invalidateModelState = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["model-configuration", instance.instanceId] }),
      queryClient.invalidateQueries({ queryKey: ["model-credential", instance.instanceId] }),
    ]);
  };

  const selectPreset = (preset: ModelProviderPreset) => {
    setProviderPresetId(preset.id);
    setModelId(preset.recommendedModels[0] ?? "");
    setApiKey("");
    if (passwordRef.current) passwordRef.current.value = "";
    setNotice("");
  };

  const save = async () => {
    if (disabled || !providerPresetId || !modelId) return;
    setBusy(true);
    setNotice("");
    setErrorCode("");
    const submittedKey = apiKey;
    try {
      if (submittedKey) {
        await client.saveModelCredential(instance.instanceId, providerPresetId, modelId, submittedKey);
      } else {
        await client.patchModelConfiguration(instance.instanceId, providerPresetId, modelId);
      }
      setNotice(copy.models.saved);
      await invalidateModelState();
    } catch (error) {
      setErrorCode(error instanceof YorvaApiError ? error.code : "INTERNAL_ERROR");
    } finally {
      setApiKey("");
      if (passwordRef.current) passwordRef.current.value = "";
      setBusy(false);
    }
  };

  const deleteCredential = async () => {
    if (disabled || !configured) return;
    setBusy(true);
    setNotice("");
    setErrorCode("");
    try {
      await client.deleteModelCredential(instance.instanceId);
      setNotice(copy.models.deleted);
      await invalidateModelState();
    } catch (error) {
      setErrorCode(error instanceof YorvaApiError ? error.code : "INTERNAL_ERROR");
    } finally {
      setApiKey("");
      if (passwordRef.current) passwordRef.current.value = "";
      setBusy(false);
    }
  };

  const startValidation = async () => {
    if (disabled || validationBusy || configurationQuery.data?.state !== "CONFIGURED") return;
    setBusy(true);
    setNotice("");
    setErrorCode("");
    try {
      const operation = await client.startModelValidation(instance.instanceId, crypto.randomUUID());
      setValidationOperationId(operation.id);
    } catch (error) {
      setErrorCode(error instanceof YorvaApiError ? error.code : "INTERNAL_ERROR");
    } finally {
      setBusy(false);
    }
  };

  const cancelValidation = async () => {
    if (!followedValidationId || busy) return;
    setBusy(true);
    try {
      await client.cancelOperation(followedValidationId);
      await validationOperationQuery.refetch();
    } catch (error) {
      setErrorCode(error instanceof YorvaApiError ? error.code : "INTERNAL_ERROR");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card className="model-panel" aria-label={`${copy.models.title}: ${instance.name}`}>
      <div className="panel-heading panel-heading-split">
        <div>
          <h3>{copy.models.title}</h3>
          <p className="panel-copy">{copy.models.description}</p>
        </div>
        <Button variant="ghost" onClick={onClose}>{copy.models.close}</Button>
      </div>
      {!available ? <p role="status" className="notice notice-warn">{copy.models.unavailable}</p> : null}
      {configurationQuery.isPending || presetsQuery.isPending ? <p className="page-copy">{copy.models.loading}</p> : null}
      <fieldset className="model-provider-grid" disabled={disabled || presetsQuery.isPending}>
        <legend>{copy.models.provider}</legend>
        <ProviderGroup
          title={copy.models.china}
          presets={presetsQuery.data?.items.filter((preset) => preset.region === "CHINA") ?? []}
          selected={providerPresetId}
          onSelect={selectPreset}
        />
        <ProviderGroup
          title={copy.models.global}
          presets={presetsQuery.data?.items.filter((preset) => preset.region === "GLOBAL") ?? []}
          selected={providerPresetId}
          onSelect={selectPreset}
        />
      </fieldset>
      {selectedPreset?.helpText ? <p className="notice notice-info">{selectedPreset.helpText}</p> : null}
      <div className="model-form-grid">
        <label htmlFor={`model-id-${instance.instanceId}`}>{copy.models.model}</label>
        <input
          id={`model-id-${instance.instanceId}`}
          className="instance-create-input"
          value={modelId}
          list={`model-options-${instance.instanceId}`}
          onChange={(event) => setModelId(event.target.value)}
          autoComplete="off"
          spellCheck={false}
          disabled={disabled}
        />
        <datalist id={`model-options-${instance.instanceId}`}>
          {selectedPreset?.recommendedModels.map((model) => <option key={model} value={model} />)}
        </datalist>
        <p className="page-copy">{copy.models.modelHint}</p>
        <label htmlFor={`model-key-${instance.instanceId}`}>{copy.models.apiKey}</label>
        <input
          ref={passwordRef}
          id={`model-key-${instance.instanceId}`}
          className="instance-create-input"
          type="password"
          value={apiKey}
          onChange={(event) => setApiKey(event.target.value)}
          placeholder={copy.models.apiKeyPlaceholder}
          autoComplete="new-password"
          spellCheck={false}
          disabled={disabled}
        />
      </div>
      <div className="model-status-row" role="status">
        <Badge tone={configurationQuery.data?.state === "CONFIGURED" ? "ok" : "warn"}>
          {copy.models.configState[configurationQuery.data?.state ?? "UNCONFIGURED"]}
        </Badge>
        <span>{configured ? copy.models.credentialConfigured : copy.models.credentialMissing}</span>
        <Badge tone={validationTone(configurationQuery.data?.validation.state)}>
          {copy.models.validationState[configurationQuery.data?.validation.state ?? "NOT_RUN"]}
        </Badge>
      </div>
      {configurationQuery.data?.observedAt ? (
        <p className="page-copy">{copy.models.observedAt}: {formatDateTime(configurationQuery.data.observedAt, locale)}</p>
      ) : null}
      {configurationQuery.data?.validation.completedAt ? (
        <p className="page-copy">{copy.models.validationAt}: {formatDateTime(configurationQuery.data.validation.completedAt, locale)}</p>
      ) : null}
      {configurationQuery.data?.validation.errorCode ? (
        <p className="notice notice-warn">
          {copy.models.errorCode}: {configurationQuery.data.validation.errorCode}. {copy.models.validationAdvice}
        </p>
      ) : null}
      <div className="inline-actions">
        <Button variant="primary" onClick={() => { void save(); }} disabled={disabled || !providerPresetId || !modelId}>
          {busy ? copy.models.saving : copy.models.save}
        </Button>
        <Button onClick={() => { void startValidation(); }} disabled={disabled || validationBusy || configurationQuery.data?.state !== "CONFIGURED"}>
          {validationBusy ? copy.models.testing : copy.models.testConnection}
        </Button>
        {validationBusy ? <Button onClick={() => { void cancelValidation(); }} disabled={busy}>{copy.models.cancelTest}</Button> : null}
        {configured ? <Button variant="danger" onClick={() => { void deleteCredential(); }} disabled={disabled}>{copy.models.deleteCredential}</Button> : null}
      </div>
      {notice ? <p className="notice notice-info" role="status">{notice}</p> : null}
      {errorCode ? <p className="notice notice-error" role="alert">{copy.models.requestFailed} {copy.models.errorCode}: {errorCode}</p> : null}
    </Card>
  );
}

function ProviderGroup({ title, presets, selected, onSelect }: {
  title: string;
  presets: ModelProviderPreset[];
  selected: string;
  onSelect: (preset: ModelProviderPreset) => void;
}) {
  return (
    <div className="model-provider-group">
      <h4>{title}</h4>
      <div className="model-provider-options">
        {presets.map((preset) => (
          <label key={preset.id} className={selected === preset.id ? "model-provider-option is-active" : "model-provider-option"}>
            <input type="radio" name="model-provider" value={preset.id} checked={selected === preset.id} onChange={() => onSelect(preset)} />
            <span>{preset.displayName}</span>
          </label>
        ))}
      </div>
    </div>
  );
}

function isActiveModelValidation(operation: Operation): boolean {
  return operation.type === "model.validate" && isActiveOperation(operation);
}

function isActiveOperation(operation?: Operation): boolean {
  return operation?.status === "PENDING" || operation?.status === "RUNNING";
}

function validationTone(state?: "NOT_RUN" | "PASSED" | "FAILED" | "UNKNOWN") {
  if (state === "PASSED") return "ok" as const;
  if (state === "FAILED") return "error" as const;
  if (state === "UNKNOWN") return "warn" as const;
  return "neutral" as const;
}
