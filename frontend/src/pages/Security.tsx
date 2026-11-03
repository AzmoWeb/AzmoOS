import { useState } from "react";
import { ShieldCheck, Shield, Terminal, Globe2, Plus, Trash2, Monitor } from "lucide-react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { Firewall, ServiceState, Session } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useUI, Modal } from "../components/overlay";
import { Badge, Button, Callout, Card, Field, Input, Loading, PageHead, Switch, ToggleRow } from "../components/ui";
import { ago } from "../lib/format";

export function Security() {
  const { toast, confirm } = useUI();
  const fw = usePoll(() => api.get<Firewall>("/api/firewall"));
  const settings = usePoll(() => api.get<{ ssh: ServiceState }>("/api/system/settings"));
  const me = usePoll(() => api.get<{ sessions: Session[] }>("/api/me"));
  const [rule, setRule] = useState(false);

  const toggleFw = async (on: boolean) => {
    if (on && !(await confirm({ title: "Activer le pare-feu ?", confirm: "Activer", body: "Seules l'interface Azmo OS et SSH resteront ouvertes, plus les ports que tu autoriseras. Les applications Docker gardent leurs propres ports." }))) return;
    try { await api.post(`/api/firewall/${on ? "enable" : "disable"}`); toast(on ? "Pare-feu activé 🛡️" : "Pare-feu désactivé", on ? "good" : "info"); fw.reload(); }
    catch (e) { toast((e as Error).message, "bad"); }
  };
  const toggleSSH = async (on: boolean) => {
    try { await api.post("/api/system/ssh", { enabled: on }); toast(on ? "SSH activé" : "SSH désactivé", "good"); settings.reload(); }
    catch (e) { toast((e as Error).message, "bad"); }
  };

  // Le pare-feu est indispensable pour être « bien protégé » ; le reste affine le conseil.
  const protectedOk = !!fw.data?.active;
  const advice = !fw.data ? "" : !fw.data.installed ? "Installe le pare-feu (ufw) pour bloquer les connexions indésirables."
    : !fw.data.active ? "Active le pare-feu ci-dessous : il bloque les connexions indésirables."
    : settings.data?.ssh.active ? "Le pare-feu filtre les connexions. Pense à désactiver SSH si tu ne t'en sers pas."
    : "Le pare-feu filtre les connexions entrantes et SSH est fermé. 👌";

  return (
    <>
      <PageHead title="Sécurité" sub="Garde ton serveur à l'abri, sans te prendre la tête." />
      <div className="stack">
        {fw.data && (
          <div className={`health ${protectedOk ? "good" : "warn"}`}>
            <div className="health-icon">{protectedOk ? <ShieldCheck /> : <Shield />}</div>
            <div>
              <h2>{protectedOk ? "Ton serveur est bien protégé" : "Un réglage renforcerait ta sécurité"}</h2>
              <p>{advice}</p>
            </div>
          </div>
        )}

        <div className="grid grid-2">
          <Card title="Pare-feu" icon={<Shield />} right={fw.data?.installed && <Switch checked={!!fw.data.active} onChange={toggleFw} label="Pare-feu" />}>
            {!fw.data ? <Loading /> : !fw.data.installed ? <Callout>Le pare-feu ufw n'est pas installé. Lance <code>sudo apt install ufw</code> sur ton Pi.</Callout> : (
              <>
                <p className="muted small" style={{ marginBottom: 12 }}>{fw.data.active ? `${fw.data.rules.filter((r) => !r.v6).length} règles actives.` : "Désactivé : toutes les connexions sont acceptées."}</p>
                {fw.data.active && fw.data.rules.filter((r) => !r.v6).map((r) => (
                  <div key={r.number} className="toggle-row" style={{ padding: "10px 0" }}>
                    <div className="txt"><b className="mono" style={{ fontSize: 13.5 }}>{r.to}</b><span>depuis {r.from === "Anywhere" ? "partout" : r.from}</span></div>
                    <Badge mood={r.action === "ALLOW" ? "good" : "bad"}>{r.action === "ALLOW" ? "Autorisé" : "Bloqué"}</Badge>
                    <button className="icon-btn" aria-label="Supprimer" onClick={async () => { try { await api.del(`/api/firewall/rules/${r.number}`); fw.reload(); } catch (e) { toast((e as Error).message, "bad"); } }}><Trash2 /></button>
                  </div>
                ))}
                {fw.data.active && <Button size="sm" icon={<Plus />} style={{ marginTop: 12 }} onClick={() => setRule(true)}>Ajouter une règle</Button>}
              </>
            )}
          </Card>

          <Card title="Accès à distance" icon={<Globe2 />}>
            {settings.data && (
              <ToggleRow title={<><Terminal size={15} style={{ verticalAlign: -2 }} /> SSH (ligne de commande)</>} desc={settings.data.ssh.installed ? "Pour administrer ton Pi depuis un terminal. Désactive-le si tu ne t'en sers pas." : "Serveur SSH non installé."}
                checked={settings.data.ssh.active} onChange={toggleSSH} disabled={!settings.data.ssh.installed} />
            )}
            <div style={{ marginTop: 14 }}>
              <b style={{ fontSize: 14.5 }}>Accéder à ton Pi depuis l'extérieur</b>
              <p className="small muted" style={{ margin: "4px 0 12px" }}>N'ouvre jamais directement les ports de ta box. Utilise plutôt un VPN, c'est plus sûr :</p>
              <div className="row"><Link to="/store" className="btn sm">🔗 Tailscale (le plus simple)</Link><Link to="/store" className="btn sm">🛡️ WireGuard</Link></div>
            </div>
          </Card>
        </div>

        <Card title="Mes sessions ouvertes" icon={<Monitor />} right={(me.data?.sessions.length ?? 0) > 1 && (
          <Button size="sm" onClick={async () => { await api.post("/api/me/sessions/revoke"); toast("Les autres appareils ont été déconnectés", "good"); me.reload(); }}>Déconnecter les autres appareils</Button>
        )}>
          {!me.data ? <Loading /> : me.data.sessions.map((s) => (
            <div key={s.id} className="toggle-row">
              <div className="txt"><b>{describeAgent(s.agent)}</b><span>{s.ip} · expire {ago(s.expires)}</span></div>
              {s.current && <Badge mood="brand">Cet appareil</Badge>}
            </div>
          ))}
        </Card>
      </div>
      {rule && <RuleDialog onClose={() => setRule(false)} onSaved={fw.reload} />}
    </>
  );
}

