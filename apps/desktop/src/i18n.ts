import type { RuntimeDiscoveryState } from "./api/types";
import type { EventStreamStatus } from "./hooks/useEventStreamStatus";

export type Locale = "en-US" | "zh-CN";
export type PageId = "dashboard" | "runtimes" | "settings";

type RuntimeStateCopy = {
  title: string;
  description: string;
};

type Messages = {
  languageName: string;
  brandTagline: string;
  navigationLabel: string;
  pages: Record<PageId, { navigation: string; title: string; description: string }>;
  switchLanguage: string;
  versionUnavailable: string;
  node: {
    starting: string;
    startingDescription: string;
    connectionUnavailable: string;
    daemonStartFailure: string;
    nodeReachFailure: string;
    connected: string;
    title: string;
    description: string;
    name: string;
    id: string;
    platform: string;
    version: string;
    events: string;
    eventStates: Record<EventStreamStatus, string>;
  };
  hermes: {
    title: string;
    summaryTitle: string;
    summaryDescription: string;
    openRuntimes: string;
    checking: string;
    checkingDescription: string;
    cancelled: string;
    cancelledDescription: string;
    unavailable: string;
    unavailableDescription: string;
    cancel: string;
    retry: string;
    checkAgain: string;
    version: string;
    executable: string;
    compatibility: string;
    supportedRange: string;
    lastChecked: string;
    candidates: string;
    candidateCount: string;
    warnings: string;
    unknownWarning: string;
    states: Record<RuntimeDiscoveryState, RuntimeStateCopy>;
    warningMessages: Record<string, string>;
  };
  settings: {
    language: string;
    languageDescription: string;
    languageLegend: string;
    savedAutomatically: string;
  };
};

const english: Messages = {
  languageName: "English",
  brandTagline: "Local-first AI runtime control",
  navigationLabel: "Primary navigation",
  pages: {
    dashboard: {
      navigation: "Dashboard",
      title: "Dashboard",
      description: "Local Node health and Runtime readiness at a glance.",
    },
    runtimes: {
      navigation: "Runtimes",
      title: "Runtimes",
      description: "Discover the official Hermes CLI and verify version compatibility.",
    },
    settings: {
      navigation: "Settings",
      title: "Settings",
      description: "Choose how YORVA appears on this device.",
    },
  },
  switchLanguage: "Switch language",
  versionUnavailable: "yorvad unavailable",
  node: {
    starting: "Starting local node",
    startingDescription: "Creating a private Desktop session and checking the local daemon.",
    connectionUnavailable: "Connection unavailable",
    daemonStartFailure: "The local daemon could not start.",
    nodeReachFailure: "The local Node could not be reached.",
    connected: "Local node connected",
    title: "Local Node",
    description: "This device is connected through the authenticated local management API.",
    name: "Node name",
    id: "Node ID",
    platform: "Platform",
    version: "yorvad version",
    events: "Events",
    eventStates: {
      idle: "Idle",
      connecting: "Connecting",
      connected: "Connected",
      disconnected: "Disconnected",
    },
  },
  hermes: {
    title: "Hermes discovery",
    summaryTitle: "Hermes Runtime",
    summaryDescription: "Read-only discovery and compatibility status.",
    openRuntimes: "View Runtime details",
    checking: "Checking Hermes",
    checkingDescription: "Looking for a safe local Hermes CLI and checking its version.",
    cancelled: "Check cancelled",
    cancelledDescription: "The Hermes check was cancelled. No Runtime state was changed.",
    unavailable: "Discovery unavailable",
    unavailableDescription: "YORVA could not complete Hermes discovery.",
    cancel: "Cancel",
    retry: "Retry",
    checkAgain: "Check again",
    version: "Version",
    executable: "Executable",
    compatibility: "Compatibility",
    supportedRange: "Supported range",
    lastChecked: "Last checked",
    candidates: "Hermes candidates",
    candidateCount: "Candidates found: {count}",
    warnings: "Warnings",
    unknownWarning: "Hermes discovery reported an additional warning.",
    states: {
      NOT_INSTALLED: {
        title: "Hermes not installed",
        description: "No trusted Hermes installation evidence was found.",
      },
      SUPPORTED: {
        title: "Hermes ready",
        description: "The detected Hermes version is compatible with YORVA.",
      },
      UNSUPPORTED: {
        title: "Hermes version unsupported",
        description: "Hermes was found, but its version is outside the supported range.",
      },
      BROKEN_EXECUTABLE: {
        title: "Hermes installation is incomplete",
        description: "Hermes installation evidence was found, but its safe CLI launcher is missing or broken.",
      },
      MALFORMED_VERSION: {
        title: "Hermes version is unreadable",
        description: "Hermes returned a version that YORVA could not safely interpret.",
      },
      TIMED_OUT: {
        title: "Hermes check timed out",
        description: "Hermes did not report its version before the discovery deadline.",
      },
      AMBIGUOUS: {
        title: "Multiple Hermes executables found",
        description: "YORVA found more than one runnable Hermes executable and did not choose between them.",
      },
    },
    warningMessages: {
      CANDIDATE_LIMIT_REACHED: "Additional Hermes executable candidates were not evaluated.",
      MULTIPLE_RUNNABLE_CANDIDATES: "Multiple runnable Hermes executables were found; none was selected.",
      OTHER_CANDIDATES_UNUSABLE: "Other Hermes candidates could not be used.",
      PRERELEASE_UNTESTED: "The detected Hermes prerelease is outside the tested compatibility range.",
      HERMES_CLI_LAUNCHER_MISSING: "The Hermes installation does not contain a safe CLI launcher.",
    },
  },
  settings: {
    language: "Language",
    languageDescription: "YORVA applies your language choice immediately and remembers it on this device.",
    languageLegend: "Interface language",
    savedAutomatically: "Saved automatically",
  },
};

