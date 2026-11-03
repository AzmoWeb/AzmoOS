import { RefreshCw, Package, Sparkles, Power } from "lucide-react";
import { api } from "../api/client";
import type { UpdatesResponse } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useJobs } from "../session";
import { useUI, Modal } from "../components/overlay";
import { Badge, Button, Callout, Card, Empty, ErrorBox, Loading, PageHead, Spinner } from "../components/ui";
import { ago, plural } from "../lib/format";
import { useEffect, useState } from "react";

export function Updates() {
  const { toast, confirm } = useUI();
  const { running, kick, onDone, jobs } = useJobs();
  const data = usePoll(() => api.get<UpdatesResponse>("/api/updates"));
  const [log, setLog] = useState<string | null>(null);
  useEffect(() => onDone((j) => (j.kind === "system" || j.kind === "updater") && data.reload()), [onDone, data]);
  const apt = running("system", "apt");
  const self = running("updater", "azmoos");

  const check = async () => {
    try { await api.post("/api/updates/check"); kick(); toast("Je cherche les nouveautés…"); setTimeout(data.reload, 1500); } catch (e) { toast((e as Error).message, "bad"); }
  };
  const upgradeSystem = async () => {
    if (!(await confirm({ title: "Mettre à jour le système ?", confirm: "Mettre à jour", body: "Les paquets Debian / Raspberry Pi OS seront mis à jour. Ça peut prendre plusieurs minutes ; tes applications continuent de tourner." }))) return;
    try { await api.post("/api/updates/system"); kick(); } catch (e) { toast((e as Error).message, "bad"); }
  };
  const upgradeSelf = async () => {
    try { await api.post("/api/updates/azmoos"); kick(); toast("Mise à jour d'Azmo OS en cours. L'interface va se recharger toute seule."); setTimeout(() => window.location.reload(), 25000); }
    catch (e) { toast((e as Error).message, "bad"); }
  };

  if (data.error && !data.data) return <ErrorBox error={data.error} onRetry={data.reload} />;
  const d = data.data;
  const lastApt = jobs.find((j) => j.kind === "system" && j.target === "apt");
  return (
    <>
      <PageHead title="Mises à jour" sub="Garde ton Pi à jour pour profiter des nouveautés et des correctifs de sécurité." actions={<Button icon={<RefreshCw />} onClick={check}>Rechercher</Button>} />
      {!d ? <Loading /> : (
        <div className="stack">
          {d.rebootRequired && (
            <Callout mood="warn" title="Un redémarrage est nécessaire">
              Certaines mises à jour (noyau, firmware) ne s'appliquent qu'après un redémarrage.
              <div style={{ marginTop: 10 }}><Button size="sm" icon={<Power />} onClick={async () => { if (await confirm({ title: "Redémarrer maintenant ?", confirm: "Redémarrer", body: "Ton Pi sera indisponible environ une minute." })) api.post("/api/system/power", { action: "reboot" }); }}>Redémarrer maintenant</Button></div>
            </Callout>
          )}
          <div className="grid grid-2">
            <Card title="Azmo OS" icon={<Sparkles />}>
              <div className="row" style={{ marginBottom: 12 }}>
                <span>Version installée</span><Badge mood="brand">{d.azmoos.current}</Badge>
              </div>
              {self ? <div className="row"><Spinner /> Installation de la nouvelle version…</div>
                : d.azmoos.available && d.azmoos.latest ? (
                  <>
                    <Callout mood="good" title={`Nouvelle version : ${d.azmoos.latest.version} 🎁`}>
                      <div className="small" style={{ whiteSpace: "pre-line", maxHeight: 160, overflow: "auto" }}>{d.azmoos.latest.notes || "Améliorations et corrections."}</div>
                    </Callout>
                    <Button variant="primary" style={{ marginTop: 14 }} onClick={upgradeSelf}>Installer {d.azmoos.latest.version}</Button>
                  </>
                ) : (
                  <p className="muted">{d.azmoos.error ? d.azmoos.error : "Tu as la dernière version. 👌"}{d.azmoos.checked ? ` Dernière vérification ${ago(d.azmoos.checked)}.` : ""}</p>
                )}
            </Card>
            <Card title="Système (Raspberry Pi OS)" icon={<Package />}>
              {apt ? <div className="row"><Spinner /> {apt.action === "upgrade" ? "Mise à jour en cours…" : "Recherche des mises à jour…"}<div className="spacer" /><Button size="sm" variant="ghost" onClick={() => setLog(apt.id)}>Voir le détail</Button></div>
                : d.packages.length ? (
                  <>
                    <p style={{ marginBottom: 14 }}><b>{plural(d.packages.length, "mise à jour disponible", "mises à jour disponibles")}</b></p>
                    <Button variant="primary" onClick={upgradeSystem}>Tout mettre à jour</Button>
                  </>
                ) : <p className="muted">Ton système est à jour. Clique sur « Rechercher » pour vérifier à nouveau.</p>}
              {!apt && lastApt && <Button size="sm" variant="ghost" style={{ marginTop: 10 }} onClick={() => setLog(lastApt.id)}>Journal de la dernière opération</Button>}
            </Card>
          </div>
          {d.packages.length > 0 && (
            <div className="card pad-0">
              <div className="table-wrap"><table className="table">
                <thead><tr><th>Paquet</th><th className="hide-sm">Version actuelle</th><th>Nouvelle version</th></tr></thead>
                <tbody>{d.packages.slice(0, 200).map((p) => <tr key={p.name}><td className="mono small">{p.name}</td><td className="hide-sm muted small mono">{p.current}</td><td className="small mono">{p.new}</td></tr>)}</tbody>
              </table></div>
            </div>
          )}
          {!d.packages.length && !apt && <Card flat><Empty emoji="✅" title="Rien à faire">Tout est à jour, profite !</Empty></Card>}
        </div>
      )}
      {log && <JobLog id={log} onClose={() => setLog(null)} />}
    </>
  );
}

function JobLog({ id, onClose }: { id: string; onClose: () => void }) {
  const { jobs } = useJobs();
  const j = jobs.find((x) => x.id === id);
  return (
    <Modal wide title="Détail de l'opération" onClose={onClose}>
      <pre className="logs" ref={(el) => { if (el) el.scrollTop = el.scrollHeight; }}>{j?.log || "…"}</pre>
    </Modal>
  );
}
