import { useEffect, useLayoutEffect, useRef, useState, type ReactElement } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router-dom";
import { Bell, Thermometer, HardDrive, MemoryStick, Package, Gift, ShieldAlert, Archive, Info, CheckCheck, Settings2 } from "lucide-react";
import { api } from "../api/client";
import type { AzmoNotification } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useSession } from "../session";
import { ago } from "../lib/format";
import { Button, Empty, IconButton } from "./ui";
import { t } from "../lib/i18n";

const kindIcon: Record<string, ReactElement> = {
  temperature: <Thermometer />, disk: <HardDrive />, memory: <MemoryStick />, apps: <Package />, updates: <Gift />,
  security: <ShieldAlert />, backups: <Archive />,
};
export const notifIcon = (k: string) => kindIcon[k] ?? <Info />;

/** Cloche avec le nombre de notifications non lues (administrateurs). */
export function NotificationBell({ size = 36 }: { size?: number }) {
  const { can } = useSession();
  const data = usePoll(() => api.get<{ items: AzmoNotification[]; unread: number }>("/api/notifications"), can("admin") ? 20000 : 0);
  const [open, setOpen] = useState(false);
  const btn = useRef<HTMLButtonElement>(null);
  if (!can("admin")) return null;
  const unread = data.data?.unread ?? 0;
  return (
    <>
      <span className="bell">
        <IconButton label={unread ? t("{0} notification(s) non lue(s)", unread) : t("Notifications")} onClick={() => setOpen(!open)} ref={btn} style={{ width: size, height: size }}>
          <Bell />
        </IconButton>
        {unread > 0 && <span className="count">{unread > 99 ? "99+" : unread}</span>}
      </span>
      {open && btn.current && <Panel anchor={btn.current} items={data.data?.items ?? []} onClose={() => setOpen(false)} reload={data.reload} />}
    </>
  );
}

function Panel({ anchor, items, onClose, reload }: { anchor: HTMLElement; items: AzmoNotification[]; onClose: () => void; reload: () => void }) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ left: -9999, top: -9999 });
  useLayoutEffect(() => {
    const r = anchor.getBoundingClientRect();
    const w = ref.current!.offsetWidth;
    const left = Math.max(12, Math.min(r.left, window.innerWidth - w - 12));
    const top = r.bottom + 8 + 400 > window.innerHeight && r.top > 420 ? Math.max(12, r.top - ref.current!.offsetHeight - 8) : r.bottom + 8;
    setPos({ left, top });
  }, [anchor]);
  useEffect(() => {
    const close = (e: Event) => { if (!ref.current?.contains(e.target as Node) && !anchor.contains(e.target as Node)) onClose(); };
    const key = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("mousedown", close);
    window.addEventListener("keydown", key);
    return () => { document.removeEventListener("mousedown", close); window.removeEventListener("keydown", key); };
  }, [anchor, onClose]);
  const readAll = async () => { await api.post("/api/notifications/read", { id: 0 }); reload(); };
  return createPortal(
    <div className="notif-panel" ref={ref} style={pos} role="dialog" aria-label={t("Notifications")}>
      <header>
        <b>{t("Notifications")}</b>
        <div className="spacer" />
        {items.some((n) => !n.read) && <Button size="sm" variant="ghost" icon={<CheckCheck />} onClick={readAll}>{t("Tout marquer comme lu")}</Button>}
        <Link to="/notifications" className="icon-btn" title={t("Réglages des notifications")} onClick={onClose}><Settings2 /></Link>
      </header>
      <div className="notif-list">
        {items.length === 0 ? <Empty emoji="🔕" title={t("Rien à signaler")}>{t("Azmo OS te préviendra ici si ton Pi a besoin de toi.")}</Empty> : items.slice(0, 40).map((n) => (
          <div key={n.id} className={`notif ${n.read ? "" : "unread"}`} onClick={async () => { if (!n.read) { await api.post("/api/notifications/read", { id: n.id }); reload(); } }}>
            <div className={`tl-icon ${n.level}`}>{notifIcon(n.kind)}</div>
            <div style={{ minWidth: 0 }}>
              <b>{n.title}</b>
              <p>{n.message}</p>
              <small>{ago(n.time)}</small>
            </div>
          </div>
        ))}
      </div>
    </div>,
    document.body,
  );
}
