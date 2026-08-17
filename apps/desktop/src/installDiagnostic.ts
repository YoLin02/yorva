import type { Operation } from "./api/types";

export type InstallRequestError = {
  code: string;
  message: string;
  retryable: boolean;
};

export function formatInstallDiagnostic(input: {
  operation?: Operation | null;
  requestError?: InstallRequestError | null;
}): string {
  const records: Record<string, unknown>[] = [];
  if (input.requestError) {
    records.push({
      msg: "runtime install",
      event: "rejected",
      errorCode: input.requestError.code,
      retryable: input.requestError.retryable,
      message: input.requestError.message,
    });
  }
  if (input.operation) {
    records.push({
      msg: "runtime install",
      event: diagnosticEvent(input.operation.status),
      operationId: input.operation.id,
      correlationId: input.operation.correlationId,
      runtimeKind: input.operation.targetId,
      stage: input.operation.stage,
      status: input.operation.status,
      errorCode: input.operation.errorCode,
      retryable: input.operation.retryable,
      createdAt: input.operation.createdAt,
      updatedAt: input.operation.updatedAt,
    });
  }
  return records.map((record) => JSON.stringify(record)).join("\n");
}

function diagnosticEvent(status: Operation["status"]): string {
  switch (status) {
    case "FAILED":
      return "failed";
    case "CANCELLED":
      return "cancelled";
    case "SUCCEEDED":
      return "succeeded";
    case "RUNNING":
      return "stage";
    default:
      return "created";
  }
}
