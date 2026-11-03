import { useRef, useState } from "react";
import { Download, Upload, Archive, ShieldCheck } from "lucide-react";
import { api } from "../api/client";
import { useUI } from "../components/overlay";
import { Button, Callout, Card, PageHead, ToggleRow } from "../components/ui";
import { dateTime } from "../lib/format";

export function Backups() {
  const { toast, confirm } = useUI();
  const [withData, setWithData] = useState(false);
  const [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);

  const restore = async (file: File) => {
    if (!(await confirm({ title: "Restaurer cette sauvegarde ?", danger: true, confirm: "Restaurer", body: <>La configuration actuelle (comptes, applications, partages) sera remplacée par celle de <b>{file.name}</b>. Les applications seront redéployées automatiquement.</> }))) return;
    setBusy(true);
    const fd = new FormData();
    fd.append("file", file);
    try {
      const m = await api.upload<{ created: number; apps: string[]; hostname: string }>("/api/backup/restore", fd);
      toast(`Sauvegarde du ${dateTime(m.created)} restaurée (${m.apps.length} apps). Reconnecte-toi si besoin.`, "good");
    } catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };

  return (
    <>
      <PageHead title="Sauvegardes" sub="Mets ta configuration à l'abri. En cas de pépin (carte SD morte, nouveau Pi…), tu retrouves tout en un clic." />
      <div className="grid grid-2">
        <Card title="Créer une sauvegarde" icon={<Archive />}>
          <p className="muted" style={{ marginBottom: 12 }}>Contient tes comptes, tes réglages, tes partages et la configuration de chaque application.</p>
          <ToggleRow title="Inclure les données des applications" desc="Bases de données, réglages internes des apps (dossier AppData). Le fichier sera plus gros. Tes fichiers personnels (films, photos…) ne sont jamais inclus." checked={withData} onChange={setWithData} />
          <a className="btn primary" style={{ marginTop: 16 }} href={`/api/backup${withData ? "?appdata=1" : ""}`} download onClick={() => toast("Préparation de la sauvegarde… le téléchargement va démarrer.")}><Download /> Télécharger la sauvegarde</a>
        </Card>
        <Card title="Restaurer" icon={<Upload />}>
          <p className="muted" style={{ marginBottom: 16 }}>Choisis un fichier <code>azmoos-sauvegarde-….tar.gz</code> créé par Azmo OS.</p>
          <Button icon={<Upload />} loading={busy} onClick={() => input.current?.click()}>Choisir un fichier</Button>
          <input ref={input} type="file" accept=".gz,.tgz,application/gzip" hidden onChange={(e) => { const f = e.target.files?.[0]; if (f) restore(f); e.target.value = ""; }} />
        </Card>
      </div>
      <div style={{ marginTop: 18 }}>
        <Callout mood="good" title="Le bon réflexe 💡"><ShieldCheck size={0} />Garde une copie de ta sauvegarde ailleurs que sur le Pi : sur ton ordinateur, une clé USB ou ton cloud. Pour tes films et photos, utilise plutôt Duplicati ou Syncthing depuis l'App Store.</Callout>
      </div>
    </>
  );
}
