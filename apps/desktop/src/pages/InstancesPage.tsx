import { useEffect, useRef, useState } from "react";
import { formatDateTime } from "../formatDateTime";
import type { DaemonClient } from "../api/client";
import type { Instance, InstanceList, Operation } from "../api/types";
import type { AppMessages, Locale } from "../i18n";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { IconRotateCw } from "../components/ui/icons";
import { ModelConfigurationPanel } from "../components/models/ModelConfigurationPanel";

const createNamePattern = /^[a-z][a-z0-9_-]{0,63}$/;

type InstancesPageProps = {
  supported: boolean;
  loading: boolean;
  error: boolean;
  inventory: InstanceList | null;
  createName: string;
  createBusy: boolean;
  createOperation: Operation | null;
  copy: AppMessages;
  locale: Locale;
  onRefresh: () => void;
  onCreateNameChange: (value: string) => void;
  onCreate: () => void;
  onCancelCreate: () => void;
  deleteTarget: Instance | null;
  deleteConfirmation: string;
  deleteBusy: boolean;
  deleteOperation: Operation | null;
  onDeleteTargetChange: (item: Instance | null) => void;
  onDeleteConfirmationChange: (value: string) => void;
  onDelete: () => void;
  onCancelDelete: () => void;
  client?: DaemonClient;
};

export function InstancesPage({
  supported,
  loading,
  error,
  inventory,
  createName,
  createBusy,
  createOperation,
  copy,
  locale,
  onRefresh,
  onCreateNameChange,
  onCreate,
  onCancelCreate,
  deleteTarget,
  deleteConfirmation,
  deleteBusy,
  deleteOperation,
  onDeleteTargetChange,
  onDeleteConfirmationChange,
  onDelete,
  onCancelDelete,
  client,
}: InstancesPageProps) {
  const items = inventory?.instances ?? [];
  const named = items.filter((item) => !item.default);
  const [modelInstanceId, setModelInstanceId] = useState<string | null>(null);

  return (
    <section className="page-stack" aria-label={copy.pages.instances.title}>
      {!supported ? (
        <div className="card instance-card">
          <h2 className="instance-card-title">{copy.instances.unsupportedTitle}</h2>
          <p className="page-copy">{copy.instances.unsupportedDescription}</p>
        </div>
      ) : (
        <>
          <div className="instance-toolbar">
            <Button type="button" onClick={onRefresh} disabled={loading}>
              <IconRotateCw />
              {copy.instances.refresh}
            </Button>
            {inventory?.lastSyncedAt ? (
              <p className="page-copy">
                {copy.instances.lastSynced}: {formatDateTime(inventory.lastSyncedAt, locale)}
              </p>
            ) : null}
          </div>

          {loading && !inventory ? <p className="page-copy">{copy.instances.loading}</p> : null}
          {error && !inventory ? <p className="page-copy">{copy.instances.loadFailure}</p> : null}
          {inventory?.freshness === "UNKNOWN" ? (
            <p className="page-copy" role="status">
              {copy.instances.freshnessUnknown}
            </p>
          ) : null}
          {inventory?.errorCode ? <p className="page-copy">{copy.instances.queryFailed}</p> : null}

          <form
            className="instance-create"
            onSubmit={(event) => {
              event.preventDefault();
              onCreate();
            }}
          >
            <label className="instance-create-label" htmlFor="instance-name">
              {copy.instances.createLabel}
            </label>
            <input
              id="instance-name"
              className="instance-create-input"
              value={createName}
              onChange={(event) => onCreateNameChange(event.target.value)}
              placeholder={copy.instances.createPlaceholder}
              autoComplete="off"
              spellCheck={false}
              disabled={createBusy}
            />
            {createName !== "" && !createNamePattern.test(createName) ? (
              <p className="page-copy">{copy.instances.createInvalid}</p>
            ) : null}
            <div className="instance-toolbar">
              <Button type="submit" disabled={createBusy || !createNamePattern.test(createName)}>
                {copy.instances.createAction}
              </Button>
              {createOperation && (createOperation.status === "PENDING") ? (
                <Button type="button" onClick={onCancelCreate} disabled={createBusy}>
                  {copy.instances.cancelCreate}
                </Button>
              ) : null}
            </div>
            {createOperation ? (
              <p className="page-copy" role="status">
                {createOperation.status === "PENDING"
                  ? copy.instances.createPending
                  : createOperation.status === "RUNNING"
                    ? copy.instances.createRunning
                    : createOperation.status === "SUCCEEDED"
                      ? copy.instances.createSucceeded
                      : copy.instances.createFailed}
              </p>
            ) : null}
          </form>

          <ul className="instance-list">
            {items.map((item) => (
              <li key={item.instanceId}>
                <InstanceRow
                  item={item}
                  copy={copy}
                  locale={locale}
                  onDelete={() => onDeleteTargetChange(item)}
                  onOpenModels={() => setModelInstanceId(item.instanceId)}
                />
                {client && modelInstanceId === item.instanceId ? (
                  <ModelConfigurationPanel
                    client={client}
                    instance={item}
                    copy={copy}
                    locale={locale}
                    onClose={() => setModelInstanceId(null)}
                  />
                ) : null}
              </li>
            ))}
          </ul>

          {supported && !loading && named.length === 0 && items.some((item) => item.default) ? (
            <p className="page-copy">{copy.instances.emptyNamed}</p>
          ) : null}

          <p className="page-copy">{copy.instances.lifecycleUnavailable}</p>

          {deleteTarget && !deleteTarget.protected && !deleteTarget.default ? (
            <DeleteConfirmDialog
              target={deleteTarget}
              confirmation={deleteConfirmation}
              busy={deleteBusy}
              operation={deleteOperation}
              copy={copy}
              onConfirmationChange={onDeleteConfirmationChange}
              onConfirm={onDelete}
              onCancelOperation={onCancelDelete}
              onDismiss={() => onDeleteTargetChange(null)}
            />
          ) : null}
        </>
      )}
    </section>
  );
}

