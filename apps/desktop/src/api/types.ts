import type { components } from "./generated/schema";

export type Health = components["schemas"]["Health"];
export type Node = components["schemas"]["Node"];
export type ErrorResponse = components["schemas"]["ErrorResponse"];
export type RuntimeDiscovery = components["schemas"]["RuntimeDiscovery"];
export type RuntimeDiscoveryState = components["schemas"]["RuntimeDiscoveryState"];
export type RuntimeCandidate = components["schemas"]["RuntimeCandidate"];

export type DaemonSession = {
  baseUrl: string;
  token: string;
  protocolVersion: string;
};
