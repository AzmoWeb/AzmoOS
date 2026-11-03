// Fonds d'écran du bureau Azmo OS : dégradés originaux, légers (aucune image à télécharger).
import { t } from "./i18n";

export interface Wallpaper {
  id: string;
  name: string;
  css: string;
  /** Fond clair : textes du bureau en foncé. */
  light?: boolean;
}

const stars =
  "radial-gradient(1.2px 1.2px at 12% 18%, #ffffffcc, transparent), radial-gradient(1px 1px at 72% 12%, #ffffffaa, transparent)," +
  "radial-gradient(1.4px 1.4px at 38% 62%, #ffffffbb, transparent), radial-gradient(1px 1px at 88% 48%, #ffffff99, transparent)," +
  "radial-gradient(1.2px 1.2px at 22% 84%, #ffffffaa, transparent), radial-gradient(1px 1px at 58% 32%, #ffffff88, transparent)," +
  "radial-gradient(1.3px 1.3px at 92% 82%, #ffffffaa, transparent), radial-gradient(1px 1px at 6% 52%, #ffffff77, transparent)";

export const wallpapers: Wallpaper[] = [
  { id: "azmo", name: t("Azmo"), css: "radial-gradient(55% 75% at 12% 18%, #8b7bffcc 0%, transparent 62%), radial-gradient(50% 70% at 88% 24%, #ff8a65bb 0%, transparent 60%), radial-gradient(70% 60% at 62% 100%, #c07cf0aa 0%, transparent 62%), linear-gradient(160deg, #1b1530 0%, #120f1c 100%)" },
  { id: "crepuscule", name: t("Crépuscule"), css: "radial-gradient(70% 50% at 50% 105%, #ffb36bdd 0%, transparent 65%), radial-gradient(60% 60% at 80% 70%, #f2706b99 0%, transparent 60%), linear-gradient(180deg, #1d1440 0%, #4a1f5f 45%, #a43f6a 75%, #f08a5d 100%)" },
  { id: "ocean", name: t("Océan"), css: "radial-gradient(60% 70% at 20% 30%, #1fb5c2bb 0%, transparent 60%), radial-gradient(55% 65% at 85% 75%, #3a6df0bb 0%, transparent 60%), radial-gradient(40% 40% at 60% 20%, #5ccfe666 0%, transparent 70%), linear-gradient(170deg, #0a1b33 0%, #07101f 100%)" },
  { id: "foret", name: t("Forêt"), css: "radial-gradient(60% 70% at 18% 75%, #2f9e6acc 0%, transparent 60%), radial-gradient(50% 60% at 82% 20%, #a3c94f88 0%, transparent 60%), radial-gradient(45% 45% at 55% 45%, #1f7a6a77 0%, transparent 70%), linear-gradient(165deg, #0e231b 0%, #081510 100%)" },
  { id: "nuit", name: t("Nuit étoilée"), css: `${stars}, radial-gradient(60% 50% at 75% 15%, #4b3aa899 0%, transparent 65%), radial-gradient(50% 50% at 15% 90%, #1f4e8c88 0%, transparent 60%), linear-gradient(180deg, #0a0b1e 0%, #070814 100%)` },
  { id: "graphite", name: t("Graphite"), css: "radial-gradient(60% 60% at 30% 20%, #3a3d4699 0%, transparent 65%), radial-gradient(50% 50% at 80% 80%, #2a2c3388 0%, transparent 60%), linear-gradient(160deg, #1c1d22 0%, #121316 100%)" },
  { id: "aube", name: t("Aube"), light: true, css: "radial-gradient(55% 65% at 15% 20%, #ffd1e8 0%, transparent 62%), radial-gradient(55% 65% at 85% 25%, #d6d1ff 0%, transparent 62%), radial-gradient(70% 60% at 55% 100%, #ffe1c2 0%, transparent 62%), linear-gradient(160deg, #fdf3ee 0%, #f3eefb 100%)" },
  { id: "lagon", name: t("Lagon"), light: true, css: "radial-gradient(55% 65% at 20% 25%, #bdf0ea 0%, transparent 62%), radial-gradient(55% 65% at 85% 70%, #c9dcff 0%, transparent 62%), radial-gradient(40% 40% at 70% 15%, #fff1c9 0%, transparent 70%), linear-gradient(160deg, #eefbf8 0%, #eef3ff 100%)" },
];

export const wallpaperById = (id?: string) => wallpapers.find((w) => w.id === id);
