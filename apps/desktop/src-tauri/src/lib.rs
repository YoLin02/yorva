mod daemon;
#[cfg(desktop)]
mod desktop;
#[cfg(desktop)]
mod diagnostics;
#[cfg(desktop)]
mod product_data;
#[cfg(desktop)]
mod updater;

use daemon::{
    DaemonLifecycle, daemon_session, discard_skill_import, select_skill_import, start_daemon,
};
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
        builder = builder
            .manage(desktop::DesktopLifecycle::new())
            .manage(updater::UpdateManager::new());
    }

    let app = builder
        .invoke_handler(tauri::generate_handler![
            daemon_session,
            select_skill_import,
            discard_skill_import,
            desktop::desktop_preferences,
            desktop::set_desktop_preferences,
            diagnostics::export_diagnostic_bundle,
            updater::yorva_update_status,
            updater::check_yorva_update,
            updater::download_yorva_update,
            updater::cancel_yorva_update,
            updater::install_yorva_update
        ])
        .setup(|app| {
            #[cfg(desktop)]
            {
                desktop::setup(app, desktop::starts_hidden())?;
                if let Err(error) = product_data::migrate_legacy_app_data(app) {
                    eprintln!(
                        "{}",
                        serde_json::json!({
                            "service": "yorva-desktop",
                            "event": "product_data_migration_failed",
                            "errorCode": error.to_string(),
                        })
                    );
                    app.state::<DaemonLifecycle>()
                        .fail_startup_with(error.command_error());
                    updater::resume_postcheck(app.handle().clone());
                    return Ok(());
                }
            }
            start_daemon(app.handle());
            #[cfg(desktop)]
            updater::resume_postcheck(app.handle().clone());
            #[cfg(all(desktop, feature = "update-qualification"))]
            updater::run_qualification_update(app.handle().clone());
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
