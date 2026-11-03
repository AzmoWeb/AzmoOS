import { useEffect, useState } from "react";
import { CalendarClock, Plus, Play, Pencil, Trash2, FolderOpen, Cloud, History, RotateCcw, CheckCircle2, XCircle } from "lucide-react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { BackupFile, BackupSchedule, SchedulesResponse } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useJobs } from "../session";
import { useUI, Modal } from "../components/overlay";
import { Badge, Button, Callout, Card, Empty, Field, Input, Loading, Spinner, ToggleRow } from "../components/ui";
import { ago, bytes, dateTime } from "../lib/format";
import { t } from "../lib/i18n";

export const weekdays = [t("Dimanche"), t("Lundi"), t("Mardi"), t("Mercredi"), t("Jeudi"), t("Vendredi"), t("Samedi")];

function when(b: BackupSchedule) {
  const h = String(b.hour).padStart(2, "0");
  return b.frequency === "daily" ? t("Tous les jours à {0} h", h) : t("Chaque {0} à {1} h", weekdays[b.weekday], h);
}

const isCloud = (target: string) => target.startsWith("rclone:");
function targetLabel(target: string) {
  if (!isCloud(target)) return `/DATA${target.startsWith("/") ? target : "/" + target}`;
  const [remote, ...path] = target.slice(7).split(":");
  return `☁️ ${remote}${path.join(":") ? " → " + path.join(":") : ""}`;
}

export function BackupSchedules() {
  const { toast, confirm } = useUI();
  const { running, kick, onDone } = useJobs();
  const data = usePoll(() => api.get<SchedulesResponse>("/api/backups/schedules"), 15000);
  const [edit, setEdit] = useState<BackupSchedule | null>(null);
  const [files, setFiles] = useState<BackupSchedule | null>(null);
  useEffect(() => onDone((j) => j.kind === "backup" && data.reload()), [onDone, data]);

  if (!data.data) return <Card title={t("Sauvegardes automatiques")} icon={<CalendarClock />}><Loading /></Card>;
  const { schedules, rclone } = data.data;
  const blank: BackupSchedule = { id: "", name: t("Sauvegarde quotidienne"), enabled: true, frequency: "daily", hour: 3, weekday: 0, target: "/Backups/Automatiques", appData: true, keep: 7, lastOk: false };

  return (
    <Card title={t("Sauvegardes automatiques")} icon={<CalendarClock />} right={<Button size="sm" variant="primary" icon={<Plus />} onClick={() => setEdit(blank)}>{t("Planifier")}</Button>}
      sub={t("Azmo OS sauvegarde tout seul, chaque jour ou chaque semaine, vers un disque ou le cloud. S'il était éteint à l'heure prévue, il rattrape au démarrage.")}>
      {schedules.length === 0 ? <Empty emoji="🗓️" title={t("Aucune sauvegarde planifiée")} action={<Button variant="primary" icon={<Plus />} onClick={() => setEdit(blank)}>{t("Planifier ma première sauvegarde")}</Button>}>{t("Le conseil : une sauvegarde chaque nuit sur un disque USB, et une chaque semaine dans le cloud.")}</Empty> : schedules.map((b) => {
        const job = running("backup", b.id);
        return (
          <div key={b.id} className="toggle-row" style={{ alignItems: "flex-start" }}>
            <div className="disk-icon" style={{ width: 40, height: 40 }}>{isCloud(b.target) ? <Cloud /> : <FolderOpen />}</div>
            <div className="txt">
              <b>{b.name} {!b.enabled && <Badge>{t("En pause")}</Badge>}</b>
              <span>{when(b)} · {targetLabel(b.target)} {t("· garde les")} {b.keep} {t("dernières")}</span>
              <div className="small" style={{ marginTop: 4 }}>
                {job ? <span className="row" style={{ gap: 6 }}><Spinner small /> {t("Sauvegarde en cours…")}</span>
                  : b.lastRun ? (b.lastOk
                    ? <span style={{ color: "var(--good)" }}><CheckCircle2 size={13} style={{ verticalAlign: -2 }} /> {t("Réussie")} {ago(b.lastRun)} · {bytes(b.lastSize ?? 0)}</span>
                    : <span style={{ color: "var(--bad)" }}><XCircle size={13} style={{ verticalAlign: -2 }} /> {t("Échec")} {ago(b.lastRun)} : {b.lastError}</span>)
                  : <span className="muted">{t("Jamais lancée")}</span>}
                {b.enabled && b.next ? <span className="muted"> {t("· prochaine")} {dateTime(b.next)}</span> : null}
              </div>
            </div>
            <div className="row" style={{ gap: 2, flexWrap: "nowrap" }}>
              <button className="icon-btn" title={t("Sauvegarder maintenant")} aria-label={t("Sauvegarder maintenant")} disabled={!!job} onClick={async () => { try { await api.post(`/api/backups/schedules/${b.id}/run`); toast(t("Sauvegarde lancée…")); kick(); data.reload(); } catch (e) { toast((e as Error).message, "bad"); } }}><Play /></button>
              <button className="icon-btn" title={t("Sauvegardes enregistrées")} aria-label={t("Sauvegardes enregistrées")} onClick={() => setFiles(b)}><History /></button>
              <button className="icon-btn" title={t("Modifier")} aria-label={t("Modifier")} onClick={() => setEdit(b)}><Pencil /></button>
              <button className="icon-btn" title={t("Supprimer")} aria-label={t("Supprimer")} onClick={async () => {
                if (await confirm({ title: t("Supprimer « {0} » ?", b.name), confirm: t("Supprimer"), body: t("La planification est supprimée. Les sauvegardes déjà créées sont conservées.") })) {
                  await api.del(`/api/backups/schedules/${b.id}`).catch((e) => toast(e.message, "bad")); data.reload();
                }
              }}><Trash2 /></button>
            </div>
          </div>
        );
      })}
      {edit && <ScheduleDialog value={edit} rclone={rclone} onClose={() => setEdit(null)} onSaved={data.reload} />}
      {files && <FilesDialog schedule={files} onClose={() => setFiles(null)} />}
    </Card>
  );
}

