import { useState } from "react";
import { Boxes, Layers, Database, MoreVertical, Play, Square, RotateCw, ScrollText, Trash2, Sparkles } from "lucide-react";
import { api, q } from "../api/client";
import type { Container, ContainerStats, DockerImage, DockerVolume } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useMenu, useUI, Modal } from "../components/overlay";
import { Badge, Button, Card, Dot, Empty, ErrorBox, Loading, PageHead, Tabs } from "../components/ui";
import { ago, bytes } from "../lib/format";
import { Logs } from "./Apps";

const host = window.location.hostname;

export function Containers() {
  const [tab, setTab] = useState<"containers" | "images" | "volumes">("containers");
  return (
    <>
      <PageHead title="Conteneurs" sub="La salle des machines : chaque application tourne dans un ou plusieurs conteneurs Docker." />
      <Tabs value={tab} onChange={setTab} tabs={[
        { id: "containers", label: "Conteneurs", icon: <Boxes /> },
        { id: "images", label: "Images", icon: <Layers /> },
        { id: "volumes", label: "Volumes", icon: <Database /> },
      ]} />
      {tab === "containers" && <ContainerList />}
      {tab === "images" && <Images />}
      {tab === "volumes" && <Volumes />}
    </>
  );
}

function ContainerList() {
  const { toast, confirm } = useUI();
  const menu = useMenu();
  const list = usePoll(() => api.get<Container[]>("/api/docker/containers"), 5000);
  const stats = usePoll(() => api.get<ContainerStats[]>("/api/docker/stats"), 6000);
  const [logs, setLogs] = useState<Container | null>(null);

  const act = async (c: Container, action: string) => {
    try { await api.post(`/api/docker/containers/${q(c.id)}/${action}`); toast(`${c.name} : c'est fait`, "good"); list.reload(); }
    catch (e) { toast((e as Error).message, "bad"); }
  };

  if (list.error && !list.data) return <ErrorBox error={list.error} onRetry={list.reload} />;
  if (!list.data) return <Loading />;
  if (!list.data.length) return <Card><Empty emoji="📦" title="Aucun conteneur">Installe une application depuis l'App Store pour voir ses conteneurs ici.</Empty></Card>;
  return (
    <div className="card pad-0">
      <div className="table-wrap">
        <table className="table">
          <thead><tr><th>Nom</th><th className="hide-sm">Image</th><th>État</th><th className="hide-sm">Ressources</th><th className="hide-sm">Ports</th><th /></tr></thead>
          <tbody>
            {list.data.map((c) => {
              const st = stats.data?.find((s) => s.id === c.id);
              return (
                <tr key={c.id}>
                  <td><div className="row" style={{ gap: 10, flexWrap: "nowrap" }}><Dot mood={c.state === "running" ? "good" : "bad"} /><b>{c.name}</b></div></td>
                  <td className="hide-sm muted small mono">{c.image}</td>
                  <td className="muted small">{c.status}</td>
                  <td className="hide-sm small">{st ? <>{st.cpu.toFixed(1)}% · {bytes(st.memUsage)}</> : <span className="muted">—</span>}</td>
                  <td className="hide-sm">{(c.ports ?? []).map((p) => <a key={p.publicPort} href={`http://${host}:${p.publicPort}`} target="_blank" rel="noopener" className="badge" style={{ marginRight: 4, textDecoration: "none" }}>{p.publicPort}</a>)}</td>
                  <td className="actions">
                    <button className="icon-btn" aria-label="Actions" onClick={(e) => menu.open(e.currentTarget, [
                      c.state === "running" ? { label: "Arrêter", icon: <Square />, onClick: () => act(c, "stop") } : { label: "Démarrer", icon: <Play />, onClick: () => act(c, "start") },
                      { label: "Redémarrer", icon: <RotateCw />, onClick: () => act(c, "restart") },
                      { label: "Journaux", icon: <ScrollText />, onClick: () => setLogs(c) },
                      "-",
                      { label: "Supprimer", icon: <Trash2 />, danger: true, onClick: async () => { if (await confirm({ title: `Supprimer ${c.name} ?`, danger: true, confirm: "Supprimer", body: "Le conteneur sera supprimé. Si une application Azmo OS l'utilise, redémarre-la pour le recréer." })) act(c, "remove"); } },
                    ])}><MoreVertical /></button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      {menu.node}
      {logs && <Modal wide title={`Journaux de ${logs.name}`} onClose={() => setLogs(null)}><Logs containers={[logs]} /></Modal>}
    </div>
  );
}

function Images() {
  const { toast, confirm } = useUI();
  const list = usePoll(() => api.get<DockerImage[]>("/api/docker/images"));
  const [busy, setBusy] = useState(false);
  if (list.error && !list.data) return <ErrorBox error={list.error} onRetry={list.reload} />;
  if (!list.data) return <Loading />;
  const total = list.data.reduce((n, i) => n + i.size, 0);
  const unused = list.data.filter((i) => !i.inUse);
  const prune = async () => {
    if (!(await confirm({ title: "Faire le ménage ?", confirm: "Nettoyer", body: `Je vais supprimer les ${unused.length} images qui ne sont utilisées par aucune application. Tes données ne sont pas touchées.` }))) return;
    setBusy(true);
    try { const r = await api.post<{ reclaimed: number }>("/api/docker/prune"); toast(`${bytes(r.reclaimed)} libérés 🧹`, "good"); list.reload(); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <div className="stack">
      <Card>
        <div className="row">
          <div><b>{list.data.length} images</b> occupent <b>{bytes(total)}</b>.<div className="muted small">{unused.length ? `${unused.length} ne servent plus à rien.` : "Toutes sont utilisées. 👌"}</div></div>
          <div className="spacer" />
          <Button icon={<Sparkles />} loading={busy} disabled={!unused.length} onClick={prune}>Faire le ménage</Button>
        </div>
      </Card>
      <div className="card pad-0"><div className="table-wrap"><table className="table">
        <thead><tr><th>Image</th><th>Taille</th><th className="hide-sm">Créée</th><th>Statut</th><th /></tr></thead>
        <tbody>{list.data.map((i) => (
          <tr key={i.id}>
            <td className="mono small">{i.tags[0] ?? i.id.slice(7, 19)}</td>
            <td className="small">{bytes(i.size)}</td>
            <td className="hide-sm muted small">{ago(i.created)}</td>
            <td>{i.inUse ? <Badge mood="good">Utilisée</Badge> : <Badge>Inutilisée</Badge>}</td>
            <td className="actions">{!i.inUse && <button className="icon-btn" aria-label="Supprimer" onClick={async () => { try { await api.del(`/api/docker/images/${q(i.id)}`); list.reload(); } catch (e) { toast((e as Error).message, "bad"); } }}><Trash2 /></button>}</td>
          </tr>
        ))}</tbody>
      </table></div></div>
    </div>
  );
}

function Volumes() {
  const { toast, confirm } = useUI();
  const list = usePoll(() => api.get<DockerVolume[]>("/api/docker/volumes"));
  if (list.error && !list.data) return <ErrorBox error={list.error} onRetry={list.reload} />;
  if (!list.data) return <Loading />;
  if (!list.data.length) return <Card><Empty emoji="🗄️" title="Aucun volume Docker">Les applications Azmo OS rangent leurs données dans <code>/DATA/AppData</code>, visibles dans Fichiers.</Empty></Card>;
  return (
    <div className="card pad-0"><div className="table-wrap"><table className="table">
      <thead><tr><th>Volume</th><th className="hide-sm">Application</th><th className="hide-sm">Emplacement</th><th>Statut</th><th /></tr></thead>
      <tbody>{list.data.map((v) => (
        <tr key={v.name}>
          <td className="mono small">{v.name.length > 28 ? v.name.slice(0, 28) + "…" : v.name}</td>
          <td className="hide-sm small">{v.project || "—"}</td>
          <td className="hide-sm muted small mono">{v.mountpoint}</td>
          <td>{v.inUse ? <Badge mood="good">Utilisé</Badge> : <Badge>Libre</Badge>}</td>
          <td className="actions">{!v.inUse && <button className="icon-btn" aria-label="Supprimer" onClick={async () => {
            if (await confirm({ title: "Supprimer ce volume ?", danger: true, confirm: "Supprimer", body: "Les données qu'il contient seront perdues définitivement." })) {
              try { await api.del(`/api/docker/volumes/${q(v.name)}`); list.reload(); } catch (e) { toast((e as Error).message, "bad"); }
            }
          }}><Trash2 /></button>}</td>
        </tr>
      ))}</tbody>
    </table></div></div>
  );
}
