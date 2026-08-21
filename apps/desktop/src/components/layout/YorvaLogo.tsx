import { APP_NAME } from "../../appMetadata";
import { AppMark } from "../ui/icons";

export function YorvaLogo({ size = 28 }: { size?: number }) {
  return (
    <div className="yorva-logo">
      <AppMark size={size} />
      <span className="yorva-logo-word">{APP_NAME}</span>
    </div>
  );
}
