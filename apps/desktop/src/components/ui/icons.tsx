import type { LucideProps } from "lucide-react";
import {
  AlertTriangle,
  Box,
  Check,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  Globe,
  Home,
  Info,
  Layers,
  Loader2,
  Monitor,
  RotateCw,
  Settings,
  XCircle,
  Zap,
} from "lucide-react";

export const ICON_STROKE = 1.75;

function glyph(Icon: typeof Home, defaultSize: number, props: LucideProps) {
  const { size = defaultSize, strokeWidth = ICON_STROKE, ...rest } = props;
  return <Icon size={size} strokeWidth={strokeWidth} aria-hidden="true" {...rest} />;
}

export function IconHome(props: LucideProps) {
  return glyph(Home, 16, props);
}

export function IconBox(props: LucideProps) {
  return glyph(Box, 16, props);
}

export function IconLayers(props: LucideProps) {
  return glyph(Layers, 16, props);
}

export function IconSettings(props: LucideProps) {
  return glyph(Settings, 16, props);
}

export function IconRotateCw(props: LucideProps) {
  return glyph(RotateCw, 16, props);
}

export function IconGlobe(props: LucideProps) {
  return glyph(Globe, 16, props);
}

export function IconMonitor(props: LucideProps) {
  return glyph(Monitor, 24, props);
}

export function IconRefresh(props: LucideProps) {
  return glyph(RotateCw, 14, props);
}

export function IconChevronRight(props: LucideProps) {
  return glyph(ChevronRight, 16, props);
}

export function IconChevronUp(props: LucideProps) {
  return glyph(ChevronUp, 14, props);
}

export function IconChevronDown(props: LucideProps) {
  return glyph(ChevronDown, 14, props);
}

export function IconCheck(props: LucideProps) {
  return glyph(Check, 14, props);
}

export function IconCheckCircle(props: LucideProps) {
  return glyph(CheckCircle2, 28, props);
}

export function IconAlert(props: LucideProps) {
  return glyph(AlertTriangle, 14, props);
}

export function IconLoader(props: LucideProps) {
  return glyph(Loader2, 14, props);
}

export function IconXCircle(props: LucideProps) {
  return glyph(XCircle, 14, props);
}

export function IconInfo(props: LucideProps) {
  return glyph(Info, 16, props);
}

export function IconZap(props: LucideProps) {
  return glyph(Zap, 14, props);
}

export function HermesMark({ size = 24, className }: { size?: number; className?: string }) {
  return (
    <svg
      className={className}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={ICON_STROKE}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242" />
      <path d="m8 19 4-4 4 4" />
      <path d="M12 15v6" />
    </svg>
  );
}
