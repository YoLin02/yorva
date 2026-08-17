use std::{
    io,
    path::Path,
    sync::{Condvar, Mutex, MutexGuard},
    thread,
    time::Duration,
};

use base64::{Engine as _, engine::general_purpose::URL_SAFE_NO_PAD};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager};
use tauri_plugin_shell::{
    ShellExt,
    process::{CommandChild, CommandEvent},
};

const PROTOCOL_VERSION: &str = "1";
const STARTUP_TIMEOUT: Duration = Duration::from_secs(10);
const SHUTDOWN_TIMEOUT: Duration = Duration::from_secs(3);
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
    code: &'static str,
    message: &'static str,
    retryable: bool,
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
        let child = {
            let mut inner = self.lock();
            if matches!(
                inner.status,
                StartupStatus::Stopping | StartupStatus::Stopped
            ) {
                return;
            }
            inner.status = StartupStatus::Failed;
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

    fn process_terminated(&self) {
        let mut inner = self.lock();
        inner.child.take();
        inner.status = if matches!(
            inner.status,
            StartupStatus::Stopping | StartupStatus::Stopped
        ) {
            StartupStatus::Stopped
        } else {
            StartupStatus::Failed
        };
        self.changed.notify_all();
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
        match &self.lock().status {
            StartupStatus::Ready(session) => Ok(session.clone()),
            StartupStatus::Starting | StartupStatus::Stopping => Err(DaemonCommandError {
                code: "DAEMON_NOT_READY",
                message: "The local daemon is still starting.",
                retryable: true,
            }),
            StartupStatus::Failed => Err(DaemonCommandError {
                code: "DAEMON_STARTUP_FAILED",
                message: STARTUP_FAILED_MESSAGE,
                retryable: false,
            }),
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

pub fn start_daemon(app: &AppHandle) {
    if let Err(error) = try_start_daemon(app) {
        log_diagnostic("daemon_startup_failed", &error.to_string());
        app.state::<DaemonLifecycle>().fail_startup();
    }
}

fn try_start_daemon(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let data_dir = app.path().app_data_dir()?;
    let data_dir_text = path_to_utf8(&data_dir)?;
    let token = generate_token()?;
    let embedded_source = resolve_hermes_resource(app, HERMES_EMBEDDED_SOURCE_NAME);
    let node_archive = resolve_hermes_resource(app, HERMES_NODE_ARCHIVE_NAME);
    let npm_archive = resolve_hermes_resource(app, HERMES_NPM_ARCHIVE_NAME);
    let bootstrap = serde_json::to_vec(&BootstrapMessage {
        protocol_version: PROTOCOL_VERSION,
        token: &token,
        data_dir: data_dir_text,
        hermes_embedded_source_path: embedded_source.as_deref(),
        hermes_node_archive_path: node_archive.as_deref(),
        hermes_npm_archive_path: npm_archive.as_deref(),
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
        let lifecycle = event_handle.state::<DaemonLifecycle>();
        let mut handshake_complete = false;
        while let Some(event) = events.recv().await {
            match event {
                CommandEvent::Stdout(line) if !handshake_complete => {
                    if let Err(error) =
                        accept_handshake(&lifecycle, &line, child_pid, token.clone())
                    {
                        log_diagnostic("daemon_handshake_rejected", &error);
                        return;
                    }
                    handshake_complete = true;
                }
                CommandEvent::Stdout(_) => {
                    log_diagnostic(
                        "daemon_stdout_contract_violation",
                        "unexpected stdout record",
                    );
                    lifecycle.fail_startup();
                    return;
                }
                CommandEvent::Stderr(line) => {
                    log_diagnostic("daemon_stderr", &String::from_utf8_lossy(&line));
                }
                CommandEvent::Error(error) => {
                    log_diagnostic("daemon_process_error", &error);
                    lifecycle.fail_startup();
                    return;
                }
                CommandEvent::Terminated(payload) => {
                    log_diagnostic(
                        "daemon_process_terminated",
                        &format!("exit_code={:?}", payload.code),
                    );
                    lifecycle.process_terminated();
                    return;
                }
                _ => {}
            }
        }
        log_diagnostic("daemon_event_channel_closed", "sidecar event stream ended");
        lifecycle.fail_startup();
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
    "hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip";
const HERMES_NODE_ARCHIVE_NAME: &str = "node-v22.23.1-win-x64.zip";
const HERMES_NPM_ARCHIVE_NAME: &str = "npm-12.0.2.tgz";

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
