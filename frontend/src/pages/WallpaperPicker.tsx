import { useEffect, useRef, useState } from "react";
import { Image as ImageIcon, Upload, Trash2 } from "lucide-react";
import { api } from "../api/client";
import { useWallpaper } from "../components/Layout";
import { useUI } from "../components/overlay";
import { Button, Card, Spinner } from "../components/ui";
import { t } from "../lib/i18n";
import { wallpapers } from "../lib/wallpapers";

/** Choix du fond d'écran du bureau (préréglages Azmo ou photo personnelle). */
export function WallpaperPicker() {
  const { id, set, bump } = useWallpaper();
  const { toast } = useUI();
  const input = useRef<HTMLInputElement>(null);
  const card = useRef<HTMLDivElement>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => { if (location.hash === "#wallpaper") card.current?.scrollIntoView({ behavior: "smooth", block: "center" }); }, []);

  const upload = async (f: File) => {
    if (f.size > 12 << 20) return toast(t("Image trop lourde (12 Mo maximum)"), "bad");
    setBusy(true);
    const fd = new FormData();
    fd.append("file", f);
    try { await api.upload("/api/me/wallpaper", fd); bump(); await set("custom"); toast(t("Nouveau fond d'écran ✨"), "good"); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };

  return (
    <div ref={card} id="wallpaper">
      <Card title={t("Fond d'écran")} icon={<ImageIcon />} sub={t("Personnalise ton bureau. Ton choix suit ton compte sur tous tes appareils.")}>
        <div className="wall-grid">
          {wallpapers.map((w) => (
            <button key={w.id} className={`wall-option ${w.light ? "light" : ""} ${id === w.id ? "active" : ""}`} style={{ background: w.css }} onClick={() => set(w.id)} aria-pressed={id === w.id}>
              <span>{w.name}</span>
            </button>
          ))}
          <button className={`wall-option upload ${id === "custom" ? "active" : ""}`} onClick={() => input.current?.click()} disabled={busy}
            style={id === "custom" ? { background: `center / cover url("/api/me/wallpaper?v=${Date.now() >> 16}")` } : undefined}>
            {busy ? <Spinner /> : <><Upload /><span style={{ position: "static", color: "inherit", textShadow: "none" }}>{t("Ma photo")}</span></>}
          </button>
        </div>
        <input ref={input} type="file" accept="image/jpeg,image/png,image/webp" hidden onChange={(e) => { const f = e.target.files?.[0]; if (f) upload(f); e.target.value = ""; }} />
        {id === "custom" && (
          <div style={{ marginTop: 12 }}>
            <Button size="sm" variant="ghost" icon={<Trash2 />} onClick={async () => { await api.del("/api/me/wallpaper"); await set("azmo"); }}>{t("Retirer ma photo")}</Button>
          </div>
        )}
      </Card>
    </div>
  );
}
