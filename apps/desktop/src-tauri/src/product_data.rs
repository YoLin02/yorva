use std::{
    fs, io,
    path::Path,
    time::{SystemTime, UNIX_EPOCH},
};

use serde::{Deserialize, Serialize};
use tauri::{App, Manager};

const STABLE_IDENTIFIER: &str = "com.yorva.desktop";
const LEGACY_IDENTIFIER: &str = "com.yorva.desktop.dev";
const MIGRATION_MARKER: &str = ".product-identity-migration.json";
const STAGING_NAME: &str = ".com.yorva.desktop.migrating";

#[derive(Debug)]
pub struct ProductDataError {
    code: &'static str,
    source: io::Error,
}

impl std::fmt::Display for ProductDataError {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(formatter, "{}", self.code)
    }
}

impl std::error::Error for ProductDataError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        Some(&self.source)
    }
}

impl ProductDataError {
    pub fn command_error(&self) -> crate::daemon::DaemonCommandError {
        let message = if self.code == "PRODUCT_DATA_IDENTITY_CONFLICT" {
            "YORVA found both legacy and current data. Resolve the data conflict before retrying."
        } else {
            "YORVA could not safely prepare the existing product data."
        };
        crate::daemon::DaemonCommandError {
            code: self.code,
            message,
            retryable: false,
        }
    }
}

#[derive(Deserialize, Serialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct ProductIdentityMigration {
    schema_version: u8,
    source_identifier: String,
    target_identifier: String,
    copied_files: u64,
    copied_bytes: u64,
    completed_unix_seconds: u64,
}

#[derive(Default)]
struct CopySummary {
    files: u64,
    bytes: u64,
}

pub fn migrate_legacy_app_data(app: &mut App) -> Result<(), ProductDataError> {
    let stable = app
        .path()
        .app_data_dir()
        .map_err(|error| ProductDataError {
            code: "PRODUCT_DATA_PATH_UNAVAILABLE",
            source: io::Error::other(error.to_string()),
        })?;
    if stable.file_name().and_then(|value| value.to_str()) != Some(STABLE_IDENTIFIER) {
        return Err(product_data_error(
            "PRODUCT_DATA_PATH_INVALID",
            io::Error::new(
                io::ErrorKind::InvalidInput,
                "unexpected stable data directory",
            ),
        ));
    }
    let parent = stable.parent().ok_or_else(|| {
        product_data_error(
            "PRODUCT_DATA_PATH_INVALID",
            io::Error::new(
                io::ErrorKind::InvalidInput,
                "stable data directory has no parent",
            ),
        )
    })?;
    migrate_legacy_directory(&parent.join(LEGACY_IDENTIFIER), &stable)
}

fn migrate_legacy_directory(legacy: &Path, stable: &Path) -> Result<(), ProductDataError> {
    if !path_entry_exists(legacy)? {
        return Ok(());
    }
    require_directory(legacy)?;
    if path_entry_exists(stable)? {
        require_directory(stable)?;
        if has_valid_marker(stable)? {
            return Ok(());
        }
        if fs::read_dir(stable)
            .map_err(|error| product_data_error("PRODUCT_DATA_READ_FAILED", error))?
            .next()
            .is_some()
        {
            return Err(product_data_error(
                "PRODUCT_DATA_IDENTITY_CONFLICT",
                io::Error::new(
                    io::ErrorKind::AlreadyExists,
                    "both product data roots contain data",
                ),
            ));
        }
        fs::remove_dir(stable)
            .map_err(|error| product_data_error("PRODUCT_DATA_STAGING_FAILED", error))?;
    }

    let parent = stable.parent().ok_or_else(|| {
        product_data_error(
            "PRODUCT_DATA_PATH_INVALID",
            io::Error::new(
                io::ErrorKind::InvalidInput,
                "stable data directory has no parent",
            ),
        )
    })?;
    let staging = parent.join(STAGING_NAME);
    if path_entry_exists(&staging)? {
        require_directory(&staging)?;
        fs::remove_dir_all(&staging)
            .map_err(|error| product_data_error("PRODUCT_DATA_STAGING_FAILED", error))?;
    }
    fs::create_dir(&staging)
        .map_err(|error| product_data_error("PRODUCT_DATA_STAGING_FAILED", error))?;

    let result = (|| {
        let mut summary = CopySummary::default();
        copy_directory(legacy, &staging, &mut summary)?;
        let marker = ProductIdentityMigration {
            schema_version: 1,
            source_identifier: LEGACY_IDENTIFIER.to_owned(),
            target_identifier: STABLE_IDENTIFIER.to_owned(),
            copied_files: summary.files,
            copied_bytes: summary.bytes,
            completed_unix_seconds: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .map_err(|error| {
                    product_data_error(
                        "PRODUCT_DATA_STAGING_FAILED",
                        io::Error::other(error.to_string()),
                    )
                })?
                .as_secs(),
        };
        let payload = serde_json::to_vec_pretty(&marker).map_err(|error| {
            product_data_error(
                "PRODUCT_DATA_STAGING_FAILED",
                io::Error::other(error.to_string()),
            )
        })?;
        fs::write(staging.join(MIGRATION_MARKER), payload)
            .map_err(|error| product_data_error("PRODUCT_DATA_STAGING_FAILED", error))?;
        fs::rename(&staging, stable)
            .map_err(|error| product_data_error("PRODUCT_DATA_PUBLISH_FAILED", error))
    })();
    if result.is_err() {
        let _ = fs::remove_dir_all(&staging);
    }
    result
}

