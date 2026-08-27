import { invoke } from "@tauri-apps/api/core";
import type { DaemonSession } from "./types";

type DaemonCommandError = {
  code?: string;
  message?: string;
  retryable?: boolean;
};

export function getDaemonSession(): Promise<DaemonSession> {
  return invoke<DaemonSession>("daemon_session");
}

export type SkillImportSelection = { sourceRef: string; suggestedSkillId: string };

export function selectSkillImport(kind: "ZIP" | "DIRECTORY"): Promise<SkillImportSelection | null> {
  return invoke<SkillImportSelection | null>("select_skill_import", { kind });
}

export function discardSkillImport(sourceRef: string): Promise<void> {
  return invoke<void>("discard_skill_import", { sourceRef });
}

export function isDaemonNotReady(error: unknown): boolean {
  return typeof error === "object" && error !== null && (error as DaemonCommandError).code === "DAEMON_NOT_READY";
}
