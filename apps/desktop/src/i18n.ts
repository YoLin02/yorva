import type { RuntimeDiscoveryState } from "./api/types";
import type { EventStreamStatus } from "./hooks/useEventStreamStatus";

export type Locale = "en-US" | "zh-CN";
export type PageId = "dashboard" | "runtimes" | "instances" | "settings";

type RuntimeStateCopy = {
  title: string;
  description: string;
};

type Messages = {
  languageName: string;
  brandTagline: string;
  navigationLabel: string;
  pages: Record<PageId, { navigation: string; title: string; description: string }>;
  languageShort: string;
  switchLanguage: string;
  versionUnavailable: string;
  dashboard: {
    connectionState: string;
    nodeInfo: string;
    systemPlatform: string;
    discoveryTitle: string;
    discoveryNotice: string;
    supportedStatuses: string;
    statusInstalled: string;
    statusUnsupported: string;
    statusChecking: string;
    statusBroken: string;
    unavailableValue: string;
  };
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
    managedInstances: string;
    viewInstances: string;
    unavailableValue: string;
    candidates: string;
    candidateCount: string;
    warnings: string;
    unknownWarning: string;
    states: Record<RuntimeDiscoveryState, RuntimeStateCopy>;
    warningMessages: Record<string, string>;
    install: {
      action: string;
      confirmTitle: string;
      confirmDescription: string;
      source: string;
      version: string;
      destination: string;
      hostChanges: string;
      hostChangeItems: string[];
      bundledSourceNote: string;
      sourceNotes: Record<string, string>;
      noProfileNote: string;
      confirm: string;
      back: string;
      unavailable: string;
      blocked: string;
      running: string;
      cancelling: string;
      cancelled: string;
      failed: string;
      interrupted: string;
      succeeded: string;
      stage: string;
      errorCode: string;
      correlation: string;
      logTitle: string;
      logHint: string;
      copyLog: string;
      logCopied: string;
      retryInstall: string;
      cancelInstall: string;
    };
    prerequisites: {
      title: string;
      nodeReady: string;
      nodeMissing: string;
      nodeUnsupported: string;
      npmUnsupported: string;
      depsNotInstalled: string;
      depsFailed: string;
      installAction: string;
      retryDeps: string;
      installingNode: string;
      installingDeps: string;
      starting: string;
      failed: string;
      cancelled: string;
      retry: string;
      continueWithoutNode: string;
      logTitle: string;
      cancel: string;
      nodeLabel: string;
      npmLabel: string;
      dependenciesLabel: string;
      lastChecked: string;
      ready: string;
      unavailable: string;
      needsAttention: string;
    };
  };
  settings: {
    generalTab: string;
    advancedTab: string;
    diagnosticsTab: string;
    aboutTab: string;
    language: string;
    languageDescription: string;
    languageLegend: string;
    appearance: string;
    appearanceDescription: string;
    lightTheme: string;
    darkTheme: string;
    systemTheme: string;
    themeUnavailable: string;
    windowBehavior: string;
    windowBehaviorDescription: string;
    launchOnLogin: string;
    launchOnLoginDescription: string;
    closeToTray: string;
    closeToTrayDescription: string;
    desktopPreferencesFailed: string;
    savedAutomatically: string;
    hermesSourcesTitle: string;
    hermesSourcesDescription: string;
    hermesSourcesChinaDefault: string;
    bundledFirstTitle: string;
    bundledFirstDescription: string;
    artifactSourcesTitle: string;
    artifactSourcesDescription: string;
    dependencySourcesTitle: string;
    dependencySourcesDescription: string;
    hermesArchiveLabel: string;
    hermesArchiveHelp: string;
    nodeArchiveLabel: string;
    nodeArchiveHelp: string;
    npmArchiveLabel: string;
    npmArchiveHelp: string;
    pythonIndexLabel: string;
    pythonIndexHelp: string;
    npmRegistryLabel: string;
    npmRegistryHelp: string;
    sourcePolicyHint: string;
    sourcesLoading: string;
    sourcesSaved: string;
    sourcesInvalid: string;
    sourcesFailed: string;
    restoreChinaDefaults: string;
    saveSources: string;
    sourcesSaving: string;
  };
  instances: {
    unsupportedTitle: string;
    unsupportedDescription: string;
    loading: string;
    refresh: string;
    allFilter: string;
    searchLabel: string;
    searchPlaceholder: string;
    noMatches: string;
    totalCount: string;
    lastSynced: string;
    freshnessUnknown: string;
    emptyNamed: string;
    defaultLabel: string;
    protectedLabel: string;
    availability: Record<"AVAILABLE" | "MISSING" | "UNKNOWN", string>;
    availabilityHint: Record<"AVAILABLE" | "MISSING" | "UNKNOWN", string>;
    lifecycleUnavailable: string;
    lifecycleReady: string;
    lifecycleRunning: string;
    lifecycleStopped: string;
    lifecycleUnknown: string;
    lifecycleStarting: string;
    lifecycleStopping: string;
    lifecycleRestarting: string;
    lifecycleStart: string;
    lifecycleStop: string;
    lifecycleRestart: string;
    lifecycleFailed: string;
    lifecycleConfirmTitle: string;
    lifecycleStopWarning: string;
    lifecycleRestartWarning: string;
    lifecycleConfirm: string;
    loadFailure: string;
    queryFailed: string;
    createLabel: string;
    createDescription: string;
    createPlaceholder: string;
    createAction: string;
    createPending: string;
    createRunning: string;
    createSucceeded: string;
    createFailed: string;
    createInvalid: string;
    cancelCreate: string;
    deleteAction: string;
    deleteTitle: string;
    deleteWarning: string;
    deleteConfirmLabel: string;
    deletePending: string;
    deleteRunning: string;
    deleteSucceeded: string;
    deleteFailed: string;
    cancelDelete: string;
    dismissDelete: string;
    tableInstance: string;
    tableAvailability: string;
    tableLastSynced: string;
    tableCapabilities: string;
    tableActions: string;
    instanceCapability: string;
    lifecycleCapability: string;
    capabilityAvailable: string;
    capabilityUnavailable: string;
    moreActions: string;
  };
  channels: {
    open: string;
    close: string;
    title: string;
    description: string;
    gatewayState: string;
    gatewayRunning: string;
    gatewayStopped: string;
    gatewayUnknown: string;
    independenceNote: string;
    weixin: string;
    weixinDescription: string;
    wecom: string;
    wecomDescription: string;
    connect: string;
    disconnect: string;
    retry: string;
    cancel: string;
    dismiss: string;
    refresh: string;
    loading: string;
    requestFailed: string;
    botId: string;
    botIdPlaceholder: string;
    secret: string;
    secretPlaceholder: string;
    qrTitle: string;
    qrDescription: string;
    qrWaiting: string;
    qrExpired: string;
    expiresIn: string;
    failureTitle: string;
    cancelledTitle: string;
    failureReason: string;
    errorCode: string;
    unknownFailure: string;
    failureMessages: Record<string, string>;
    safeIdentity: string;
    noIdentity: string;
    pairingTitle: string;
    pairingDescription: string;
    pairingChecking: string;
    pendingPairings: string;
    noPendingPairings: string;
    pairingCode: string;
    pairingCodePlaceholder: string;
    approvePairing: string;
    approvingPairing: string;
    pairingApproved: string;
    pairingInvalid: string;
    pairingLocked: string;
    pairingCheckFailed: string;
    pairingApprovalFailed: string;
    pairingRefresh: string;
    connecting: string;
	disconnecting: string;
    revocationNote: string;
    disconnectConfirm: string;
    state: Record<"NOT_CONFIGURED" | "CONNECTING" | "CONNECTED" | "DISCONNECTED" | "FAILED" | "CANCELLED" | "UNKNOWN", string>;
  };
  models: {
    open: string;
    close: string;
    title: string;
    description: string;
    china: string;
    global: string;
    provider: string;
    model: string;
    modelHint: string;
    apiKey: string;
    apiKeyPlaceholder: string;
    credentialConfigured: string;
    credentialMissing: string;
    save: string;
    saving: string;
    saved: string;
    deleteCredential: string;
    deleted: string;
    testConnection: string;
    testing: string;
    cancelTest: string;
    loading: string;
    unavailable: string;
    unsupported: string;
    requestFailed: string;
    observedAt: string;
    validationAt: string;
    errorCode: string;
    validationAdvice: string;
    providerHelp: Record<"qwen" | "glm", string>;
    configState: Record<"UNCONFIGURED" | "CONFIGURED", string>;
    validationState: Record<"NOT_RUN" | "PASSED" | "FAILED" | "UNKNOWN", string>;
  };
};

