import { invoke } from "@tauri-apps/api/core";

export type DiagnosticExportResult = {
  fileName: string;
  sizeBytes: number;
  exportedAtUnixMs: number;
};

export type DiagnosticExportError = {
  code: string;
  message: string;
  retryable: boolean;
};

export function isDiagnosticExportError(value: unknown): value is DiagnosticExportError {
  return typeof value === "object" && value !== null && "code" in value;
}

export function exportDiagnosticBundle(): Promise<DiagnosticExportResult | null> {
  return invoke<DiagnosticExportResult | null>("export_diagnostic_bundle");
}
