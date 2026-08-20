import { useEffect, useMemo, useRef, useState, type MouseEvent, type ReactNode } from "react";
import { formatDateTime } from "../formatDateTime";
import type { DaemonClient } from "../api/client";
import type { Instance, InstanceList, Operation } from "../api/types";
import type { AppMessages, Locale } from "../i18n";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import {
  IconAlert,
  IconClose,
  IconPlus,
  IconRefresh,
  IconSearch,
  IconSliders,
  IconTrash,
} from "../components/ui/icons";
import { ModelConfigurationPanel } from "../components/models/ModelConfigurationPanel";

const createNamePattern = /^[a-z][a-z0-9_-]{0,63}$/;
type AvailabilityFilter = "ALL" | Instance["availability"];

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
  onPrepareCreate: () => void;
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
  onPrepareCreate,
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
  const items = useMemo(() => inventory?.instances ?? [], [inventory?.instances]);
  const named = items.filter((item) => !item.default);
  const [availabilityFilter, setAvailabilityFilter] = useState<AvailabilityFilter>("ALL");
  const [searchQuery, setSearchQuery] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [modelInstanceId, setModelInstanceId] = useState<string | null>(null);
  const modelInstance = items.find((item) => item.instanceId === modelInstanceId) ?? null;

  const createOperationActive = createOperation?.status === "PENDING" || createOperation?.status === "RUNNING";

  const filteredItems = useMemo(() => {
    const query = searchQuery.trim().toLocaleLowerCase(locale);
    return items.filter((item) => {
      const matchesAvailability = availabilityFilter === "ALL" || item.availability === availabilityFilter;
      const matchesQuery = query === ""
        || item.name.toLocaleLowerCase(locale).includes(query)
        || item.instanceId.toLocaleLowerCase(locale).includes(query);
      return matchesAvailability && matchesQuery;
    });
  }, [availabilityFilter, items, locale, searchQuery]);

  const availabilityCounts = useMemo(() => ({
    ALL: items.length,
    AVAILABLE: items.filter((item) => item.availability === "AVAILABLE").length,
    MISSING: items.filter((item) => item.availability === "MISSING").length,
    UNKNOWN: items.filter((item) => item.availability === "UNKNOWN").length,
  }), [items]);

  const openCreate = () => {
    onPrepareCreate();
    setCreateOpen(true);
  };

  return (
    <section className="page-stack instances-page" aria-label={copy.pages.instances.title}>
      {!supported ? (
        <div className="card empty-state-card">
          <div className="empty-state-icon"><IconAlert /></div>
          <div>
            <h2 className="instance-card-title">{copy.instances.unsupportedTitle}</h2>
            <p className="page-copy">{copy.instances.unsupportedDescription}</p>
          </div>
        </div>
      ) : (
        <>
          <div className="instance-control-bar">
            <div className="instance-filter-tabs" role="group" aria-label={copy.instances.tableAvailability}>
              {(["ALL", "AVAILABLE", "MISSING", "UNKNOWN"] as const).map((filter) => (
                <button
                  type="button"
                  key={filter}
                  className={availabilityFilter === filter ? "instance-filter is-active" : "instance-filter"}
                  aria-pressed={availabilityFilter === filter}
                  onClick={() => setAvailabilityFilter(filter)}
                >
                  {filter !== "ALL" ? <span className={`availability-dot is-${filter.toLowerCase()}`} /> : null}
                  <span>{filter === "ALL" ? copy.instances.allFilter : copy.instances.availability[filter]}</span>
                  <span className="instance-filter-count">{availabilityCounts[filter]}</span>
                </button>
              ))}
            </div>

            <div className="instance-control-actions">
              <label className="instance-search">
                <span className="sr-only">{copy.instances.searchLabel}</span>
                <IconSearch />
                <input
                  type="search"
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
                  placeholder={copy.instances.searchPlaceholder}
                  spellCheck={false}
                />
              </label>
              <Button type="button" onClick={onRefresh} disabled={loading} className="button-compact">
                <IconRefresh className={loading ? "spin" : undefined} />
                {copy.instances.refresh}
              </Button>
              <Button type="button" variant="primary" onClick={openCreate} className="button-compact">
                <IconPlus />
                {copy.instances.createAction}
              </Button>
            </div>
          </div>

          <InventoryNotices inventory={inventory} loading={loading} error={error} copy={copy} />

          <div className="instance-table-card">
            <div className="instance-table-scroll">
              <table className="instance-table">
                <thead>
                  <tr>
                    <th>{copy.instances.tableInstance}</th>
                    <th>{copy.instances.tableAvailability}</th>
                    <th>{copy.instances.tableLastSynced}</th>
                    <th>{copy.instances.tableCapabilities}</th>
                    <th className="instance-actions-column">{copy.instances.tableActions}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredItems.map((item) => (
                    <InstanceRow
                      key={item.instanceId}
                      item={item}
                      copy={copy}
                      locale={locale}
                      onDelete={() => onDeleteTargetChange(item)}
                      onOpenModels={() => setModelInstanceId(item.instanceId)}
                    />
                  ))}
                  {!loading && filteredItems.length === 0 ? (
                    <tr><td className="instance-table-empty" colSpan={5}>{copy.instances.noMatches}</td></tr>
                  ) : null}
                </tbody>
              </table>
            </div>
            <div className="instance-table-footer">
              <span>{copy.instances.totalCount.replace("{count}", String(filteredItems.length))}</span>
              <span>{copy.instances.lifecycleUnavailable}</span>
            </div>
          </div>

          {!loading && named.length === 0 && items.some((item) => item.default) ? (
            <p className="notice notice-info">{copy.instances.emptyNamed}</p>
          ) : null}

          {createOpen || createOperationActive ? (
            <CreateInstanceDialog
              name={createName}
              busy={createBusy}
              operation={createOperation}
              copy={copy}
              onNameChange={onCreateNameChange}
              onCreate={onCreate}
              onCancelOperation={onCancelCreate}
              onDismiss={() => setCreateOpen(false)}
            />
          ) : null}

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

          {client && modelInstance ? (
            <div className="instance-modal-backdrop model-modal-backdrop">
              <div className="model-modal" role="dialog" aria-modal="true" aria-label={`${copy.models.title}: ${modelInstance.name}`}>
                <ModelConfigurationPanel
                  client={client}
                  instance={modelInstance}
                  copy={copy}
                  locale={locale}
                  onClose={() => setModelInstanceId(null)}
                />
              </div>
            </div>
          ) : null}
        </>
      )}
    </section>
  );
}

