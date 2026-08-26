import type { DaemonClient } from "../api/client";
import type { InstanceList } from "../api/types";
import { ManagementPanel } from "../components/management/ManagementPanel";
import { Button } from "../components/ui/Button";
import type { AppMessages, Locale } from "../i18n";

export function RuntimeManagementPage({ client, inventory, copy, locale, onBack }: {
  client: DaemonClient;
  inventory: InstanceList;
  copy: AppMessages;
  locale: Locale;
  onBack: () => void;
}) {
  const defaultInstance = inventory.instances.find((item) => item.default) ?? inventory.instances[0];

  return (
    <section className="runtime-management-page" aria-label={copy.management.runtimeTitle}>
      <div className="runtime-management-page-toolbar">
        <Button variant="ghost" onClick={onBack}>← {copy.management.backToRuntime}</Button>
      </div>
      <ManagementPanel
        client={client}
        instance={defaultInstance}
        instances={inventory.instances}
        runtimeCapabilities={inventory.capabilities}
        scope="runtime"
        copy={copy}
        locale={locale}
      />
    </section>
  );
}
