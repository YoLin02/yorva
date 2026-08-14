import type { components } from "./generated/schema";

export type Health = components["schemas"]["Health"];
export type Node = components["schemas"]["Node"];
export type ErrorResponse = components["schemas"]["ErrorResponse"];

export type DaemonSession = {
  baseUrl: string;
  token: string;
  protocolVersion: string;
};
