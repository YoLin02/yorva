import type { ReactNode } from "react";
import { cn } from "../../lib/cn";
import type { BadgeTone } from "../../types/ui";

export function Badge({ tone, children }: { tone: BadgeTone; children: ReactNode }) {
  return <span className={cn("badge", `badge-${tone}`)}>{children}</span>;
}