function DeleteConfirmDialog({
  target,
  confirmation,
  busy,
  operation,
  copy,
  onConfirmationChange,
  onConfirm,
  onCancelOperation,
  onDismiss,
}: {
  target: Instance;
  confirmation: string;
  busy: boolean;
  operation: Operation | null;
  copy: AppMessages;
  onConfirmationChange: (value: string) => void;
  onConfirm: () => void;
  onCancelOperation: () => void;
  onDismiss: () => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const status = operation?.status;
  const locked = busy || status === "RUNNING";
  const canCancelOperation = status === "PENDING";

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") {
        return;
      }
      if (locked) {
        return;
      }
      if (canCancelOperation) {
        onCancelOperation();
        return;
      }
      onDismiss();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [canCancelOperation, locked, onCancelOperation, onDismiss]);

  return (
    <div
      className="instance-modal-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !locked) {
          if (canCancelOperation) {
            onCancelOperation();
            return;
          }
          onDismiss();
        }
      }}
    >
      <div
        className="instance-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-instance-title"
        aria-describedby="delete-instance-warning"
      >
        <h2 id="delete-instance-title" className="instance-modal-title">
          {copy.instances.deleteTitle}
        </h2>
        <p className="instance-modal-name">{target.name}</p>
        <p id="delete-instance-warning" className="page-copy">
          {copy.instances.deleteWarning}
        </p>
        <label className="instance-create-label" htmlFor="delete-confirm">
          {copy.instances.deleteConfirmLabel}
        </label>
        <input
          ref={inputRef}
          id="delete-confirm"
          className="instance-create-input"
          value={confirmation}
          onChange={(event) => onConfirmationChange(event.target.value)}
          autoComplete="off"
          spellCheck={false}
          disabled={locked}
        />
        <div className="instance-modal-actions">
          <Button
            type="button"
            variant="danger"
            disabled={locked || confirmation !== target.name}
            onClick={onConfirm}
          >
            {copy.instances.deleteAction}
          </Button>
          {canCancelOperation ? (
            <Button type="button" onClick={onCancelOperation} disabled={busy}>
              {copy.instances.cancelDelete}
            </Button>
          ) : (
            <Button type="button" onClick={onDismiss} disabled={locked}>
              {copy.instances.dismissDelete}
            </Button>
          )}
        </div>
        {operation ? (
          <p className="page-copy" role="status">
            {operation.status === "PENDING"
              ? copy.instances.deletePending
              : operation.status === "RUNNING"
                ? copy.instances.deleteRunning
                : operation.status === "SUCCEEDED"
                  ? copy.instances.deleteSucceeded
                  : copy.instances.deleteFailed}
          </p>
        ) : null}
      </div>
    </div>
  );
}

function InstanceRow({
  item,
  copy,
  locale,
  onDelete,
  onOpenModels,
}: {
  item: Instance;
  copy: AppMessages;
  locale: Locale;
  onDelete: () => void;
  onOpenModels: () => void;
}) {
  const availability = item.availability;
  return (
    <article className="card instance-card">
      <div className="instance-card-head">
        <h2 className="instance-card-title">{item.name}</h2>
        <div className="instance-tags">
          {item.default ? <Badge tone="info">{copy.instances.defaultLabel}</Badge> : null}
          {item.protected ? <Badge tone="neutral">{copy.instances.protectedLabel}</Badge> : null}
          <span className={`instance-availability is-${availability.toLowerCase()}`}>
            {copy.instances.availability[availability]}
          </span>
        </div>
      </div>
      <p className="page-copy">{copy.instances.availabilityHint[availability]}</p>
      {item.lastSyncedAt ? (
        <p className="page-copy">
          {copy.instances.lastSynced}: {formatDateTime(item.lastSyncedAt, locale)}
        </p>
      ) : null}
      {!item.default && !item.protected && item.availability === "AVAILABLE" ? (
        <Button type="button" variant="danger" onClick={onDelete}>
          {copy.instances.deleteAction}
        </Button>
      ) : null}
      <Button type="button" onClick={onOpenModels} disabled={item.availability !== "AVAILABLE"}>
        {copy.models.open}
      </Button>
    </article>
  );
}
