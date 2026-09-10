import type { RuntimeDiscovery } from "../api/types";
import type { AppMessages, Locale } from "../i18n";
import { formatDateTime } from "../formatDateTime";
import { Button } from "./ui/Button";
import { Card } from "./ui/Card";
import { IconRefresh } from "./ui/icons";

export function OpenClawDiscoveryView({ discovery, loading, failed, copy, locale, instanceCount, onRetry, onOpenInstances, onOpenManagement }: {
  discovery?: RuntimeDiscovery;
  loading: boolean;
  failed: boolean;
  copy: AppMessages;
  locale: Locale;
  instanceCount: number | null;
  onRetry: () => void;
  onOpenInstances: () => void;
  onOpenManagement?: () => void;
}) {
  const zh = locale === "zh-CN";
  const supported = !loading && !failed && discovery?.state === "SUPPORTED";
  const stateLabels: Record<RuntimeDiscovery["state"], string> = zh ? {
    SUPPORTED: "已就绪", NOT_INSTALLED: "未安装", UNSUPPORTED: "版本不受支持", BROKEN_EXECUTABLE: "无法运行",
    MALFORMED_VERSION: "无法识别版本", TIMED_OUT: "检测超时", AMBIGUOUS: "发现多个安装",
  } : {
    SUPPORTED: "Ready", NOT_INSTALLED: "Not installed", UNSUPPORTED: "Unsupported version", BROKEN_EXECUTABLE: "Unable to run",
    MALFORMED_VERSION: "Unrecognized version", TIMED_OUT: "Detection timed out", AMBIGUOUS: "Multiple installations found",
  };
  const stateText = loading ? (zh ? "正在检测" : "Checking") : failed ? (zh ? "检测失败" : "Detection failed") : discovery ? stateLabels[discovery.state] : "—";
  return (
    <Card className="runtime-overview-card" aria-labelledby="openclaw-title">
      <div className="runtime-overview-primary">
        <h2 id="openclaw-title">OpenClaw</h2>
        <p className="runtime-overview-description">{zh ? "独立 Gateway 实例，共享 YORVA 管理入口。" : "Independent Gateway instances, managed in YORVA."}</p>
        <dl className="runtime-overview-meta">
          <div><dt>{copy.hermes.version}</dt><dd>{discovery?.selected?.version || "—"}</dd></div>
          <div><dt>{copy.hermes.lastChecked}</dt><dd>{discovery ? formatDateTime(discovery.detectedAt, locale) : "—"}</dd></div>
        </dl>
        {!supported && !loading && <p className="panel-copy">{zh
          ? `使用官方 npm 安装 OpenClaw ${discovery?.supportedRange || "2026.9.3"}，并确保 PATH 中有 Node.js 24.16.0 或更高的 24 系列版本，然后重新检测。`
          : `Install official OpenClaw ${discovery?.supportedRange || "2026.9.3"} with npm and Node.js 24.16.0 or later in the 24.x series on PATH, then check again.`}</p>}
      </div>
      <div className="runtime-overview-secondary">
        <div className="runtime-secondary-head">
          <div><span className="runtime-field-label">{copy.hermes.compatibility}</span><strong className={supported ? "runtime-compatibility is-ready" : "runtime-compatibility"}>{stateText}</strong></div>
          <Button onClick={onRetry} disabled={loading} className="button-compact button-neutral"><IconRefresh />{copy.hermes.checkAgain}</Button>
        </div>
        {discovery?.selected?.path && <div className="runtime-path-field"><span className="runtime-field-label">{copy.hermes.executable}</span><code>{discovery.selected.path}</code></div>}
        {supported && <div className="runtime-instance-summary">
          <div><span className="runtime-field-label">{copy.hermes.managedInstances}</span><strong>{instanceCount ?? "—"}</strong></div>
          <Button onClick={onOpenInstances}>{zh ? "管理实例" : "Manage instances"}</Button>
          {onOpenManagement && <Button onClick={onOpenManagement}>{copy.management.open}</Button>}
        </div>}
      </div>
    </Card>
  );
}
