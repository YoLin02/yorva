import { invoke } from "@tauri-apps/api/core";

export type DesktopPreferences = {
  launchOnLogin: boolean;
  closeToTray: boolean;
};

export function getDesktopPreferences(): Promise<DesktopPreferences> {
  return invoke<DesktopPreferences>("desktop_preferences");
}

export function setDesktopPreferences(preferences: DesktopPreferences): Promise<DesktopPreferences> {
  return invoke<DesktopPreferences>("set_desktop_preferences", { preferences });
}
