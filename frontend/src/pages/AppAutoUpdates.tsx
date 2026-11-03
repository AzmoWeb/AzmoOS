// Mises à jour automatiques des applications (Docker) : planning, apps concernées, historique.
import { useEffect, useState } from "react";
import { Boxes, Play, CheckCircle2, XCircle } from "lucide-react";
import { api } from "../api/client";
import type { AppUpdateSettings, AppUpdatesResponse } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useUI } from "../components/overlay";
import { AppIcon } from "../components/AppIcon";
import { Badge, Button, Card, Field, Loading, Spinner, Switch } from "../components/ui";
import { ago, dateTime } from "../lib/format";
import { t, tServer } from "../lib/i18n";
import { weekdays } from "./BackupSchedules";

export function AppAutoUpdates() {
  const { toast } = useUI();
  const data = usePoll(() => api.get<AppUpdatesResponse>("/api/app-updates"), 5000);
  const [cfg, setCfg] = useState<AppUpdateSettings | null>(null);
  useEffect(() => { if (data.data && !cfg) setCfg(data.data.settings); }, [data.data, cfg]);
  if (!data.data || !cfg) return <Card title={t("Mises à jour automatiques des applications")} icon={<Boxes />}><Loading /></Card>;
  const d = data.data;

  const save = async (next: AppUpdateSettings) => {
    setCfg(next);
    try {
      await api.put("/api/app-updates", { enabled: next.enabled, frequency: next.frequency, hour: next.hour, weekday: next.weekday, exclude: next.exclude });
      data.reload();
    } catch (e) { toast((e as Error).message, "bad"); setCfg(d.settings); }
  };
  const set = <K extends keyof AppUpdateSettings>(k: K, v: AppUpdateSettings[K]) => save({ ...cfg, [k]: v });
  const toggleApp = (id: string, on: boolean) => set("exclude", on ? cfg.exclude.filter((x) => x !== id) : [...cfg.exclude, id]);
  const runNow = async () => {
    try { await api.post("/api/app-updates/run"); toast(t("Je vérifie tes applications une par une…")); setTimeout(data.reload, 800); }
    catch (e) { toast((e as Error).message, "bad"); }
  };
  const current = d.apps.find((a) => a.id === d.current)?.name ?? d.current;
  const s = d.settings;

  return (
    <Card title={t("Mises à jour automatiques des applications")} icon={<Boxes />}
      right={<Switch checked={cfg.enabled} onChange={(v) => set("enabled", v)} label={t("Mises à jour automatiques des applications")} />}>
      <p className="muted" style={{ marginBottom: 14 }}>
        {t("Chaque nuit, Azmo OS regarde si tes applications ont une nouvelle version et l'installe. Tes réglages et tes données sont gardés. Une app à jour n'est pas redémarrée, et une app que tu as arrêtée reste arrêtée.")}
      </p>
      <div className="grid grid-3" style={{ gap: 14, opacity: cfg.enabled ? 1 : 0.55 }}>
        <Field label={t("Fréquence")}>
          <select className="select" value={cfg.frequency} disabled={!cfg.enabled} onChange={(e) => set("frequency", e.target.value as AppUpdateSettings["frequency"])}>
            <option value="daily">{t("Tous les jours")}</option><option value="weekly">{t("Toutes les semaines")}</option>
          </select>
        </Field>
        {cfg.frequency === "weekly" && (
          <Field label={t("Jour")}>
            <select className="select" value={cfg.weekday} disabled={!cfg.enabled} onChange={(e) => set("weekday", +e.target.value)}>
              {weekdays.map((w, i) => <option key={i} value={i}>{w}</option>)}
            </select>
          </Field>
        )}
        <Field label={t("Heure")} hint={t("La nuit, ton Pi est moins occupé.")}>
          <select className="select" value={cfg.hour} disabled={!cfg.enabled} onChange={(e) => set("hour", +e.target.value)}>
            {Array.from({ length: 24 }, (_, h) => <option key={h} value={h}>{String(h).padStart(2, "0")} {t("h 00")}</option>)}
          </select>
        </Field>
      </div>

      <div className="row" style={{ gap: 10, margin: "6px 0 16px", flexWrap: "wrap" }}>
        {d.running ? <span className="row"><Spinner small /> {current ? t("Vérification de « {0} »…", current) : t("Vérification en cours…")}</span>
          : s.lastRun ? <span className="small muted">
            {t("Dernier passage {0} : {1} app(s) vérifiée(s), {2} mise(s) à jour.", ago(s.lastRun), s.lastCheck, s.lastCount)}
            {s.lastFail > 0 && <> <Badge mood="bad">{t("{0} échec(s)", s.lastFail)}</Badge></>}
          </span> : null}
        {cfg.enabled && d.next > 0 && !d.running && <Badge mood="brand">{t("Prochain passage : {0}", dateTime(d.next))}</Badge>}
        <div className="spacer" />
        <Button size="sm" icon={<Play />} disabled={d.running || d.apps.length === 0} onClick={runNow}>{t("Vérifier maintenant")}</Button>
      </div>

      {d.apps.length > 0 && (
        <>
          <div className="small muted" style={{ marginBottom: 8 }}>{t("Applications concernées")}</div>
          <div className="autoupd-apps">
            {d.apps.map((a) => (
              <label key={a.id} className="autoupd-app">
                <AppIcon name={a.name} icon={a.icon} size={34} />
                <span>{a.name}</span>
                <Switch checked={!cfg.exclude.includes(a.id)} onChange={(v) => toggleApp(a.id, v)} label={a.name} />
              </label>
            ))}
          </div>
          <p className="small muted" style={{ marginTop: 8 }}>{t("Décoche une app si tu préfères choisir toi-même quand la mettre à jour (par exemple pour une base de données).")}</p>
        </>
      )}

      {s.history.length > 0 && (
        <>
          <div className="small muted" style={{ margin: "18px 0 8px" }}>{t("Historique")}</div>
          <ul className="autoupd-history">
            {s.history.slice(0, 8).map((h, i) => (
              <li key={i}>
                {h.result === "updated" ? <CheckCircle2 className="mood-good" /> : <XCircle className="mood-bad" />}
                <b>{h.name}</b>
                <span className="muted">{h.result === "updated" ? t("nouvelle version installée") : tServer(h.error ?? "")}</span>
                <span className="spacer" />
                <span className="small muted">{dateTime(h.time)}</span>
              </li>
            ))}
          </ul>
        </>
      )}
    </Card>
  );
}
