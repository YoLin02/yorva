use std::{
    fs, io,
    path::Path,
    sync::{Condvar, Mutex, MutexGuard},
    thread,
    time::Duration,
};

use base64::{Engine as _, engine::general_purpose::URL_SAFE_NO_PAD};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, Manager};
use tauri_plugin_shell::{
    ShellExt,
    process::{CommandChild, CommandEvent},
};

const PROTOCOL_VERSION: &str = "1";
// Startup includes database recovery and one bounded Runtime discovery/reconcile pass.
// Runtime discovery owns a 35-second deadline, so the Desktop must not abandon the
// daemon while that qualified recovery work is still inside its own bound.
const STARTUP_TIMEOUT: Duration = Duration::from_secs(45);
const SHUTDOWN_TIMEOUT: Duration = Duration::from_secs(3);
const MAX_SKILL_IMPORT_BYTES: u64 = 4 * 1024 * 1024;
const MAX_SKILL_IMPORT_ENTRIES: usize = 160;
const SHUTDOWN_MESSAGE: &[u8] = b"{\"type\":\"shutdown\"}\n";
const STARTUP_FAILED_MESSAGE: &str = "The local daemon could not be started.";

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DaemonSession {
    base_url: String,
    token: String,
    protocol_version: String,
}

#[derive(Clone, Serialize)]
pub struct DaemonCommandError {
    pub(crate) code: &'static str,
    pub(crate) message: &'static str,
    pub(crate) retryable: bool,
}

enum StartupStatus {
    Starting,
    Ready(DaemonSession),
    Failed,
    Stopping,
    Stopped,
}

trait ChildControl: Send {
    fn write(&mut self, bytes: &[u8]) -> io::Result<()>;
    fn kill(self: Box<Self>) -> io::Result<()>;
}

impl ChildControl for CommandChild {
    fn write(&mut self, bytes: &[u8]) -> io::Result<()> {
        CommandChild::write(self, bytes).map_err(|error| io::Error::other(error.to_string()))
    }

    fn kill(self: Box<Self>) -> io::Result<()> {
        (*self)
            .kill()
            .map_err(|error| io::Error::other(error.to_string()))
    }
}

struct LifecycleInner {
    status: StartupStatus,
    child: Option<Box<dyn ChildControl>>,
    failure: Option<DaemonCommandError>,
    restart_attempted: bool,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SkillImportSelection {
    source_ref: String,
    suggested_skill_id: String,
}

pub struct DaemonLifecycle {
    inner: Mutex<LifecycleInner>,
    changed: Condvar,
}

impl DaemonLifecycle {
    pub fn new() -> Self {
        Self {
            inner: Mutex::new(LifecycleInner {
                status: StartupStatus::Starting,
                child: None,
                failure: None,
                restart_attempted: false,
            }),
            changed: Condvar::new(),
        }
    }

