import { describe, expect, it } from "vitest";
import { formatInstallDiagnostic } from "./installDiagnostic";

describe("formatInstallDiagnostic", () => {
  it("formats a failed Operation as copyable JSON lines", () => {
    const text = formatInstallDiagnostic({
      operation: {
        id: "op_Q9uKg6FNLej8hdeLIaWXdlOh",
        type: "runtime.install",
        targetType: "runtime-kind",
        targetId: "hermes",
        status: "FAILED",
        stage: "preflight",
        progress: null,
        message: "",
        errorCode: "RUNTIME_INSTALL_TARGET_OCCUPIED",
        retryable: false,
        correlationId: "cor_z2rAtd992j_dSYzU",
        createdAt: "2026-08-17T03:06:24.5335554Z",
        startedAt: null,
        completedAt: "2026-08-17T03:06:24.5381932Z",
        updatedAt: "2026-08-17T03:06:24.5381932Z",
      },
    });
    expect(text).toContain('"event":"failed"');
    expect(text).toContain('"stage":"preflight"');
    expect(text).toContain("RUNTIME_INSTALL_TARGET_OCCUPIED");
    expect(text).toContain("cor_z2rAtd992j_dSYzU");
  });

  it("includes a rejected start error before an Operation exists", () => {
    const text = formatInstallDiagnostic({
      requestError: {
        code: "RUNTIME_INSTALL_ALREADY_PRESENT",
        message: "The Runtime installation request was rejected.",
        retryable: false,
      },
    });
    expect(text).toContain('"event":"rejected"');
    expect(text).toContain("RUNTIME_INSTALL_ALREADY_PRESENT");
  });
});
