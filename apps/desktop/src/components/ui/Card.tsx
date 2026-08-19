import type { HTMLAttributes, ReactNode } from "react";
import { cn } from "../../lib/cn";

type CardProps = HTMLAttributes<HTMLElement> & {
  as?: "section" | "div" | "article";
  children: ReactNode;
};

export function Card({ as: Tag = "section", className, children, ...props }: CardProps) {
  return (
    <Tag className={cn("card", className)} {...props}>
      {children}
    </Tag>
  );
}
