use std::{
    fs,
    path::PathBuf,
    sync::atomic::{AtomicBool, Ordering},
};

use serde::{Deserialize, Serialize};
use tauri::{
    App, AppHandle, Manager, RunEvent, State, WindowEvent,
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
};

const MAIN_WINDOW_LABEL: &str = "main";
const OPEN_MENU_ID: &str = "open";
const QUIT_MENU_ID: &str = "quit";
const AUTOSTART_ARG: &str = "--hidden";
const PREFERENCES_FILE: &str = "desktop-preferences.json";

#[derive(Debug, Clone, Copy, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct DesktopPreferences {
    pub launch_on_login: bool,
    pub close_to_tray: bool,
}

impl Default for DesktopPreferences {
    fn default() -> Self {
        Self {
            launch_on_login: true,
            close_to_tray: true,
        }
    }
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct DesktopPreferenceError {
    code: &'static str,
    message: &'static str,
    retryable: bool,
}

impl DesktopPreferenceError {
    fn read() -> Self {
        Self {
            code: "DESKTOP_PREFERENCES_READ_FAILED",
            message: "Yorva could not read the desktop preferences.",
            retryable: true,
        }
    }

    fn write() -> Self {
        Self {
            code: "DESKTOP_PREFERENCES_WRITE_FAILED",
            message: "Yorva could not save the desktop preferences.",
            retryable: true,
        }
    }

    fn autostart() -> Self {
        Self {
            code: "DESKTOP_AUTOSTART_UPDATE_FAILED",
            message: "Yorva could not update the login startup setting.",
            retryable: true,
        }
    }
}

pub struct DesktopLifecycle {
    exit_requested: AtomicBool,
    launch_on_login: AtomicBool,
    close_to_tray: AtomicBool,
}

impl DesktopLifecycle {
    pub fn new() -> Self {
        Self {
            exit_requested: AtomicBool::new(false),
            launch_on_login: AtomicBool::new(true),
            close_to_tray: AtomicBool::new(true),
        }
    }

    fn request_exit(&self) {
        self.exit_requested.store(true, Ordering::SeqCst);
    }

    fn should_hide_on_close(&self) -> bool {
        self.close_to_tray.load(Ordering::SeqCst) && !self.exit_requested.load(Ordering::SeqCst)
    }

    fn preferences(&self) -> DesktopPreferences {
        DesktopPreferences {
            launch_on_login: self.launch_on_login.load(Ordering::SeqCst),
            close_to_tray: self.close_to_tray.load(Ordering::SeqCst),
        }
    }

    fn apply_preferences(&self, preferences: DesktopPreferences) {
        self.launch_on_login
            .store(preferences.launch_on_login, Ordering::SeqCst);
        self.close_to_tray
            .store(preferences.close_to_tray, Ordering::SeqCst);
    }
}

pub fn starts_hidden() -> bool {
    starts_hidden_from(std::env::args())
}

fn starts_hidden_from<I, S>(args: I) -> bool
where
    I: IntoIterator<Item = S>,
    S: AsRef<str>,
{
    args.into_iter().any(|arg| arg.as_ref() == AUTOSTART_ARG)
}

pub fn setup(app: &mut App, start_hidden: bool) -> tauri::Result<()> {
    let open = MenuItem::with_id(app, OPEN_MENU_ID, "Open Yorva", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, QUIT_MENU_ID, "Quit Yorva", true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&open, &quit])?;

    let mut tray = TrayIconBuilder::new()
        .tooltip("Yorva")
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| match event.id().as_ref() {
            OPEN_MENU_ID => show_main_window(app),
            QUIT_MENU_ID => {
                app.state::<DesktopLifecycle>().request_exit();
                app.exit(0);
            }
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if matches!(
                event,
                TrayIconEvent::Click {
                    button: MouseButton::Left,
                    button_state: MouseButtonState::Up,
                    ..
                }
            ) {
                show_main_window(tray.app_handle());
            }
        });

    if let Some(icon) = app.default_window_icon() {
        tray = tray.icon(icon.clone());
    }
    tray.build(app)?;

    let preferences = load_preferences(app.handle()).unwrap_or_default();
    app.state::<DesktopLifecycle>()
        .apply_preferences(preferences);

    if let Err(error) = apply_autostart(app.handle(), preferences.launch_on_login) {
        eprintln!("Yorva autostart registration failed: {error}");
    }

    if start_hidden {
        hide_main_window(app.handle());
    }

    Ok(())
}

