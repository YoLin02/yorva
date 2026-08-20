import type { RuntimeDiscoveryState } from "../../api/types";
import type { HermesDiscoveryViewState } from "../HermesDiscoveryView";
import type { AppMessages } from "../../i18n";

export type DiscoveryTone = "pending" | "ready" | "error";

export type DiscoveryPresentation = {
  label: string;
  description: string;
  tone: DiscoveryTone;
  version?: string;
  checkedAt?: string;
  state?: RuntimeDiscoveryState;
};

export function discoveryStatus(state: HermesDiscoveryViewState, copy: AppMessages): DiscoveryPresentation {
  if (state.kind === "checking") {
    return { label: copy.hermes.checking, description: copy.hermes.checkingDescription, tone: "pending" };
  }
  if (state.kind === "cancelled") {
    return { label: copy.hermes.cancelled, description: copy.hermes.cancelledDescription, tone: "error" };
  }
  if (state.kind === "failure") {
    return { label: copy.hermes.unavailable, description: copy.hermes.unavailableDescription, tone: "error" };
  }
  const ready = state.discovery.state === "SUPPORTED";
  const pending = state.discovery.state === "NOT_INSTALLED";
  return {
    label: copy.hermes.states[state.discovery.state].title,
    description: copy.hermes.states[state.discovery.state].description,
    tone: ready ? "ready" : pending ? "pending" : "error",
    version: state.discovery.selected?.version,
    checkedAt: state.discovery.detectedAt,
    state: state.discovery.state,
  };
}

export function tileClass(tone: DiscoveryTone) {
  if (tone === "ready") return "icon-tile icon-tile-ok";
  if (tone === "error") return "icon-tile icon-tile-error";
  return "icon-tile icon-tile-warn";
}
