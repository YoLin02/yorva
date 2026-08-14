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

export function isDaemonNotReady(error: unknown): boolean {
  return typeof error === "object" && error !== null && (error as DaemonCommandError).code === "DAEMON_NOT_READY";
}
