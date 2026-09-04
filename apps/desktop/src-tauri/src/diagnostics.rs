use std::{
    fs::{self, OpenOptions},
    io::{self, Read, Write},
    path::Path,
    time::{SystemTime, UNIX_EPOCH},
};

use serde::Serialize;
use tauri::State;

use crate::daemon::DaemonLifecycle;

const MAX_BUNDLE_BYTES: u64 = 2 * 1024 * 1024;

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticExportError {
    code: &'static str,
    message: &'static str,
    retryable: bool,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DiagnosticExportResult {
    file_name: String,
    size_bytes: u64,
    exported_at_unix_ms: u64,
}

impl DiagnosticExportError {
    fn failed() -> Self {
        Self {
            code: "DIAGNOSTICS_EXPORT_FAILED",
            message: "YORVA could not export the sanitized diagnostic bundle.",
            retryable: true,
        }
    }
}

#[tauri::command]
pub async fn export_diagnostic_bundle(
    lifecycle: State<'_, DaemonLifecycle>,
) -> Result<Option<DiagnosticExportResult>, DiagnosticExportError> {
    let session = lifecycle
        .session()
        .map_err(|_| DiagnosticExportError::failed())?;
    let suggested_name = format!("YORVA-diagnostics-{}.zip", unix_millis());
    let bytes = tauri::async_runtime::spawn_blocking(move || download_bundle(session))
        .await
        .map_err(|_| DiagnosticExportError::failed())??;
    let selected = rfd::AsyncFileDialog::new()
        .set_title("Export sanitized YORVA diagnostics")
        .set_file_name(&suggested_name)
        .add_filter("ZIP archive", &["zip"])
        .save_file()
        .await;
    let Some(selected) = selected else {
        return Ok(None);
    };
    let path = selected.path().to_path_buf();
    let size_bytes = bytes.len() as u64;
    tauri::async_runtime::spawn_blocking(move || publish_atomic(&path, &bytes))
        .await
        .map_err(|_| DiagnosticExportError::failed())?
        .map_err(|_| DiagnosticExportError::failed())?;
    Ok(Some(DiagnosticExportResult {
        file_name: selected.file_name(),
        size_bytes,
        exported_at_unix_ms: unix_millis(),
    }))
}

fn download_bundle(
    session: crate::daemon::DaemonSession,
) -> Result<Vec<u8>, DiagnosticExportError> {
    let response = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(30))
        .build()
        .map_err(|_| DiagnosticExportError::failed())?
        .post(format!("{}/api/v1/diagnostics/bundle", session.base_url()))
        .bearer_auth(session.token())
        .send()
        .map_err(|_| DiagnosticExportError::failed())?;
    if !response.status().is_success()
        || response
            .content_length()
            .is_some_and(|size| size > MAX_BUNDLE_BYTES)
    {
        return Err(DiagnosticExportError::failed());
    }
    let mut bytes = Vec::new();
    response
        .take(MAX_BUNDLE_BYTES + 1)
        .read_to_end(&mut bytes)
        .map_err(|_| DiagnosticExportError::failed())?;
    if bytes.is_empty() || bytes.len() as u64 > MAX_BUNDLE_BYTES || !bytes.starts_with(b"PK") {
        return Err(DiagnosticExportError::failed());
    }
    Ok(bytes)
}

fn publish_atomic(destination: &Path, bytes: &[u8]) -> io::Result<()> {
    let parent = destination
        .parent()
        .filter(|path| path.is_dir())
        .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidInput, "invalid export destination"))?;
    let file_name = destination
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidInput, "invalid export filename"))?;
    let partial = parent.join(format!(".{file_name}.{}.partial", unix_millis()));
    let result = (|| {
        let mut file = OpenOptions::new()
            .create_new(true)
            .write(true)
            .open(&partial)?;
        file.write_all(bytes)?;
        file.sync_all()?;
        drop(file);
        atomic_replace(&partial, destination)
    })();
    if result.is_err() {
        let _ = fs::remove_file(&partial);
    }
    result
}

#[cfg(windows)]
fn atomic_replace(source: &Path, destination: &Path) -> io::Result<()> {
    use std::os::windows::ffi::OsStrExt;
    use windows_sys::Win32::Storage::FileSystem::{
        REPLACE_FILE_FLAGS, REPLACEFILE_WRITE_THROUGH, ReplaceFileW,
    };
    if !destination.exists() {
        return fs::rename(source, destination);
    }
    let source: Vec<u16> = source.as_os_str().encode_wide().chain(Some(0)).collect();
    let destination: Vec<u16> = destination
        .as_os_str()
        .encode_wide()
        .chain(Some(0))
        .collect();
    let replaced = unsafe {
        ReplaceFileW(
            destination.as_ptr(),
            source.as_ptr(),
            std::ptr::null(),
            REPLACEFILE_WRITE_THROUGH as REPLACE_FILE_FLAGS,
            std::ptr::null_mut(),
            std::ptr::null_mut(),
        )
    };
    if replaced == 0 {
        Err(io::Error::last_os_error())
    } else {
        Ok(())
    }
}

#[cfg(not(windows))]
fn atomic_replace(source: &Path, destination: &Path) -> io::Result<()> {
    fs::rename(source, destination)
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

    #[test]
    fn atomic_publish_replaces_destination_and_leaves_no_partial() {
        let root = std::env::temp_dir().join(format!("yorva-diagnostics-test-{}", unix_millis()));
        fs::create_dir_all(&root).unwrap();
        let destination = root.join("diagnostics.zip");
        fs::write(&destination, b"old").unwrap();
        publish_atomic(&destination, b"PK-new").unwrap();
        assert_eq!(fs::read(&destination).unwrap(), b"PK-new");
        assert_eq!(fs::read_dir(&root).unwrap().count(), 1);
        fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn failed_publish_cleans_partial_file() {
        let root = std::env::temp_dir().join(format!("yorva-diagnostics-fail-{}", unix_millis()));
        fs::create_dir_all(&root).unwrap();
        let destination = root.join("occupied");
        fs::create_dir(&destination).unwrap();
        assert!(publish_atomic(&destination, b"PK-new").is_err());
        let names: Vec<_> = fs::read_dir(&root)
            .unwrap()
            .map(|entry| entry.unwrap().file_name())
            .collect();
        assert_eq!(names, vec![std::ffi::OsString::from("occupied")]);
        fs::remove_dir_all(root).unwrap();
    }
}
