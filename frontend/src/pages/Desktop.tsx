// Le bureau : widgets à gauche, grille d'applications à droite, sur le fond d'écran.
import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Cpu, MemoryStick, Thermometer, HardDrive, ArrowDown, ArrowUp, Plus, Smile, AlertTriangle, Flame, SlidersHorizontal, ChevronUp, ChevronDown, X, GripVertical, Check, RotateCcw, Clock3, Activity, Network, Boxes, StickyNote } from "lucide-react";
import { api } from "../api/client";
import type { AppsResponse, SystemStats, WidgetId } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useSession } from "../session";
import { bytes, pct, uptime } from "../lib/format";
import { greeting, health, machineName, tempMood, usageMood } from "../lib/human";
import { locale, t } from "../lib/i18n";
import { sections, SectionTile } from "../lib/sections";
import { Sparkline } from "../components/Sparkline";
import { Loading } from "../components/ui";
import { useUI } from "../components/overlay";
import { AppGrid } from "./Apps";

function Clock() {
  const { status } = useSession();
  const [now, setNow] = useState(new Date());
  useEffect(() => { const i = setInterval(() => setNow(new Date()), 1000 * 15); return () => clearInterval(i); }, []);
  return (
    <section className="widget clock">
      <div className="clock-time">{now.toLocaleTimeString(locale(), { hour: "2-digit", minute: "2-digit" })}</div>
      <div className="clock-date">{now.toLocaleDateString(locale(), { weekday: "long", day: "numeric", month: "long" })}</div>
      <div className="clock-hello">{greeting(status.user!.display)}</div>
    </section>
  );
}

/** Jauge circulaire. */
function Ring({ value, label, sub, mood, icon }: { value: number; label: string; sub: string; mood: "good" | "warn" | "bad"; icon: React.ReactNode }) {
  const r = 30, c = 2 * Math.PI * r;
  const color = mood === "bad" ? "var(--bad)" : mood === "warn" ? "var(--warn)" : "url(#ringGrad)";
  return (
    <div className="ring">
      <svg viewBox="0 0 76 76" width="76" height="76" aria-hidden>
        <defs><linearGradient id="ringGrad" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stopColor="#8b7bff" /><stop offset="1" stopColor="#ff8a65" /></linearGradient></defs>
        <circle cx="38" cy="38" r={r} fill="none" stroke="currentColor" strokeOpacity=".14" strokeWidth="7" />
        <circle cx="38" cy="38" r={r} fill="none" stroke={color} strokeWidth="7" strokeLinecap="round"
          strokeDasharray={c} strokeDashoffset={c * (1 - Math.min(100, Math.max(0, value)) / 100)} transform="rotate(-90 38 38)" style={{ transition: "stroke-dashoffset .8s cubic-bezier(.2,.8,.2,1)" }} />
      </svg>
      <div className="ring-center">{icon}<b>{Math.round(value)}<small>%</small></b></div>
      <div className="ring-label">{label}</div>
      <div className="ring-sub">{sub}</div>
    </div>
  );
}

function SystemWidget({ s }: { s: SystemStats }) {
  const mem = pct(s.memUsed, s.memTotal);
  const hl = health(s);
  const HealthIcon = hl.mood === "bad" ? Flame : hl.mood === "warn" ? AlertTriangle : Smile;
  return (
    <section className="widget">
      <div className={`widget-health ${hl.mood}`}><HealthIcon /><div><b>{hl.title}</b><span>{hl.detail}</span></div></div>
      <div className="rings">
        <Ring value={s.cpu} label={t("Processeur")} sub={t("{0} cœurs", s.cores)} mood={usageMood(s.cpu)} icon={<Cpu />} />
        <Ring value={mem} label={t("Mémoire")} sub={bytes(s.memTotal, 0)} mood={usageMood(mem)} icon={<MemoryStick />} />
      </div>
      <div className="widget-rows">
        <div><Thermometer /> <span>{t("Température")}</span><b className={`mood-${tempMood(s.temp)}`}>{s.temp ? `${s.temp.toFixed(0)} °C` : "—"}</b></div>
        <div><span className="dot good" /> <span>{t("Allumé depuis")}</span><b>{uptime(s.uptime)}</b></div>
        <div className="widget-machine">{machineName(s.model)}</div>
      </div>
    </section>
  );
}

function StorageWidget({ s }: { s: SystemStats }) {
  return (
    <section className="widget">
      <div className="widget-title"><HardDrive /> {t("Stockage")}</div>
      {s.disks.map((d) => {
        const p = pct(d.used, d.total);
        return (
          <div key={d.path} className="storage-row">
            <div className="row between small"><span>{d.path === "/" ? t("Système") : d.path}</span><span className="muted">{t("{0} libres", bytes(d.free))}</span></div>
            <div className={`progress thin ${usageMood(p) !== "good" ? usageMood(p) : ""}`}><i style={{ width: `${p}%` }} /></div>
          </div>
        );
      })}
    </section>
  );
}

