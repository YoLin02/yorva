import { invoke } from "@tauri-apps/api/core";

export type YorvaUpdatePhase =
  | "IDLE"
  | "UP_TO_DATE"
  | "AVAILABLE"
  | "DOWNLOADING"
  | "READY_TO_INSTALL"
  | "INSTALLING"
  | "POSTCHECK"
  | "SUCCEEDED"
  | "FAILED";

export type YorvaUpdateCandidate = {
  version: string;
  releaseNotes: string;
  publishedAtUtc: string;
  sizeBytes: number;
  authenticodeRequired: boolean;
};

export type YorvaUpdateStatus = {
  installedVersion: string;
  phase: YorvaUpdatePhase;
  verificationKeyConfigured: boolean;
  metadataSource: string;
  candidate: YorvaUpdateCandidate | null;
  errorCode: string | null;
};

export type YorvaUpdateError = {
  code: string;
  message: string;
  retryable: boolean;
};

export function isYorvaUpdateError(value: unknown): value is YorvaUpdateError {
  return typeof value === "object" && value !== null && "code" in value
    && typeof (value as { code?: unknown }).code === "string";
}

export function getYorvaUpdateStatus(): Promise<YorvaUpdateStatus> {
  return invoke<YorvaUpdateStatus>("yorva_update_status");
}

export function checkYorvaUpdate(): Promise<YorvaUpdateStatus> {
  return invoke<YorvaUpdateStatus>("check_yorva_update");
}

export function downloadYorvaUpdate(): Promise<YorvaUpdateStatus> {
  return invoke<YorvaUpdateStatus>("download_yorva_update");
}

export function cancelYorvaUpdate(): Promise<void> {
  return invoke<void>("cancel_yorva_update");
}

export function installYorvaUpdate(): Promise<YorvaUpdateStatus> {
  return invoke<YorvaUpdateStatus>("install_yorva_update");
}
