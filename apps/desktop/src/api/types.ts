import type { components } from "./generated/schema";

export type Health = components["schemas"]["Health"];
export type Node = components["schemas"]["Node"];
export type ErrorResponse = components["schemas"]["ErrorResponse"];
export type RuntimeDiscovery = components["schemas"]["RuntimeDiscovery"];
export type RuntimeDiscoveryState = components["schemas"]["RuntimeDiscoveryState"];
export type RuntimeCandidate = components["schemas"]["RuntimeCandidate"];
export type Operation = components["schemas"]["Operation"];
export type OperationList = components["schemas"]["OperationList"];
export type Instance = components["schemas"]["Instance"];
export type InstanceList = components["schemas"]["InstanceList"];
export type Lifecycle = components["schemas"]["Lifecycle"];
export type Channel = components["schemas"]["Channel"];
export type ChannelList = components["schemas"]["ChannelList"];
export type ChannelQr = components["schemas"]["ChannelQr"];
export type ChannelPairingStatus = components["schemas"]["ChannelPairingStatus"];
export type ChannelPairingApproval = components["schemas"]["ChannelPairingApproval"];
export type ModelProviderPreset = components["schemas"]["ModelProviderPreset"];
export type ModelProviderPresetList = components["schemas"]["ModelProviderPresetList"];
export type ModelConfiguration = components["schemas"]["ModelConfiguration"];
export type ModelCredential = components["schemas"]["ModelCredential"];
export type HermesDownloadSources = components["schemas"]["HermesDownloadSources"];
export type HermesPrerequisites = {
  node: { state: string; version: string; errorCode: string | null; retryable: boolean };
  npm: { state: string; version: string; errorCode: string | null; retryable: boolean };
  nodeDependencies: { state: string; version: string; errorCode: string | null; retryable: boolean };
  checkedAt: string;
  activeOperationId: string | null;
};

export type DaemonSession = {
  baseUrl: string;
  token: string;
  protocolVersion: string;
};