fn has_valid_marker(stable: &Path) -> Result<bool, ProductDataError> {
    let marker = stable.join(MIGRATION_MARKER);
    let metadata = match fs::symlink_metadata(&marker) {
        Ok(value) => value,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(false),
        Err(error) => return Err(product_data_error("PRODUCT_DATA_READ_FAILED", error)),
    };
    if !metadata.file_type().is_file()
        || metadata.file_type().is_symlink()
        || is_reparse_point(&metadata)
        || metadata.len() > 4096
    {
        return Err(product_data_error(
            "PRODUCT_DATA_MARKER_INVALID",
            io::Error::new(
                io::ErrorKind::InvalidData,
                "invalid product data migration marker",
            ),
        ));
    }
    let payload =
        fs::read(marker).map_err(|error| product_data_error("PRODUCT_DATA_READ_FAILED", error))?;
    let value: ProductIdentityMigration = serde_json::from_slice(&payload).map_err(|error| {
        product_data_error(
            "PRODUCT_DATA_MARKER_INVALID",
            io::Error::other(error.to_string()),
        )
    })?;
    if value.schema_version != 1
        || value.source_identifier != LEGACY_IDENTIFIER
        || value.target_identifier != STABLE_IDENTIFIER
        || value.completed_unix_seconds == 0
    {
        return Err(product_data_error(
            "PRODUCT_DATA_MARKER_INVALID",
            io::Error::new(
                io::ErrorKind::InvalidData,
                "product data migration marker mismatch",
            ),
        ));
    }
    Ok(true)
}

fn copy_directory(
    source: &Path,
    destination: &Path,
    summary: &mut CopySummary,
) -> Result<(), ProductDataError> {
    for entry in fs::read_dir(source)
        .map_err(|error| product_data_error("PRODUCT_DATA_READ_FAILED", error))?
    {
        let entry = entry.map_err(|error| product_data_error("PRODUCT_DATA_READ_FAILED", error))?;
        let metadata = fs::symlink_metadata(entry.path())
            .map_err(|error| product_data_error("PRODUCT_DATA_READ_FAILED", error))?;
        let file_type = metadata.file_type();
        let target = destination.join(entry.file_name());
        if file_type.is_symlink() || is_reparse_point(&metadata) {
            return Err(product_data_error(
                "PRODUCT_DATA_SOURCE_UNSAFE",
                io::Error::new(io::ErrorKind::InvalidData, "legacy data contains a link"),
            ));
        }
        if file_type.is_dir() {
            fs::create_dir(&target)
                .map_err(|error| product_data_error("PRODUCT_DATA_STAGING_FAILED", error))?;
            copy_directory(&entry.path(), &target, summary)?;
            continue;
        }
        if !file_type.is_file() {
            return Err(product_data_error(
                "PRODUCT_DATA_SOURCE_UNSAFE",
                io::Error::new(
                    io::ErrorKind::InvalidData,
                    "legacy data contains a special file",
                ),
            ));
        }
        let source_size = metadata.len();
        let copied = fs::copy(entry.path(), &target)
            .map_err(|error| product_data_error("PRODUCT_DATA_STAGING_FAILED", error))?;
        let target_size = fs::metadata(&target)
            .map_err(|error| product_data_error("PRODUCT_DATA_STAGING_FAILED", error))?
            .len();
        if copied != source_size || target_size != source_size {
            return Err(product_data_error(
                "PRODUCT_DATA_VERIFY_FAILED",
                io::Error::new(io::ErrorKind::InvalidData, "copied file size mismatch"),
            ));
        }
        summary.files = summary.files.checked_add(1).ok_or_else(|| {
            product_data_error(
                "PRODUCT_DATA_VERIFY_FAILED",
                io::Error::other("copied file count overflow"),
            )
        })?;
        summary.bytes = summary.bytes.checked_add(source_size).ok_or_else(|| {
            product_data_error(
                "PRODUCT_DATA_VERIFY_FAILED",
                io::Error::other("copied byte count overflow"),
            )
        })?;
    }
    Ok(())
}