function ScheduleDialog({ value, rclone, onClose, onSaved }: { value: BackupSchedule; rclone: SchedulesResponse["rclone"]; onClose: () => void; onSaved: () => void }) {
  const { toast } = useUI();
  const [b, setB] = useState(value);
  const cloud = isCloud(b.target);
  const [remote, setRemote] = useState(() => (cloud ? b.target.slice(7).split(":")[0] : rclone.remotes[0] ?? ""));
  const [cloudPath, setCloudPath] = useState(() => (cloud ? b.target.slice(7).split(":").slice(1).join(":") : "AzmoOS"));
  const [busy, setBusy] = useState(false);
  const set = <K extends keyof BackupSchedule>(k: K, v: BackupSchedule[K]) => setB({ ...b, [k]: v });

  const save = async () => {
    setBusy(true);
    const target = cloud ? `rclone:${remote}:${cloudPath}` : b.target;
    try { await api.post("/api/backups/schedules", { ...b, target }); toast(t("Sauvegarde planifiée ✨"), "good"); onSaved(); onClose(); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };

  return (
    <Modal wide title={value.id ? t("Modifier la sauvegarde") : t("Planifier une sauvegarde")} onClose={onClose}
      footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="primary" loading={busy} disabled={cloud && !remote} onClick={save}>{t("Enregistrer")}</Button></>}>
      <Field label={t("Nom")}><Input value={b.name} onChange={(e) => set("name", e.target.value)} /></Field>
      <div className="grid grid-3" style={{ gap: 14 }}>
        <Field label={t("Fréquence")}>
          <select className="select" value={b.frequency} onChange={(e) => set("frequency", e.target.value as BackupSchedule["frequency"])}>
            <option value="daily">{t("Tous les jours")}</option><option value="weekly">{t("Toutes les semaines")}</option>
          </select>
        </Field>
        {b.frequency === "weekly" && (
          <Field label={t("Jour")}>
            <select className="select" value={b.weekday} onChange={(e) => set("weekday", +e.target.value)}>
              {weekdays.map((d, i) => <option key={i} value={i}>{d}</option>)}
            </select>
          </Field>
        )}
        <Field label={t("Heure")} hint={t("La nuit, ton Pi est moins occupé.")}>
          <select className="select" value={b.hour} onChange={(e) => set("hour", +e.target.value)}>
            {Array.from({ length: 24 }, (_, h) => <option key={h} value={h}>{String(h).padStart(2, "0")} {t("h 00")}</option>)}
          </select>
        </Field>
      </div>

      <Field label={t("Destination")}>
        <div className="grid grid-2" style={{ gap: 10 }}>
          {([["local", "💽 Disque ou dossier", "Un disque USB, un partage réseau monté, ou un dossier du Pi."], ["cloud", "☁️ Cloud", "Google Drive, Dropbox, OneDrive, S3… avec rclone."]] as const).map(([id, t, d]) => (
            <button type="button" key={id} className="card flat" onClick={() => set("target", id === "cloud" ? "rclone:" : "/Backups/Automatiques")}
              style={{ textAlign: "left", cursor: "pointer", padding: 14, borderColor: (id === "cloud") === cloud ? "var(--brand)" : undefined, boxShadow: (id === "cloud") === cloud ? "0 0 0 3px var(--brand-soft)" : undefined }}>
              <b>{t}</b><div className="small muted">{d}</div>
            </button>
          ))}
        </div>
      </Field>
      {!cloud ? (
        <Field label={t("Dossier (dans /DATA)")} hint={<>{t("Pour un disque USB :")} <code>/Disks/NomDuDisque/Sauvegardes</code>{t(". Branche-le et clique « Utiliser ce disque » dans Disques & partages.")}</>}>
          <Input value={b.target} onChange={(e) => set("target", e.target.value)} />
        </Field>
      ) : !rclone.installed ? (
        <Callout mood="warn" title={t("rclone n'est pas installé")}>{t("Ouvre le")} <Link to="/terminal">{t("Terminal")}</Link> {t("et lance")} <code>sudo apt install -y rclone</code>.</Callout>
      ) : rclone.remotes.length === 0 ? (
        <Callout title={t("Connecte d'abord ton compte cloud")}>
          {t("Ouvre le")} <Link to="/terminal">{t("Terminal")}</Link> {t("et lance :")}<br /><code style={{ userSelect: "all" }}>sudo rclone config --config {rclone.config}</code><br />
          {t("Choisis « n » (nouveau), donne un nom (ex :")} <code>gdrive</code>{t("), choisis ton service et suis les étapes. Reviens ensuite ici.")}
        </Callout>
      ) : (
        <div className="grid grid-2" style={{ gap: 14 }}>
          <Field label={t("Compte cloud")}>
            <select className="select" value={remote} onChange={(e) => setRemote(e.target.value)}>{rclone.remotes.map((r) => <option key={r}>{r}</option>)}</select>
          </Field>
          <Field label={t("Dossier dans le cloud")}><Input value={cloudPath} onChange={(e) => setCloudPath(e.target.value)} /></Field>
        </div>
      )}
      <div className="grid grid-2" style={{ gap: 14 }}>
        <Field label={t("Nombre de sauvegardes à garder")} hint={t("Les plus anciennes sont supprimées automatiquement.")}>
          <Input type="number" min={1} max={365} value={b.keep} onChange={(e) => set("keep", Math.max(1, +e.target.value))} />
        </Field>
      </div>
      <ToggleRow title={t("Inclure les données des applications")} desc={t("Bases de données et réglages internes des apps (conseillé). Tes films et photos ne sont jamais inclus.")} checked={b.appData} onChange={(v) => set("appData", v)} />
      <ToggleRow title={t("Activée")} desc={t("Décoche pour mettre en pause sans supprimer.")} checked={b.enabled} onChange={(v) => set("enabled", v)} />
    </Modal>
  );
}

