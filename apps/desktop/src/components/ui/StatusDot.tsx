import { cn } from "../../lib/cn";
import type { StatusTone } from "../../types/ui";

export function StatusDot({ tone }: { tone: StatusTone }) {
  return <span className={cn("status-dot", `status-dot-${tone}`)} aria-hidden="true" />;
}