const english: Messages = {
  languageName: "English",
  languageShort: "EN",
  brandTagline: "Local-first AI runtime control",
  navigationLabel: "Primary navigation",
  pages: {
    dashboard: {
      navigation: "Dashboard",
      title: "Runtime Overview",
      description: "Monitor your local node and Hermes runtime status",
    },
    runtimes: {
      navigation: "Runtimes",
      title: "Runtimes",
      description: "Discover the official Hermes CLI and verify version compatibility.",
    },
    instances: {
      navigation: "Instances",
      title: "Instances",
      description: "List Hermes Profiles as Yorva Instances. Availability is not Agent, model, or login readiness.",
    },
    settings: {
      navigation: "Settings",
      title: "Settings",
      description: "Choose how Yorva appears on this device.",
    },
  },
  switchLanguage: "Switch language",
  versionUnavailable: "yorvad unavailable",
  dashboard: {
    connectionState: "Connection State",
    nodeInfo: "Node Information",
    systemPlatform: "System & Platform",
    discoveryTitle: "Hermes Runtime Discovery",
    discoveryNotice: "Hermes is required for Runtime capabilities. Install a supported official version to continue.",
    supportedStatuses: "Supported Statuses",
    statusInstalled: "Installed",
    statusUnsupported: "Unsupported",
    statusChecking: "Checking",
    statusBroken: "Broken",
    unavailableValue: "—",
  },
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
    summaryDescription: "Hermes is a local AI agent runtime for models, tools, and messaging workflows.",
    openRuntimes: "View details",
    checking: "Checking Hermes",
    checkingDescription: "Looking for a safe local Hermes CLI and checking its version.",
    cancelled: "Check cancelled",
    cancelledDescription: "The Hermes check was cancelled. No Runtime state was changed.",
    unavailable: "Discovery unavailable",
    unavailableDescription: "Yorva could not complete Hermes discovery.",
    cancel: "Cancel",
    retry: "Retry",
    checkAgain: "Check again",
    version: "Version",
    executable: "Executable",
    compatibility: "Compatibility",
    supportedRange: "Supported range",
    lastChecked: "Last checked",
    managedInstances: "Managed instances",
    viewInstances: "View instances",
    unavailableValue: "—",
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
        description: "The detected Hermes version is compatible with Yorva.",
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
        description: "Hermes returned a version that Yorva could not safely interpret.",
      },
      TIMED_OUT: {
        title: "Hermes check timed out",
        description: "Hermes did not report its version before the discovery deadline.",
      },
      AMBIGUOUS: {
        title: "Multiple Hermes executables found",
        description: "Yorva found more than one runnable Hermes executable and did not choose between them.",
      },
    },
    warningMessages: {
      CANDIDATE_LIMIT_REACHED: "Additional Hermes executable candidates were not evaluated.",
      MULTIPLE_RUNNABLE_CANDIDATES: "Multiple runnable Hermes executables were found; none was selected.",
      OTHER_CANDIDATES_UNUSABLE: "Other Hermes candidates could not be used.",
      PRERELEASE_UNTESTED: "The detected Hermes prerelease is outside the tested compatibility range.",
      HERMES_CLI_LAUNCHER_MISSING: "The Hermes installation does not contain a safe CLI launcher.",
      HERMES_LAUNCHER_ALIAS: "Official Hermes launchers in bin and venv are the same installation.",
    },
    install: {
      action: "Install Hermes",
      confirmTitle: "Install official Hermes",
      confirmDescription: "Review the official source and host changes before Yorva starts the installation Operation.",
      source: "Official source",
      version: "Hermes version",
      destination: "User-scope destination",
      hostChanges: "This official installer may:",
      hostChangeItems: [
        "Use the verified Hermes, Node.js and npm files packaged in this Yorva installer when present",
        "Use the configured fallback sources only when a corresponding packaged file is absent",
        "Create an isolated Python environment and install Node dependencies",
        "Install Hermes-managed uv and PortableGit when needed",
        "Use approved Windows package sources for reviewed prerequisites",
        "Create official Hermes bootstrap and config-template directories",
        "Add only the official Hermes launcher directory to this user's PATH",
        "Set this user's HERMES_HOME",
        "Preserve an existing Hermes .env or config.yaml rather than overwrite it",
      ],
      bundledSourceNote: "Bundled source prepared; dependencies may still require network.",
      sourceNotes: {
        HERMES_SOURCE_OFFICIAL_UNAVAILABLE: "Official source download unavailable.",
        HERMES_SOURCE_BUNDLED_USED: "Verified bundled source used.",
        HERMES_SOURCE_PREPARED: "Bundled source prepared; dependencies may still require network.",
      },
      noProfileNote: "Yorva will not configure a model, API key, profile or messaging channel.",
      confirm: "Install",
      back: "Back",
      unavailable: "Hermes installation is available on Windows only.",
      blocked: "Installation is not available because the current Hermes state must be resolved first.",
      running: "Installing Hermes",
      cancelling: "Cancelling installation",
      cancelled: "Installation cancelled",
      failed: "Installation failed",
      interrupted: "Installation was interrupted",
      succeeded: "Hermes installation succeeded",
      stage: "Current stage",
      errorCode: "Error code",
      correlation: "Correlation ID",
      logTitle: "Install log",
      logHint: "Install log on this computer: %APPDATA%\\com.yorva.desktop.dev\\logs\\install.ndjson",
      copyLog: "Copy log",
      logCopied: "Copied",
      retryInstall: "Retry installation",
      cancelInstall: "Cancel installation",
    },
    prerequisites: {
      title: "Node.js / npm components",
      nodeReady: "Node.js is ready",
      nodeMissing: "Node.js was not detected",
      nodeUnsupported: "The Node.js version is not supported",
      npmUnsupported: "The npm version is not supported",
      depsNotInstalled: "Node dependencies are not installed",
      depsFailed: "Node dependency installation failed",
      installAction: "Install or reinstall Node.js / npm",
      retryDeps: "Retry Node dependencies",
      installingNode: "Installing Node.js",
      installingDeps: "Installing Node dependencies",
      starting: "Starting Node.js / npm installation",
      failed: "Node.js / npm installation failed",
      cancelled: "Node.js / npm installation was cancelled",
      retry: "Retry Node.js / npm installation",
      continueWithoutNode: "Node.js / npm can be installed before Hermes. That step installs only the managed Node and npm. Install Hermes after it finishes; the two Operations cannot run at the same time. If this stays on Installing, cancel first. Do not wait on the official hermes command in a terminal; close that session with Ctrl+C.",
      logTitle: "Install log",
      cancel: "Cancel",
      nodeLabel: "Node.js",
      npmLabel: "npm",
      dependenciesLabel: "Hermes Node dependencies",
      lastChecked: "Last checked",
      ready: "Ready",
      unavailable: "Not ready",
      needsAttention: "Needs attention",
    },
  },
  settings: {
    generalTab: "General",
    advancedTab: "Advanced",
    diagnosticsTab: "Diagnostics",
    aboutTab: "About",
    language: "Language",
    languageDescription: "Yorva applies your language choice immediately and remembers it on this device.",
    languageLegend: "Interface language",
    appearance: "Appearance",
    appearanceDescription: "Choose the interface appearance. Yorva currently supports the light theme.",
    lightTheme: "Light",
    darkTheme: "Dark",
    systemTheme: "System",
    themeUnavailable: "This appearance option is not available yet.",
    windowBehavior: "Window behavior",
    windowBehaviorDescription: "Choose how Yorva starts and what happens when its main window is closed.",
    launchOnLogin: "Launch Yorva when I sign in",
    launchOnLoginDescription: "The packaged app starts hidden in the system tray without starting a Hermes instance.",
    closeToTray: "Minimize to tray when closing",
    closeToTrayDescription: "Closing the main window keeps Yorva available in the system tray.",
    desktopPreferencesFailed: "Yorva could not update the window behavior setting. Please try again.",
    savedAutomatically: "Saved automatically",
    hermesSourcesTitle: "Hermes download sources",
    hermesSourcesDescription: "Configure the artifact fallbacks and package registries used by new Hermes installation operations.",
    hermesSourcesChinaDefault: "Mainland China defaults",
    bundledFirstTitle: "Verified bundled artifacts are used first",
    bundledFirstDescription: "The packaged Hermes, Node.js, and npm archives are preferred. Artifact URLs below are used only when the corresponding bundled file is unavailable.",
    artifactSourcesTitle: "Artifact fallback URLs",
    artifactSourcesDescription: "Downloaded archives must still match Yorva's pinned size and SHA-256 before they can be extracted.",
    dependencySourcesTitle: "Dependency registries",
    dependencySourcesDescription: "These registries are applied to Python/uv/pip and npm dependency work for the next operation.",
    hermesArchiveLabel: "Hermes source archive",
    hermesArchiveHelp: "Pinned Hermes 0.20.2 commit archive fallback.",
    nodeArchiveLabel: "Node.js archive",
    nodeArchiveHelp: "Pinned Node.js 22.23.1 Windows x64 archive fallback.",
    npmArchiveLabel: "npm archive",
    npmArchiveHelp: "Pinned npm 12.0.2 tarball fallback.",
    pythonIndexLabel: "Python package index",
    pythonIndexHelp: "Used by uv and pip. The default is the Tsinghua TUNA PyPI mirror.",
    npmRegistryLabel: "npm registry",
    npmRegistryHelp: "Used for Hermes Node dependencies. The default is npmmirror.",
    sourcePolicyHint: "HTTPS URLs without usernames, passwords, query strings, or fragments are accepted. Changes apply only to operations started after saving.",
    sourcesLoading: "Loading source settings…",
    sourcesSaved: "Download sources saved.",
    sourcesInvalid: "Enter a credential-free HTTPS URL in every field.",
    sourcesFailed: "Yorva could not read or save the download sources. Please try again.",
    restoreChinaDefaults: "Restore China defaults",
    saveSources: "Save changes",
    sourcesSaving: "Saving…",
  },
  instances: {
    unsupportedTitle: "Instances unavailable",
    unsupportedDescription: "Instance management is available only when Hermes discovery is SUPPORTED.",
    loading: "Refreshing instance inventory",
    refresh: "Refresh",
    allFilter: "All",
    searchLabel: "Search instances",
    searchPlaceholder: "Search by name or instance ID",
    noMatches: "No instances match the current filters.",
    totalCount: "{count} instances",
    lastSynced: "Last successful sync",
    freshnessUnknown: "The latest Hermes query did not succeed. Showing last known rows as unknown, not deleted.",
    emptyNamed: "No named Instances yet. The built-in default Profile remains visible and protected.",
    defaultLabel: "Default",
    protectedLabel: "Protected",
    availability: {
      AVAILABLE: "Available",
      MISSING: "Deleted",
      UNKNOWN: "Unknown",
    },
    availabilityHint: {
      AVAILABLE: "Present in the latest successful Hermes query. This is not login, model, or process readiness.",
      MISSING: "This instance was deleted. The Yorva identity is retained as a tombstone.",
      UNKNOWN: "The latest Hermes query failed or could not be parsed. Previous rows were not marked deleted.",
    },
    lifecycleUnavailable: "Lifecycle controls are unavailable for this Runtime.",
    lifecycleReady: "Manual Start, Stop, and Restart are available. Login auto-start is not changed.",
    lifecycleRunning: "Running",
    lifecycleStopped: "Stopped",
    lifecycleUnknown: "Unknown",
    lifecycleStarting: "Starting",
    lifecycleStopping: "Stopping",
    lifecycleRestarting: "Restarting",
    lifecycleStart: "Start",
    lifecycleStop: "Stop",
    lifecycleRestart: "Restart",
    lifecycleFailed: "Lifecycle action failed. Refresh the authoritative state and retry.",
    lifecycleConfirmTitle: "Confirm lifecycle action",
    lifecycleStopWarning: "Stopping this instance may interrupt active work. Continue?",
    lifecycleRestartWarning: "Restarting this instance may interrupt active work. Continue?",
    lifecycleConfirm: "Continue",
    loadFailure: "Yorva could not load Instances.",
    queryFailed: "Hermes Profile query failed. Inventory freshness is unknown.",
    createLabel: "New instance name",
    createDescription: "Initialize an isolated Hermes Profile. Model configuration can be added after creation.",
    createPlaceholder: "coder",
    createAction: "Create instance",
    createPending: "Create queued",
    createRunning: "Creating instance",
    createSucceeded: "Instance created",
    createFailed: "Instance create failed",
    createInvalid: "Use a lowercase letter, then letters, digits, _ or -.",
    cancelCreate: "Cancel create",
    deleteAction: "Delete",
    deleteTitle: "Delete instance",
    deleteWarning: "This permanently deletes Hermes-owned profile data. The Yorva identity is kept as deleted. The default profile cannot be deleted.",
    deleteConfirmLabel: "Type the instance name to confirm",
    deletePending: "Delete queued",
    deleteRunning: "Deleting instance",
    deleteSucceeded: "Instance deleted",
    deleteFailed: "Instance delete failed",
    cancelDelete: "Cancel delete",
    dismissDelete: "Cancel",
    tableInstance: "Instance",
    tableAvailability: "Status",
    tableLastSynced: "Last synced",
    tableCapabilities: "Capabilities",
    tableActions: "Actions",
    instanceCapability: "Instance management",
    lifecycleCapability: "Lifecycle",
    capabilityAvailable: "Available",
    capabilityUnavailable: "Not available",
    moreActions: "More actions",
  },
  channels: {
    open: "Channels",
    close: "Close channels",
    title: "Messaging channels",
    description: "Connect Weixin or WeCom to this exact Hermes instance.",
    gatewayState: "Gateway",
    gatewayRunning: "Running",
    gatewayStopped: "Stopped",
    gatewayUnknown: "Unknown",
    independenceNote: "Channel connection and gateway runtime are independent states. A connected channel does not mean the gateway is running.",
    weixin: "Weixin",
    weixinDescription: "Scan an expiring iLink QR code with Weixin to authorize this instance.",
    wecom: "WeCom",
    wecomDescription: "Enter the Bot ID and Secret from WeCom. Yorva verifies them before saving.",
    connect: "Connect",
    disconnect: "Disconnect locally",
    retry: "Retry",
    cancel: "Cancel",
    dismiss: "Close",
    refresh: "Refresh",
    loading: "Loading channel state",
    requestFailed: "The channel request could not be completed.",
    botId: "Bot ID",
    botIdPlaceholder: "Enter the WeCom Bot ID",
    secret: "Secret",
    secretPlaceholder: "Enter the WeCom Secret",
    qrTitle: "Scan with Weixin",
    qrDescription: "Keep this window open and confirm the login in Weixin.",
    qrWaiting: "Preparing the expiring QR code…",
    qrExpired: "This QR code has expired. Cancel and retry.",
    expiresIn: "Expires in {seconds}s",
    failureTitle: "Connection failed",
    cancelledTitle: "Connection cancelled",
    failureReason: "Reason",
    errorCode: "Error code",
    unknownFailure: "The channel operation could not be completed.",
    failureMessages: {
      CHANNEL_AUTH_FAILED: "Weixin or WeCom rejected the authentication request. Check the account or credential and retry.",
      CHANNEL_AUTH_TIMEOUT: "The authorization timed out before it was confirmed. Generate a new QR code and retry.",
      CHANNEL_AUTH_CANCELLED: "The authorization was cancelled before it completed.",
      CHANNEL_STATE_UNKNOWN: "Yorva could not confirm the final channel state. Refresh the channel state before retrying.",
      CHANNEL_DISCONNECT_FAILED: "The local channel binding could not be removed.",
      CHANNEL_DEPENDENCY_MISSING: "A required Hermes channel dependency is unavailable.",
      CHANNEL_NOT_SUPPORTED: "This Hermes installation does not support the requested channel.",
      CHANNEL_CONFLICT: "Another operation for this instance is still running.",
      OPERATION_INTERRUPTED: "The operation was interrupted when the local service restarted.",
    },
    safeIdentity: "Connected identity",
    noIdentity: "No verified identity",
    pairingTitle: "Sender pairing",
    pairingDescription: "When Weixin sends a pairing code, enter it here to approve this sender for the current instance.",
    pairingChecking: "Checking pending requests…",
    pendingPairings: "{count} pending pairing request(s)",
    noPendingPairings: "No pending pairing requests",
    pairingCode: "Pairing code",
    pairingCodePlaceholder: "8-character code",
    approvePairing: "Approve",
    approvingPairing: "Approving…",
    pairingApproved: "Sender approved. New messages can now reach Hermes.",
    pairingInvalid: "The code is invalid, expired, or does not match a pending request.",
    pairingLocked: "Approval is temporarily locked after too many failed attempts. Wait before retrying.",
    pairingCheckFailed: "Pending pairing requests could not be checked. Refresh and try again.",
    pairingApprovalFailed: "The pairing request could not be approved. Refresh and try again.",
    pairingRefresh: "Refresh pairing requests",
    connecting: "Connecting",
	disconnecting: "Disconnecting",
    revocationNote: "Disconnect removes only this instance's local Hermes binding. It does not delete or revoke the remote account or bot identity.",
    disconnectConfirm: "Remove this channel's local binding? The remote account or bot will not be deleted.",
    state: {
      NOT_CONFIGURED: "Not configured",
      CONNECTING: "Connecting",
      CONNECTED: "Connected",
      DISCONNECTED: "Disconnected",
      FAILED: "Failed",
      CANCELLED: "Cancelled",
      UNKNOWN: "Unknown",
    },
  },
  models: {
    open: "Models",
    close: "Close models",
    title: "Add model Provider",
    description: "Choose a qualified Provider and model for this Instance. Saving does not contact the Provider.",
    china: "Recommended in China",
    global: "Other compatible Providers",
    provider: "Provider",
    model: "Model",
    modelHint: "Choose a recommendation or enter a reviewed model ID.",
    apiKey: "API Key",
    apiKeyPlaceholder: "Enter a new key to save or replace it",
    credentialConfigured: "Credential configured",
    credentialMissing: "Credential not configured",
    save: "Save configuration",
    saving: "Saving configuration",
    saved: "Configuration saved",
    deleteCredential: "Delete credential",
    deleted: "Credential deleted",
    testConnection: "Test connection",
    testing: "Testing connection",
    cancelTest: "Cancel test",
    loading: "Loading model configuration",
    unavailable: "Models are disabled because this Instance is not available.",
    unsupported: "Models are disabled because this Hermes version does not support the qualified model surface.",
    requestFailed: "The model request could not be completed.",
    observedAt: "Observed",
    validationAt: "Last tested",
    errorCode: "Error code",
    validationAdvice: "Check the Provider key and model access, then run the test again.",
    providerHelp: {
      qwen: "Hermes 0.20.2 uses the DashScope international compatible endpoint.",
      glm: "Hermes selects the qualified Z.AI endpoint for this provider.",
    },
    configState: { UNCONFIGURED: "Unconfigured", CONFIGURED: "Configured" },
    validationState: { NOT_RUN: "Not tested", PASSED: "Passed", FAILED: "Failed", UNKNOWN: "Unknown" },
  },
};