    fn lock(&self) -> MutexGuard<'_, LifecycleInner> {
        self.inner
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner())
    }

    fn set_child(&self, child: Box<dyn ChildControl>) {
        let mut inner = self.lock();
        if matches!(inner.status, StartupStatus::Starting) && inner.child.is_none() {
            inner.child = Some(child);
        } else {
            drop(inner);
            let _ = child.kill();
        }
    }

    fn set_ready(&self, session: DaemonSession) {
        let mut inner = self.lock();
        if matches!(inner.status, StartupStatus::Starting) {
            inner.status = StartupStatus::Ready(session);
            self.changed.notify_all();
        }
    }

    fn fail_startup(&self) {
        self.fail_startup_with(DaemonCommandError {
            code: "DAEMON_STARTUP_FAILED",
            message: STARTUP_FAILED_MESSAGE,
            retryable: false,
        });
    }

    pub(crate) fn fail_startup_with(&self, failure: DaemonCommandError) {
        let child = {
            let mut inner = self.lock();
            if matches!(
                inner.status,
                StartupStatus::Stopping | StartupStatus::Stopped
            ) {
                return;
            }
            inner.status = StartupStatus::Failed;
            inner.failure = Some(failure);
            self.changed.notify_all();
            inner.child.take()
        };
        if let Some(child) = child {
            let _ = child.kill();
        }
    }

    fn wait_for_startup_deadline(&self, timeout: Duration) -> bool {
        let inner = self.lock();
        let (mut inner, wait_result) = self
            .changed
            .wait_timeout_while(inner, timeout, |state| {
                matches!(state.status, StartupStatus::Starting)
            })
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        if wait_result.timed_out() && matches!(inner.status, StartupStatus::Starting) {
            log_diagnostic("daemon_startup_timeout", "handshake deadline elapsed");
            inner.status = StartupStatus::Failed;
            let child = inner.child.take();
            self.changed.notify_all();
            drop(inner);
            if let Some(child) = child {
                let _ = child.kill();
            }
            true
        } else {
            false
        }
    }

    fn process_terminated(&self) -> bool {
        let mut inner = self.lock();
        inner.child.take();
        let restart = matches!(inner.status, StartupStatus::Ready(_)) && !inner.restart_attempted;
        if restart {
            inner.status = StartupStatus::Starting;
            inner.failure = None;
            inner.restart_attempted = true;
        } else if matches!(
            inner.status,
            StartupStatus::Stopping | StartupStatus::Stopped
        ) {
            inner.status = StartupStatus::Stopped;
        } else {
            inner.status = StartupStatus::Failed;
        }
        self.changed.notify_all();
        restart
    }

    pub fn stop(&self) {
        let _ = self.stop_with_timeout(SHUTDOWN_TIMEOUT);
    }

    fn stop_with_timeout(&self, timeout: Duration) -> bool {
        let mut inner = self.lock();
        if inner.child.is_none() {
            inner.status = StartupStatus::Stopped;
            self.changed.notify_all();
            return false;
        }

        inner.status = StartupStatus::Stopping;
        let write_failed = inner
            .child
            .as_mut()
            .is_some_and(|child| child.write(SHUTDOWN_MESSAGE).is_err());
        if write_failed {
            let child = inner.child.take();
            inner.status = StartupStatus::Stopped;
            self.changed.notify_all();
            drop(inner);
            if let Some(child) = child {
                let _ = child.kill();
            }
            return true;
        }

        let (mut inner, _) = self
            .changed
            .wait_timeout_while(inner, timeout, |state| state.child.is_some())
            .unwrap_or_else(|poisoned| poisoned.into_inner());
        let child = inner.child.take();
        inner.status = StartupStatus::Stopped;
        self.changed.notify_all();
        drop(inner);

        if let Some(child) = child {
            log_diagnostic("daemon_shutdown_timeout", "forcing child termination");
            let _ = child.kill();
            true
        } else {
            false
        }
    }

    fn session(&self) -> Result<DaemonSession, DaemonCommandError> {
        let inner = self.lock();
        match &inner.status {
            StartupStatus::Ready(session) => Ok(session.clone()),
            StartupStatus::Starting | StartupStatus::Stopping => Err(DaemonCommandError {
                code: "DAEMON_NOT_READY",
                message: "The local daemon is still starting.",
                retryable: true,
            }),
            StartupStatus::Failed => Err(inner.failure.clone().unwrap_or(DaemonCommandError {
                code: "DAEMON_STARTUP_FAILED",
                message: STARTUP_FAILED_MESSAGE,
                retryable: false,
            })),
            StartupStatus::Stopped => Err(DaemonCommandError {
                code: "DAEMON_NOT_READY",
                message: "The local daemon is stopped.",
                retryable: true,
            }),
        }
    }
}

struct StartupChildGuard {
    child: Option<Box<dyn ChildControl>>,
}

impl StartupChildGuard {
    fn new(child: Box<dyn ChildControl>) -> Self {
        Self { child: Some(child) }
    }

    fn write(&mut self, bytes: &[u8]) -> io::Result<()> {
        self.child
            .as_mut()
            .expect("startup child guard must own a child")
            .write(bytes)
    }

    fn take(&mut self) -> Box<dyn ChildControl> {
        self.child
            .take()
            .expect("startup child guard must own a child")
    }
}

