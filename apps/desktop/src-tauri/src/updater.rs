use std::{
    fs::{self, File, OpenOptions},
    io::{self, Read, Write},
    path::{Path, PathBuf},
    process::Command,
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
    },
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use base64::{Engine as _, engine::general_purpose::STANDARD};
use ed25519_dalek::{Signature, Verifier, VerifyingKey};
use semver::Version;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use tauri::{AppHandle, Manager, State};

use crate::daemon::DaemonLifecycle;

#[cfg(not(feature = "update-qualification"))]
const METADATA_URL: &str =
    "https://github.com/YoLin02/yorva/releases/latest/download/yorva-update.json";
#[cfg(feature = "update-qualification")]
const METADATA_URL: &str = "https://10.0.2.2:8443/yorva-update.json";
const MAX_METADATA_BYTES: u64 = 64 * 1024;
const MAX_PACKAGE_BYTES: u64 = 512 * 1024 * 1024;
const UPDATE_DIRECTORY: &str = "update-staging";
const STATE_FILE: &str = "state.json";
const PACKAGE_FILE: &str = "candidate.msi";
const PARTIAL_FILE: &str = "candidate.partial";
const INSTALLER_RESULT_FILE: &str = "installer.exit";
const STATE_SCHEMA: u32 = 1;
const USER_AGENT: &str = "YORVA-Desktop-Updater/1";

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateCommandError {
    code: &'static str,
    message: &'static str,
    retryable: bool,
}

impl UpdateCommandError {
    fn new(code: &'static str, message: &'static str, retryable: bool) -> Self {
        Self {
            code,
            message,
            retryable,
        }
    }

    fn metadata() -> Self {
        Self::new(
            "UPDATE_METADATA_INVALID",
            "YORVA could not verify the update information.",
            true,
        )
    }

    fn signing_unavailable() -> Self {
        Self::new(
            "UPDATE_SIGNING_UNAVAILABLE",
            "This build does not contain an approved YORVA update verification key.",
            false,
        )
    }

    fn download() -> Self {
        Self::new(
            "UPDATE_DOWNLOAD_FAILED",
            "YORVA could not download the complete update package.",
            true,
        )
    }

    fn integrity() -> Self {
        Self::new(
            "UPDATE_INTEGRITY_FAILED",
            "The update package failed verification and was not opened.",
            false,
        )
    }

    fn install() -> Self {
        Self::new(
            "UPDATE_INSTALL_FAILED",
            "Windows could not start the verified YORVA update.",
            true,
        )
    }

    fn state() -> Self {
        Self::new(
            "UPDATE_STATE_FAILED",
            "YORVA could not safely read or save the update state.",
            true,
        )
    }
}

#[derive(Debug, Clone, Copy, Deserialize, Serialize, PartialEq, Eq)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
enum UpdatePhase {
    Idle,
    UpToDate,
    Available,
    Downloading,
    ReadyToInstall,
    Installing,
    Postcheck,
    Succeeded,
    Failed,
}

