import { useRef, useState } from "react";
import { Download, Upload, Archive, ShieldCheck } from "lucide-react";
import { api } from "../api/client";
import { useUI } from "../components/overlay";
import { Button, Callout, Card, PageHead, ToggleRow } from "../components/ui";
import { dateTime } from "../lib/format";
import { BackupSchedules } from "./BackupSchedules";
import { t } from "../lib/i18n";

export function Backups() {
  const { toast, confirm } = useUI();
  const [withData, setWithData] = useState(false);
  const [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);

  const restore = async (file: File) => {
    if (!(await confirm({ title: t("Restaurer cette sauvegarde ?"), danger: true, confirm: t("Restaurer"), body: <>{t("La configuration actuelle (comptes, applications, partages) sera remplacée par celle de")} <b>{file.name}</b>{t(". Les applications seront redéployées automatiquement.")}</> }))) return;
    setBusy(true);
    const fd = new FormData();
    fd.append("file", file);
    try {
      const m = await api.upload<{ created: number; apps: string[]; hostname: string }>("/api/backup/restore", fd);
      toast(t("Sauvegarde du {0} restaurée ({1} apps). Reconnecte-toi si besoin.", dateTime(m.created), m.apps.length), "good");
    } catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };

  return (
    <>
      <PageHead title={t("Sauvegardes")} sub={t("Mets ta configuration à l'abri. En cas de pépin (carte SD morte, nouveau Pi…), tu retrouves tout en un clic.")} />
      <div style={{ marginBottom: 18 }}><BackupSchedules /></div>
      <div className="grid grid-2">
        <Card title={t("Sauvegarde manuelle")} icon={<Archive />}>
          <p className="muted" style={{ marginBottom: 12 }}>{t("Contient tes comptes, tes réglages, tes partages et la configuration de chaque application.")}</p>
          <ToggleRow title={t("Inclure les données des applications")} desc={t("Bases de données, réglages internes des apps (dossier AppData). Le fichier sera plus gros. Tes fichiers personnels (films, photos…) ne sont jamais inclus.")} checked={withData} onChange={setWithData} />
          <a className="btn primary" style={{ marginTop: 16 }} href={`/api/backup${withData ? "?appdata=1" : ""}`} download onClick={() => toast(t("Préparation de la sauvegarde… le téléchargement va démarrer."))}><Download /> {t("Télécharger la sauvegarde")}</a>
        </Card>
        <Card title={t("Restaurer")} icon={<Upload />}>
          <p className="muted" style={{ marginBottom: 16 }}>{t("Choisis un fichier")} <code>azmoos-sauvegarde-….tar.gz</code> {t("créé par Azmo OS.")}</p>
          <Button icon={<Upload />} loading={busy} onClick={() => input.current?.click()}>{t("Choisir un fichier")}</Button>
          <input ref={input} type="file" accept=".gz,.tgz,application/gzip" hidden onChange={(e) => { const f = e.target.files?.[0]; if (f) restore(f); e.target.value = ""; }} />
        </Card>
      </div>
      <div style={{ marginTop: 18 }}>
        <Callout mood="good" title={t("Le bon réflexe 💡")}><ShieldCheck size={0} />{t("Garde une copie de ta sauvegarde ailleurs que sur le Pi : sur ton ordinateur, une clé USB ou ton cloud. Pour tes films et photos, utilise plutôt Duplicati ou Syncthing depuis l'App Store.")}</Callout>
      </div>
    </>
  );
}
