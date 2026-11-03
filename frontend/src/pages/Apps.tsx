import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { MoreVertical, ExternalLink, Play, Square, RotateCw, Download, ScrollText, Code2, Trash2, Plus, Store, Info, Cpu, MemoryStick, Network, FolderTree } from "lucide-react";
import { api, q } from "../api/client";
import type { AppsResponse, Container, ContainerDetails, ContainerStats, InstalledApp } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useJobs, useSession } from "../session";
import { useMenu, useUI, Modal } from "../components/overlay";
import { AppIcon } from "../components/AppIcon";
import { Badge, Button, Card, Dot, Empty, ErrorBox, Field, IconButton, Input, Loading, PageHead, Spinner, Tabs, Callout } from "../components/ui";
import { actionLabel, explainError } from "../lib/human";
import { bytes, slug } from "../lib/format";

const host = window.location.hostname;

export function appURL(a: { port: number; scheme?: string; path?: string }) {
  return a.port ? `${a.scheme || "http"}://${host}:${a.port}${a.path || ""}` : null;
}

const stateMood = { running: "good", partial: "warn", stopped: "bad", unknown: undefined } as const;
const stateLabel = { running: "En marche", partial: "Partiellement démarrée", stopped: "Arrêtée", unknown: "État inconnu" };