#[derive(Debug, Clone, Copy, Deserialize, Serialize, PartialEq, Eq)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
enum AuthenticodePolicy {
    Required,
    InternalUnsigned,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct UpdatePackage {
    url: String,
    size_bytes: u64,
    sha256: String,
    authenticode_policy: AuthenticodePolicy,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    signer_sha256: Option<String>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct UpdateMetadata {
    schema_version: u32,
    channel: String,
    version: String,
    minimum_version: String,
    published_at_utc: String,
    release_notes: String,
    package: UpdatePackage,
    signature: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct SignedMetadata<'a> {
    schema_version: u32,
    channel: &'a str,
    version: &'a str,
    minimum_version: &'a str,
    published_at_utc: &'a str,
    release_notes: &'a str,
    package: &'a UpdatePackage,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct UpdateRecord {
    schema_version: u32,
    phase: UpdatePhase,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    metadata: Option<UpdateMetadata>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    error_code: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    started_at_unix_ms: Option<u64>,
}

impl Default for UpdateRecord {
    fn default() -> Self {
        Self {
            schema_version: STATE_SCHEMA,
            phase: UpdatePhase::Idle,
            metadata: None,
            error_code: None,
            started_at_unix_ms: None,
        }
    }
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateCandidate {
    version: String,
    release_notes: String,
    published_at_utc: String,
    size_bytes: u64,
    authenticode_required: bool,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateStatus {
    installed_version: String,
    phase: UpdatePhase,
    verification_key_configured: bool,
    metadata_source: &'static str,
    candidate: Option<UpdateCandidate>,
    error_code: Option<String>,
}

#[derive(Clone)]
pub struct UpdateManager {
    gate: Arc<Mutex<()>>,
    cancel_download: Arc<AtomicBool>,
}

impl UpdateManager {
    pub fn new() -> Self {
        Self {
            gate: Arc::new(Mutex::new(())),
            cancel_download: Arc::new(AtomicBool::new(false)),
        }
    }
}

struct UpdatePaths {
    root: PathBuf,
    state: PathBuf,
    package: PathBuf,
    partial: PathBuf,
    installer_result: PathBuf,
}

impl UpdatePaths {
    fn from_app(app: &AppHandle) -> Result<Self, UpdateCommandError> {
        let root = app
            .path()
            .app_data_dir()
            .map_err(|_| UpdateCommandError::state())?
            .join(UPDATE_DIRECTORY);
        Self::at(root)
    }

    fn at(root: PathBuf) -> Result<Self, UpdateCommandError> {
        ensure_update_root(&root)?;
        Ok(Self {
            state: root.join(STATE_FILE),
            package: root.join(PACKAGE_FILE),
            partial: root.join(PARTIAL_FILE),
            installer_result: root.join(INSTALLER_RESULT_FILE),
            root,
        })
    }
}

#[tauri::command]
pub fn yorva_update_status(
    app: AppHandle,
    manager: State<'_, UpdateManager>,
    daemon: State<'_, DaemonLifecycle>,
) -> Result<UpdateStatus, UpdateCommandError> {
    let paths = UpdatePaths::from_app(&app)?;
    let _guard = manager
        .gate
        .lock()
        .map_err(|_| UpdateCommandError::state())?;
    let mut record = load_record(&paths)?;
    reconcile_postcheck(&paths, &mut record, &daemon)?;
    Ok(status_from(&record))
}

#[tauri::command]
pub async fn check_yorva_update(
    app: AppHandle,
    manager: State<'_, UpdateManager>,
) -> Result<UpdateStatus, UpdateCommandError> {
    let paths = UpdatePaths::from_app(&app)?;
    let manager = manager.inner().clone();
    run_blocking(move || {
        let _guard = manager
            .gate
            .lock()
            .map_err(|_| UpdateCommandError::state())?;
        let result =
            compiled_verifying_key().and_then(|key| check_update(&paths, METADATA_URL, key));
        match result {
            Ok(record) => Ok(status_from(&record)),
            Err(error) => {
                persist_failure(&paths, error.code)?;
                Err(error)
            }
        }
    })
    .await
}

#[tauri::command]
pub async fn download_yorva_update(
    app: AppHandle,
    manager: State<'_, UpdateManager>,
) -> Result<UpdateStatus, UpdateCommandError> {
    let paths = UpdatePaths::from_app(&app)?;
    let manager = manager.inner().clone();
    run_blocking(move || {
        let _guard = manager
            .gate
            .lock()
            .map_err(|_| UpdateCommandError::state())?;
        manager.cancel_download.store(false, Ordering::SeqCst);
        let result = compiled_verifying_key()
            .and_then(|key| download_update(&paths, &manager.cancel_download, key));
        match result {
            Ok(record) => Ok(status_from(&record)),
            Err(error) => {
                remove_if_regular(&paths.partial);
                persist_failure(&paths, error.code)?;
                Err(error)
            }
        }
    })
    .await
}

#[tauri::command]
pub fn cancel_yorva_update(manager: State<'_, UpdateManager>) {
    manager.cancel_download.store(true, Ordering::SeqCst);
}

#[tauri::command]
pub fn install_yorva_update(
    app: AppHandle,
    manager: State<'_, UpdateManager>,
    daemon: State<'_, DaemonLifecycle>,
) -> Result<UpdateStatus, UpdateCommandError> {
    let paths = UpdatePaths::from_app(&app)?;
    let _guard = manager
        .gate
        .lock()
        .map_err(|_| UpdateCommandError::state())?;
    match install_update(&app, &paths, &daemon) {
        Ok(status) => Ok(status),
        Err(error) => {
            cleanup_staging(&paths);
            persist_failure(&paths, error.code)?;
            Err(error)
        }
    }
}

fn install_update(
    app: &AppHandle,
    paths: &UpdatePaths,
    daemon: &DaemonLifecycle,
) -> Result<UpdateStatus, UpdateCommandError> {
    let mut record = load_record(paths)?;
    let metadata = record
        .metadata
        .clone()
        .ok_or_else(UpdateCommandError::metadata)?;
    if record.phase != UpdatePhase::ReadyToInstall {
        return Err(UpdateCommandError::install());
    }
    validate_metadata(&metadata, &compiled_verifying_key()?)?;
    verify_staged_package(&paths.package, &metadata)?;
    remove_existing_regular_file(&paths.installer_result)?;

    record.phase = UpdatePhase::Installing;
    record.error_code = None;
    record.started_at_unix_ms = Some(unix_millis());
    persist_record(paths, &record)?;

    daemon.stop();
    if let Err(error) = launch_installer(&paths.package, &paths.installer_result) {
        record.phase = UpdatePhase::Failed;
        record.error_code = Some(error.code.to_owned());
        persist_record(paths, &record)?;
        if daemon.prepare_start() {
            crate::daemon::start_daemon(app);
        }
        return Err(error);
    }
    let status = status_from(&record);
    app.exit(0);
    Ok(status)
}

pub fn resume_postcheck(app: AppHandle) {
    std::thread::spawn(move || {
        const ATTEMPTS: usize = 92;
        for attempt in 0..ATTEMPTS {
            let result: Result<bool, UpdateCommandError> = (|| {
                let paths = UpdatePaths::from_app(&app)?;
                let manager = app.state::<UpdateManager>();
                let daemon = app.state::<DaemonLifecycle>();
                let _guard = manager
                    .gate
                    .lock()
                    .map_err(|_| UpdateCommandError::state())?;
                let mut record = load_record(&paths)?;
                if record.phase != UpdatePhase::Installing && record.phase != UpdatePhase::Postcheck
                {
                    return Ok(true);
                }
                reconcile_postcheck(&paths, &mut record, &daemon)?;
                if attempt + 1 == ATTEMPTS && record.phase == UpdatePhase::Postcheck {
                    record.phase = UpdatePhase::Failed;
                    record.error_code = Some("UPDATE_POSTCHECK_FAILED".to_owned());
                    persist_record(&paths, &record)?;
                }
                Ok(matches!(
                    record.phase,
                    UpdatePhase::Succeeded | UpdatePhase::Failed
                ))
            })();

            match result {
                Ok(true) => return,
                Ok(false) => std::thread::sleep(Duration::from_millis(500)),
                Err(error) => {
                    eprintln!(
                        "{}",
                        serde_json::json!({
                            "service": "yorva-desktop",
                            "event": "update_postcheck_failed",
                            "errorCode": error.code,
                        })
                    );
                    return;
                }
            }
        }
    });
}

#[cfg(feature = "update-qualification")]
pub fn run_qualification_update(app: AppHandle) {
    const ARGUMENT: &str = "--qualify-fixed-update";
    if !std::env::args_os().any(|argument| argument == ARGUMENT) {
        return;
    }

    std::thread::spawn(move || {
        let mut daemon_ready = false;
        for _ in 0..92 {
            match app.state::<DaemonLifecycle>().session() {
                Ok(_) => {
                    daemon_ready = true;
                    break;
                }
                Err(error) if error.retryable => std::thread::sleep(Duration::from_millis(500)),
                Err(_) => return,
            }
        }
        if !daemon_ready {
            if let Ok(paths) = UpdatePaths::from_app(&app) {
                let _ = persist_failure(&paths, "UPDATE_POSTCHECK_FAILED");
            }
            return;
        }

        let result = (|| {
            let paths = UpdatePaths::from_app(&app)?;
            let manager = app.state::<UpdateManager>();
            let daemon = app.state::<DaemonLifecycle>();
            let _guard = manager
                .gate
                .lock()
                .map_err(|_| UpdateCommandError::state())?;
            let key = compiled_verifying_key()?;
            let checked = check_update(&paths, METADATA_URL, key)?;
            if checked.phase != UpdatePhase::Available {
                return Err(UpdateCommandError::metadata());
            }
            download_update(&paths, &manager.cancel_download, key)?;
            install_update(&app, &paths, &daemon)
        })();

        if let Err(error) = result
            && let Ok(paths) = UpdatePaths::from_app(&app)
        {
            cleanup_staging(&paths);
            let _ = persist_failure(&paths, error.code);
        }
    });
}

async fn run_blocking<T: Send + 'static>(
    task: impl FnOnce() -> Result<T, UpdateCommandError> + Send + 'static,
) -> Result<T, UpdateCommandError> {
    tauri::async_runtime::spawn_blocking(task)
        .await
        .map_err(|_| UpdateCommandError::state())?
}

fn check_update(
    paths: &UpdatePaths,
    metadata_url: &str,
    key: VerifyingKey,
) -> Result<UpdateRecord, UpdateCommandError> {
    let bytes = fetch_bounded(metadata_url, MAX_METADATA_BYTES, None)?;
    let metadata: UpdateMetadata =
        serde_json::from_slice(&bytes).map_err(|_| UpdateCommandError::metadata())?;
    validate_metadata(&metadata, &key)?;

    let current =
        Version::parse(env!("CARGO_PKG_VERSION")).map_err(|_| UpdateCommandError::metadata())?;
    let minimum =
        Version::parse(&metadata.minimum_version).map_err(|_| UpdateCommandError::metadata())?;
    let candidate =
        Version::parse(&metadata.version).map_err(|_| UpdateCommandError::metadata())?;
    if current < minimum {
        return Err(UpdateCommandError::new(
            "UPDATE_VERSION_UNSUPPORTED",
            "This YORVA version cannot update directly to the selected release.",
            false,
        ));
    }

    let record = UpdateRecord {
        schema_version: STATE_SCHEMA,
        phase: if candidate > current {
            UpdatePhase::Available
        } else {
            UpdatePhase::UpToDate
        },
        metadata: Some(metadata),
        error_code: None,
        started_at_unix_ms: None,
    };
    persist_record(paths, &record)?;
    Ok(record)
}

fn download_update(
    paths: &UpdatePaths,
    cancel: &AtomicBool,
    key: VerifyingKey,
) -> Result<UpdateRecord, UpdateCommandError> {
    let mut record = load_record(paths)?;
    let metadata = record
        .metadata
        .clone()
        .ok_or_else(UpdateCommandError::metadata)?;
    if record.phase != UpdatePhase::Available && record.phase != UpdatePhase::Failed {
        return Err(UpdateCommandError::download());
    }
    validate_metadata(&metadata, &key)?;
    record.phase = UpdatePhase::Downloading;
    record.error_code = None;
    persist_record(paths, &record)?;

    remove_if_regular(&paths.partial);
    remove_if_regular(&paths.package);
    let response = http_client()?
        .get(&metadata.package.url)
        .header(reqwest::header::USER_AGENT, USER_AGENT)
        .header(reqwest::header::ACCEPT, "application/octet-stream")
        .send()
        .map_err(|_| UpdateCommandError::download())?;
    if !response.status().is_success()
        || !allowed_redirect_target(response.url())
        || response
            .content_length()
            .is_some_and(|length| length != metadata.package.size_bytes)
    {
        return Err(UpdateCommandError::download());
    }

    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&paths.partial)
        .map_err(|_| UpdateCommandError::state())?;
    if let Err(error) = copy_and_hash(response, &mut file, metadata.package.size_bytes, cancel) {
        return Err(
            if error.code == "UPDATE_CANCELLED" || error.code == "UPDATE_STATE_FAILED" {
                error
            } else {
                UpdateCommandError::download()
            },
        );
    }
    file.sync_all().map_err(|_| UpdateCommandError::state())?;
    drop(file);
    verify_staged_package(&paths.partial, &metadata)?;
    fs::rename(&paths.partial, &paths.package).map_err(|_| UpdateCommandError::state())?;

    record.phase = UpdatePhase::ReadyToInstall;
    record.error_code = None;
    persist_record(paths, &record)?;
    Ok(record)
}

fn fetch_bounded(
    url: &str,
    limit: u64,
    cancel: Option<&AtomicBool>,
) -> Result<Vec<u8>, UpdateCommandError> {
    if url != METADATA_URL {
        return Err(UpdateCommandError::metadata());
    }
    let response = http_client()?
        .get(url)
        .header(reqwest::header::USER_AGENT, USER_AGENT)
        .header(reqwest::header::ACCEPT, "application/json")
        .send()
        .map_err(|_| UpdateCommandError::download())?;
    if !response.status().is_success()
        || !allowed_redirect_target(response.url())
        || response
            .content_length()
            .is_some_and(|length| length > limit)
    {
        return Err(UpdateCommandError::metadata());
    }
    let mut reader = response.take(limit + 1);
    let mut bytes = Vec::new();
    let mut buffer = [0_u8; 8192];
    loop {
        if cancel.is_some_and(|flag| flag.load(Ordering::SeqCst)) {
            return Err(UpdateCommandError::new(
                "UPDATE_CANCELLED",
                "The update download was cancelled.",
                true,
            ));
        }
        let count = reader
            .read(&mut buffer)
            .map_err(|_| UpdateCommandError::download())?;
        if count == 0 {
            break;
        }
        bytes.extend_from_slice(&buffer[..count]);
        if bytes.len() as u64 > limit {
            return Err(UpdateCommandError::metadata());
        }
    }
    Ok(bytes)
}

fn http_client() -> Result<reqwest::blocking::Client, UpdateCommandError> {
    let builder = reqwest::blocking::Client::builder()
        .connect_timeout(Duration::from_secs(10))
        .timeout(Duration::from_secs(120))
        .redirect(reqwest::redirect::Policy::custom(|attempt| {
            if attempt.previous().len() >= 5 || !allowed_redirect_target(attempt.url()) {
                attempt.stop()
            } else {
                attempt.follow()
            }
        }));
    #[cfg(feature = "update-qualification")]
    let builder = builder
        .add_root_certificate(
            reqwest::Certificate::from_pem(
                option_env!("YORVA_UPDATE_QUALIFICATION_CA_PEM")
                    .unwrap_or_default()
                    .as_bytes(),
            )
            .map_err(|_| UpdateCommandError::download())?,
        )
        .no_proxy();
    builder.build().map_err(|_| UpdateCommandError::download())
}

fn allowed_redirect_target(url: &reqwest::Url) -> bool {
    if url.scheme() != "https" || !url.username().is_empty() || url.password().is_some() {
        return false;
    }
    let production = matches!(
        url.host_str(),
        Some(
            "github.com" | "objects.githubusercontent.com" | "release-assets.githubusercontent.com"
        )
    );
    #[cfg(feature = "update-qualification")]
    let qualification = url.host_str() == Some("10.0.2.2") && url.port() == Some(8443);
    #[cfg(not(feature = "update-qualification"))]
    let qualification = false;
    production || qualification
}

fn validate_metadata(
    metadata: &UpdateMetadata,
    key: &VerifyingKey,
) -> Result<(), UpdateCommandError> {
    if metadata.schema_version != STATE_SCHEMA
        || metadata.channel != "stable"
        || metadata.release_notes.is_empty()
        || metadata.release_notes.len() > 8 * 1024
        || !is_utc_timestamp(&metadata.published_at_utc)
        || metadata.package.size_bytes == 0
        || metadata.package.size_bytes > MAX_PACKAGE_BYTES
        || !is_sha256(&metadata.package.sha256)
    {
        return Err(UpdateCommandError::metadata());
    }
    let version = Version::parse(&metadata.version).map_err(|_| UpdateCommandError::metadata())?;
    let minimum =
        Version::parse(&metadata.minimum_version).map_err(|_| UpdateCommandError::metadata())?;
    if !version.pre.is_empty()
        || !version.build.is_empty()
        || !minimum.pre.is_empty()
        || !minimum.build.is_empty()
        || minimum > version
    {
        return Err(UpdateCommandError::metadata());
    }
    let expected_url = expected_package_url(&metadata.version);
    if metadata.package.url != expected_url {
        return Err(UpdateCommandError::metadata());
    }
    match metadata.package.authenticode_policy {
        AuthenticodePolicy::Required => {
            if !metadata
                .package
                .signer_sha256
                .as_deref()
                .is_some_and(is_sha256)
            {
                return Err(UpdateCommandError::metadata());
            }
        }
        AuthenticodePolicy::InternalUnsigned => {
            if metadata.package.signer_sha256.is_some() {
                return Err(UpdateCommandError::metadata());
            }
        }
    }
    let signature_bytes = STANDARD
        .decode(&metadata.signature)
        .map_err(|_| UpdateCommandError::metadata())?;
    let signature =
        Signature::from_slice(&signature_bytes).map_err(|_| UpdateCommandError::metadata())?;
    let signed = SignedMetadata {
        schema_version: metadata.schema_version,
        channel: &metadata.channel,
        version: &metadata.version,
        minimum_version: &metadata.minimum_version,
        published_at_utc: &metadata.published_at_utc,
        release_notes: &metadata.release_notes,
        package: &metadata.package,
    };
    let payload = serde_json::to_vec(&signed).map_err(|_| UpdateCommandError::metadata())?;
    key.verify(&payload, &signature)
        .map_err(|_| UpdateCommandError::integrity())
}

#[cfg(not(feature = "update-qualification"))]
fn expected_package_url(version: &str) -> String {
    format!(
        "https://github.com/YoLin02/yorva/releases/download/v{version}/YORVA_{version}_x64_en-US.msi"
    )
}

#[cfg(feature = "update-qualification")]
fn expected_package_url(version: &str) -> String {
    format!("https://10.0.2.2:8443/YORVA_{version}_x64_en-US.msi")
}

fn compiled_verifying_key() -> Result<VerifyingKey, UpdateCommandError> {
    let encoded = option_env!("YORVA_UPDATE_PUBLIC_KEY")
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(UpdateCommandError::signing_unavailable)?;
    parse_verifying_key(encoded)
}

fn parse_verifying_key(encoded: &str) -> Result<VerifyingKey, UpdateCommandError> {
    let bytes = STANDARD
        .decode(encoded)
        .map_err(|_| UpdateCommandError::signing_unavailable())?;
    let bytes: [u8; 32] = bytes
        .try_into()
        .map_err(|_| UpdateCommandError::signing_unavailable())?;
    VerifyingKey::from_bytes(&bytes).map_err(|_| UpdateCommandError::signing_unavailable())
}

fn copy_and_hash(
    mut reader: impl Read,
    writer: &mut impl Write,
    expected_size: u64,
    cancel: &AtomicBool,
) -> Result<String, UpdateCommandError> {
    let mut total = 0_u64;
    let mut digest = Sha256::new();
    let mut buffer = [0_u8; 64 * 1024];
    loop {
        if cancel.load(Ordering::SeqCst) {
            return Err(UpdateCommandError::new(
                "UPDATE_CANCELLED",
                "The update download was cancelled.",
                true,
            ));
        }
        let count = reader
            .read(&mut buffer)
            .map_err(|_| UpdateCommandError::download())?;
        if count == 0 {
            break;
        }
        total = total
            .checked_add(count as u64)
            .ok_or_else(UpdateCommandError::integrity)?;
        if total > expected_size || total > MAX_PACKAGE_BYTES {
            return Err(UpdateCommandError::integrity());
        }
        writer
            .write_all(&buffer[..count])
            .map_err(|_| UpdateCommandError::state())?;
        digest.update(&buffer[..count]);
    }
    if total != expected_size {
        return Err(UpdateCommandError::integrity());
    }
    Ok(format!("{:X}", digest.finalize()))
}

fn verify_staged_package(path: &Path, metadata: &UpdateMetadata) -> Result<(), UpdateCommandError> {
    require_regular_file(path)?;
    let info = fs::metadata(path).map_err(|_| UpdateCommandError::integrity())?;
    if info.len() != metadata.package.size_bytes {
        return Err(UpdateCommandError::integrity());
    }
    let mut file = File::open(path).map_err(|_| UpdateCommandError::integrity())?;
    let digest = copy_and_hash(
        &mut file,
        &mut io::sink(),
        metadata.package.size_bytes,
        &AtomicBool::new(false),
    )?;
    if !digest.eq_ignore_ascii_case(&metadata.package.sha256) {
        return Err(UpdateCommandError::integrity());
    }
    verify_msi_version(path, &metadata.version)?;
    if metadata.package.authenticode_policy == AuthenticodePolicy::Required {
        verify_authenticode(
            path,
            metadata
                .package
                .signer_sha256
                .as_deref()
                .ok_or_else(UpdateCommandError::metadata)?,
        )?;
    }
    Ok(())
}

#[cfg(windows)]
fn verify_msi_version(path: &Path, expected: &str) -> Result<(), UpdateCommandError> {
    let script = concat!(
        "& { param($p)",
        "$i=New-Object -ComObject WindowsInstaller.Installer;",
        "$d=$i.GetType().InvokeMember('OpenDatabase','InvokeMethod',$null,$i,@($p,0));",
        "$v=$d.GetType().InvokeMember('OpenView','InvokeMethod',$null,$d,@(\"SELECT `Value` FROM `Property` WHERE `Property`='ProductVersion'\"));",
        "$null=$v.GetType().InvokeMember('Execute','InvokeMethod',$null,$v,$null);",
        "$r=$v.GetType().InvokeMember('Fetch','InvokeMethod',$null,$v,$null);",
        "if($null -eq $r){exit 4};",
        "$r.GetType().InvokeMember('StringData','GetProperty',$null,$r,1) }"
    );
    let output = Command::new(powershell_path()?)
        .args(["-NoProfile", "-NonInteractive", "-Command", script])
        .arg(path)
        .output()
        .map_err(|_| UpdateCommandError::integrity())?;
    if !output.status.success()
        || String::from_utf8(output.stdout)
            .map_err(|_| UpdateCommandError::integrity())?
            .trim()
            != expected
    {
        return Err(UpdateCommandError::integrity());
    }
    Ok(())
}

#[cfg(not(windows))]
fn verify_msi_version(_path: &Path, _expected: &str) -> Result<(), UpdateCommandError> {
    Err(UpdateCommandError::integrity())
}

#[cfg(windows)]
fn verify_authenticode(path: &Path, signer_sha256: &str) -> Result<(), UpdateCommandError> {
    let script = concat!(
        "& { param($p)",
        "$s=Get-AuthenticodeSignature -LiteralPath $p;",
        "if($s.Status -ne 'Valid' -or $null -eq $s.SignerCertificate){exit 4};",
        "$s.SignerCertificate.GetCertHashString('SHA256') }"
    );
    let output = Command::new(powershell_path()?)
        .args(["-NoProfile", "-NonInteractive", "-Command", script])
        .arg(path)
        .output()
        .map_err(|_| UpdateCommandError::integrity())?;
    if !output.status.success()
        || !String::from_utf8(output.stdout)
            .map_err(|_| UpdateCommandError::integrity())?
            .trim()
            .eq_ignore_ascii_case(signer_sha256)
    {
        return Err(UpdateCommandError::integrity());
    }
    Ok(())
}

#[cfg(windows)]
fn powershell_path() -> Result<PathBuf, UpdateCommandError> {
    let path = windows_system_directory()?
        .join("WindowsPowerShell")
        .join("v1.0")
        .join("powershell.exe");
    require_regular_file(&path)?;
    Ok(path)
}

#[cfg(windows)]
fn launch_installer(path: &Path, result_path: &Path) -> Result<(), UpdateCommandError> {
    let executable = windows_system_directory()
        .map_err(|_| UpdateCommandError::install())?
        .join("msiexec.exe");
    require_regular_file(&executable).map_err(|_| UpdateCommandError::install())?;
    let application = std::env::current_exe().map_err(|_| UpdateCommandError::install())?;
    require_regular_file(&application).map_err(|_| UpdateCommandError::install())?;
    if application.file_name().and_then(|name| name.to_str()) != Some("yorva-desktop.exe") {
        return Err(UpdateCommandError::install());
    }
    let script = concat!(
        "& { param($engine,$msi,$app,$result)",
        "& $engine /i $msi /qn /norestart;",
        "$code=$LASTEXITCODE;",
        "[IO.File]::WriteAllText($result,[string]$code,[Text.Encoding]::ASCII);",
        "for($i=0;$i -lt 20 -and -not [IO.File]::Exists($app);$i++){Start-Sleep -Milliseconds 250};",
        "if([IO.File]::Exists($app)){Start-Process -FilePath $app -ArgumentList '--hidden'} }"
    );
    Command::new(powershell_path().map_err(|_| UpdateCommandError::install())?)
        .args([
            "-NoProfile",
            "-NonInteractive",
            "-WindowStyle",
            "Hidden",
            "-Command",
            script,
        ])
        .arg(executable)
        .arg(path)
        .arg(application)
        .arg(result_path)
        .spawn()
        .map(|_| ())
        .map_err(|_| UpdateCommandError::install())
}

#[cfg(windows)]
fn windows_system_directory() -> Result<PathBuf, UpdateCommandError> {
    use std::os::windows::ffi::OsStringExt;
    use windows_sys::Win32::System::SystemInformation::GetSystemDirectoryW;

    let mut buffer = vec![0_u16; 32_768];
    // SAFETY: the vector owns a writable buffer of the length passed to Win32.
    let length = unsafe { GetSystemDirectoryW(buffer.as_mut_ptr(), buffer.len() as u32) };
    if length == 0 || length as usize >= buffer.len() {
        return Err(UpdateCommandError::integrity());
    }
    buffer.truncate(length as usize);
    Ok(PathBuf::from(std::ffi::OsString::from_wide(&buffer)))
}

#[cfg(not(windows))]
fn launch_installer(_path: &Path, _result_path: &Path) -> Result<(), UpdateCommandError> {
    Err(UpdateCommandError::install())
}

fn reconcile_postcheck(
    paths: &UpdatePaths,
    record: &mut UpdateRecord,
    daemon: &DaemonLifecycle,
) -> Result<(), UpdateCommandError> {
    if record.phase != UpdatePhase::Installing && record.phase != UpdatePhase::Postcheck {
        return Ok(());
    }
    let Some(metadata) = &record.metadata else {
        record.phase = UpdatePhase::Failed;
        record.error_code = Some("UPDATE_POSTCHECK_FAILED".to_owned());
        persist_record(paths, record)?;
        cleanup_staging(paths);
        return Ok(());
    };
    if let Some(exit_code) = read_installer_result(&paths.installer_result)?
        && !matches!(exit_code, 0 | 1641 | 3010)
    {
        record.phase = UpdatePhase::Failed;
        record.error_code = Some("UPDATE_INSTALL_FAILED".to_owned());
        persist_record(paths, record)?;
        cleanup_staging(paths);
        return Ok(());
    }
    if metadata.version != env!("CARGO_PKG_VERSION") {
        if record
            .started_at_unix_ms
            .is_some_and(|started| unix_millis().saturating_sub(started) > 10 * 60 * 1000)
        {
            record.phase = UpdatePhase::Failed;
            record.error_code = Some("UPDATE_INSTALL_FAILED".to_owned());
            persist_record(paths, record)?;
            cleanup_staging(paths);
        }
        return Ok(());
    }
    match daemon.session() {
        Ok(_) => {
            record.phase = UpdatePhase::Succeeded;
            record.error_code = None;
            persist_record(paths, record)?;
            cleanup_staging(paths);
        }
        Err(error) if error.retryable => {
            record.phase = UpdatePhase::Postcheck;
            persist_record(paths, record)?;
        }
        Err(_) => {
            record.phase = UpdatePhase::Failed;
            record.error_code = Some("UPDATE_POSTCHECK_FAILED".to_owned());
            persist_record(paths, record)?;
            cleanup_staging(paths);
        }
    }
    Ok(())
}

fn status_from(record: &UpdateRecord) -> UpdateStatus {
    UpdateStatus {
        installed_version: env!("CARGO_PKG_VERSION").to_owned(),
        phase: record.phase,
        verification_key_configured: option_env!("YORVA_UPDATE_PUBLIC_KEY")
            .is_some_and(|value| !value.trim().is_empty()),
        metadata_source: METADATA_URL,
        candidate: record.metadata.as_ref().map(|metadata| UpdateCandidate {
            version: metadata.version.clone(),
            release_notes: metadata.release_notes.clone(),
            published_at_utc: metadata.published_at_utc.clone(),
            size_bytes: metadata.package.size_bytes,
            authenticode_required: metadata.package.authenticode_policy
                == AuthenticodePolicy::Required,
        }),
        error_code: record.error_code.clone(),
    }
}

fn ensure_update_root(root: &Path) -> Result<(), UpdateCommandError> {
    if root.exists() {
        require_directory(root)?;
    } else {
        fs::create_dir_all(root).map_err(|_| UpdateCommandError::state())?;
        require_directory(root)?;
    }
    Ok(())
}

fn require_directory(path: &Path) -> Result<(), UpdateCommandError> {
    let metadata = fs::symlink_metadata(path).map_err(|_| UpdateCommandError::state())?;
    if !metadata.is_dir() || metadata.file_type().is_symlink() || is_reparse_point(&metadata) {
        return Err(UpdateCommandError::state());
    }
    Ok(())
}

fn require_regular_file(path: &Path) -> Result<(), UpdateCommandError> {
    let metadata = fs::symlink_metadata(path).map_err(|_| UpdateCommandError::integrity())?;
    if !metadata.is_file() || metadata.file_type().is_symlink() || is_reparse_point(&metadata) {
        return Err(UpdateCommandError::integrity());
    }
    Ok(())
}

#[cfg(windows)]
fn is_reparse_point(metadata: &fs::Metadata) -> bool {
    use std::os::windows::fs::MetadataExt;
    metadata.file_attributes() & 0x400 != 0
}

#[cfg(not(windows))]
fn is_reparse_point(_metadata: &fs::Metadata) -> bool {
    false
}

fn load_record(paths: &UpdatePaths) -> Result<UpdateRecord, UpdateCommandError> {
    if !paths.state.exists() {
        return Ok(UpdateRecord::default());
    }
    require_regular_file(&paths.state).map_err(|_| UpdateCommandError::state())?;
    let bytes = fs::read(&paths.state).map_err(|_| UpdateCommandError::state())?;
    if bytes.len() > MAX_METADATA_BYTES as usize {
        return Err(UpdateCommandError::state());
    }
    let record: UpdateRecord =
        serde_json::from_slice(&bytes).map_err(|_| UpdateCommandError::state())?;
    if record.schema_version != STATE_SCHEMA {
        return Err(UpdateCommandError::state());
    }
    Ok(record)
}

fn persist_record(paths: &UpdatePaths, record: &UpdateRecord) -> Result<(), UpdateCommandError> {
    require_directory(&paths.root)?;
    let bytes = serde_json::to_vec_pretty(record).map_err(|_| UpdateCommandError::state())?;
    if bytes.len() > MAX_METADATA_BYTES as usize {
        return Err(UpdateCommandError::state());
    }
    let temporary = paths.root.join("state.tmp");
    remove_if_regular(&temporary);
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&temporary)
        .map_err(|_| UpdateCommandError::state())?;
    file.write_all(&bytes)
        .map_err(|_| UpdateCommandError::state())?;
    file.sync_all().map_err(|_| UpdateCommandError::state())?;
    drop(file);
    if paths.state.exists() {
        require_regular_file(&paths.state).map_err(|_| UpdateCommandError::state())?;
        fs::remove_file(&paths.state).map_err(|_| UpdateCommandError::state())?;
    }
    fs::rename(temporary, &paths.state).map_err(|_| UpdateCommandError::state())
}

fn persist_failure(paths: &UpdatePaths, code: &str) -> Result<(), UpdateCommandError> {
    let mut record = load_record(paths).unwrap_or_default();
    record.phase = UpdatePhase::Failed;
    record.error_code = Some(code.to_owned());
    persist_record(paths, &record)
}

fn remove_if_regular(path: &Path) {
    if fs::symlink_metadata(path)
        .is_ok_and(|metadata| metadata.is_file() && !metadata.file_type().is_symlink())
    {
        let _ = fs::remove_file(path);
    }
}

fn cleanup_staging(paths: &UpdatePaths) {
    remove_if_regular(&paths.partial);
    remove_if_regular(&paths.package);
    remove_if_regular(&paths.installer_result);
}

fn remove_existing_regular_file(path: &Path) -> Result<(), UpdateCommandError> {
    match fs::symlink_metadata(path) {
        Ok(metadata)
            if metadata.is_file()
                && !metadata.file_type().is_symlink()
                && !is_reparse_point(&metadata) =>
        {
            fs::remove_file(path).map_err(|_| UpdateCommandError::state())
        }
        Ok(_) => Err(UpdateCommandError::state()),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(()),
        Err(_) => Err(UpdateCommandError::state()),
    }
}

fn read_installer_result(path: &Path) -> Result<Option<i32>, UpdateCommandError> {
    if !path.exists() {
        return Ok(None);
    }
    require_regular_file(path).map_err(|_| UpdateCommandError::state())?;
    let bytes = fs::read(path).map_err(|_| UpdateCommandError::state())?;
    if bytes.is_empty() || bytes.len() > 16 || !bytes.iter().all(u8::is_ascii_digit) {
        return Err(UpdateCommandError::state());
    }
    let text = std::str::from_utf8(&bytes).map_err(|_| UpdateCommandError::state())?;
    text.parse::<i32>()
        .map(Some)
        .map_err(|_| UpdateCommandError::state())
}

fn is_sha256(value: &str) -> bool {
    value.len() == 64 && value.bytes().all(|byte| byte.is_ascii_hexdigit())
}

fn is_utc_timestamp(value: &str) -> bool {
    let bytes = value.as_bytes();
    if bytes.len() < 20 || bytes.len() > 24 || bytes.last() != Some(&b'Z') {
        return false;
    }
    if bytes.get(4) != Some(&b'-')
        || bytes.get(7) != Some(&b'-')
        || bytes.get(10) != Some(&b'T')
        || bytes.get(13) != Some(&b':')
        || bytes.get(16) != Some(&b':')
    {
        return false;
    }
    if bytes.len() > 20 && (bytes.get(19) != Some(&b'.') || bytes.len() == 21) {
        return false;
    }
    bytes.iter().enumerate().all(|(index, byte)| {
        matches!(index, 4 | 7 | 10 | 13 | 16 | 19)
            || byte.is_ascii_digit()
            || index + 1 == bytes.len()
    }) && value[5..7]
        .parse::<u8>()
        .is_ok_and(|month| (1..=12).contains(&month))
        && value[8..10]
            .parse::<u8>()
            .is_ok_and(|day| (1..=31).contains(&day))
        && value[11..13].parse::<u8>().is_ok_and(|hour| hour <= 23)
        && value[14..16].parse::<u8>().is_ok_and(|minute| minute <= 59)
        && value[17..19].parse::<u8>().is_ok_and(|second| second <= 59)
}

fn unix_millis() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis()
        .try_into()
        .unwrap_or(u64::MAX)
}