fn require_directory(path: &Path) -> Result<(), ProductDataError> {
    let metadata = fs::symlink_metadata(path)
        .map_err(|error| product_data_error("PRODUCT_DATA_READ_FAILED", error))?;
    if !metadata.is_dir() || metadata.file_type().is_symlink() || is_reparse_point(&metadata) {
        return Err(product_data_error(
            "PRODUCT_DATA_SOURCE_UNSAFE",
            io::Error::new(
                io::ErrorKind::InvalidData,
                "product data root is not a regular directory",
            ),
        ));
    }
    Ok(())
}

fn path_entry_exists(path: &Path) -> Result<bool, ProductDataError> {
    match fs::symlink_metadata(path) {
        Ok(_) => Ok(true),
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(false),
        Err(error) => Err(product_data_error("PRODUCT_DATA_READ_FAILED", error)),
    }
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

fn product_data_error(code: &'static str, source: io::Error) -> ProductDataError {
    ProductDataError { code, source }
}

#[cfg(test)]
mod tests {
    use super::{MIGRATION_MARKER, migrate_legacy_directory, path_entry_exists};
    use std::fs;

    #[test]
    fn copies_legacy_data_without_removing_the_source() {
        let root = tempfile_root();
        let legacy = root.join("com.yorva.desktop.dev");
        let stable = root.join("com.yorva.desktop");
        fs::create_dir_all(legacy.join("backups")).unwrap();
        fs::write(legacy.join("yorva.db"), b"schema-016").unwrap();
        fs::write(legacy.join("backups").join("safe.age"), b"encrypted").unwrap();

        migrate_legacy_directory(&legacy, &stable).unwrap();

        assert_eq!(fs::read(legacy.join("yorva.db")).unwrap(), b"schema-016");
        assert_eq!(fs::read(stable.join("yorva.db")).unwrap(), b"schema-016");
        assert_eq!(
            fs::read(stable.join("backups").join("safe.age")).unwrap(),
            b"encrypted"
        );
        assert!(stable.join(MIGRATION_MARKER).is_file());
        fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn refuses_to_merge_two_unconfirmed_data_roots() {
        let root = tempfile_root();
        let legacy = root.join("com.yorva.desktop.dev");
        let stable = root.join("com.yorva.desktop");
        fs::create_dir_all(&legacy).unwrap();
        fs::create_dir_all(&stable).unwrap();
        fs::write(legacy.join("yorva.db"), b"legacy").unwrap();
        fs::write(stable.join("yorva.db"), b"stable").unwrap();

        let error = migrate_legacy_directory(&legacy, &stable).unwrap_err();
        assert_eq!(error.code, "PRODUCT_DATA_IDENTITY_CONFLICT");
        assert_eq!(fs::read(legacy.join("yorva.db")).unwrap(), b"legacy");
        assert_eq!(fs::read(stable.join("yorva.db")).unwrap(), b"stable");
        fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn completed_migration_is_repeat_safe() {
        let root = tempfile_root();
        let legacy = root.join("com.yorva.desktop.dev");
        let stable = root.join("com.yorva.desktop");
        fs::create_dir_all(&legacy).unwrap();
        fs::write(legacy.join("yorva.db"), b"first").unwrap();
        migrate_legacy_directory(&legacy, &stable).unwrap();
        fs::write(legacy.join("yorva.db"), b"later-legacy-change").unwrap();

        migrate_legacy_directory(&legacy, &stable).unwrap();

        assert_eq!(fs::read(stable.join("yorva.db")).unwrap(), b"first");
        fs::remove_dir_all(root).unwrap();
    }

    #[test]
    fn path_entry_presence_does_not_require_a_directory() {
        let root = tempfile_root();
        let present = root.join("present");
        let missing = root.join("missing");
        fs::write(&present, b"present").unwrap();

        assert!(path_entry_exists(&present).unwrap());
        assert!(!path_entry_exists(&missing).unwrap());
        fs::remove_dir_all(root).unwrap();
    }

    #[cfg(unix)]
    #[test]
    fn refuses_a_dangling_legacy_link_instead_of_treating_it_as_absent() {
        use std::os::unix::fs::symlink;

        let root = tempfile_root();
        let legacy = root.join("com.yorva.desktop.dev");
        let stable = root.join("com.yorva.desktop");
        symlink(root.join("missing-target"), &legacy).unwrap();

        let error = migrate_legacy_directory(&legacy, &stable).unwrap_err();
        assert_eq!(error.code, "PRODUCT_DATA_SOURCE_UNSAFE");
        assert!(!stable.exists());
        fs::remove_file(legacy).unwrap();
        fs::remove_dir(root).unwrap();
    }

    fn tempfile_root() -> std::path::PathBuf {
        let mut random = [0_u8; 12];
        getrandom::fill(&mut random).unwrap();
        let root = std::env::temp_dir().join(format!(
            "yorva-product-data-test-{}",
            random
                .iter()
                .map(|byte| format!("{byte:02x}"))
                .collect::<String>()
        ));
        fs::create_dir(&root).unwrap();
        root
    }
}
