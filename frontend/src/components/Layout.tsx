import { useEffect, useState, type ReactNode } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import {
  Home, LayoutGrid, Store, Boxes, FolderOpen, HardDrive, Wifi, ShieldCheck, Users, Archive, RefreshCw,
  ScrollText, Settings, Menu as MenuIcon, LogOut, Moon, Sun, TerminalSquare, Bell,
} from "lucide-react";
import { NotificationBell } from "./Notifications";
import { api } from "../api/client";
import type { Perm } from "../api/types";
import { useJobs, useSession } from "../session";
import { Avatar, IconButton } from "./ui";
import { t } from "../lib/i18n";

interface NavItem { to: string; label: string; icon: ReactNode; perm?: Perm }

const groups: { title?: string; items: NavItem[] }[] = [
  { items: [{ to: "/", label: t("Accueil"), icon: <Home /> }] },
  {
    title: t("Applications"),
    items: [
      { to: "/apps", label: t("Mes applications"), icon: <LayoutGrid /> },
      { to: "/store", label: t("App Store"), icon: <Store />, perm: "apps.manage" },
      { to: "/containers", label: t("Conteneurs"), icon: <Boxes />, perm: "apps.manage" },
    ],
  },
  {
    title: t("Stockage"),
    items: [
      { to: "/files", label: t("Fichiers"), icon: <FolderOpen />, perm: "files.read" },
      { to: "/storage", label: t("Disques & partages"), icon: <HardDrive />, perm: "admin" },
    ],
  },
  {
    title: t("Système"),
    items: [
      { to: "/terminal", label: t("Terminal"), icon: <TerminalSquare />, perm: "admin" },
      { to: "/network", label: t("Réseau"), icon: <Wifi />, perm: "admin" },
      { to: "/security", label: t("Sécurité"), icon: <ShieldCheck />, perm: "admin" },
      { to: "/users", label: t("Utilisateurs"), icon: <Users />, perm: "admin" },
      { to: "/backups", label: t("Sauvegardes"), icon: <Archive />, perm: "admin" },
      { to: "/notifications", label: t("Notifications"), icon: <Bell />, perm: "admin" },
      { to: "/updates", label: t("Mises à jour"), icon: <RefreshCw />, perm: "admin" },
      { to: "/activity", label: t("Journal"), icon: <ScrollText />, perm: "admin" },
      { to: "/settings", label: t("Réglages"), icon: <Settings /> },
    ],
  },
];

function useTheme() {
  const [theme, setTheme] = useState(() => {
    try { return localStorage.getItem("azmo-theme") ?? "dark"; } catch { return "dark"; }
  });
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    try { localStorage.setItem("azmo-theme", theme); } catch { /* navigation privée */ }
  }, [theme]);
  return [theme, () => setTheme(theme === "dark" ? "light" : "dark")] as const;
}

export function Layout() {
  const { status, refresh, can } = useSession();
  const { jobs } = useJobs();
  const [open, setOpen] = useState(false);
  const [theme, toggleTheme] = useTheme();
  const loc = useLocation();
  useEffect(() => { setOpen(false); window.scrollTo(0, 0); }, [loc.pathname]);
  const user = status.user!;
  const busy = jobs.filter((j) => j.status === "running").length;

  const logout = async () => { await api.post("/api/logout"); await refresh(); };

  return (
    <div className="shell">
      <div className="topbar">
        <IconButton label={t("Menu")} onClick={() => setOpen(true)}><MenuIcon /></IconButton>
        <img src="/logo.svg" alt="" />
        <b>{t("Azmo OS")}</b>
        <div className="spacer" />
        <NotificationBell />
        <Avatar name={user.display} size={30} />
      </div>
      {open && <div className="scrim" onClick={() => setOpen(false)} />}
      <aside className={`sidebar ${open ? "open" : ""}`}>
        <NavLink to="/" className="brand">
          <img src="/logo.svg" alt="" />
          <div style={{ flex: 1 }}>
            <div className="brand-name">{t("Azmo OS")}</div>
            <div className="brand-by">{t("Made by")} <b>Azmo</b></div>
          </div>
        </NavLink>
        <div className="sidebar-bell hide-sm"><NotificationBell /></div>
        {groups.map((g, i) => {
          const items = g.items.filter((it) => !it.perm || can(it.perm));
          if (!items.length) return null;
          return (
            <div key={i} className={g.title ? "nav-group" : ""}>
              {g.title && <div className="nav-title">{g.title}</div>}
              {items.map((it) => (
                <NavLink key={it.to} to={it.to} end={it.to === "/"} className={({ isActive }) => `nav-link ${isActive ? "active" : ""}`}>
                  {it.icon}<span>{it.label}</span>
                  {it.to === "/apps" && busy > 0 && <span className="nav-badge">{busy}</span>}
                </NavLink>
              ))}
            </div>
          );
        })}
        <div className="sidebar-foot">
          <div className="me">
            <Avatar name={user.display} />
            <div className="me-text">
              <b>{user.display}</b>
              <span>{user.role === "admin" ? t("Administrateur") : t("Membre")}</span>
            </div>
            <IconButton label={theme === "dark" ? t("Thème clair") : t("Thème sombre")} onClick={toggleTheme}>{theme === "dark" ? <Sun /> : <Moon />}</IconButton>
            <IconButton label={t("Se déconnecter")} onClick={logout}><LogOut /></IconButton>
          </div>
        </div>
      </aside>
      <main className="main">
        <div className="page" key={loc.pathname}><Outlet /></div>
      </main>
    </div>
  );
}