impl Drop for StartupChildGuard {
    fn drop(&mut self) {
        if let Some(child) = self.child.take() {
            let _ = child.kill();
        }
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct BootstrapMessage<'a> {
    protocol_version: &'static str,
    token: &'a str,
    data_dir: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    hermes_embedded_source_path: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    hermes_node_archive_path: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    hermes_npm_archive_path: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    hermes_python_archive_path: Option<&'a str>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct Handshake {
    protocol_version: String,
    port: u16,
    pid: u32,
}

#[tauri::command]
pub fn daemon_session(
    lifecycle: tauri::State<'_, DaemonLifecycle>,
) -> Result<DaemonSession, DaemonCommandError> {
    lifecycle.session()
}

#[tauri::command]
pub fn select_skill_import(
    app: AppHandle,
    kind: String,
) -> Result<Option<SkillImportSelection>, DaemonCommandError> {
    let selected = match kind.as_str() {
        "ZIP" => rfd::FileDialog::new()
            .set_title("Install Skill from ZIP")
            .add_filter("Skill ZIP", &["zip"])
            .pick_file(),
        "DIRECTORY" => rfd::FileDialog::new()
            .set_title("Import existing Skill")
            .pick_folder(),
        _ => return Err(skill_import_error()),
    };
    let Some(source) = selected else {
        return Ok(None);
    };
    let source_ref = generate_token().map_err(|_| skill_import_error())?;
    let imports_root = app
        .path()
        .app_data_dir()
        .map_err(|_| skill_import_error())?
        .join("skill-imports");
    let staging = imports_root.join(&source_ref);
    fs::create_dir_all(&staging).map_err(|_| skill_import_error())?;
    let result = if kind == "ZIP" {
        stage_skill_zip(&source, &staging)
    } else {
        stage_skill_directory(&source, &staging.join("directory"))
    };
    if result.is_err() {
        let _ = fs::remove_dir_all(&staging);
        return Err(skill_import_error());
    }
    Ok(Some(SkillImportSelection {
        source_ref,
        suggested_skill_id: suggested_skill_id(&source),
    }))
}

#[tauri::command]
pub fn discard_skill_import(app: AppHandle, source_ref: String) -> Result<(), DaemonCommandError> {
    if source_ref.len() != 43
        || !source_ref
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || byte == b'_' || byte == b'-')
    {
        return Err(skill_import_error());
    }
    let imports_root = app
        .path()
        .app_data_dir()
        .map_err(|_| skill_import_error())?
        .join("skill-imports");
    let staging = imports_root.join(source_ref);
    if staging.parent() != Some(imports_root.as_path()) {
        return Err(skill_import_error());
    }
    match fs::remove_dir_all(staging) {
        Ok(()) => Ok(()),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(()),
        Err(_) => Err(skill_import_error()),
    }
}

fn stage_skill_zip(source: &Path, staging: &Path) -> Result<(), ()> {
    let metadata = fs::symlink_metadata(source).map_err(|_| ())?;
    if !metadata.file_type().is_file()
        || metadata.file_type().is_symlink()
        || metadata.len() == 0
        || metadata.len() > MAX_SKILL_IMPORT_BYTES
    {
        return Err(());
    }
    fs::copy(source, staging.join("source.zip")).map_err(|_| ())?;
    Ok(())
}

fn stage_skill_directory(source: &Path, destination: &Path) -> Result<(), ()> {
    let metadata = fs::symlink_metadata(source).map_err(|_| ())?;
    if !metadata.is_dir() || metadata.file_type().is_symlink() {
        return Err(());
    }
    fs::create_dir(destination).map_err(|_| ())?;
    let mut pending = vec![(source.to_path_buf(), destination.to_path_buf(), 0_usize)];
    let mut entries = 0_usize;
    let mut total = 0_u64;
    while let Some((from, to, depth)) = pending.pop() {
        if depth > 8 {
            return Err(());
        }
        for item in fs::read_dir(&from).map_err(|_| ())? {
            let item = item.map_err(|_| ())?;
            entries += 1;
            if entries > MAX_SKILL_IMPORT_ENTRIES {
                return Err(());
            }
            let item_type = item.file_type().map_err(|_| ())?;
            if item_type.is_symlink() {
                return Err(());
            }
            let target = to.join(item.file_name());
            if item_type.is_dir() {
                fs::create_dir(&target).map_err(|_| ())?;
                pending.push((item.path(), target, depth + 1));
            } else if item_type.is_file() {
                let size = item.metadata().map_err(|_| ())?.len();
                total = total.checked_add(size).ok_or(())?;
                if size > 512 * 1024 || total > MAX_SKILL_IMPORT_BYTES {
                    return Err(());
                }
                fs::copy(item.path(), target).map_err(|_| ())?;
            } else {
                return Err(());
            }
        }
    }
    Ok(())
}