function FilesDialog({ schedule, onClose }: { schedule: BackupSchedule; onClose: () => void }) {
  const { toast, confirm } = useUI();
  const files = usePoll(() => api.get<BackupFile[]>(`/api/backups/schedules/${schedule.id}/files`));
  const restore = async (f: BackupFile) => {
    if (!(await confirm({ title: t("Restaurer cette sauvegarde ?"), danger: true, confirm: t("Restaurer"), body: <>{t("La configuration actuelle sera remplacée par celle du")} <b>{dateTime(f.time)}</b>{t(". Les applications seront redéployées.")}</> }))) return;
    try { await api.post(`/api/backups/schedules/${schedule.id}/restore`, { file: f.name }); toast(t("Sauvegarde restaurée ✨"), "good"); onClose(); }
    catch (e) { toast((e as Error).message, "bad"); }
  };
  return (
    <Modal title={t("Sauvegardes de « {0} »", schedule.name)} sub={targetLabel(schedule.target)} onClose={onClose}>
      {files.error ? <Callout mood="bad">{files.error}</Callout> : !files.data ? <Loading /> : files.data.length === 0 ? <Empty emoji="📭" title={t("Aucune sauvegarde pour l'instant")} /> : (
        files.data.map((f) => (
          <div key={f.name} className="toggle-row">
            <div className="txt"><b>{dateTime(f.time)}</b><span className="mono">{f.name} · {bytes(f.size)}</span></div>
            <Button size="sm" icon={<RotateCcw />} onClick={() => restore(f)}>{t("Restaurer")}</Button>
          </div>
        ))
      )}
    </Modal>
  );
}
