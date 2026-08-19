import { formatDateTime } from "../formatDateTime";
import type { Instance, InstanceList, Operation } from "../api/types";
import type { AppMessages, Locale } from "../i18n";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { IconRotateCw } from "../components/ui/icons";

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
}: InstancesPageProps) {
  const items = inventory?.instances ?? [];
  const named = items.filter((item) => !item.default);

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
                <InstanceRow item={item} copy={copy} locale={locale} />
              </li>
            ))}
          </ul>

          {supported && !loading && named.length === 0 && items.some((item) => item.default) ? (
            <p className="page-copy">{copy.instances.emptyNamed}</p>
          ) : null}

          <p className="page-copy">{copy.instances.lifecycleUnavailable}</p>
        </>
      )}
    </section>
  );
}

function InstanceRow({
  item,
  copy,
  locale,
}: {
  item: Instance;
  copy: AppMessages;
  locale: Locale;
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
    </article>
  );
}