#[tauri::command]
pub fn desktop_preferences(state: State<'_, DesktopLifecycle>) -> DesktopPreferences {
    state.preferences()
}

#[tauri::command]
pub fn set_desktop_preferences(
    app: AppHandle,
    state: State<'_, DesktopLifecycle>,
    preferences: DesktopPreferences,
) -> Result<DesktopPreferences, DesktopPreferenceError> {
    let previous = state.preferences();
    apply_autostart(&app, preferences.launch_on_login)
        .map_err(|_| DesktopPreferenceError::autostart())?;

    if persist_preferences(&app, preferences).is_err() {
        let _ = apply_autostart(&app, previous.launch_on_login);
        return Err(DesktopPreferenceError::write());
    }

    state.apply_preferences(preferences);
    Ok(preferences)
}

pub fn handle_run_event(app: &AppHandle, event: &RunEvent) {
    match event {
        RunEvent::WindowEvent {
            label,
            event: WindowEvent::CloseRequested { api, .. },
            ..
        } if label == MAIN_WINDOW_LABEL
            && app.state::<DesktopLifecycle>().should_hide_on_close() =>
        {
            api.prevent_close();
            hide_main_window(app);
        }
        RunEvent::WindowEvent {
            label,
            event: WindowEvent::CloseRequested { .. },
            ..
        } if label == MAIN_WINDOW_LABEL => {
            app.state::<DesktopLifecycle>().request_exit();
            app.exit(0);
        }
        _ => {}
    }
}

pub fn show_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window(MAIN_WINDOW_LABEL) {
        let _ = window.unminimize();
        let _ = window.show();
        let _ = window.set_focus();
    }
}

fn hide_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window(MAIN_WINDOW_LABEL) {
        let _ = window.hide();
    }
}

fn preferences_path(app: &AppHandle) -> Result<PathBuf, DesktopPreferenceError> {
    app.path()
        .app_config_dir()
        .map(|directory| directory.join(PREFERENCES_FILE))
        .map_err(|_| DesktopPreferenceError::read())
}

fn load_preferences(app: &AppHandle) -> Result<DesktopPreferences, DesktopPreferenceError> {
    let path = preferences_path(app)?;
    if !path.exists() {
        return Ok(DesktopPreferences::default());
    }
    let bytes = fs::read(path).map_err(|_| DesktopPreferenceError::read())?;
    serde_json::from_slice(&bytes).map_err(|_| DesktopPreferenceError::read())
}

fn persist_preferences(
    app: &AppHandle,
    preferences: DesktopPreferences,
) -> Result<(), DesktopPreferenceError> {
    let path = preferences_path(app)?;
    let parent = path.parent().ok_or_else(DesktopPreferenceError::write)?;
    fs::create_dir_all(parent).map_err(|_| DesktopPreferenceError::write())?;
    let bytes =
        serde_json::to_vec_pretty(&preferences).map_err(|_| DesktopPreferenceError::write())?;
    fs::write(path, bytes).map_err(|_| DesktopPreferenceError::write())
}

#[cfg(not(debug_assertions))]
fn apply_autostart(app: &AppHandle, enabled: bool) -> Result<(), String> {
    use tauri_plugin_autostart::ManagerExt;

    let manager = app.autolaunch();
    let current = manager.is_enabled().map_err(|error| error.to_string())?;
    if enabled && !current {
        manager.enable().map_err(|error| error.to_string())?;
    } else if !enabled && current {
        manager.disable().map_err(|error| error.to_string())?;
    }
    Ok(())
}

#[cfg(debug_assertions)]
fn apply_autostart(_app: &AppHandle, _enabled: bool) -> Result<(), String> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::{DesktopLifecycle, DesktopPreferences, starts_hidden_from};

    #[test]
    fn recognizes_only_the_explicit_hidden_start_argument() {
        assert!(starts_hidden_from(["yorva", "--hidden"]));
        assert!(!starts_hidden_from(["yorva", "--hidden-window"]));
    }

    #[test]
    fn explicit_exit_disables_close_to_tray() {
        let lifecycle = DesktopLifecycle::new();
        assert!(lifecycle.should_hide_on_close());
        lifecycle.request_exit();
        assert!(!lifecycle.should_hide_on_close());
    }

    #[test]
    fn close_to_tray_preference_controls_window_close_behavior() {
        let lifecycle = DesktopLifecycle::new();
        lifecycle.apply_preferences(DesktopPreferences {
            launch_on_login: true,
            close_to_tray: false,
        });
        assert!(!lifecycle.should_hide_on_close());
        assert!(!lifecycle.preferences().close_to_tray);
    }
}