function NetworkWidget({ s, hist }: { s: SystemStats; hist: number[] }) {
  return (
    <section className="widget">
      <div className="widget-title">{t("Réseau")}<span className="spacer" /><span className="muted small">{s.interfaces.flatMap((i) => i.ips)[0] ?? ""}</span></div>
      <div className="net-rates">
        <div><ArrowDown /> <b>{bytes(s.netRx)}/s</b></div>
        <div><ArrowUp /> <b>{bytes(s.netTx)}/s</b></div>
      </div>
      <Sparkline values={hist} color="#5fd39a" />
    </section>
  );
}

function AppsWidget({ data }: { data?: AppsResponse | null }) {
  if (!data) return <section className="widget"><Loading /></section>;
  const up = data.apps.filter((a) => a.state === "running");
  const down = data.apps.filter((a) => a.state !== "running");
  return (
    <section className="widget">
      <div className="widget-title"><Boxes /> {t("Applications")}</div>
      <div className="apps-stats">
        <Link to="/apps"><b className="mood-good">{up.length}</b><span>{t("en marche")}</span></Link>
        <Link to="/apps"><b className={down.length ? "mood-warn" : ""}>{down.length}</b><span>{down.length > 1 ? t("arrêtées") : t("arrêtée")}</span></Link>
      </div>
      {down.length > 0 ? <div className="small muted">{t("À l'arrêt : {0}", down.slice(0, 3).map((a) => a.name).join(", ") + (down.length > 3 ? "…" : ""))}</div>
        : data.apps.length > 0 && <div className="small muted">{t("Toutes tes applications tournent. 👌")}</div>}
    </section>
  );
}

function NoteWidget() {
  const [text, setText] = useState<string | null>(null);
  const [state, setState] = useState<"idle" | "saving" | "saved">("idle");
  const timer = useRef<number>(undefined);
  useEffect(() => { api.get<{ text: string }>("/api/me/note").then((r) => setText(r.text)).catch(() => setText("")); }, []);
  const change = (v: string) => {
    setText(v); setState("saving");
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => { api.put("/api/me/note", { text: v }).then(() => setState("saved")).catch(() => setState("idle")); }, 700);
  };
  return (
    <section className="widget">
      <div className="widget-title"><StickyNote /> {t("Pense-bête")}<span className="spacer" /><span className="small muted">{state === "saving" ? "…" : state === "saved" ? t("Enregistré ✓") : ""}</span></div>
      {text === null ? <Loading /> : (
        <textarea className="note-area" value={text} maxLength={4000} onChange={(e) => change(e.target.value)}
          placeholder={t("Une idée, une liste de courses, le code du Wi-Fi des invités… C'est enregistré sur ton Pi.")} />
      )}
    </section>
  );
}

const DEFAULT_WIDGETS: WidgetId[] = ["clock", "system", "storage", "network"];

function widgetCatalog(): { id: WidgetId; label: string; desc: string; icon: React.ReactNode }[] {
  return [
    { id: "clock", label: t("Horloge"), desc: t("L'heure, la date et un petit mot."), icon: <Clock3 /> },
    { id: "system", label: t("Système"), desc: t("Processeur, mémoire, température."), icon: <Activity /> },
    { id: "storage", label: t("Stockage"), desc: t("La place libre sur tes disques."), icon: <HardDrive /> },
    { id: "network", label: t("Réseau"), desc: t("Le débit en direct."), icon: <Network /> },
    { id: "apps", label: t("Applications"), desc: t("Combien tournent, lesquelles sont arrêtées."), icon: <Boxes /> },
    { id: "note", label: t("Pense-bête"), desc: t("Des notes rapides, gardées sur ton Pi."), icon: <StickyNote /> },
  ];
}