/** Grille des apps installées, partagée avec l'accueil. */
export function AppGrid({ data, reload }: { data: AppsResponse; reload: () => void }) {
  const { can } = useSession();
  const { running, kick, onDone, jobs } = useJobs();
  const { toast, confirm } = useUI();
  const menu = useMenu();
  const failed = (id: string) => jobs.find((j) => j.kind === "app" && j.target === id)?.status === "error";
  const [detail, setDetail] = useState<InstalledApp | null>(null);
  useEffect(() => onDone((j) => j.kind === "app" && reload()), [onDone, reload]);

  const act = async (a: InstalledApp, action: string) => {
    try {
      await api.post(`/api/apps/${a.id}/${action}`);
      toast(`${actionLabel[action]} de ${a.name} en cours…`);
      kick();
    } catch (e) { toast((e as Error).message, "bad"); }
  };

  const openMenu = (el: HTMLElement, a: InstalledApp) => {
    const url = appURL(a);
    menu.open(el, [
      ...(url ? [{ label: "Ouvrir", icon: <ExternalLink />, onClick: () => window.open(url, "_blank", "noopener") }] : []),
      { label: "Détails et ressources", icon: <Info />, onClick: () => setDetail(a) },
      ...(can("apps.manage") ? [
        a.state === "running" ? { label: "Arrêter", icon: <Square />, onClick: () => act(a, "stop") } : { label: "Démarrer", icon: <Play />, onClick: () => act(a, "start") },
        { label: "Redémarrer", icon: <RotateCw />, onClick: () => act(a, "restart") },
        { label: "Mettre à jour", icon: <Download />, onClick: () => act(a, "update") },
        "-" as const,
        { label: "Désinstaller", icon: <Trash2 />, danger: true, onClick: async () => {
          if (await confirm({ title: `Désinstaller ${a.name} ?`, danger: true, confirm: "Désinstaller", body: <>L'application sera retirée, mais tes données restent en sécurité dans <code>/DATA/AppData/{a.id}</code>. Tu pourras la réinstaller sans rien perdre.</> })) act(a, "uninstall");
        } },
      ] : []),
    ]);
  };

  return (
    <>
      <div className="app-grid">
        {data.apps.map((a, i) => {
          const job = running("app", a.id);
          const url = appURL(a);
          return (
            <div key={a.id} className="app-tile" style={{ animationDelay: `${i * 30}ms` }} title={stateLabel[a.state]}
              onClick={(e) => {
                if ((e.target as HTMLElement).closest(".icon-btn")) return;
                if (url && a.state === "running" && !failed(a.id)) window.open(url, "_blank", "noopener");
                else setDetail(a);
              }}>
              <Dot mood={stateMood[a.state]} />
              <IconButton label="Actions" onClick={(e) => openMenu(e.currentTarget, a)}><MoreVertical /></IconButton>
              <AppIcon name={a.name} icon={a.icon} size={64} />
              <div className="name">{a.name}</div>
              <div className="sub" style={failed(a.id) && !job ? { color: "var(--bad)" } : undefined}>
                {failed(a.id) && !job ? "Problème, clique ici" : a.state === "running" ? (url ? `Port ${a.port}` : "En marche") : stateLabel[a.state]}
              </div>
              {job && <div className="busy"><Spinner />{actionLabel[job.action]}…</div>}
            </div>
          );
        })}
      </div>
      {menu.node}
      {detail && <AppDetails app={detail} onClose={() => setDetail(null)} reload={reload} />}
    </>
  );
}

function AppDetails({ app, onClose, reload }: { app: InstalledApp; onClose: () => void; reload: () => void }) {
  const { can } = useSession();
  const { jobs, kick, running } = useJobs();
  const { toast } = useUI();
  const [showLog, setShowLog] = useState(false);
  const lastJob = jobs.find((j) => j.kind === "app" && j.target === app.id);
  const busy = running("app", app.id);
  const retry = async () => {
    try { await api.post(`/api/apps/${app.id}/update`); toast(`Nouvelle tentative pour ${app.name}…`); kick(); } catch (e) { toast((e as Error).message, "bad"); }
  };
  const [tab, setTab] = useState<"overview" | "logs" | "config">("overview");
  const ids = app.containers.map((c) => c.id);
  const stats = usePoll(async () => {
    const all = await api.get<ContainerStats[]>("/api/docker/stats");
    return all.filter((s) => ids.includes(s.id));
  }, tab === "overview" ? 4000 : 0, [tab]);
  const url = appURL(app);
  const cpu = stats.data?.reduce((n, s) => n + s.cpu, 0) ?? 0;
  const mem = stats.data?.reduce((n, s) => n + s.memUsage, 0) ?? 0;

  return (
    <Modal wide title={app.name} sub={<><Dot mood={stateMood[app.state]} /> &nbsp;{stateLabel[app.state]}{app.tagline ? ` · ${app.tagline}` : ""}</>} icon={<AppIcon name={app.name} icon={app.icon} size={52} />} onClose={onClose}
      footer={url && app.state === "running" ? <a className="btn primary" href={url} target="_blank" rel="noopener"><ExternalLink /> Ouvrir {app.name}</a> : undefined}>
      <Tabs value={tab} onChange={setTab} tabs={[
        { id: "overview", label: "Aperçu", icon: <Info /> },
        ...(can("apps.manage") ? [{ id: "logs" as const, label: "Journaux", icon: <ScrollText /> }, { id: "config" as const, label: "Configuration", icon: <Code2 /> }] : []),
      ]} />
      {tab === "overview" && (
        <div className="stack">
          <div className="grid grid-3">
            <Card flat><div className="metric-head"><div className="metric-icon"><Cpu /></div>Processeur</div><div className="metric-value">{cpu.toFixed(1).replace(".", ",")}<small>%</small></div></Card>
            <Card flat><div className="metric-head"><div className="metric-icon"><MemoryStick /></div>Mémoire</div><div className="metric-value" style={{ fontSize: 26 }}>{bytes(mem)}</div></Card>
            <Card flat><div className="metric-head"><div className="metric-icon"><Network /></div>Conteneurs</div><div className="metric-value">{app.containers.length}</div></Card>
          </div>
          {busy && <Callout title={`${actionLabel[busy.action]} en cours…`}>Le téléchargement peut prendre quelques minutes selon ta connexion.</Callout>}
          {!busy && lastJob?.status === "error" && (
            <Callout mood="bad" title={`${actionLabel[lastJob.action]} : ça n'a pas marché`}>
              {explainError(lastJob.log)}
              <div className="row" style={{ marginTop: 10 }}>
                {can("apps.manage") && <Button size="sm" variant="primary" icon={<RotateCw />} onClick={retry}>Réessayer</Button>}
                <Button size="sm" variant="ghost" onClick={() => setShowLog(!showLog)}>{showLog ? "Masquer" : "Voir"} le détail technique</Button>
              </div>
              {showLog && <pre className="logs" style={{ marginTop: 10, maxHeight: 260 }}>{lastJob.log}</pre>}
            </Callout>
          )}
          {app.notes && <Callout>{app.notes}</Callout>}
          {app.containers.map((c) => <ContainerInfo key={c.id} c={c} stats={stats.data?.find((s) => s.id === c.id)} />)}
          {app.containers.length === 0 && !busy && lastJob?.status !== "error" && <Callout mood="warn">Cette application n'a aucun conteneur actif. Essaie de la démarrer depuis le menu ⋮.</Callout>}
        </div>
      )}
      {tab === "logs" && <Logs containers={app.containers} />}
      {tab === "config" && <ComposeEditor app={app} onSaved={() => { reload(); onClose(); }} />}
    </Modal>
  );
}

function ContainerInfo({ c, stats }: { c: Container; stats?: ContainerStats }) {
  const [d, setD] = useState<ContainerDetails | null>(null);
  useEffect(() => { api.get<ContainerDetails>(`/api/docker/containers/${q(c.id)}`).then(setD).catch(() => undefined); }, [c.id]);
  return (
    <Card flat>
      <div className="row" style={{ marginBottom: 12 }}>
        <Dot mood={c.state === "running" ? "good" : "bad"} />
        <b>{c.name}</b>
        <span className="muted small mono">{c.image}</span>
        <div className="spacer" />
        {stats && <Badge>{stats.cpu.toFixed(1)}% · {bytes(stats.memUsage)}</Badge>}
      </div>
      {!d ? <Spinner small /> : (
        <div className="grid grid-2" style={{ gap: 14 }}>
          <div>
            <div className="small bold row" style={{ marginBottom: 6, gap: 6 }}><Network size={15} /> Ports</div>
            {d.ports.length ? d.ports.map((p) => <div key={p} className="small mono">{p}</div>) : <div className="small muted">{d.network === "host" ? "Réseau de l'hôte (tous les ports)" : "Aucun port exposé"}</div>}
          </div>
          <div>
            <div className="small bold row" style={{ marginBottom: 6, gap: 6 }}><FolderTree size={15} /> Volumes et dossiers</div>
            {d.mounts.length ? d.mounts.map((m) => (
              <div key={m.destination} className="small mono" style={{ wordBreak: "break-all" }}>{m.name || m.source} → {m.destination}{!m.rw && " (lecture seule)"}</div>
            )) : <div className="small muted">Aucun</div>}
          </div>
          <div className="small muted">Redémarrage : <b>{d.restart || "non"}</b> · Redémarré {d.restarts} fois</div>
        </div>
      )}
    </Card>
  );
}

export function Logs({ containers }: { containers: { id: string; name: string }[] }) {
  const [id, setId] = useState(containers[0]?.id ?? "");
  const logs = usePoll(() => (id ? api.get<{ logs: string }>(`/api/docker/containers/${q(id)}/logs?tail=500`) : Promise.resolve({ logs: "" })), 5000, [id]);
  if (!containers.length) return <Callout>Aucun conteneur.</Callout>;
  return (
    <>
      {containers.length > 1 && (
        <div className="chips">{containers.map((c) => <button key={c.id} className={`chip ${c.id === id ? "active" : ""}`} onClick={() => setId(c.id)}>{c.name}</button>)}</div>
      )}
      <pre className="logs" ref={(el) => { if (el) el.scrollTop = el.scrollHeight; }}>{logs.error ?? (logs.data?.logs || "Pas encore de journaux…")}</pre>
    </>
  );
}

function ComposeEditor({ app, onSaved }: { app: InstalledApp; onSaved: () => void }) {
  const { toast } = useUI();
  const { kick } = useJobs();
  const [text, setText] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => { api.get<{ compose: string }>(`/api/apps/${app.id}/compose`).then((r) => setText(r.compose)).catch((e) => toast(e.message, "bad")); }, [app.id, toast]);
  if (text === null) return <Loading />;
  const save = async () => {
    setBusy(true);
    try {
      await api.post(`/api/apps/${app.id}/install`, { compose: text, name: app.name, icon: app.icon, port: app.port, path: app.path });
      toast(`${app.name} est en cours de redéploiement…`);
      kick();
      onSaved();
    } catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <>
      <Callout>Ici tu peux changer les ports, les dossiers ou les variables de l'application. L'application redémarre quand tu enregistres.</Callout>
      <textarea className="textarea" style={{ marginTop: 14, minHeight: 340 }} value={text} onChange={(e) => setText(e.target.value)} spellCheck={false} />
      <div className="row" style={{ justifyContent: "flex-end", marginTop: 14 }}><Button variant="primary" loading={busy} onClick={save}>Enregistrer et redéployer</Button></div>
    </>
  );
}

export function Apps() {
  const { can } = useSession();
  const apps = usePoll(() => api.get<AppsResponse>("/api/apps"), 6000);
  const [custom, setCustom] = useState(false);
  return (
    <>
      <PageHead title="Mes applications" sub="Toutes les applications installées sur ton Pi. Clique sur une carte pour l'ouvrir."
        actions={can("apps.manage") && <>
          <Button icon={<Plus />} onClick={() => setCustom(true)}>App personnalisée</Button>
          <Link to="/store" className="btn primary"><Store /> App Store</Link>
        </>} />
      {apps.error && !apps.data ? <ErrorBox error={apps.error} onRetry={apps.reload} /> : !apps.data ? <Loading /> : (
        <>
          {!apps.data.docker && <div style={{ marginBottom: 18 }}><Callout mood="warn" title="Docker ne répond pas">Les applications ne peuvent pas être gérées tant que Docker est arrêté. Essaie de redémarrer ton Pi.</Callout></div>}
          {apps.data.apps.length ? <AppGrid data={apps.data} reload={apps.reload} /> : (
            <Card><Empty emoji="🧺" title="Rien d'installé pour le moment" action={can("apps.manage") && <Link to="/store" className="btn primary"><Store /> Découvrir l'App Store</Link>}>Choisis parmi plus de 25 applications prêtes à l'emploi, installées en un clic.</Empty></Card>
          )}
          {apps.data.others.length > 0 && can("apps.manage") && (
            <>
              <div className="section-title">Autres conteneurs <span className="count">{apps.data.others.length}</span></div>
              <p className="muted small" style={{ marginTop: -8, marginBottom: 14 }}>Ces conteneurs n'ont pas été installés via Azmo OS. Tu peux les gérer depuis la page <Link to="/containers">Conteneurs</Link>.</p>
              <div className="app-grid">
                {apps.data.others.map((c) => {
                  const p = c.ports?.[0]?.publicPort;
                  return (
                    <div key={c.id} className="app-tile" onClick={() => p && window.open(`http://${host}:${p}`, "_blank", "noopener")}>
                      <Dot mood={c.state === "running" ? "good" : "bad"} />
                      <AppIcon name={c.name} size={56} />
                      <div className="name">{c.name}</div>
                      <div className="sub">{p ? `Port ${p}` : c.state}</div>
                    </div>
                  );
                })}
              </div>
            </>
          )}
        </>
      )}
      {custom && <CustomApp onClose={() => setCustom(false)} onDone={apps.reload} />}
    </>
  );
}

const sampleCompose = `services:
  monapp:
    image: nginx:alpine
    container_name: monapp
    restart: unless-stopped
    ports:
      - "8080:80"
    volumes:
      - /DATA/AppData/monapp:/usr/share/nginx/html
`;

function CustomApp({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const { toast } = useUI();
  const { kick } = useJobs();
  const [name, setName] = useState("");
  const [port, setPort] = useState("8080");
  const [icon, setIcon] = useState("");
  const [compose, setCompose] = useState(sampleCompose);
  const [busy, setBusy] = useState(false);
  const submit = async () => {
    const id = slug(name) || "app";
    setBusy(true);
    try {
      await api.post(`/api/apps/${id}/install`, { compose, name, icon, port: Number(port) || 0 });
      toast(`Installation de ${name} lancée, je télécharge les images…`);
      kick();
      onDone();
      onClose();
    } catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <Modal wide title="Ajouter une application personnalisée" sub="Colle un fichier docker-compose.yml, Azmo OS s'occupe du reste." onClose={onClose}
      footer={<><Button variant="ghost" onClick={onClose}>Annuler</Button><Button variant="primary" loading={busy} disabled={!name.trim()} onClick={submit}>Installer</Button></>}>
      <div className="grid grid-3" style={{ gap: 14 }}>
        <Field label="Nom"><Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Mon application" autoFocus /></Field>
        <Field label="Port de l'interface web"><Input value={port} onChange={(e) => setPort(e.target.value.replace(/\D/g, ""))} inputMode="numeric" /></Field>
        <Field label="Icône (URL, optionnel)"><Input value={icon} onChange={(e) => setIcon(e.target.value)} placeholder="https://…/icone.png" /></Field>
      </div>
      <Field label="docker-compose.yml"><textarea className="textarea" value={compose} onChange={(e) => setCompose(e.target.value)} spellCheck={false} /></Field>
    </Modal>
  );
}
