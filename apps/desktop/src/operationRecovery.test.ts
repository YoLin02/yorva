import { describe, expect, it } from "vitest";
import { YorvaApiError } from "./api/client";
import type { Operation } from "./api/types";
import {
  isHermesPrerequisite,
  isHermesRuntimeInstall,
  isInstanceCreate,
  isInstanceDelete,
  newestActiveOperation,
  operationIdFromConflict,
} from "./operationRecovery";

function op(partial: Partial<Operation> & Pick<Operation, "id" | "type" | "status" | "createdAt">): Operation {
  return {
    targetType: "runtime-kind",
    targetId: "hermes",
    stage: "preflight",
    progress: null,
    message: "",
    errorCode: null,
    retryable: true,
    correlationId: "cor",
    startedAt: null,
    completedAt: null,
    updatedAt: partial.createdAt,
    ...partial,
  };
}

describe("operation recovery helpers", () => {
  it("selects the newest active runtime.install and ignores terminal history", () => {
    const selected = newestActiveOperation(
      [
        op({ id: "op_old", type: "runtime.install", status: "RUNNING", createdAt: "2026-08-17T01:00:00Z" }),
        op({ id: "op_new", type: "runtime.install", status: "PENDING", createdAt: "2026-08-17T02:00:00Z" }),
        op({ id: "op_done", type: "runtime.install", status: "SUCCEEDED", createdAt: "2026-08-17T03:00:00Z" }),
        op({ id: "op_prereq", type: "hermes.prerequisites", status: "RUNNING", createdAt: "2026-08-17T04:00:00Z" }),
      ],
      isHermesRuntimeInstall,
    );
    expect(selected?.id).toBe("op_new");
  });

  it("rejects a conflict operationId that is not a Hermes prerequisite", () => {
    const install = op({ id: "op_install", type: "runtime.install", status: "RUNNING", createdAt: "2026-08-17T01:00:00Z" });
    expect(isHermesPrerequisite(install)).toBe(false);
    expect(isHermesRuntimeInstall(install)).toBe(true);
  });

  it("selects the newest active instance.create and instance.delete after restart", () => {
    const operations = [
      op({
        id: "op_create_old",
        type: "instance.create",
        status: "RUNNING",
        createdAt: "2026-08-19T01:00:00Z",
        targetType: "runtime-installation",
        targetId: "rtinst_test",
      }),
      op({
        id: "op_create_new",
        type: "instance.create",
        status: "PENDING",
        createdAt: "2026-08-19T02:00:00Z",
        targetType: "runtime-installation",
        targetId: "rtinst_test",
      }),
      op({
        id: "op_create_done",
        type: "instance.create",
        status: "SUCCEEDED",
        createdAt: "2026-08-19T03:00:00Z",
        targetType: "runtime-installation",
        targetId: "rtinst_test",
      }),
      op({
        id: "op_delete",
        type: "instance.delete",
        status: "RUNNING",
        createdAt: "2026-08-19T04:00:00Z",
        targetType: "runtime-installation",
        targetId: "rtinst_test",
        message: "coder",
      }),
    ];
    expect(newestActiveOperation(operations, isInstanceCreate)?.id).toBe("op_create_new");
    expect(newestActiveOperation(operations, isInstanceDelete)?.id).toBe("op_delete");
    expect(isInstanceCreate(operations[3])).toBe(false);
    expect(isInstanceDelete(operations[3])).toBe(true);
  });

  it("reads only typed conflict operation identifiers", () => {
    expect(operationIdFromConflict(new Error("in progress"))).toBeNull();
    expect(
      operationIdFromConflict(
        new YorvaApiError({
          code: "RUNTIME_INSTALL_IN_PROGRESS",
          message: "The Runtime installation request was rejected.",
          retryable: true,
          details: { operationId: "op_live" },
        }),
      ),
    ).toBe("op_live");
  });
});