export function Desktop() {
  const { status, can, refresh } = useSession();
  const { toast } = useUI();
  const hist = useRef<number[]>([]);
  const sys = usePoll(async () => {
    const s = await api.get<SystemStats>("/api/system");
    hist.current.push(s.netRx + s.netTx);
    if (hist.current.length > 30) hist.current.shift();
    return s;
  }, status.hardware?.refresh ?? 3000);
  const apps = usePoll(() => api.get<AppsResponse>("/api/apps"), 6000);
  const shortcuts = sections.filter((x) => x.desktop && (!x.perm || can(x.perm)));
  const s = sys.data;
  const [widgets, setWidgets] = useState<WidgetId[]>(status.user?.widgets ?? DEFAULT_WIDGETS);
  const [editing, setEditing] = useState(false);
  const [dragging, setDragging] = useState<WidgetId | null>(null);

  const save = useCallback((next: WidgetId[] | null) => {
    setWidgets(next ?? DEFAULT_WIDGETS);
    api.put("/api/me/widgets", { widgets: next }).then(() => refresh()).catch((e) => toast((e as Error).message, "bad"));
  }, [refresh, toast]);
  const move = (id: WidgetId, to: number) => {
    const next = widgets.filter((w) => w !== id);
    next.splice(Math.max(0, Math.min(next.length, to)), 0, id);
    save(next);
  };

  const render = (id: WidgetId) => {
    switch (id) {
      case "clock": return <Clock />;
      case "apps": return <AppsWidget data={apps.data} />;
      case "note": return <NoteWidget />;
    }
    if (!s) return <section className="widget"><Loading /></section>;
    if (id === "system") return <SystemWidget s={s} />;
    if (id === "storage") return <StorageWidget s={s} />;
    return <NetworkWidget s={s} hist={hist.current} />;
  };
  const catalog = widgetCatalog();
  const hidden = catalog.filter((w) => !widgets.includes(w.id));

  return (
    <div className="desktop-grid">
      <aside className={`widgets${editing ? " editing" : ""}`}>
        {widgets.map((id, i) => editing ? (
          <div key={id} className={`widget-slot${dragging === id ? " dragging" : ""}`} draggable
            onDragStart={(e) => { setDragging(id); e.dataTransfer.effectAllowed = "move"; }}
            onDragEnd={() => setDragging(null)}
            onDragOver={(e) => { e.preventDefault(); e.dataTransfer.dropEffect = "move"; }}
            onDrop={(e) => { e.preventDefault(); if (dragging && dragging !== id) move(dragging, i); setDragging(null); }}>
            <div className="widget-tools">
              <GripVertical className="grip" aria-hidden />
              <span>{catalog.find((c) => c.id === id)?.label}</span>
              <span className="spacer" />
              <button type="button" aria-label={t("Monter")} disabled={i === 0} onClick={() => move(id, i - 1)}><ChevronUp /></button>
              <button type="button" aria-label={t("Descendre")} disabled={i === widgets.length - 1} onClick={() => move(id, i + 1)}><ChevronDown /></button>
              <button type="button" aria-label={t("Retirer")} onClick={() => save(widgets.filter((w) => w !== id))}><X /></button>
            </div>
            <div className="widget-preview">{render(id)}</div>
          </div>
        ) : <div key={id}>{render(id)}</div>)}
        {editing && (
          <section className="widget widget-add">
            <div className="widget-title"><Plus /> {t("Ajouter un widget")}</div>
            {hidden.length === 0 ? <p className="small muted">{t("Tous les widgets sont déjà sur ton bureau.")}</p> : (
              <div className="widget-add-list">
                {hidden.map((w) => (
                  <button key={w.id} type="button" onClick={() => save([...widgets, w.id])}>
                    <span className="widget-add-icon">{w.icon}</span>
                    <span><b>{w.label}</b><small>{w.desc}</small></span>
                    <Plus className="plus" />
                  </button>
                ))}
              </div>
            )}
          </section>
        )}
        <div className="widgets-actions">
          {editing ? (
            <>
              <button type="button" className="btn sm glass-btn" onClick={() => save(null)}><RotateCcw /> {t("Par défaut")}</button>
              <button type="button" className="btn sm primary" onClick={() => setEditing(false)}><Check /> {t("Terminé")}</button>
            </>
          ) : (
            <button type="button" className="btn sm glass-btn" onClick={() => setEditing(true)}><SlidersHorizontal /> {t("Personnaliser")}</button>
          )}
        </div>
      </aside>
      <section className="desk-apps">
        <div className="desk-apps-head">
          <h2>{t("Mes applications")}</h2>
          {apps.data && <span className="count">{apps.data.apps.length}</span>}
          <div className="spacer" />
          {can("apps.manage") && <Link to="/store" className="btn sm glass-btn"><Plus /> {t("App Store")}</Link>}
        </div>
        <div className="desk-icons">
          {shortcuts.map((x) => (
            <Link key={x.to} to={x.to} className="desk-icon">
              <SectionTile s={x} size={68} />
              <span className="desk-label">{x.label}</span>
            </Link>
          ))}
        </div>
        {apps.data ? (
          <div className="desktop-apps">
            <AppGrid data={apps.data} reload={apps.reload} extra={can("apps.manage") && (
              <Link to="/store" className="desk-icon add-tile">
                <div className="add-tile-box"><Plus /></div>
                <span className="desk-label">{t("Ajouter une app")}</span>
              </Link>
            )} />
          </div>
        ) : <Loading />}
        {apps.data && apps.data.apps.length === 0 && (
          <p className="desk-hint">{t("Ton bureau est prêt ! Installe Jellyfin pour tes films, Nextcloud pour tes fichiers ou Home Assistant pour ta maison depuis l'App Store.")}</p>
        )}
      </section>
    </div>
  );
}
