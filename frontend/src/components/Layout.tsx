import { createContext, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { LayoutGrid, Search as SearchIcon, X, LogOut, Moon, Sun, Settings, ChevronLeft, Image as ImageIcon } from "lucide-react";
import { api } from "../api/client";
import type { AppsResponse, CatalogApp } from "../api/types";
import { useSession } from "../session";
import { sections, sectionFor, SectionTile, type Section } from "../lib/sections";
import { wallpaperById } from "../lib/wallpapers";
import { t } from "../lib/i18n";
import { NotificationBell } from "./Notifications";
import { AppIcon } from "./AppIcon";
import { Avatar, IconButton } from "./ui";

/* ---------- Fond d'écran ---------- */

interface WallCtx { id: string; light: boolean; set: (id: string) => Promise<void>; bump: () => void }
const WallpaperCtx = createContext<WallCtx>(null as unknown as WallCtx);
export const useWallpaper = () => useContext(WallpaperCtx);

function WallpaperProvider({ children }: { children: ReactNode }) {
  const { status, refresh } = useSession();
  const [id, setId] = useState(status.user?.wallpaper || "azmo");
  const [version, setVersion] = useState(Date.now());
  useEffect(() => { setId(status.user?.wallpaper || "azmo"); }, [status.user?.wallpaper]);
  const preset = wallpaperById(id) ?? wallpaperById("azmo")!;
  const light = id !== "custom" && !!preset.light;
  const css = id === "custom" ? `center / cover no-repeat url("/api/me/wallpaper?v=${version}")` : preset.css;
  useEffect(() => { document.documentElement.dataset.wall = light ? "light" : "dark"; }, [light]);
  const set = async (next: string) => { setId(next); await api.post("/api/me/wallpaper/preset", { id: next }).catch(() => undefined); await refresh(); };
  return (
    <WallpaperCtx.Provider value={{ id, light, set, bump: () => setVersion(Date.now()) }}>
      <div className="wallpaper" style={{ background: css }} aria-hidden />
      <div className="wallpaper-veil" aria-hidden />
      {children}
    </WallpaperCtx.Provider>
  );
}

/* ---------- Thème ---------- */

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

/* ---------- Panneau flottant (lanceur, compte, recherche) ---------- */

function Popover({ anchor, onClose, children, className = "", align = "right" }: { anchor: HTMLElement; onClose: () => void; children: ReactNode; className?: string; align?: "left" | "right" | "center" }) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState<{ left: number; top: number }>({ left: -9999, top: -9999 });
  useEffect(() => {
    const r = anchor.getBoundingClientRect();
    const w = ref.current!.offsetWidth;
    const left = align === "left" ? r.left : align === "center" ? r.left + r.width / 2 - w / 2 : r.right - w;
    setPos({ left: Math.max(10, Math.min(left, window.innerWidth - w - 10)), top: r.bottom + 10 });
    const close = (e: Event) => { if (!ref.current?.contains(e.target as Node) && !anchor.contains(e.target as Node)) onClose(); };
    const key = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("mousedown", close);
    window.addEventListener("keydown", key);
    return () => { document.removeEventListener("mousedown", close); window.removeEventListener("keydown", key); };
  }, [anchor, onClose, align]);
  return createPortal(<div ref={ref} className={`popover ${className}`} style={pos}>{children}</div>, document.body);
}

function Launcher({ anchor, onClose }: { anchor: HTMLElement; onClose: () => void }) {
  const { can } = useSession();
  const list = sections.filter((s) => !s.perm || can(s.perm));
  return (
    <Popover anchor={anchor} onClose={onClose} className="launcher" align="right">
      <div className="launcher-title">{t("Toutes les sections")}</div>
      <div className="launcher-grid">
        {list.map((s) => (
          <Link key={s.to} to={s.to} className="launcher-item" onClick={onClose}>
            <SectionTile s={s} size={52} />
            <span>{s.label}</span>
          </Link>
        ))}
      </div>
    </Popover>
  );
}

function AccountMenu({ anchor, onClose, theme, toggleTheme }: { anchor: HTMLElement; onClose: () => void; theme: string; toggleTheme: () => void }) {
  const { status, refresh } = useSession();
  const user = status.user!;
  return (
    <Popover anchor={anchor} onClose={onClose} className="account" align="right">
      <div className="row" style={{ gap: 12, padding: "6px 6px 12px", flexWrap: "nowrap" }}>
        <Avatar name={user.display} size={42} />
        <div style={{ minWidth: 0 }}><b>{user.display}</b><div className="small muted">@{user.username} · {user.role === "admin" ? t("Administrateur") : t("Membre")}</div></div>
      </div>
      <Link to="/settings" className="menu-link" onClick={onClose}><Settings /> {t("Réglages")}</Link>
      <Link to="/settings#wallpaper" className="menu-link" onClick={onClose}><ImageIcon /> {t("Fond d'écran")}</Link>
      <button className="menu-link" onClick={() => { toggleTheme(); }}>{theme === "dark" ? <Sun /> : <Moon />} {theme === "dark" ? t("Thème clair") : t("Thème sombre")}</button>
      <hr />
      <button className="menu-link danger" onClick={async () => { await api.post("/api/logout"); await refresh(); }}><LogOut /> {t("Se déconnecter")}</button>
    </Popover>
  );
}

/* ---------- Recherche globale ---------- */

interface Result { key: string; label: string; sub: string; icon: ReactNode; go: () => void }