#[cfg(test)]
mod tests {
    use super::*;
    use ed25519_dalek::{Signer, SigningKey};
    use std::sync::atomic::AtomicBool;

    fn signed_metadata() -> (UpdateMetadata, VerifyingKey) {
        let signing = SigningKey::from_bytes(&[7_u8; 32]);
        let mut metadata = UpdateMetadata {
            schema_version: 1,
            channel: "stable".to_owned(),
            version: "0.4.1".to_owned(),
            minimum_version: "0.3.2".to_owned(),
            published_at_utc: "2026-09-04T00:00:00Z".to_owned(),
            release_notes: "A bounded update test.".to_owned(),
            package: UpdatePackage {
                url: expected_package_url("0.4.1"),
                size_bytes: 4,
                sha256: "A".repeat(64),
                authenticode_policy: AuthenticodePolicy::InternalUnsigned,
                signer_sha256: None,
            },
            signature: String::new(),
        };
        let signed = SignedMetadata {
            schema_version: metadata.schema_version,
            channel: &metadata.channel,
            version: &metadata.version,
            minimum_version: &metadata.minimum_version,
            published_at_utc: &metadata.published_at_utc,
            release_notes: &metadata.release_notes,
            package: &metadata.package,
        };
        let payload = serde_json::to_vec(&signed).unwrap();
        metadata.signature = STANDARD.encode(signing.sign(&payload).to_bytes());
        (metadata, signing.verifying_key())
    }

