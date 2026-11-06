import { useState } from "react";
import { HardDrive, Usb, MemoryStick, Share2, Plus, Trash2, KeyRound, FolderOpen, Copy } from "lucide-react";
import { api, q } from "../api/client";
import type { Disk, Partition, Share, SharesResponse, User } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useJobs, useSession } from "../session";
import { useUI, Modal } from "../components/overlay";
import { Badge, Button, Callout, Card, Empty, ErrorBox, Field, Input, Loading, PageHead, PasswordInput, Progress, Switch, Tabs, ToggleRow } from "../components/ui";
import { bytes, pct } from "../lib/format";
import { usageMood } from "../lib/human";
import { useEffect } from "react";
import { t, tServer } from "../lib/i18n";

export function Storage() {
  const [tab, setTab] = useState<"disks" | "shares">("disks");
  return (
    <>
      <PageHead title={t("Disques & partages")} sub={t("Branche un disque dur ou une clé USB, et partage tes dossiers avec tous les appareils de la maison.")} />
      <Tabs value={tab} onChange={setTab} tabs={[{ id: "disks", label: t("Disques"), icon: <HardDrive /> }, { id: "shares", label: t("Partages réseau (NAS)"), icon: <Share2 /> }]} />
      {tab === "disks" ? <Disks /> : <Shares />}
    </>
  );
}

const transportIcon = (t: string) => (t === "usb" ? <Usb /> : t === "mmc" ? <MemoryStick /> : <HardDrive />);
const transportLabel: Record<string, string> = { usb: "USB", nvme: "NVMe", mmc: "microSD", sata: "SATA" };

function Disks() {
  const { toast, confirm } = useUI();
  const { running, kick, onDone } = useJobs();
  const disks = usePoll(() => api.get<Disk[]>("/api/storage/disks"), 5000);
  const [format, setFormat] = useState<Disk | null>(null);
  useEffect(() => onDone((j) => j.kind === "storage" && disks.reload()), [onDone, disks]);

  const op = async (name: string, body: object, ok: string) => {
    try { await api.post(`/api/storage/${name}`, body); toast(ok, "good"); disks.reload(); } catch (e) { toast((e as Error).message, "bad"); }
  };

  if (disks.error && !disks.data) return <ErrorBox error={disks.error} onRetry={disks.reload} />;
  if (!disks.data) return <Loading label={t("Je cherche tes disques…")} />;
  const external = disks.data.filter((d) => !d.system);
  return (
    <div className="stack">
      {external.length === 0 && <Callout title={t("Aucun disque externe détecté")}>{t("Branche un disque dur ou une clé USB sur ton Pi : il apparaîtra ici automatiquement en quelques secondes.")}</Callout>}
      {disks.data.map((d) => {
        const job = running("storage", d.name);
        return (
          <Card key={d.path}>
            <div className="disk">
              <div className="disk-icon">{transportIcon(d.transport)}</div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="row">
                  <h3 style={{ fontSize: 16.5 }}>{tServer(d.model || d.name)}</h3>
                  {d.system && <Badge mood="brand">{t("Disque système")}</Badge>}
                  {d.transport && <Badge>{transportLabel[d.transport] ?? d.transport}</Badge>}
                  <div className="spacer" />
                  {!d.system && <Button size="sm" variant="danger" loading={!!job} onClick={() => setFormat(d)}>{job ? t("Formatage…") : t("Formater")}</Button>}
                </div>
                <div className="muted small">{bytes(d.size)} · {d.path}</div>
                {d.partitions.length === 0 && <div className="part muted small">{t("Ce disque n'a aucune partition. Formate-le pour pouvoir l'utiliser.")}</div>}
                {d.partitions.map((p) => <PartitionRow key={p.path} p={p} onOp={op} confirm={confirm} />)}
              </div>
            </div>
          </Card>
        );
      })}
      {format && <FormatDialog disk={format} onClose={() => setFormat(null)} onDone={() => { kick(); disks.reload(); }} />}
    </div>
  );
}