const simplifiedChinese: Messages = {
  languageName: "简体中文",
  languageShort: "中文",
  brandTagline: "本地优先的 AI 运行引擎控制台",
  navigationLabel: "主导航",
  pages: {
    dashboard: {
      navigation: "仪表盘",
      title: "运行引擎总览",
      description: "查看本地节点与 Hermes 运行引擎状态",
    },
    runtimes: {
      navigation: "运行引擎",
      title: "运行引擎",
      description: "检测官方 Hermes CLI 并核验版本兼容性。",
    },
    instances: {
      navigation: "实例",
      title: "实例",
      description: "将 Hermes Profile 列为 Yorva 实例。可用状态不代表 Agent、模型或登录已就绪。",
    },
    settings: {
      navigation: "设置",
      title: "设置",
      description: "选择 Yorva 在此设备上的显示方式。",
    },
  },
  switchLanguage: "切换语言",
  versionUnavailable: "yorvad 不可用",
  dashboard: {
    connectionState: "连接状态",
    nodeInfo: "节点信息",
    systemPlatform: "系统与平台",
    discoveryTitle: "Hermes 运行引擎检测",
    discoveryNotice: "完整运行引擎能力需要 Hermes。请安装受支持的官方版本后再继续。",
    supportedStatuses: "支持的状态",
    statusInstalled: "已安装",
    statusUnsupported: "不受支持",
    statusChecking: "检测中",
    statusBroken: "已损坏",
    unavailableValue: "—",
  },
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
    summaryTitle: "Hermes 运行引擎",
    summaryDescription: "Hermes 是用于模型、工具与消息工作流的本地 AI Agent 运行引擎。",
    openRuntimes: "查看详细",
    checking: "正在检测 Hermes",
    checkingDescription: "正在查找安全的本地 Hermes CLI 并检查版本。",
    cancelled: "检测已取消",
    cancelledDescription: "Hermes 检测已取消，未修改任何运行引擎状态。",
    unavailable: "检测不可用",
    unavailableDescription: "Yorva 无法完成 Hermes 检测。",
    cancel: "取消",
    retry: "重试",
    checkAgain: "重新检测",
    version: "版本",
    executable: "可执行文件",
    compatibility: "兼容性",
    supportedRange: "支持范围",
    lastChecked: "上次检测",
    managedInstances: "管理实例",
    viewInstances: "查看实例",
    unavailableValue: "—",
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
        description: "检测到的 Hermes 版本与 Yorva 兼容。",
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
        description: "Hermes 返回的版本信息无法被 Yorva 安全解析。",
      },
      TIMED_OUT: {
        title: "Hermes 检测超时",
        description: "Hermes 未在检测时限内返回版本信息。",
      },
      AMBIGUOUS: {
        title: "发现多个 Hermes 可执行文件",
        description: "Yorva 发现多个可运行的 Hermes，未擅自选择其中任何一个。",
      },
    },
    warningMessages: {
      CANDIDATE_LIMIT_REACHED: "还有其他 Hermes 候选项未被检测。",
      MULTIPLE_RUNNABLE_CANDIDATES: "发现多个可运行的 Hermes，未选择任何一个。",
      OTHER_CANDIDATES_UNUSABLE: "其他 Hermes 候选项无法使用。",
      PRERELEASE_UNTESTED: "检测到的 Hermes 预发布版本不在已测试的兼容范围内。",
      HERMES_CLI_LAUNCHER_MISSING: "Hermes 安装中缺少安全的 CLI 启动器。",
      HERMES_LAUNCHER_ALIAS: "官方 bin 与 venv 中的 Hermes 启动器属于同一安装。",
    },
    install: {
      action: "安装 Hermes",
      confirmTitle: "安装官方 Hermes",
      confirmDescription: "在 Yorva 开始安装操作前，请确认官方来源和本机将发生的变更。",
      source: "官方来源",
      version: "Hermes 版本",
      destination: "当前用户安装位置",
      hostChanges: "该官方安装程序可能会：",
      hostChangeItems: [
        "优先使用本安装包内已校验的 Hermes、Node.js 和 npm 文件",
        "仅在对应内置文件缺失时使用高级设置中的备用来源",
        "创建隔离的 Python 环境并安装 Node 依赖",
        "在需要时安装 Hermes 管理的 uv 和 PortableGit",
        "通过已审核的 Windows 软件源安装必要前置组件",
        "创建官方 Hermes 引导和配置模板目录",
        "仅将官方 Hermes 启动器目录加入当前用户 PATH",
        "设置当前用户的 HERMES_HOME",
        "保留已有的 Hermes .env 或 config.yaml，而不会覆盖",
      ],
      bundledSourceNote: "已准备内置源码；依赖安装仍可能需要网络。",
      sourceNotes: {
        HERMES_SOURCE_OFFICIAL_UNAVAILABLE: "官方源码下载不可用。",
        HERMES_SOURCE_BUNDLED_USED: "已使用经验证的内置源码。",
        HERMES_SOURCE_PREPARED: "已准备内置源码；依赖安装仍可能需要网络。",
      },
      noProfileNote: "Yorva 不会配置模型、API 密钥、配置档案或消息通道。",
      confirm: "安装",
      back: "返回",
      unavailable: "Hermes 安装目前仅支持 Windows。",
      blocked: "当前 Hermes 状态需要先处理，因此无法安装。",
      running: "正在安装 Hermes",
      cancelling: "正在取消安装",
      cancelled: "安装已取消",
      failed: "安装失败",
      interrupted: "安装被中断",
      succeeded: "Hermes 安装成功",
      stage: "当前阶段",
      errorCode: "错误码",
      correlation: "关联 ID",
      logTitle: "安装日志",
      logHint: "本机安装日志：%APPDATA%\\com.yorva.desktop.dev\\logs\\install.ndjson",
      copyLog: "复制日志",
      logCopied: "已复制",
      retryInstall: "重试安装",
      cancelInstall: "取消安装",
    },
    prerequisites: {
      title: "Node.js / npm 组件",
      nodeReady: "Node.js 已就绪",
      nodeMissing: "未检测到 Node.js",
      nodeUnsupported: "Node.js 版本不受支持",
      npmUnsupported: "npm 版本不受支持",
      depsNotInstalled: "Node 依赖尚未安装",
      depsFailed: "Node 依赖安装失败",
      installAction: "重新安装 Node.js / npm",
      retryDeps: "重试 Node 依赖",
      installingNode: "正在安装 Node.js",
      installingDeps: "正在安装 Node 依赖",
      starting: "正在启动 Node.js / npm 安装",
      failed: "Node.js / npm 安装失败",
      cancelled: "Node.js / npm 安装已取消",
      retry: "重试 Node.js / npm 安装",
      continueWithoutNode: "可以先安装 Node.js / npm，这一步只安装托管的 Node 和 npm。完成后再安装 Hermes；这两项操作不能同时进行。若一直停在「正在安装」，请先取消。不要在终端里等待官方 hermes 命令；用 Ctrl+C 结束那个会话。",
      logTitle: "安装日志",
      cancel: "取消",
      nodeLabel: "Node.js",
      npmLabel: "npm",
      dependenciesLabel: "Hermes Node 依赖",
      lastChecked: "上次检查",
      ready: "就绪",
      unavailable: "未就绪",
      needsAttention: "需要处理",
    },
  },
  settings: {
    generalTab: "通用",
    advancedTab: "高级",
    diagnosticsTab: "诊断",
    aboutTab: "关于",
    language: "语言",
    languageDescription: "Yorva 会立即应用语言选择，并在此设备上记住该设置。",
    languageLegend: "界面语言",
    appearance: "外观主题",
    appearanceDescription: "选择界面外观主题。Yorva 当前支持浅色模式。",
    lightTheme: "浅色",
    darkTheme: "深色",
    systemTheme: "跟随系统",
    themeUnavailable: "该外观选项暂未开放。",
    windowBehavior: "窗口行为",
    windowBehaviorDescription: "设置 Yorva 的启动方式以及关闭主窗口时的行为。",
    launchOnLogin: "登录后自动启动 Yorva",
    launchOnLoginDescription: "安装版将隐藏启动到系统托盘，不会自动启动 Hermes 实例。",
    closeToTray: "关闭时最小化到托盘",
    closeToTrayDescription: "关闭主窗口后 Yorva 继续在系统托盘中运行。",
    desktopPreferencesFailed: "无法更新窗口行为设置，请重试。",
    savedAutomatically: "已自动保存",
    hermesSourcesTitle: "Hermes 下载与依赖源",
    hermesSourcesDescription: "配置后续 Hermes 安装操作使用的安装包备用地址和依赖仓库。",
    hermesSourcesChinaDefault: "中国大陆默认配置",
    bundledFirstTitle: "优先使用已校验的内置安装包",
    bundledFirstDescription: "安装版会优先使用内置的 Hermes、Node.js 和 npm 文件；只有对应内置文件不可用时，才会访问下方备用地址。",
    artifactSourcesTitle: "安装包备用地址",
    artifactSourcesDescription: "下载后的文件仍须匹配 Yorva 固定的大小和 SHA-256，校验通过后才能解压。",
    dependencySourcesTitle: "依赖仓库",
    dependencySourcesDescription: "这些地址会用于下一次安装中的 Python/uv/pip 与 npm 依赖下载。",
    hermesArchiveLabel: "Hermes 源码归档",
    hermesArchiveHelp: "固定 Hermes 0.20.2 提交归档的备用地址。",
    nodeArchiveLabel: "Node.js 安装包",
    nodeArchiveHelp: "固定 Node.js 22.23.1 Windows x64 压缩包的备用地址。",
    npmArchiveLabel: "npm 安装包",
    npmArchiveHelp: "固定 npm 12.0.2 压缩包的备用地址。",
    pythonIndexLabel: "Python 软件源",
    pythonIndexHelp: "供 uv 和 pip 使用，默认采用清华 TUNA PyPI 镜像。",
    npmRegistryLabel: "npm 软件源",
    npmRegistryHelp: "供 Hermes Node 依赖使用，默认采用 npmmirror。",
    sourcePolicyHint: "仅支持不包含用户名、密码、查询参数或片段的 HTTPS 地址。保存后只影响新启动的安装操作。",
    sourcesLoading: "正在读取下载源配置…",
    sourcesSaved: "下载源配置已保存。",
    sourcesInvalid: "请为每一项填写不含凭据的 HTTPS 地址。",
    sourcesFailed: "无法读取或保存下载源配置，请重试。",
    restoreChinaDefaults: "恢复国内默认值",
    saveSources: "保存更改",
    sourcesSaving: "正在保存…",
  },
  instances: {
    unsupportedTitle: "实例不可用",
    unsupportedDescription: "仅在 Hermes 检测结果为 SUPPORTED 时可以管理实例。",
    loading: "正在刷新实例清单",
    refresh: "刷新",
    allFilter: "全部",
    searchLabel: "搜索实例",
    searchPlaceholder: "按名称或实例 ID 搜索",
    noMatches: "没有符合当前筛选条件的实例。",
    totalCount: "共 {count} 个实例",
    lastSynced: "最近一次成功同步",
    freshnessUnknown: "最近一次 Hermes 查询未成功。正在显示上次已知记录，状态为未知，不是已删除。",
    emptyNamed: "还没有命名实例。内置 default Profile 仍然可见且受保护。",
    defaultLabel: "默认",
    protectedLabel: "受保护",
    availability: {
      AVAILABLE: "可用",
      MISSING: "已删除",
      UNKNOWN: "未知",
    },
    availabilityHint: {
      AVAILABLE: "最近一次成功的 Hermes 查询确认存在。这不代表已登录、已配置模型或进程已就绪。",
      MISSING: "此实例已删除。Yorva 身份会作为墓碑保留。",
      UNKNOWN: "最近一次 Hermes 查询失败或无法解析。旧记录不会被误判为已删除。",
    },
    lifecycleUnavailable: "此 Runtime 不支持生命周期控制。",
    lifecycleReady: "可手动启动、停止和重启；不会更改登录自启设置。",
    lifecycleRunning: "运行中",
    lifecycleStopped: "已停止",
    lifecycleUnknown: "未知",
    lifecycleStarting: "正在启动",
    lifecycleStopping: "正在停止",
    lifecycleRestarting: "重启中",
    lifecycleStart: "启动",
    lifecycleStop: "停止",
    lifecycleRestart: "重启",
    lifecycleFailed: "生命周期操作失败。请刷新权威状态后重试。",
    lifecycleConfirmTitle: "确认生命周期操作",
    lifecycleStopWarning: "停止此实例可能会中断正在进行的任务。是否继续？",
    lifecycleRestartWarning: "重启此实例可能会中断正在进行的任务。是否继续？",
    lifecycleConfirm: "继续",
    loadFailure: "无法加载实例。",
    queryFailed: "Hermes Profile 查询失败。清单新鲜度为未知。",
    createLabel: "新实例名称",
    createDescription: "初始化一个隔离的 Hermes Profile。创建后可继续配置模型。",
    createPlaceholder: "coder",
    createAction: "创建实例",
    createPending: "创建已排队",
    createRunning: "正在创建实例",
    createSucceeded: "实例已创建",
    createFailed: "创建实例失败",
    createInvalid: "首字符必须是小写字母，后续只能是小写字母、数字、_ 或 -。",
    cancelCreate: "取消创建",
    deleteAction: "删除",
    deleteTitle: "删除实例",
    deleteWarning: "这将永久删除 Hermes 所有的 Profile 数据。Yorva 身份会保留为已删除。default 不可删除。",
    deleteConfirmLabel: "输入实例名称以确认",
    deletePending: "删除已排队",
    deleteRunning: "正在删除实例",
    deleteSucceeded: "实例已删除",
    deleteFailed: "删除实例失败",
    cancelDelete: "取消删除",
    dismissDelete: "取消",
    tableInstance: "实例",
    tableAvailability: "状态",
    tableLastSynced: "最近同步",
    tableCapabilities: "能力",
    tableActions: "操作",
    instanceCapability: "实例管理",
    lifecycleCapability: "生命周期",
    capabilityAvailable: "可用",
    capabilityUnavailable: "不可用",
    moreActions: "更多操作",
  },
  channels: {
    open: "通道",
    close: "关闭通道",
    title: "消息通道",
    description: "为当前这一 Hermes 实例连接微信或企业微信。",
    gatewayState: "网关状态",
    gatewayRunning: "运行中",
    gatewayStopped: "已停止",
    gatewayUnknown: "未知",
    independenceNote: "通道连接与网关运行是两个独立状态。通道已连接不代表网关正在运行。",
    weixin: "微信",
    weixinDescription: "使用微信扫描有时效的 iLink 二维码，为此实例授权。",
    wecom: "企业微信",
    wecomDescription: "输入企业微信 Bot ID 和 Secret；Yorva 会先验证，再保存到 Hermes。",
    connect: "连接",
    disconnect: "断开本地连接",
    retry: "重试",
    cancel: "取消",
    dismiss: "关闭",
    refresh: "刷新",
    loading: "正在加载通道状态",
    requestFailed: "无法完成通道请求。",
    botId: "Bot ID",
    botIdPlaceholder: "输入企业微信 Bot ID",
    secret: "Secret",
    secretPlaceholder: "输入企业微信 Secret",
    qrTitle: "使用微信扫码",
    qrDescription: "请保持此窗口打开，并在微信中确认登录。",
    qrWaiting: "正在准备有时效的二维码…",
    qrExpired: "二维码已过期，请取消后重试。",
    expiresIn: "{seconds} 秒后过期",
    failureTitle: "连接失败",
    cancelledTitle: "连接已取消",
    failureReason: "原因",
    errorCode: "错误码",
    unknownFailure: "通道操作未能完成。",
    failureMessages: {
      CHANNEL_AUTH_FAILED: "微信或企业微信拒绝了认证请求，请检查账号或凭据后重试。",
      CHANNEL_AUTH_TIMEOUT: "授权确认已超时，请生成新的二维码后重试。",
      CHANNEL_AUTH_CANCELLED: "授权在完成前被取消。",
      CHANNEL_STATE_UNKNOWN: "Yorva 无法确认通道的最终状态，请先刷新通道状态再重试。",
      CHANNEL_DISCONNECT_FAILED: "无法移除本地通道绑定。",
      CHANNEL_DEPENDENCY_MISSING: "Hermes 所需的通道依赖不可用。",
      CHANNEL_NOT_SUPPORTED: "当前 Hermes 安装不支持此消息通道。",
      CHANNEL_CONFLICT: "此实例仍有另一项操作正在进行。",
      OPERATION_INTERRUPTED: "本地服务重启时中断了此操作。",
    },
    safeIdentity: "已连接身份",
    noIdentity: "尚无已验证身份",
    pairingTitle: "发送者配对",
    pairingDescription: "微信返回配对码后，在这里输入并批准，让该发送者可与当前实例中的 Hermes 对话。",
    pairingChecking: "正在检查待配对请求…",
    pendingPairings: "有 {count} 个待配对请求",
    noPendingPairings: "当前没有待配对请求",
    pairingCode: "配对码",
    pairingCodePlaceholder: "输入 8 位配对码",
    approvePairing: "批准配对",
    approvingPairing: "正在批准…",
    pairingApproved: "发送者已批准，现在发送的新消息可以到达 Hermes。",
    pairingInvalid: "配对码无效、已过期，或与待处理请求不匹配。",
    pairingLocked: "连续失败次数过多，配对批准已暂时锁定，请稍后再试。",
    pairingCheckFailed: "无法检查待配对请求，请刷新后重试。",
    pairingApprovalFailed: "未能批准配对请求，请刷新后重试。",
    pairingRefresh: "刷新配对请求",
    connecting: "正在连接",
	disconnecting: "正在断开",
    revocationNote: "断开连接只会移除此实例在 Hermes 中的本地绑定，不会删除或撤销远端账号或机器人身份。",
    disconnectConfirm: "确定移除此通道的本地绑定吗？远端账号或机器人不会被删除。",
    state: {
      NOT_CONFIGURED: "未配置",
      CONNECTING: "连接中",
      CONNECTED: "已连接",
      DISCONNECTED: "已断开",
      FAILED: "失败",
      CANCELLED: "已取消",
      UNKNOWN: "未知",
    },
  },
  models: {
    open: "模型",
    close: "关闭模型设置",
    title: "添加模型 Provider",
    description: "为此实例选择已验证的 Provider 和模型。保存配置不会请求 Provider。",
    china: "国内推荐",
    global: "其他兼容 Provider",
    provider: "Provider",
    model: "模型",
    modelHint: "选择推荐模型，或输入已审核的 model ID。",
    apiKey: "API Key",
    apiKeyPlaceholder: "输入新密钥以保存或替换",
    credentialConfigured: "凭据已配置",
    credentialMissing: "凭据未配置",
    save: "保存配置",
    saving: "正在保存配置",
    saved: "配置已保存",
    deleteCredential: "删除凭据",
    deleted: "凭据已删除",
    testConnection: "测试连接",
    testing: "正在测试连接",
    cancelTest: "取消测试",
    loading: "正在加载模型配置",
    unavailable: "此实例不可用，模型操作已禁用。",
    unsupported: "当前 Hermes 版本不支持已验证的模型配置接口，模型操作已禁用。",
    requestFailed: "模型请求未能完成。",
    observedAt: "状态时间",
    validationAt: "最近测试",
    errorCode: "错误码",
    validationAdvice: "请检查 Provider 密钥和模型访问权限，然后重新测试。",
    providerHelp: {
      qwen: "Hermes 0.20.2 使用 DashScope 国际兼容端点。",
      glm: "Hermes 会为此 Provider 选择已验证的 Z.AI 端点。",
    },
    configState: { UNCONFIGURED: "未配置", CONFIGURED: "已配置" },
    validationState: { NOT_RUN: "未测试", PASSED: "通过", FAILED: "失败", UNKNOWN: "未知" },
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
