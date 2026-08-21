import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useRef, useState, type MouseEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { formatRelativeTime } from "../formatDateTime";
import type { DaemonClient } from "../api/client";
import type { Instance, InstanceList, Lifecycle, Operation } from "../api/types";
import type { AppMessages, Locale } from "../i18n";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import {
  HermesMark,
  IconAlert,
  IconClose,
  IconMessage,
  IconMore,
  IconPlay,
  IconPlus,
  IconRefresh,
  IconSearch,
  IconSliders,
  IconStop,
  IconTrash,
} from "../components/ui/icons";
import { ModelConfigurationPanel } from "../components/models/ModelConfigurationPanel";
import { ChannelPanel } from "../components/channels/ChannelPanel";

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
  const [channelInstanceId, setChannelInstanceId] = useState<string | null>(null);
  const [relativeNow, setRelativeNow] = useState(() => Date.now());
  const modelInstance = items.find((item) => item.instanceId === modelInstanceId) ?? null;
  const channelInstance = items.find((item) => item.instanceId === channelInstanceId) ?? null;

  const createOperationActive = createOperation?.status === "PENDING" || createOperation?.status === "RUNNING";

  useEffect(() => {
    const interval = window.setInterval(() => setRelativeNow(Date.now()), 30_000);
    return () => window.clearInterval(interval);
  }, []);

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
              <Button type="button" onClick={onRefresh} disabled={loading} className="button-compact button-neutral">
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
                      now={relativeNow}
                      client={client}
                      onDelete={() => onDeleteTargetChange(item)}
                      onOpenModels={() => setModelInstanceId(item.instanceId)}
                      onOpenChannels={() => setChannelInstanceId(item.instanceId)}
                    />
                  ))}
                  {!loading && filteredItems.length === 0 ? (
                    <tr><td className="instance-table-empty" colSpan={4}>{copy.instances.noMatches}</td></tr>
                  ) : null}
                </tbody>
              </table>
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

          {client && channelInstance ? (
            <div className="instance-modal-backdrop model-modal-backdrop">
              <div className="channel-modal" role="dialog" aria-modal="true" aria-label={`${copy.channels.title}: ${channelInstance.name}`}>
                <ChannelPanel client={client} instance={channelInstance} copy={copy} onClose={() => setChannelInstanceId(null)} />
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

function InstanceRow({ item, copy, locale, now, client, onDelete, onOpenModels, onOpenChannels }: {
  item: Instance;
  copy: AppMessages;
  locale: Locale;
  now: number;
  client?: DaemonClient;
  onDelete: () => void;
  onOpenModels: () => void;
  onOpenChannels: () => void;
}) {
  const availability = item.availability;
  const canLifecycle = Boolean(client && item.capabilities.lifecycle && availability === "AVAILABLE");
  const synced = <td className="instance-sync-time">{item.lastSyncedAt ? formatRelativeTime(item.lastSyncedAt, locale, now) : "—"}</td>;
  const moreActions = (
    <InstanceMoreActions
      copy={copy}
      disabled={availability !== "AVAILABLE"}
      canDelete={!item.default && !item.protected}
      onOpenModels={onOpenModels}
      onOpenChannels={onOpenChannels}
      onDelete={onDelete}
    />
  );
  return (
    <tr>
      <td>
        <div className="instance-identity">
          <span className="instance-avatar" aria-hidden="true"><HermesMark size={36} /></span>
          <div className="instance-identity-copy">
            <div className="instance-name-row">
              <strong>{item.name}</strong>
              {item.default ? <Badge tone="neutral">{copy.instances.defaultLabel}</Badge> : null}
            </div>
          </div>
        </div>
      </td>
      {canLifecycle ? (
        <LifecycleControls client={client!} item={item} copy={copy}>
          {(presentation, controls) => (
            <>
              <td>
                <InstanceStatusChip kind={presentation.kind} label={presentation.label} />
              </td>
              {synced}
              <td>
                <div className="instance-row-actions">
                  {controls}
                  {moreActions}
                </div>
              </td>
            </>
          )}
        </LifecycleControls>
      ) : (
        <>
          <td>
            <InstanceStatusChip
              kind={availability === "MISSING" ? "deleted" : "unknown"}
              label={availability === "AVAILABLE" ? copy.instances.lifecycleUnknown : copy.instances.availability[availability]}
              hint={copy.instances.availabilityHint[availability]}
            />
          </td>
          {synced}
          <td>
            <div className="instance-row-actions">{moreActions}</div>
          </td>
        </>
      )}
    </tr>
  );
}

function InstanceStatusChip({ kind, label, hint }: { kind: string; label: string; hint?: string }) {
  return (
    <span className={`instance-availability is-${kind}`} title={hint}>
      <span className={`availability-dot is-${kind}`} />
      {label}
    </span>
  );
}

function InstanceMoreActions({ copy, disabled, canDelete, onOpenModels, onOpenChannels, onDelete }: {
  copy: AppMessages;
  disabled: boolean;
  canDelete: boolean;
  onOpenModels: () => void;
  onOpenChannels: () => void;
  onDelete: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const closeFromOutside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (triggerRef.current?.contains(target) || menuRef.current?.contains(target)) return;
      setOpen(false);
    };
    const closeFromKeyboard = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    const closeFromViewport = () => setOpen(false);
    document.addEventListener("pointerdown", closeFromOutside);
    window.addEventListener("keydown", closeFromKeyboard);
    window.addEventListener("resize", closeFromViewport);
    window.addEventListener("scroll", closeFromViewport, true);
    return () => {
      document.removeEventListener("pointerdown", closeFromOutside);
      window.removeEventListener("keydown", closeFromKeyboard);
      window.removeEventListener("resize", closeFromViewport);
      window.removeEventListener("scroll", closeFromViewport, true);
    };
  }, [open]);

  const toggle = () => {
    if (open) {
      setOpen(false);
      return;
    }
    const rect = triggerRef.current?.getBoundingClientRect();
    if (!rect) return;
    const width = 176;
    const estimatedHeight = canDelete ? 132 : 92;
    setPosition({
      left: Math.max(12, Math.min(window.innerWidth - width - 12, rect.right - width)),
      top: window.innerHeight - rect.bottom >= estimatedHeight + 8
        ? rect.bottom + 6
        : Math.max(12, rect.top - estimatedHeight - 6),
    });
    setOpen(true);
  };
  const choose = (action: () => void) => {
    setOpen(false);
    action();
  };

  return (
    <div className="instance-more-actions">
      <button
        ref={triggerRef}
        type="button"
        className="instance-icon-action instance-more-trigger"
        aria-label={copy.instances.moreActions}
        title={copy.instances.moreActions}
        aria-haspopup="menu"
        aria-expanded={open}
        disabled={disabled}
        onClick={toggle}
      >
        <IconMore />
      </button>
      {open ? createPortal(
        <div ref={menuRef} className="instance-actions-menu" role="menu" style={position}>
          <button type="button" role="menuitem" onClick={() => choose(onOpenModels)}><IconSliders />{copy.models.open}</button>
          <button type="button" role="menuitem" onClick={() => choose(onOpenChannels)}><IconMessage />{copy.channels.open}</button>
          {canDelete ? (
            <button type="button" role="menuitem" className="is-danger" onClick={() => choose(onDelete)}><IconTrash />{copy.instances.deleteAction}</button>
          ) : null}
        </div>,
        document.body,
      ) : null}
    </div>
  );
}