function describeAgent(ua: string): string {
  const os = /iPhone|iPad/.test(ua) ? "iPhone / iPad" : /Android/.test(ua) ? "Android" : /Windows/.test(ua) ? "Windows" : /Mac OS/.test(ua) ? "Mac" : /Linux/.test(ua) ? "Linux" : "";
  const br = /Edg\//.test(ua) ? "Edge" : /Firefox\//.test(ua) ? "Firefox" : /Chrome\//.test(ua) ? "Chrome" : /Safari\//.test(ua) ? "Safari" : "";
  if (!os && !br) return "Appareil ou programme inconnu";
  return `${br || "Navigateur"} sur ${os || "un appareil"}`;
}

function RuleDialog({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const { toast } = useUI();
  const [r, setR] = useState({ action: "allow", port: "", proto: "tcp", from: "" });
  const save = async () => {
    try { await api.post("/api/firewall/rules", r); toast("Règle ajoutée", "good"); onSaved(); onClose(); } catch (e) { toast((e as Error).message, "bad"); }
  };
  return (
    <Modal title="Nouvelle règle de pare-feu" onClose={onClose} footer={<><Button variant="ghost" onClick={onClose}>Annuler</Button><Button variant="primary" disabled={!r.port} onClick={save}>Ajouter</Button></>}>
      <div className="grid grid-2" style={{ gap: 12 }}>
        <Field label="Action"><select className="select" value={r.action} onChange={(e) => setR({ ...r, action: e.target.value })}><option value="allow">Autoriser</option><option value="deny">Bloquer</option></select></Field>
        <Field label="Protocole"><select className="select" value={r.proto} onChange={(e) => setR({ ...r, proto: e.target.value })}><option value="tcp">TCP</option><option value="udp">UDP</option><option value="any">Les deux</option></select></Field>
      </div>
      <Field label="Port" hint="Un port (8096) ou une plage (6000:6010)."><Input value={r.port} onChange={(e) => setR({ ...r, port: e.target.value.replace(/[^\d:]/g, "") })} placeholder="8096" autoFocus /></Field>
      <Field label="Depuis (optionnel)" hint="Laisse vide pour tout le monde, ou limite à ton réseau : 192.168.1.0/24"><Input value={r.from} onChange={(e) => setR({ ...r, from: e.target.value })} /></Field>
    </Modal>
  );
}