function PartitionRow({ p, onOp, confirm }: { p: Partition; onOp: (n: string, b: object, ok: string) => void; confirm: ReturnType<typeof useUI>["confirm"] }) {
  const used = p.fsUsed + p.fsAvail ? pct(p.fsUsed, p.fsUsed + p.fsAvail) : 0;
  return (
    <div className="part">
      <div className="row">
        <b style={{ fontSize: 14 }}>{p.label || p.name}</b>
        <Badge>{p.fstype || t("non formatée")}</Badge>
        <span className="muted small">{bytes(p.size)}</span>
        <div className="spacer" />
        {!p.system && p.fstype && (p.mountpoint
          ? <Button size="sm" onClick={async () => { if (await confirm({ title: t("Éjecter ce disque ?"), confirm: t("Éjecter"), body: t("Les applications qui l'utilisent n'y auront plus accès. Tu pourras ensuite le débrancher sans risque.") })) onOp("unmount", { path: p.path }, t("Disque éjecté, tu peux le débrancher 👍")); }}>{t("Éjecter")}</Button>
          : <Button size="sm" variant="primary" onClick={() => onOp("mount", { path: p.path }, t("Disque prêt ! Retrouve-le dans Fichiers → Disques"))}>{t("Utiliser ce disque")}</Button>)}
      </div>
      {p.mountpoint && (
        <>
          <div className="row small muted" style={{ margin: "8px 0 6px" }}>
            <FolderOpen size={14} /> <span className="mono">{p.mountpoint}</span>
            <div className="spacer" />{bytes(p.fsAvail)} {t("libres")}
          </div>
          <Progress value={used} mood={usageMood(used)} thin />
        </>
      )}
      {!p.system && p.uuid && (
        <div className="row small" style={{ marginTop: 10, gap: 10 }}>
          <Switch checked={p.autoMount} onChange={(v) => onOp("automount", { path: p.path, enabled: v }, v ? t("Ce disque sera monté à chaque démarrage") : t("Montage automatique désactivé"))} label={t("Monter au démarrage")} />
          <span className="muted">{t("Monter automatiquement au démarrage du Pi")}</span>
        </div>
      )}
    </div>
  );
}

function FormatDialog({ disk, onClose, onDone }: { disk: Disk; onClose: () => void; onDone: () => void }) {
  const { toast, confirm } = useUI();
  const [fs, setFs] = useState<"ext4" | "exfat">("ext4");
  const [label, setLabel] = useState("AzmoDisk");
  const go = async () => {
    if (!(await confirm({ title: t("Dernière vérification"), danger: true, confirm: t("Tout effacer et formater"), typeToConfirm: "EFFACER", body: <>{t("Toutes les données de")} <b>{disk.model || disk.name}</b> ({bytes(disk.size)}{t(") seront")} <b>{t("définitivement effacées")}</b>.</> }))) return;
    try { await api.post("/api/storage/format", { path: disk.path, fstype: fs, label }); toast(t("Formatage en cours…")); onDone(); onClose(); }
    catch (e) { toast((e as Error).message, "bad"); }
  };
  return (
    <Modal title={t("Formater {0}", disk.model || disk.name)} sub={`${bytes(disk.size)} · ${disk.path}`} onClose={onClose}
      footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="danger-solid" onClick={go}>{t("Formater")}</Button></>}>
      <Callout mood="warn">{t("Formater efface tout le contenu du disque.")}</Callout>
      <div style={{ marginTop: 18 }}>
        <Field label={t("Format")}>
          <div className="grid grid-2" style={{ gap: 10 }}>
            {([["ext4", "Linux (ext4)", "Le plus fiable et rapide pour ton Pi. Conseillé."], ["exfat", "Universel (exFAT)", "Lisible aussi sur Windows et Mac si tu débranches le disque."]] as const).map(([id, t, d]) => (
              <button type="button" key={id} className="card flat" onClick={() => setFs(id)} style={{ textAlign: "left", cursor: "pointer", padding: 14, borderColor: fs === id ? "var(--brand)" : undefined, boxShadow: fs === id ? "0 0 0 3px var(--brand-soft)" : undefined }}>
                <b>{t}</b><div className="small muted">{d}</div>
              </button>
            ))}
          </div>
        </Field>
        <Field label={t("Nom du disque")} hint={t("11 caractères maximum.")}><Input value={label} maxLength={11} onChange={(e) => setLabel(e.target.value.replace(/[^A-Za-z0-9_-]/g, ""))} /></Field>
      </div>
    </Modal>
  );
}