fn suggested_skill_id(source: &Path) -> String {
    let stem = source
        .file_stem()
        .and_then(|value| value.to_str())
        .unwrap_or("imported-skill");
    let mut result = String::new();
    for ch in stem.chars().flat_map(char::to_lowercase) {
        if ch.is_ascii_lowercase() || ch.is_ascii_digit() {
            result.push(ch);
        } else if !result.ends_with('-') {
            result.push('-');
        }
    }
    let result = result.trim_matches('-');
    let mut result = if result
        .chars()
        .next()
        .is_some_and(|ch| ch.is_ascii_lowercase())
    {
        result.to_owned()
    } else {
        format!("skill-{result}")
    };
    result.truncate(64);
    if result.is_empty() {
        "imported-skill".to_owned()
    } else {
        result
    }
}

fn skill_import_error() -> DaemonCommandError {
    DaemonCommandError {
        code: "SKILL_IMPORT_SOURCE_UNAVAILABLE",
        message: "Yorva could not stage the selected Skill source.",
        retryable: true,
    }
}

pub fn start_daemon(app: &AppHandle) {
    if let Err(error) = try_start_daemon(app) {
        log_diagnostic("daemon_startup_failed", &error.to_string());
        app.state::<DaemonLifecycle>().fail_startup();
    }
}

fn try_start_daemon(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let data_dir = app.path().app_data_dir()?;
    match fs::remove_dir_all(data_dir.join("skill-imports")) {
        Ok(()) => {}
        Err(error) if error.kind() == io::ErrorKind::NotFound => {}
        Err(error) => log_diagnostic("skill_import_staging_cleanup_failed", &error.to_string()),
    }
    let data_dir_text = path_to_utf8(&data_dir)?;
    let token = generate_token()?;
    let embedded_source = resolve_hermes_resource(app, HERMES_EMBEDDED_SOURCE_NAME);
    let node_archive = resolve_hermes_resource(app, HERMES_NODE_ARCHIVE_NAME);
    let npm_archive = resolve_hermes_resource(app, HERMES_NPM_ARCHIVE_NAME);
    let python_archive = resolve_hermes_resource(app, HERMES_PYTHON_ARCHIVE_NAME);
    let bootstrap = serde_json::to_vec(&BootstrapMessage {
        protocol_version: PROTOCOL_VERSION,
        token: &token,
        data_dir: data_dir_text,
        hermes_embedded_source_path: embedded_source.as_deref(),
        hermes_node_archive_path: node_archive.as_deref(),
        hermes_npm_archive_path: npm_archive.as_deref(),
        hermes_python_archive_path: python_archive.as_deref(),
    })?;

    let (mut events, child) = app
        .shell()
        .sidecar("yorvad")?
        .arg("--bootstrap-stdio")
        .spawn()?;
    let child_pid = child.pid();
    let mut child_guard = StartupChildGuard::new(Box::new(child));

    let mut message = bootstrap;
    message.push(b'\n');
    child_guard.write(&message)?;
    app.state::<DaemonLifecycle>().set_child(child_guard.take());

    let event_handle = app.clone();
    tauri::async_runtime::spawn(async move {
        let mut handshake_complete = false;
        while let Some(event) = events.recv().await {
            match event {
                CommandEvent::Stdout(line) if !handshake_complete => {
                    if let Err(error) = accept_handshake(
                        &event_handle.state::<DaemonLifecycle>(),
                        &line,
                        child_pid,
                        token.clone(),
                    ) {
                        log_diagnostic("daemon_handshake_rejected", &error);
                        event_handle.state::<DaemonLifecycle>().fail_startup();
                        return;
                    }
                    handshake_complete = true;
                    let _ = event_handle.emit("daemon-session-changed", ());
                }
                CommandEvent::Stdout(_) => {
                    log_diagnostic(
                        "daemon_stdout_contract_violation",
                        "unexpected stdout after handshake",
                    );
                    event_handle.state::<DaemonLifecycle>().fail_startup();
                    return;
                }
                CommandEvent::Stderr(line) => {
                    log_diagnostic("daemon_stderr", &String::from_utf8_lossy(&line));
                }
                CommandEvent::Error(error) => {
                    log_diagnostic("daemon_process_error", &error);
                    event_handle.state::<DaemonLifecycle>().fail_startup();
                    return;
                }
                CommandEvent::Terminated(payload) => {
                    log_diagnostic(
                        "daemon_process_terminated",
                        &format!("exit_code={:?}", payload.code),
                    );
                    let should_restart =
                        event_handle.state::<DaemonLifecycle>().process_terminated();
                    if should_restart {
                        log_diagnostic(
                            "daemon_bounded_restart",
                            "restarting once after unexpected termination",
                        );
                        if let Err(error) = try_start_daemon(&event_handle) {
                            log_diagnostic("daemon_restart_failed", &error.to_string());
                            event_handle.state::<DaemonLifecycle>().fail_startup();
                        }
                    }
                    return;
                }
                _ => {}
            }
        }
        log_diagnostic("daemon_event_channel_closed", "sidecar event stream ended");
        event_handle.state::<DaemonLifecycle>().fail_startup();
    });

    let timeout_handle = app.clone();
    thread::spawn(move || {
        timeout_handle
            .state::<DaemonLifecycle>()
            .wait_for_startup_deadline(STARTUP_TIMEOUT);
    });

    Ok(())
}

