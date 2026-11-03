// Sections d'Azmo OS, présentées comme des « applications système » sur le bureau et dans le lanceur.
import type { ReactElement } from "react";
import {
  LayoutGrid, Store, Boxes, FolderOpen, HardDrive, Wifi, ShieldCheck, Users, Archive, RefreshCw, ScrollText, Settings, TerminalSquare, Bell,
} from "lucide-react";
import type { Perm } from "../api/types";
import { t } from "./i18n";

export interface Section {
  to: string;
  label: string;
  icon: ReactElement;
  /** Dégradé de la tuile. */
  colors: [string, string];
  perm?: Perm;
  /** Affichée directement sur le bureau. */
  desktop?: boolean;
}

export const sections: Section[] = [
  { to: "/files", label: t("Fichiers"), icon: <FolderOpen />, colors: ["#ffc35a", "#ff8a4c"], perm: "files.read", desktop: true },
  { to: "/store", label: t("App Store"), icon: <Store />, colors: ["#8b7bff", "#d07cf0"], perm: "apps.manage", desktop: true },
  { to: "/terminal", label: t("Terminal"), icon: <TerminalSquare />, colors: ["#3a3350", "#211c30"], perm: "admin", desktop: true },
  { to: "/settings", label: t("Réglages"), icon: <Settings />, colors: ["#8e8aa3", "#5d5872"], desktop: true },
  { to: "/apps", label: t("Mes applications"), icon: <LayoutGrid />, colors: ["#5d8bff", "#7b5cff"] },
  { to: "/containers", label: t("Conteneurs"), icon: <Boxes />, colors: ["#3a8df0", "#38c6e6"], perm: "apps.manage" },
  { to: "/storage", label: t("Disques & partages"), icon: <HardDrive />, colors: ["#36c98b", "#1f9a74"], perm: "admin" },
  { to: "/network", label: t("Réseau"), icon: <Wifi />, colors: ["#2ec5d3", "#3a8df0"], perm: "admin" },
  { to: "/security", label: t("Sécurité"), icon: <ShieldCheck />, colors: ["#5fd39a", "#2f9e6a"], perm: "admin" },
  { to: "/users", label: t("Utilisateurs"), icon: <Users />, colors: ["#ff8fb1", "#e0609a"], perm: "admin" },
  { to: "/backups", label: t("Sauvegardes"), icon: <Archive />, colors: ["#f6b44b", "#e58a2e"], perm: "admin" },
  { to: "/notifications", label: t("Notifications"), icon: <Bell />, colors: ["#ff8a65", "#f2706b"], perm: "admin" },
  { to: "/updates", label: t("Mises à jour"), icon: <RefreshCw />, colors: ["#7b8cff", "#5a6df0"], perm: "admin" },
  { to: "/activity", label: t("Journal"), icon: <ScrollText />, colors: ["#a59cc0", "#6f6782"], perm: "admin" },
];

export const sectionFor = (path: string) => sections.find((s) => path === s.to || path.startsWith(s.to + "/"));

/** Tuile d'icône en dégradé pour une section. */
export function SectionTile({ s, size = 64 }: { s: Section; size?: number }) {
  return (
    <div className="sys-tile" style={{ width: size, height: size, borderRadius: size * 0.28, background: `linear-gradient(145deg, ${s.colors[0]}, ${s.colors[1]})` }}>
      <span style={{ width: size * 0.48, height: size * 0.48 }}>{s.icon}</span>
    </div>
  );
}