function Shares() {
  const { toast, confirm } = useUI();
  const { status } = useSession();
  const data = usePoll(() => api.get<SharesResponse>("/api/shares"));
  const users = usePoll(() => api.get<{ users: User[] }>("/api/users"));
  const [edit, setEdit] = useState<Share | null>(null);
  const [pw, setPw] = useState<string | null>(null);

  if (data.error && !data.data) return <ErrorBox error={data.error} onRetry={data.reload} />;
  if (!data.data) return <Loading />;
  const { status: st, shares } = data.data;
  const host = status.hostname;
  const copy = (text: string) => navigator.clipboard?.writeText(text).then(() => toast(t("Copié !"), "good"));

  if (!st.installed) return (
    <Card><Empty emoji="🗂️" title={t("Le partage réseau n'est pas installé")}>{t("Installe Samba sur ton Pi avec")} <code>sudo apt install samba</code> {t("(déjà inclus dans l'image Azmo OS), puis reviens ici.")}</Empty></Card>
  );

  return (
    <div className="stack">
      <Card title={t("Comment s'y connecter ?")} icon={<Share2 />}>
        <div className="grid grid-3" style={{ gap: 14 }}>
          {[["🪟 Windows", `\\\\${host}`], ["🍎 Mac", `smb://${host}.local`], ["🐧 Linux / 📱 Mobile", `smb://${host}.local`]].map(([os, addr]) => (
            <div key={os} className="card flat" style={{ padding: 14 }}>
              <div className="small bold">{os}</div>
              <div className="row" style={{ marginTop: 4 }}><code style={{ flex: 1 }}>{addr}</code><button className="icon-btn" onClick={() => copy(addr)} aria-label={t("Copier")}><Copy /></button></div>
            </div>
          ))}
        </div>
        <p className="small muted" style={{ marginTop: 12 }}>{t("Dans l'explorateur de fichiers de ton ordinateur, saisis cette adresse puis connecte-toi avec ton nom d'utilisateur et ton")} <b>{t("mot de passe réseau")}</b>.</p>
      </Card>

      <Card title={t("Dossiers partagés")} icon={<FolderOpen />} right={<Button size="sm" variant="primary" icon={<Plus />} onClick={() => setEdit({ name: "", path: "/", readOnly: false, guest: false, users: [] })}>{t("Partager un dossier")}</Button>}>
        {shares.length === 0 ? <Empty emoji="📂" title={t("Aucun dossier partagé")}>{t("Partage par exemple ton dossier Media pour regarder tes films depuis la télé ou l'ordinateur.")}</Empty> : (
          shares.map((s) => (
            <div key={s.name} className="toggle-row">
              <div className="txt">
                <b>{s.name}</b>
                <span className="mono">{"/DATA"}{s.path}</span>
                <div className="row" style={{ gap: 6, marginTop: 6 }}>
                  {s.guest ? <Badge mood="warn">{t("Accès libre sur le réseau")}</Badge> : <Badge>{s.users?.length ? s.users.join(", ") : t("Tous les comptes réseau")}</Badge>}
                  {s.readOnly && <Badge>{t("Lecture seule")}</Badge>}
                </div>
              </div>
              <Button size="sm" onClick={() => setEdit(s)}>{t("Modifier")}</Button>
              <button className="icon-btn" aria-label={t("Supprimer")} onClick={async () => {
                if (await confirm({ title: t("Arrêter de partager « {0} » ?", s.name), confirm: t("Arrêter le partage"), body: t("Les fichiers ne sont pas supprimés, ils ne seront juste plus visibles sur le réseau.") })) {
                  try { await api.del(`/api/shares/${q(s.name)}`); data.reload(); } catch (e) { toast((e as Error).message, "bad"); }
                }
              }}><Trash2 /></button>
            </div>
          ))
        )}
      </Card>

      <Card title={t("Comptes réseau")} icon={<KeyRound />} sub={t("Pour des raisons de sécurité, l'accès réseau utilise un mot de passe dédié, distinct de celui de l'interface.")}>
        {(users.data?.users ?? []).map((u) => {
          const on = st.smbUsers.includes(u.username);
          return (
            <div key={u.username} className="toggle-row">
              <div className="txt"><b>{u.display}</b><span>{u.username} · {on ? t("accès réseau activé") : t("pas d'accès réseau")}</span></div>
              <Button size="sm" onClick={() => setPw(u.username)}>{on ? t("Changer le mot de passe") : t("Activer l'accès")}</Button>
              {on && <button className="icon-btn" aria-label={t("Retirer")} onClick={async () => { try { await api.del(`/api/shares/users/${q(u.username)}`); data.reload(); } catch (e) { toast((e as Error).message, "bad"); } }}><Trash2 /></button>}
            </div>
          );
        })}
      </Card>

      {edit && <ShareDialog share={edit} users={st.smbUsers} onClose={() => setEdit(null)} onSaved={data.reload} />}
      {pw && <NetPasswordDialog username={pw} onClose={() => setPw(null)} onSaved={data.reload} />}
    </div>
  );
}

