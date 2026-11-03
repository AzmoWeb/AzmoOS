import { useEffect, useState, type ReactNode } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import {
  Home, LayoutGrid, Store, Boxes, FolderOpen, HardDrive, Wifi, ShieldCheck, Users, Archive, RefreshCw,
  ScrollText, Settings, Menu as MenuIcon, LogOut, Moon, Sun,
} from "lucide-react";
import { api } from "../api/client";
import type { Perm } from "../api/types";
import { useJobs, useSession } from "../session";
import { Avatar, IconButton } from "./ui";

interface NavItem { to: string; label: string; icon: ReactNode; perm?: Perm }

const groups: { title?: string; items: NavItem[] }[] = [
  { items: [{ to: "/", label: "Accueil", icon: <Home /> }] },
  {
    title: "Applications",
    items: [
      { to: "/apps", label: "Mes applications", icon: <LayoutGrid /> },
      { to: "/store", label: "App Store", icon: <Store />, perm: "apps.manage" },
      { to: "/containers", label: "Conteneurs", icon: <Boxes />, perm: "apps.manage" },
    ],
  },
  {
    title: "Stockage",
    items: [
      { to: "/files", label: "Fichiers", icon: <FolderOpen />, perm: "files.read" },
      { to: "/storage", label: "Disques & partages", icon: <HardDrive />, perm: "admin" },
    ],
  },
  {
    title: "Système",
    items: [
      { to: "/network", label: "Réseau", icon: <Wifi />, perm: "admin" },
      { to: "/security", label: "Sécurité", icon: <ShieldCheck />, perm: "admin" },
      { to: "/users", label: "Utilisateurs", icon: <Users />, perm: "admin" },
      { to: "/backups", label: "Sauvegardes", icon: <Archive />, perm: "admin" },
      { to: "/updates", label: "Mises à jour", icon: <RefreshCw />, perm: "admin" },
      { to: "/activity", label: "Journal", icon: <ScrollText />, perm: "admin" },
      { to: "/settings", label: "Réglages", icon: <Settings /> },
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
        <IconButton label="Menu" onClick={() => setOpen(true)}><MenuIcon /></IconButton>
        <img src="/logo.svg" alt="" />
        <b>Azmo OS</b>
        <div className="spacer" />
        <Avatar name={user.display} size={30} />
      </div>
      {open && <div className="scrim" onClick={() => setOpen(false)} />}
      <aside className={`sidebar ${open ? "open" : ""}`}>
        <NavLink to="/" className="brand">
          <img src="/logo.svg" alt="" />
          <div>
            <div className="brand-name">Azmo OS</div>
            <div className="brand-by">Made by <b>Azmo</b></div>
          </div>
        </NavLink>
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
              <span>{user.role === "admin" ? "Administrateur" : "Membre"}</span>
            </div>
            <IconButton label={theme === "dark" ? "Thème clair" : "Thème sombre"} onClick={toggleTheme}>{theme === "dark" ? <Sun /> : <Moon />}</IconButton>
            <IconButton label="Se déconnecter" onClick={logout}><LogOut /></IconButton>
          </div>
        </div>
      </aside>
      <main className="main">
        <div className="page" key={loc.pathname}><Outlet /></div>
      </main>
    </div>
  );
}