function GlobalSearch() {
  const { can } = useSession();
  const nav = useNavigate();
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [apps, setApps] = useState<AppsResponse | null>(null);
  const [catalog, setCatalog] = useState<CatalogApp[] | null>(null);
  const box = useRef<HTMLDivElement>(null);
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const k = (e: KeyboardEvent) => {
      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || (e.key === "/" && !(e.target as HTMLElement).closest("input, textarea, .xterm"))) { e.preventDefault(); input.current?.focus(); }
    };
    window.addEventListener("keydown", k);
    return () => window.removeEventListener("keydown", k);
  }, []);

  const load = () => {
    if (!apps) api.get<AppsResponse>("/api/apps").then(setApps).catch(() => undefined);
    if (!catalog && can("apps.manage")) api.get<CatalogApp[]>("/api/store/catalog").then(setCatalog).catch(() => undefined);
  };

  const results = useMemo<Result[]>(() => {
    const s = q.trim().toLowerCase();
    if (!s) return [];
    const out: Result[] = [];
    for (const a of apps?.apps ?? []) {
      if (a.name.toLowerCase().includes(s)) out.push({ key: "a" + a.id, label: a.name, sub: t("Mes applications"), icon: <AppIcon name={a.name} icon={a.icon} size={34} />, go: () => nav("/apps") });
    }
    for (const sec of sections) {
      if ((!sec.perm || can(sec.perm)) && sec.label.toLowerCase().includes(s)) out.push({ key: "s" + sec.to, label: sec.label, sub: t("Section"), icon: <SectionTile s={sec} size={34} />, go: () => nav(sec.to) });
    }
    for (const a of catalog ?? []) {
      if (!a.installed && `${a.name} ${a.tagline} ${a.category}`.toLowerCase().includes(s)) out.push({ key: "c" + a.id, label: a.name, sub: t("App Store"), icon: <AppIcon name={a.name} icon={a.icon} size={34} />, go: () => nav("/store?q=" + encodeURIComponent(a.name)) });
    }
    return out.slice(0, 8);
  }, [q, apps, catalog, can, nav]);

  const pick = (r?: Result) => { if (!r) return; r.go(); setQ(""); setOpen(false); input.current?.blur(); };

  return (
    <div className="gsearch" ref={box}>
      <SearchIcon />
      <input ref={input} value={q} placeholder={t("Rechercher une app, une section…")} aria-label={t("Rechercher")}
        onFocus={() => { setOpen(true); load(); }} onChange={(e) => { setQ(e.target.value); setOpen(true); }}
        onKeyDown={(e) => { if (e.key === "Enter") pick(results[0]); if (e.key === "Escape") { setQ(""); input.current?.blur(); } }} />
      {q && <button className="icon-btn" aria-label={t("Effacer")} onClick={() => setQ("")}><X /></button>}
      <kbd className="hide-sm">/</kbd>
      {open && q && box.current && (
        <Popover anchor={box.current} onClose={() => setOpen(false)} className="search-results" align="left">
          {results.length === 0 ? <div className="muted small" style={{ padding: 14 }}>{t("Aucun résultat pour « {0} »", q)}</div> : results.map((r) => (
            <button key={r.key} className="search-item" onClick={() => pick(r)}>
              {r.icon}<span><b>{r.label}</b><small>{r.sub}</small></span>
            </button>
          ))}
        </Popover>
      )}
    </div>
  );
}

/* ---------- Coquille ---------- */

export function Layout() {
  const { status } = useSession();
  const loc = useLocation();
  const nav = useNavigate();
  const [theme, toggleTheme] = useTheme();
  const [launcher, setLauncher] = useState(false);
  const [account, setAccount] = useState(false);
  const launcherBtn = useRef<HTMLButtonElement>(null);
  const accountBtn = useRef<HTMLButtonElement>(null);
  useEffect(() => { setLauncher(false); setAccount(false); window.scrollTo(0, 0); }, [loc.pathname]);
  const user = status.user!;
  const home = loc.pathname === "/";
  const section: Section | undefined = sectionFor(loc.pathname);

  return (
    <WallpaperProvider>
      <header className="topbar">
        <NavLink to="/" className="topbar-brand" aria-label={t("Bureau")}>
          <img src="/logo.svg" alt="" />
          <span className="hide-sm"><b>Azmo OS</b><small>Made by <i>Azmo</i></small></span>
        </NavLink>
        <GlobalSearch />
        <div className="topbar-actions">
          <IconButton label={t("Toutes les sections")} ref={launcherBtn} onClick={() => setLauncher(!launcher)}><LayoutGrid /></IconButton>
          <NotificationBell />
          <button ref={accountBtn} className="topbar-avatar" aria-label={t("Mon compte")} onClick={() => setAccount(!account)}><Avatar name={user.display} size={34} /></button>
        </div>
      </header>
      {launcher && launcherBtn.current && <Launcher anchor={launcherBtn.current} onClose={() => setLauncher(false)} />}
      {account && accountBtn.current && <AccountMenu anchor={accountBtn.current} onClose={() => setAccount(false)} theme={theme} toggleTheme={toggleTheme} />}

      {home ? (
        <main className="desktop page" key="desktop"><Outlet /></main>
      ) : (
        <main className="window-wrap" key={loc.pathname}>
          <div className="window">
            <div className="window-bar">
              <button className="btn ghost sm" onClick={() => nav("/")}><ChevronLeft /> {t("Bureau")}</button>
              {section && <span className="window-title"><SectionTile s={section} size={26} /> {section.label}</span>}
              <div className="spacer" />
              <IconButton label={t("Fermer")} onClick={() => nav("/")}><X /></IconButton>
            </div>
            <div className="window-body page"><Outlet /></div>
          </div>
        </main>
      )}
    </WallpaperProvider>
  );
}