function ShareDialog({ share, users, onClose, onSaved }: { share: Share; users: string[]; onClose: () => void; onSaved: () => void }) {
  const { toast } = useUI();
  const [s, setS] = useState<Share>({ ...share, users: share.users ?? [] });
  const [busy, setBusy] = useState(false);
  const save = async () => {
    setBusy(true);
    try { await api.post("/api/shares", s); toast(t("« {0} » est partagé sur le réseau 🎉", s.name), "good"); onSaved(); onClose(); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <Modal title={share.name ? t("Modifier « {0} »", share.name) : t("Partager un dossier")} onClose={onClose}
      footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="primary" loading={busy} disabled={!s.name} onClick={save}>{t("Enregistrer")}</Button></>}>
      <Field label={t("Nom du partage")} hint={t("C'est le nom qui apparaîtra sur ton ordinateur.")}><Input value={s.name} onChange={(e) => setS({ ...s, name: e.target.value })} placeholder={t("Films")} disabled={!!share.name} autoFocus /></Field>
      <Field label={t("Dossier à partager")} hint={t("Chemin dans /DATA, par exemple /Media ou /Disks/MonDisque.")}><Input value={s.path} onChange={(e) => setS({ ...s, path: e.target.value })} /></Field>
      <ToggleRow title={t("Lecture seule")} desc={t("Personne ne pourra modifier ou supprimer les fichiers.")} checked={s.readOnly} onChange={(v) => setS({ ...s, readOnly: v })} />
      <ToggleRow title={t("Accès libre (invité)")} desc={t("Tout appareil de ton réseau peut y accéder sans mot de passe.")} checked={s.guest} onChange={(v) => setS({ ...s, guest: v })} />
      {!s.guest && users.length > 0 && (
        <Field label={t("Qui peut y accéder ?")} hint={t("Si personne n'est coché, tous les comptes réseau y ont accès.")}>
          <div className="chips" style={{ marginBottom: 0 }}>
            {users.map((u) => {
              const on = s.users!.includes(u);
              return <button type="button" key={u} className={`chip ${on ? "active" : ""}`} onClick={() => setS({ ...s, users: on ? s.users!.filter((x) => x !== u) : [...s.users!, u] })}>{u}</button>;
            })}
          </div>
        </Field>
      )}
    </Modal>
  );
}

function NetPasswordDialog({ username, onClose, onSaved }: { username: string; onClose: () => void; onSaved: () => void }) {
  const { toast } = useUI();
  const [pw, setPw] = useState("");
  const [busy, setBusy] = useState(false);
  const save = async () => {
    setBusy(true);
    try { await api.post("/api/shares/users", { username, password: pw }); toast(t("Accès réseau prêt pour {0}", username), "good"); onSaved(); onClose(); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <Modal title={t("Mot de passe réseau de {0}", username)} onClose={onClose} footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="primary" loading={busy} disabled={pw.length < 8} onClick={save}>{t("Enregistrer")}</Button></>}>
      <Field label={t("Mot de passe réseau")} hint={t("8 caractères minimum. Il te sera demandé par ton ordinateur pour ouvrir les dossiers partagés.")}><PasswordInput value={pw} onChange={(e) => setPw(e.target.value)} autoFocus /></Field>
    </Modal>
  );
}
