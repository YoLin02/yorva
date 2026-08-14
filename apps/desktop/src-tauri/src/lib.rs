mod daemon;

use daemon::{DaemonLifecycle, daemon_session, start_daemon};
use tauri::{Manager, RunEvent};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let app = tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(DaemonLifecycle::new())
        .invoke_handler(tauri::generate_handler![daemon_session])
        .setup(|app| {
            start_daemon(app.handle());
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("failed to build YORVA desktop");

    app.run(|app_handle, event| {
        if matches!(event, RunEvent::ExitRequested { .. } | RunEvent::Exit) {
            app_handle.state::<DaemonLifecycle>().stop();
        }
    });
}