fn log_diagnostic(event: &str, detail: &str) {
    eprintln!(
        "{}",
        serde_json::json!({
            "service": "yorva-desktop",
            "event": event,
            "detail": detail.trim(),
        })
    );
}

fn generate_token() -> Result<String, getrandom::Error> {
    let mut bytes = [0_u8; 32];
    getrandom::fill(&mut bytes)?;
    Ok(URL_SAFE_NO_PAD.encode(bytes))
}

const HERMES_EMBEDDED_SOURCE_NAME: &str =
    "hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip";
const HERMES_NODE_ARCHIVE_NAME: &str = "node-v22.23.1-win-x64.zip";
const HERMES_NPM_ARCHIVE_NAME: &str = "npm-12.0.2.tgz";
const HERMES_PYTHON_ARCHIVE_NAME: &str =
    "cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz";

fn hermes_resource_candidates(resource_dir: &Path, name: &str) -> [std::path::PathBuf; 2] {
    [
        resource_dir.join("hermes").join("source").join(name),
        resource_dir
            .join("resources")
            .join("hermes")
            .join("source")
            .join(name),
    ]
}

fn resolve_hermes_resource(app: &AppHandle, name: &str) -> Option<String> {
    let resource_dir = app.path().resource_dir().ok()?;
    hermes_resource_candidates(&resource_dir, name)
        .into_iter()
        .find(|path| path.is_file())
        .and_then(|path| path.to_str().map(str::to_owned))
}

fn path_to_utf8(path: &Path) -> Result<&str, Box<dyn std::error::Error>> {
    path.to_str()
        .ok_or_else(|| "The application data path is not valid UTF-8.".into())
}

fn parse_handshake(line: &[u8], expected_pid: u32, token: String) -> Result<DaemonSession, String> {
    let handshake: Handshake =
        serde_json::from_slice(line).map_err(|_| "invalid bootstrap handshake JSON".to_owned())?;
    if handshake.protocol_version != PROTOCOL_VERSION {
        return Err("bootstrap protocol version mismatch".into());
    }
    if handshake.port == 0 || handshake.pid != expected_pid {
        return Err("inconsistent bootstrap process details".into());
    }
    Ok(DaemonSession {
        base_url: format!("http://127.0.0.1:{}", handshake.port),
        token,
        protocol_version: handshake.protocol_version,
    })
}

fn accept_handshake(
    lifecycle: &DaemonLifecycle,
    line: &[u8],
    expected_pid: u32,
    token: String,
) -> Result<(), String> {
    match parse_handshake(line, expected_pid, token) {
        Ok(session) => {
            lifecycle.set_ready(session);
            Ok(())
        }
        Err(error) => {
            lifecycle.fail_startup();
            Err(error)
        }
    }
}