const simplifiedChinese: Messages = {
  languageName: "简体中文",
  brandTagline: "本地优先的 AI 运行时控制台",
  navigationLabel: "主导航",
  pages: {
    dashboard: {
      navigation: "仪表盘",
      title: "仪表盘",
      description: "集中查看本地节点健康状态和运行时就绪情况。",
    },
    runtimes: {
      navigation: "运行时",
      title: "运行时",
      description: "检测官方 Hermes CLI 并核验版本兼容性。",
    },
    settings: {
      navigation: "设置",
      title: "设置",
      description: "选择 YORVA 在此设备上的显示方式。",
    },
  },
  switchLanguage: "切换语言",
  versionUnavailable: "yorvad 不可用",
  node: {
    starting: "正在启动本地节点",
    startingDescription: "正在创建私有桌面会话并检查本地守护进程。",
    connectionUnavailable: "连接不可用",
    daemonStartFailure: "本地守护进程无法启动。",
    nodeReachFailure: "无法连接本地节点。",
    connected: "本地节点已连接",
    title: "本地节点",
    description: "此设备已通过经过身份验证的本地管理 API 连接。",
    name: "节点名称",
    id: "节点 ID",
    platform: "平台",
    version: "yorvad 版本",
    events: "事件连接",
    eventStates: {
      idle: "空闲",
      connecting: "连接中",
      connected: "已连接",
      disconnected: "已断开",
    },
  },
  hermes: {
    title: "Hermes 检测",
    summaryTitle: "Hermes 运行时",
    summaryDescription: "只读检测和版本兼容状态。",
    openRuntimes: "查看运行时详情",
    checking: "正在检测 Hermes",
    checkingDescription: "正在查找安全的本地 Hermes CLI 并检查版本。",
    cancelled: "检测已取消",
    cancelledDescription: "Hermes 检测已取消，未修改任何运行时状态。",
    unavailable: "检测不可用",
    unavailableDescription: "YORVA 无法完成 Hermes 检测。",
    cancel: "取消",
    retry: "重试",
    checkAgain: "重新检测",
    version: "版本",
    executable: "可执行文件",
    compatibility: "兼容性",
    supportedRange: "支持范围",
    lastChecked: "上次检测",
    candidates: "Hermes 候选项",
    candidateCount: "发现 {count} 个候选项",
    warnings: "警告",
    unknownWarning: "Hermes 检测返回了一项附加警告。",
    states: {
      NOT_INSTALLED: {
        title: "未检测到 Hermes",
        description: "未找到可信的 Hermes 安装证据。",
      },
      SUPPORTED: {
        title: "Hermes 已就绪",
        description: "检测到的 Hermes 版本与 YORVA 兼容。",
      },
      UNSUPPORTED: {
        title: "Hermes 版本不受支持",
        description: "已检测到 Hermes，但其版本超出当前支持范围。",
      },
      BROKEN_EXECUTABLE: {
        title: "Hermes 安装不完整",
        description: "已找到 Hermes 安装证据，但安全 CLI 启动器缺失或损坏。",
      },
      MALFORMED_VERSION: {
        title: "无法识别 Hermes 版本",
        description: "Hermes 返回的版本信息无法被 YORVA 安全解析。",
      },
      TIMED_OUT: {
        title: "Hermes 检测超时",
        description: "Hermes 未在检测时限内返回版本信息。",
      },
      AMBIGUOUS: {
        title: "发现多个 Hermes 可执行文件",
        description: "YORVA 发现多个可运行的 Hermes，未擅自选择其中任何一个。",
      },
    },
    warningMessages: {
      CANDIDATE_LIMIT_REACHED: "还有其他 Hermes 候选项未被检测。",
      MULTIPLE_RUNNABLE_CANDIDATES: "发现多个可运行的 Hermes，未选择任何一个。",
      OTHER_CANDIDATES_UNUSABLE: "其他 Hermes 候选项无法使用。",
      PRERELEASE_UNTESTED: "检测到的 Hermes 预发布版本不在已测试的兼容范围内。",
      HERMES_CLI_LAUNCHER_MISSING: "Hermes 安装中缺少安全的 CLI 启动器。",
    },
  },
  settings: {
    language: "语言",
    languageDescription: "YORVA 会立即应用语言选择，并在此设备上记住该设置。",
    languageLegend: "界面语言",
    savedAutomatically: "已自动保存",
  },
};

export const messages: Record<Locale, Messages> = {
  "en-US": english,
  "zh-CN": simplifiedChinese,
};

export const supportedLocales = Object.keys(messages) as Locale[];
export const localeStorageKey = "yorva.locale";

export function resolveLocale(savedLocale: string | null, systemLocale: string): Locale {
  if (savedLocale === "en-US" || savedLocale === "zh-CN") return savedLocale;
  return systemLocale.toLowerCase().startsWith("zh") ? "zh-CN" : "en-US";
}

export function loadLocale(): Locale {
  let savedLocale: string | null = null;
  try {
    savedLocale = window.localStorage.getItem(localeStorageKey);
  } catch {
    // A denied storage API should not block the Desktop UI.
  }
  return resolveLocale(savedLocale, window.navigator.language);
}

export function saveLocale(locale: Locale): void {
  try {
    window.localStorage.setItem(localeStorageKey, locale);
  } catch {
    // Keep the in-memory choice when persistence is unavailable.
  }
}

export type AppMessages = Messages;