    #[test]
    fn accepts_only_signed_fixed_release_metadata() {
        let (metadata, key) = signed_metadata();
        validate_metadata(&metadata, &key).unwrap();

        let mut tampered = metadata.clone();
        tampered.package.size_bytes += 1;
        assert_eq!(
            validate_metadata(&tampered, &key).unwrap_err().code,
            "UPDATE_INTEGRITY_FAILED"
        );

        let mut alternate = metadata.clone();
        alternate.package.url = "https://example.com/update.msi".to_owned();
        assert_eq!(
            validate_metadata(&alternate, &key).unwrap_err().code,
            "UPDATE_METADATA_INVALID"
        );

        let mut malformed_time = metadata.clone();
        malformed_time.published_at_utc = "not-a-timestamp-value".to_owned();
        assert_eq!(
            validate_metadata(&malformed_time, &key).unwrap_err().code,
            "UPDATE_METADATA_INVALID"
        );
    }

    #[test]
    fn bounded_copy_rejects_truncation_growth_and_cancellation() {
        let mut output = Vec::new();
        let digest = copy_and_hash(
            io::Cursor::new(b"test"),
            &mut output,
            4,
            &AtomicBool::new(false),
        )
        .unwrap();
        assert_eq!(
            digest,
            "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08"
        );

        assert_eq!(
            copy_and_hash(
                io::Cursor::new(b"bad"),
                &mut Vec::new(),
                4,
                &AtomicBool::new(false),
            )
            .unwrap_err()
            .code,
            "UPDATE_INTEGRITY_FAILED"
        );
        assert_eq!(
            copy_and_hash(
                io::Cursor::new(b"large"),
                &mut Vec::new(),
                4,
                &AtomicBool::new(false),
            )
            .unwrap_err()
            .code,
            "UPDATE_INTEGRITY_FAILED"
        );
        assert_eq!(
            copy_and_hash(
                io::Cursor::new(b"test"),
                &mut Vec::new(),
                4,
                &AtomicBool::new(true),
            )
            .unwrap_err()
            .code,
            "UPDATE_CANCELLED"
        );
    }