#[cfg(test)]
mod tests {
    use std::sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    };

    use super::*;

    struct FakeChild {
        killed: Arc<AtomicBool>,
        writes: Arc<Mutex<Vec<Vec<u8>>>>,
        fail_write: bool,
    }

    #[derive(Clone)]
    struct FakeChildProbe {
        killed: Arc<AtomicBool>,
        writes: Arc<Mutex<Vec<Vec<u8>>>>,
    }

    impl FakeChild {
        fn new(fail_write: bool) -> (Self, FakeChildProbe) {
            let probe = FakeChildProbe {
                killed: Arc::new(AtomicBool::new(false)),
                writes: Arc::new(Mutex::new(Vec::new())),
            };
            (
                Self {
                    killed: Arc::clone(&probe.killed),
                    writes: Arc::clone(&probe.writes),
                    fail_write,
                },
                probe,
            )
        }
    }

    impl ChildControl for FakeChild {
        fn write(&mut self, bytes: &[u8]) -> io::Result<()> {
            if self.fail_write {
                return Err(io::Error::other("simulated write failure"));
            }
            self.writes.lock().unwrap().push(bytes.to_vec());
            Ok(())
        }

        fn kill(self: Box<Self>) -> io::Result<()> {
            self.killed.store(true, Ordering::SeqCst);
            Ok(())
        }
    }

    #[test]
    fn hermes_resource_candidates_stay_inside_resource_dir() {
        let root = Path::new(r"C:\app\resources");
        let got = hermes_resource_candidates(root, HERMES_NODE_ARCHIVE_NAME);
        assert!(got[0].starts_with(root));
        assert!(got[1].starts_with(root));
        assert!(got[0].ends_with(HERMES_NODE_ARCHIVE_NAME));
        assert!(
            !got.iter()
                .any(|path| { path.to_string_lossy().contains("api/v1") })
        );
    }

    #[test]
    fn generated_token_contains_at_least_256_bits() {
        let token = generate_token().expect("token generation should succeed");
        let decoded = URL_SAFE_NO_PAD
            .decode(token)
            .expect("generated token should be base64url");
        assert_eq!(decoded.len(), 32);
    }

    #[test]
    fn handshake_never_supplies_the_session_token() {
        let session = parse_handshake(
            br#"{"protocolVersion":"1","port":49152,"pid":123}"#,
            123,
            "desktop-owned-token".into(),
        )
        .expect("handshake should be valid");
        assert_eq!(session.token, "desktop-owned-token");
        assert_eq!(session.base_url, "http://127.0.0.1:49152");
    }

    #[test]
    fn handshake_rejects_wrong_process_and_malformed_json() {
        assert!(
            parse_handshake(
                br#"{"protocolVersion":"1","port":49152,"pid":999}"#,
                123,
                "token".into(),
            )
            .is_err()
        );
        assert!(parse_handshake(b"not json", 123, "token".into()).is_err());

        let lifecycle = DaemonLifecycle::new();
        let (child, probe) = FakeChild::new(false);
        lifecycle.set_child(Box::new(child));
        assert!(accept_handshake(&lifecycle, b"not json", 123, "token".into()).is_err());
        assert!(probe.killed.load(Ordering::SeqCst));
        assert!(matches!(lifecycle.lock().status, StartupStatus::Failed));
    }

    #[test]
    fn startup_guard_kills_child_on_partial_initialization_failure() {
        let (child, probe) = FakeChild::new(true);
        let mut guard = StartupChildGuard::new(Box::new(child));
        assert!(guard.write(b"bootstrap\n").is_err());
        drop(guard);
        assert!(probe.killed.load(Ordering::SeqCst));
    }

    #[test]
    fn startup_timeout_and_early_exit_fail_safely_and_kill_when_needed() {
        let lifecycle = DaemonLifecycle::new();
        let (child, probe) = FakeChild::new(false);
        lifecycle.set_child(Box::new(child));
        assert!(lifecycle.wait_for_startup_deadline(Duration::from_millis(1)));
        assert!(probe.killed.load(Ordering::SeqCst));
        let error = match lifecycle.session() {
            Ok(_) => panic!("timed out startup must fail"),
            Err(error) => error,
        };
        assert_eq!(error.code, "DAEMON_STARTUP_FAILED");
        assert_eq!(error.message, STARTUP_FAILED_MESSAGE);

        let lifecycle = DaemonLifecycle::new();
        let (child, _) = FakeChild::new(false);
        lifecycle.set_child(Box::new(child));
        lifecycle.process_terminated();
        let error = match lifecycle.session() {
            Ok(_) => panic!("early exit must fail"),
            Err(error) => error,
        };
        assert_eq!(error.code, "DAEMON_STARTUP_FAILED");
    }

    #[test]
    fn successful_handshake_releases_startup_deadline_owner() {
        let lifecycle = Arc::new(DaemonLifecycle::new());
        let (child, _) = FakeChild::new(false);
        lifecycle.set_child(Box::new(child));
        let waiter = Arc::clone(&lifecycle);
        let deadline =
            thread::spawn(move || waiter.wait_for_startup_deadline(Duration::from_secs(1)));

        accept_handshake(
            &lifecycle,
            br#"{"protocolVersion":"1","port":49152,"pid":123}"#,
            123,
            "desktop-owned-token".into(),
        )
        .expect("handshake should succeed");

        assert!(!deadline.join().unwrap());
        assert!(lifecycle.session().is_ok());
        lifecycle.process_terminated();
    }

    #[test]
    fn synchronous_startup_failure_preserves_queryable_failure_state() {
        let lifecycle = DaemonLifecycle::new();
        lifecycle.fail_startup();
        let error = match lifecycle.session() {
            Ok(_) => panic!("failure state must be queryable"),
            Err(error) => error,
        };
        assert_eq!(error.code, "DAEMON_STARTUP_FAILED");
        assert!(!error.retryable);
        assert_eq!(error.message, STARTUP_FAILED_MESSAGE);
    }

    #[test]
    fn typed_startup_failure_preserves_safe_error_contract() {
        let lifecycle = DaemonLifecycle::new();
        lifecycle.fail_startup_with(DaemonCommandError {
            code: "PRODUCT_DATA_IDENTITY_CONFLICT",
            message: "Resolve the product data conflict before retrying.",
            retryable: false,
        });
        let error = match lifecycle.session() {
            Ok(_) => panic!("failure state must be queryable"),
            Err(error) => error,
        };
        assert_eq!(error.code, "PRODUCT_DATA_IDENTITY_CONFLICT");
        assert_eq!(
            error.message,
            "Resolve the product data conflict before retrying."
        );
        assert!(!error.retryable);
    }

    #[test]
    fn unexpected_ready_process_exit_allows_only_one_bounded_restart() {
        let lifecycle = DaemonLifecycle::new();
        let (first_child, _) = FakeChild::new(false);
        lifecycle.set_child(Box::new(first_child));
        accept_handshake(
            &lifecycle,
            br#"{"protocolVersion":"1","port":49152,"pid":123}"#,
            123,
            "first-token".into(),
        )
        .expect("first handshake should succeed");

        assert!(lifecycle.process_terminated());
        let restarting = match lifecycle.session() {
            Ok(_) => panic!("restart must not expose stale session"),
            Err(error) => error,
        };
        assert_eq!(restarting.code, "DAEMON_NOT_READY");

        let (second_child, _) = FakeChild::new(false);
        lifecycle.set_child(Box::new(second_child));
        accept_handshake(
            &lifecycle,
            br#"{"protocolVersion":"1","port":49153,"pid":124}"#,
            124,
            "second-token".into(),
        )
        .expect("restart handshake should succeed");
        assert!(!lifecycle.process_terminated());
        let failed = match lifecycle.session() {
            Ok(_) => panic!("second exit must be terminal"),
            Err(error) => error,
        };
        assert_eq!(failed.code, "DAEMON_STARTUP_FAILED");
    }

    #[test]
    fn graceful_stop_sends_control_record_and_observes_termination() {
        let lifecycle = Arc::new(DaemonLifecycle::new());
        let (child, probe) = FakeChild::new(false);
        lifecycle.set_child(Box::new(child));
        let observer = Arc::clone(&lifecycle);
        let terminator = thread::spawn(move || {
            thread::sleep(Duration::from_millis(10));
            observer.process_terminated();
        });

        assert!(!lifecycle.stop_with_timeout(Duration::from_secs(1)));
        terminator.join().unwrap();
        assert!(!probe.killed.load(Ordering::SeqCst));
        let writes = probe.writes.lock().unwrap();
        assert_eq!(writes.len(), 1);
        assert_eq!(writes[0].as_slice(), SHUTDOWN_MESSAGE);
    }

    #[test]
    fn stop_forces_kill_after_timeout_or_write_failure() {
        let lifecycle = DaemonLifecycle::new();
        let (child, probe) = FakeChild::new(false);
        lifecycle.set_child(Box::new(child));
        assert!(lifecycle.stop_with_timeout(Duration::from_millis(1)));
        assert!(probe.killed.load(Ordering::SeqCst));

        let lifecycle = DaemonLifecycle::new();
        let (child, probe) = FakeChild::new(true);
        lifecycle.set_child(Box::new(child));
        assert!(lifecycle.stop_with_timeout(Duration::from_secs(1)));
        assert!(probe.killed.load(Ordering::SeqCst));
    }
}