function InventoryNotices({ inventory, loading, error, copy }: {
  inventory: InstanceList | null;
  loading: boolean;
  error: boolean;
  copy: AppMessages;
}) {
  if (loading && !inventory) return <p className="notice notice-info">{copy.instances.loading}</p>;
  if (error && !inventory) return <p className="notice notice-error">{copy.instances.loadFailure}</p>;
  if (inventory?.freshness === "UNKNOWN") return <p className="notice notice-warn" role="status">{copy.instances.freshnessUnknown}</p>;
  if (inventory?.errorCode) return <p className="notice notice-warn">{copy.instances.queryFailed}</p>;
  return null;
}

function CreateInstanceDialog({ name, busy, operation, copy, onNameChange, onCreate, onCancelOperation, onDismiss }: {
  name: string;
  busy: boolean;
  operation: Operation | null;
  copy: AppMessages;
  onNameChange: (value: string) => void;
  onCreate: () => void;
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
  useDialogDismiss({ locked, canCancelOperation, onCancelOperation, onDismiss });

  return (
    <div className="instance-modal-backdrop" onMouseDown={(event) => dismissFromBackdrop(event, locked, canCancelOperation, onCancelOperation, onDismiss)}>
      <div className="instance-modal" role="dialog" aria-modal="true" aria-labelledby="create-instance-title" aria-describedby="create-instance-description">
        <ModalHeader
          icon={<IconPlus />}
          title={copy.instances.createAction}
          description={copy.instances.createDescription}
          titleId="create-instance-title"
          descriptionId="create-instance-description"
          closeLabel={copy.instances.dismissDelete}
          onClose={canCancelOperation ? onCancelOperation : onDismiss}
          closeDisabled={locked}
        />
        <form onSubmit={(event) => { event.preventDefault(); onCreate(); }}>
          <label className="instance-create-label" htmlFor="instance-name">{copy.instances.createLabel}</label>
          <input
            ref={inputRef}
            id="instance-name"
            className="instance-create-input"
            value={name}
            onChange={(event) => onNameChange(event.target.value)}
            placeholder={copy.instances.createPlaceholder}
            autoComplete="off"
            spellCheck={false}
            disabled={locked || canCancelOperation}
          />
          {name !== "" && !createNamePattern.test(name) ? <p className="field-error">{copy.instances.createInvalid}</p> : null}
          {operation ? <p className="modal-operation-status" role="status">{operationStatus(operation, copy, "create")}</p> : null}
          <div className="instance-modal-actions">
            {canCancelOperation ? (
              <Button type="button" onClick={onCancelOperation} disabled={busy}>{copy.instances.cancelCreate}</Button>
            ) : (
              <Button type="button" onClick={onDismiss} disabled={locked}>{copy.instances.dismissDelete}</Button>
            )}
            <Button type="submit" variant="primary" disabled={locked || canCancelOperation || !createNamePattern.test(name)}>
              <IconPlus />
              {copy.instances.createAction}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

function DeleteConfirmDialog({ target, confirmation, busy, operation, copy, onConfirmationChange, onConfirm, onCancelOperation, onDismiss }: {
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
  useDialogDismiss({ locked, canCancelOperation, onCancelOperation, onDismiss });

  return (
    <div className="instance-modal-backdrop" onMouseDown={(event) => dismissFromBackdrop(event, locked, canCancelOperation, onCancelOperation, onDismiss)}>
      <div className="instance-modal" role="dialog" aria-modal="true" aria-labelledby="delete-instance-title" aria-describedby="delete-instance-warning">
        <ModalHeader
          icon={<IconAlert />}
          danger
          title={copy.instances.deleteTitle}
          description={copy.instances.deleteWarning}
          titleId="delete-instance-title"
          descriptionId="delete-instance-warning"
          closeLabel={copy.instances.dismissDelete}
          onClose={canCancelOperation ? onCancelOperation : onDismiss}
          closeDisabled={locked}
        />
        <p className="instance-modal-name">{target.name}</p>
        <label className="instance-create-label" htmlFor="delete-confirm">{copy.instances.deleteConfirmLabel}</label>
        <input
          ref={inputRef}
          id="delete-confirm"
          className="instance-create-input"
          value={confirmation}
          onChange={(event) => onConfirmationChange(event.target.value)}
          autoComplete="off"
          spellCheck={false}
          disabled={locked || canCancelOperation}
        />
        {operation ? <p className="modal-operation-status" role="status">{operationStatus(operation, copy, "delete")}</p> : null}
        <div className="instance-modal-actions">
          {canCancelOperation ? (
            <Button type="button" onClick={onCancelOperation} disabled={busy}>{copy.instances.cancelDelete}</Button>
          ) : (
            <Button type="button" onClick={onDismiss} disabled={locked}>{copy.instances.dismissDelete}</Button>
          )}
          <Button type="button" variant="danger" disabled={locked || canCancelOperation || confirmation !== target.name} onClick={onConfirm}>
            <IconTrash />
            {copy.instances.deleteAction}
          </Button>
        </div>
      </div>
    </div>
  );
}

function ModalHeader({ icon, title, description, titleId, descriptionId, danger = false, closeLabel, onClose, closeDisabled }: {
  icon: ReactNode;
  title: string;
  description: string;
  titleId: string;
  descriptionId: string;
  danger?: boolean;
  closeLabel: string;
  onClose: () => void;
  closeDisabled: boolean;
}) {
  return (
    <div className="instance-modal-header">
      <div className={danger ? "modal-icon is-danger" : "modal-icon"}>{icon}</div>
      <div className="instance-modal-heading">
        <h2 id={titleId} className="instance-modal-title">{title}</h2>
        <p id={descriptionId} className="page-copy">{description}</p>
      </div>
      <button type="button" className="modal-close" onClick={onClose} disabled={closeDisabled} aria-label={closeLabel}>
        <IconClose />
      </button>
    </div>
  );
}

function useDialogDismiss({ locked, canCancelOperation, onCancelOperation, onDismiss }: {
  locked: boolean;
  canCancelOperation: boolean;
  onCancelOperation: () => void;
  onDismiss: () => void;
}) {
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || locked) return;
      if (canCancelOperation) onCancelOperation();
      else onDismiss();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [canCancelOperation, locked, onCancelOperation, onDismiss]);
}

function dismissFromBackdrop(event: MouseEvent<HTMLDivElement>, locked: boolean, canCancelOperation: boolean, onCancelOperation: () => void, onDismiss: () => void) {
  if (event.target !== event.currentTarget || locked) return;
  if (canCancelOperation) onCancelOperation();
  else onDismiss();
}

function InstanceRow({ item, copy, locale, onDelete, onOpenModels }: {
  item: Instance;
  copy: AppMessages;
  locale: Locale;
  onDelete: () => void;
  onOpenModels: () => void;
}) {
  const availability = item.availability;
  return (
    <tr>
      <td>
        <div className="instance-identity">
          <span className="instance-avatar">{item.name.slice(0, 2).toUpperCase()}</span>
          <div className="instance-identity-copy">
            <div className="instance-name-row">
              <strong>{item.name}</strong>
              {item.default ? <Badge tone="info">{copy.instances.defaultLabel}</Badge> : null}
              {item.protected ? <Badge tone="neutral">{copy.instances.protectedLabel}</Badge> : null}
            </div>
            <span className="mono muted">{item.instanceId}</span>
          </div>
        </div>
      </td>
      <td>
        <span className={`instance-availability is-${availability.toLowerCase()}`} title={copy.instances.availabilityHint[availability]}>
          <span className={`availability-dot is-${availability.toLowerCase()}`} />
          {copy.instances.availability[availability]}
        </span>
      </td>
      <td className="instance-sync-time">{item.lastSyncedAt ? formatDateTime(item.lastSyncedAt, locale) : "—"}</td>
      <td>
        <div className="instance-capabilities">
          <span>{copy.instances.instanceCapability}: <b>{item.capabilities.instances ? copy.instances.capabilityAvailable : copy.instances.capabilityUnavailable}</b></span>
          <span>{copy.instances.lifecycleCapability}: <b>{item.capabilities.lifecycle ? copy.instances.capabilityAvailable : copy.instances.capabilityUnavailable}</b></span>
        </div>
      </td>
      <td>
        <div className="instance-row-actions">
          <Button type="button" variant="ghost" onClick={onOpenModels} disabled={availability !== "AVAILABLE"}>
            <IconSliders />
            {copy.models.open}
          </Button>
          {!item.default && !item.protected && availability === "AVAILABLE" ? (
            <Button type="button" variant="ghost" className="button-danger-ghost" onClick={onDelete}>
              <IconTrash />
              {copy.instances.deleteAction}
            </Button>
          ) : null}
        </div>
      </td>
    </tr>
  );
}

function operationStatus(operation: Operation, copy: AppMessages, kind: "create" | "delete") {
  const statusCopy = kind === "create"
    ? {
        PENDING: copy.instances.createPending,
        RUNNING: copy.instances.createRunning,
        SUCCEEDED: copy.instances.createSucceeded,
        FAILED: copy.instances.createFailed,
        CANCELLED: copy.instances.createFailed,
      }
    : {
        PENDING: copy.instances.deletePending,
        RUNNING: copy.instances.deleteRunning,
        SUCCEEDED: copy.instances.deleteSucceeded,
        FAILED: copy.instances.deleteFailed,
        CANCELLED: copy.instances.deleteFailed,
      };
  return statusCopy[operation.status];
}
