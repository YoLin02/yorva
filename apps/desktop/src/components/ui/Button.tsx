import type { ButtonHTMLAttributes } from "react";
import { cn } from "../../lib/cn";
import type { ButtonVariant } from "../../types/ui";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
};

export function Button({ variant = "secondary", className, type = "button", ...props }: ButtonProps) {
  return <button type={type} className={cn("button", `button-${variant}`, className)} {...props} />;
}
