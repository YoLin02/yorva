import { YorvaApiError } from "./api/client";
import type { Operation } from "./api/types";

export function isActiveOperationStatus(status: Operation["status"]): boolean {
  return status === "PENDING" || status === "RUNNING";
}

export function isHermesRuntimeInstall(operation: Operation): boolean {
  return operation.type === "runtime.install" && operation.targetType === "runtime-kind" && operation.targetId === "hermes";
}

export function isHermesPrerequisite(operation: Operation): boolean {
  return operation.type === "hermes.prerequisites" && operation.targetType === "runtime-kind" && operation.targetId === "hermes";
}

export function newestActiveOperation(
  operations: Operation[] | undefined,
  match: (operation: Operation) => boolean,
): Operation | null {
  if (!operations) {
    return null;
  }
  const active = operations.filter((operation) => match(operation) && isActiveOperationStatus(operation.status));
  if (active.length === 0) {
    return null;
  }
  return active.reduce((latest, current) => (current.createdAt > latest.createdAt ? current : latest));
}

export function operationIdFromConflict(error: unknown): string | null {
  if (!(error instanceof YorvaApiError) || error.code !== "RUNTIME_INSTALL_IN_PROGRESS") {
    return null;
  }
  const id = error.details?.operationId;
  return typeof id === "string" && id.length > 0 ? id : null;
}
