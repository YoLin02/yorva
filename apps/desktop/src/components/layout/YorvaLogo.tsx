import { APP_NAME } from "../../appMetadata";

export function YorvaLogo({ size = 28 }: { size?: number }) {
  return (
    <div className="yorva-logo">
      <svg width={size} height={size} viewBox="0 0 40 40" fill="none" aria-hidden="true">
        <path d="M13.5 6L23 15.5L18.5 20L9 10.5L13.5 6Z" fill="#10B981" />
        <path d="M26.5 34L17 24.5L21.5 20L31 29.5L26.5 34Z" fill="#059669" />
        <path d="M9 10.5L13.5 6L18 10.5L13.5 15L9 10.5Z" fill="#34D399" />
        <path d="M22 29.5L26.5 34L31 29.5L26.5 25L22 29.5Z" fill="#047857" />
      </svg>
      <span className="yorva-logo-word">{APP_NAME}</span>
    </div>
  );
}
