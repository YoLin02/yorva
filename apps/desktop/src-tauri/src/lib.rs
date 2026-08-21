mod daemon;
#[cfg(desktop)]
mod desktop;

use daemon::{DaemonLifecycle, daemon_session, start_daemon};
use tauri::{Manager, RunEvent};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let mut builder = tauri::Builder::default();

    #[cfg(desktop)]
    {
        builder = builder
            .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
                desktop::show_main_window(app);
            }))
            .plugin(tauri_plugin_autostart::init(
                tauri_plugin_autostart::MacosLauncher::LaunchAgent,
                Some(vec!["--hidden"]),
            ));
    }

    builder = builder
        .plugin(tauri_plugin_shell::init())
        .manage(DaemonLifecycle::new());

    #[cfg(desktop)]
    {
        builder = builder.manage(desktop::DesktopLifecycle::new());
    }

    let app = builder
        .invoke_handler(tauri::generate_handler![
            daemon_session,
            desktop::desktop_preferences,
            desktop::set_desktop_preferences
        ])
        .setup(|app| {
            #[cfg(desktop)]
            desktop::setup(app, desktop::starts_hidden())?;
            start_daemon(app.handle());
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("failed to build YORVA desktop");

    app.run(|app_handle, event| {
        #[cfg(desktop)]
        desktop::handle_run_event(app_handle, &event);
        if matches!(event, RunEvent::ExitRequested { .. } | RunEvent::Exit) {
            app_handle.state::<DaemonLifecycle>().stop();
        }
    });
}