function LifecycleControls({ client, item, copy, children }: {
  client: DaemonClient;
  item: Instance;
  copy: AppMessages;
  children: (presentation: ReturnType<typeof lifecyclePresentation>, controls: ReactNode) => ReactNode;
}) {
  const queryClient = useQueryClient();
  const [submittedOperationId, setSubmittedOperationId] = useState<string | null>(null);
  const [lifecycleError, setLifecycleError] = useState(false);
  const [confirmAction, setConfirmAction] = useState<"stop" | "restart" | null>(null);
  const lifecycleQuery = useQuery({
    queryKey: ["instance-lifecycle", item.instanceId, client.scope],
    queryFn: ({ signal }) => client.getInstanceLifecycle(item.instanceId, signal),
    retry: false,
    refetchInterval: 5000,
  });
  const followedOperationId = submittedOperationId ?? lifecycleQuery.data?.activeOperationId ?? null;
  const lifecycleOperationQuery = useQuery({
    queryKey: ["instance-lifecycle-operation", followedOperationId, client.scope],
    queryFn: ({ signal }) => client.getOperation(followedOperationId!, signal),
    enabled: followedOperationId !== null,
    retry: false,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "PENDING" || status === "RUNNING" ? 750 : false;
    },
  });
  const lifecycleOperation = lifecycleOperationQuery.data;
  const lifecycleOperationId = lifecycleOperation?.id;
  const lifecycleOperationStatus = lifecycleOperation?.status;
  const lifecycleBusy = lifecycleOperationStatus === "PENDING" || lifecycleOperationStatus === "RUNNING";
  const refetchLifecycle = lifecycleQuery.refetch;

  useEffect(() => {
    if (!lifecycleOperationId || lifecycleBusy) return;
    void refetchLifecycle();
    void queryClient.invalidateQueries({ queryKey: ["hermes-instances"] });
  }, [lifecycleBusy, lifecycleOperationId, lifecycleOperationStatus, queryClient, refetchLifecycle]);

  const runLifecycle = async (action: "start" | "stop" | "restart") => {
    if (lifecycleBusy) return;
    setLifecycleError(false);
    try {
      const operation = await client.startInstanceLifecycle(item.instanceId, action, crypto.randomUUID());
      setSubmittedOperationId(operation.id);
    } catch {
      setLifecycleError(true);
      void lifecycleQuery.refetch();
    }
  };

  const presentation = lifecyclePresentation(lifecycleQuery.data, lifecycleOperation, copy);
  const refreshLifecycle = () => {
    setSubmittedOperationId(null);
    setLifecycleError(false);
    void lifecycleQuery.refetch();
  };
  const controls = (
    <div className="instance-lifecycle-actions" title={lifecycleError ? copy.instances.lifecycleFailed : undefined}>
      {presentation.state === "STOPPED" ? (
        <Button type="button" variant="ghost" className="instance-icon-action" aria-label={copy.instances.lifecycleStart} title={copy.instances.lifecycleStart} disabled={lifecycleBusy} onClick={() => { void runLifecycle("start"); }}>
          <IconPlay />
        </Button>
      ) : null}
      {presentation.state === "RUNNING" ? (
        <>
          <Button type="button" variant="ghost" className="instance-icon-action" aria-label={copy.instances.lifecycleStop} title={copy.instances.lifecycleStop} disabled={lifecycleBusy} onClick={() => { setConfirmAction("stop"); }}>
            <IconStop />
          </Button>
          <Button type="button" variant="ghost" className="instance-icon-action" aria-label={copy.instances.lifecycleRestart} title={copy.instances.lifecycleRestart} disabled={lifecycleBusy} onClick={() => { setConfirmAction("restart"); }}>
            <IconRefresh className={lifecycleBusy ? "spin" : undefined} />
          </Button>
        </>
      ) : null}
      {presentation.state === "UNKNOWN" && !lifecycleBusy ? (
        <Button type="button" variant="ghost" className="instance-icon-action" aria-label={copy.instances.refresh} title={copy.instances.refresh} onClick={refreshLifecycle}><IconRefresh /></Button>
      ) : null}
      {confirmAction ? (
        <div className="instance-modal-backdrop" onMouseDown={(event) => {
          if (event.target === event.currentTarget) setConfirmAction(null);
        }}>
          <div className="instance-modal" role="dialog" aria-modal="true" aria-labelledby={`lifecycle-confirm-${item.instanceId}`}>
            <h2 id={`lifecycle-confirm-${item.instanceId}`} className="instance-modal-title">{copy.instances.lifecycleConfirmTitle}</h2>
            <p>{confirmAction === "stop" ? copy.instances.lifecycleStopWarning : copy.instances.lifecycleRestartWarning}</p>
            <p className="instance-modal-name">{item.name}</p>
            <div className="instance-modal-actions">
              <Button type="button" variant="secondary" onClick={() => { setConfirmAction(null); }}>{copy.instances.dismissDelete}</Button>
              <Button type="button" variant="primary" onClick={() => {
                const action = confirmAction;
                setConfirmAction(null);
                void runLifecycle(action);
              }}>{copy.instances.lifecycleConfirm}</Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
  return children(presentation, controls);
}

function lifecyclePresentation(lifecycle: Lifecycle | undefined, operation: Operation | undefined, copy: AppMessages): {
  state: Lifecycle["state"];
  kind: "running" | "stopped" | "starting" | "stopping" | "restarting" | "unknown";
  label: string;
  tone: "ok" | "warn" | "neutral" | "info" | "error";
} {
  if (operation?.status === "PENDING" || operation?.status === "RUNNING") {
    if (operation.type === "instance.stop") return { state: "UNKNOWN", kind: "stopping", label: copy.instances.lifecycleStopping, tone: "info" };
    if (operation.type === "instance.restart") return { state: "UNKNOWN", kind: "restarting", label: copy.instances.lifecycleRestarting, tone: "info" };
    return { state: "UNKNOWN", kind: "starting", label: copy.instances.lifecycleStarting, tone: "info" };
  }
  if (operation?.status === "FAILED" || operation?.status === "CANCELLED") {
    return { state: "UNKNOWN", kind: "unknown", label: copy.instances.lifecycleFailed, tone: "error" };
  }
  if (lifecycle?.state === "RUNNING") return { state: "RUNNING", kind: "running", label: copy.instances.lifecycleRunning, tone: "ok" };
  if (lifecycle?.state === "STOPPED") return { state: "STOPPED", kind: "stopped", label: copy.instances.lifecycleStopped, tone: "warn" };
  return { state: "UNKNOWN", kind: "unknown", label: copy.instances.lifecycleUnknown, tone: "neutral" };
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