    #[test]
    fn state_rejects_symlink_root_and_unknown_schema() {
        let root = std::env::temp_dir().join(format!("yorva-update-test-{}", unix_millis()));
        let paths = UpdatePaths::at(root.clone()).unwrap();
        let record = UpdateRecord {
            phase: UpdatePhase::Available,
            ..UpdateRecord::default()
        };
        persist_record(&paths, &record).unwrap();
        assert_eq!(load_record(&paths).unwrap().phase, UpdatePhase::Available);

        fs::write(&paths.state, br#"{"schemaVersion":2,"phase":"IDLE"}"#).unwrap();
        assert_eq!(load_record(&paths).unwrap_err().code, "UPDATE_STATE_FAILED");
        fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn installer_result_accepts_only_bounded_numeric_exit_codes() {
        let root = std::env::temp_dir().join(format!("yorva-installer-result-{}", unix_millis()));
        fs::create_dir_all(&root).unwrap();
        let result = root.join(INSTALLER_RESULT_FILE);
        assert_eq!(read_installer_result(&result).unwrap(), None);
        fs::write(&result, b"3010").unwrap();
        assert_eq!(read_installer_result(&result).unwrap(), Some(3010));
        fs::write(&result, b"not-an-exit-code").unwrap();
        assert_eq!(
            read_installer_result(&result).unwrap_err().code,
            "UPDATE_STATE_FAILED"
        );
        fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn postcheck_never_reports_success_for_installer_or_reconcile_failure() {
        let root = std::env::temp_dir().join(format!("yorva-postcheck-test-{}", unix_millis()));
        let paths = UpdatePaths::at(root.clone()).unwrap();
        let (mut metadata, _) = signed_metadata();
        metadata.version = env!("CARGO_PKG_VERSION").to_owned();

        let mut record = UpdateRecord {
            phase: UpdatePhase::Installing,
            metadata: Some(metadata.clone()),
            started_at_unix_ms: Some(unix_millis()),
            ..UpdateRecord::default()
        };
        persist_record(&paths, &record).unwrap();
        fs::write(&paths.installer_result, b"1603").unwrap();
        reconcile_postcheck(&paths, &mut record, &DaemonLifecycle::new()).unwrap();
        assert_eq!(record.phase, UpdatePhase::Failed);
        assert_eq!(record.error_code.as_deref(), Some("UPDATE_INSTALL_FAILED"));

        remove_if_regular(&paths.installer_result);
        record.phase = UpdatePhase::Installing;
        record.error_code = None;
        persist_record(&paths, &record).unwrap();
        let daemon = DaemonLifecycle::new();
        daemon.fail_startup_with(crate::daemon::DaemonCommandError {
            code: "DAEMON_STARTUP_FAILED",
            message: "fixture",
            retryable: false,
        });
        reconcile_postcheck(&paths, &mut record, &daemon).unwrap();
        assert_eq!(record.phase, UpdatePhase::Failed);
        assert_eq!(
            record.error_code.as_deref(),
            Some("UPDATE_POSTCHECK_FAILED")
        );
        fs::remove_dir_all(root).unwrap();
    }
}
