import { useState } from "react";
import { avatarHue } from "../lib/human";

/** Icône d'application avec repli élégant sur l'initiale si l'image ne charge pas (Pi hors ligne). */
export function AppIcon({ name, icon, size = 64 }: { name: string; icon?: string; size?: number }) {
  const [broken, setBroken] = useState(false);
  if (icon && !broken) {
    return <img className="app-icon" src={icon} alt="" width={size} height={size} style={{ padding: size * 0.13, borderRadius: size * 0.28 }} loading="lazy" onError={() => setBroken(true)} />;
  }
  const hue = avatarHue(name);
  return (
    <div className="app-letter" style={{ width: size, height: size, fontSize: size * 0.42, borderRadius: size * 0.28, background: `linear-gradient(135deg, hsl(${hue} 65% 60%), hsl(${(hue + 50) % 360} 70% 55%))` }}>
      {(name[0] ?? "?").toUpperCase()}
    </div>
  );
}
