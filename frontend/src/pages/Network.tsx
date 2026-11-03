import { useState } from "react";
import { Cable, Wifi, Lock, Check, RefreshCw, Settings2 } from "lucide-react";
import { api } from "../api/client";
import type { NetInterface, WiFi } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useUI, Modal } from "../components/overlay";
import { Badge, Button, Callout, Card, Empty, ErrorBox, Field, Input, Loading, PageHead, PasswordInput, Spinner } from "../components/ui";

export function Network() {
  const net = usePoll(() => api.get<{ managed: boolean; interfaces: NetInterface[] }>("/api/network"), 8000);
  const [ipEdit, setIpEdit] = useState<NetInterface | null>(null);
  if (net.error && !net.data) return <><PageHead title="Réseau" /><ErrorBox error={net.error} onRetry={net.reload} /></>;
  const hasWifi = net.data?.interfaces.some((i) => i.type === "wifi");
  return (
    <>
      <PageHead title="Réseau" sub="Comment ton Pi est connecté à ta box et au reste de la maison." />
      {!net.data ? <Loading /> : (
        <div className="stack">
          {!net.data.managed && <Callout mood="warn" title="Gestion avancée indisponible">NetworkManager n'est pas actif sur ce système : tu peux voir les connexions mais pas les modifier depuis ici.</Callout>}
          <div className="grid grid-2">
            {net.data.interfaces.map((i) => {
              const up = i.state.startsWith("connected") || i.state === "connecté";
              return (
                <Card key={i.name} title={i.type === "wifi" ? "Wi-Fi" : "Câble Ethernet"} icon={i.type === "wifi" ? <Wifi /> : <Cable />}
                  right={<Badge mood={up ? "good" : undefined}>{up ? "Connecté" : "Déconnecté"}</Badge>}>
                  {up ? (
                    <div className="grid grid-2" style={{ gap: 12 }}>
                      <div><div className="tiny muted">Adresse IP</div><b className="mono">{i.ips[0]?.split("/")[0] ?? "—"}</b></div>
                      <div><div className="tiny muted">Mode</div><b>{i.method === "manual" ? "IP fixe" : "Automatique (DHCP)"}</b></div>
                      {i.connection && <div><div className="tiny muted">Réseau</div><b>{i.connection}</b></div>}
                      {i.speed > 0 && <div><div className="tiny muted">Vitesse</div><b>{i.speed >= 1000 ? `${i.speed / 1000} Gb/s` : `${i.speed} Mb/s`}</b></div>}
                      <div><div className="tiny muted">Passerelle</div><b className="mono">{i.gateway || "—"}</b></div>
                      <div><div className="tiny muted">Adresse MAC</div><b className="mono small">{i.mac}</b></div>
                    </div>
                  ) : <p className="muted small">{i.type === "wifi" ? "Choisis un réseau ci-dessous pour te connecter." : "Branche un câble Ethernet pour une connexion plus stable."}</p>}
                  {up && net.data!.managed && i.connection && <div style={{ marginTop: 16 }}><Button size="sm" icon={<Settings2 />} onClick={() => setIpEdit(i)}>Configurer l'adresse IP</Button></div>}
                </Card>
              );
            })}
          </div>
          {hasWifi && net.data.managed && <WifiCard onChanged={net.reload} />}
        </div>
      )}
      {ipEdit && <IPDialog iface={ipEdit} onClose={() => setIpEdit(null)} />}
    </>
  );
}

function WifiCard({ onChanged }: { onChanged: () => void }) {
  const { toast, confirm } = useUI();
  const scan = usePoll(() => api.get<WiFi[]>("/api/network/wifi"));
  const [sel, setSel] = useState<WiFi | null>(null);
  const [pw, setPw] = useState("");
  const [busy, setBusy] = useState(false);
  const connect = async () => {
    if (!sel) return;
    setBusy(true);
    try { await api.post("/api/network/wifi/connect", { ssid: sel.ssid, password: pw }); toast(`Connecté à ${sel.ssid} 📶`, "good"); setSel(null); scan.reload(); onChanged(); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <Card title="Réseaux Wi-Fi à proximité" icon={<Wifi />} right={<Button size="sm" variant="ghost" icon={scan.loading ? <Spinner small /> : <RefreshCw />} onClick={scan.reload}>Actualiser</Button>}>
      {scan.error ? <Callout mood="warn">{scan.error}</Callout> : !scan.data ? <Loading label="Je scanne les ondes…" /> : scan.data.length === 0 ? <Empty emoji="📡" title="Aucun réseau trouvé" /> : (
        scan.data.map((w) => (
          <div key={w.ssid} className="toggle-row">
            <Wifi style={{ opacity: 0.35 + w.signal / 140, flex: "none" }} />
            <div className="txt"><b>{w.ssid}</b><span>Signal {w.signal >= 70 ? "excellent" : w.signal >= 45 ? "correct" : "faible"} · {w.security && w.security !== "--" ? w.security : "ouvert"}</span></div>
            {w.security && w.security !== "--" && <Lock size={15} className="muted" />}
            {w.active ? <Badge mood="good" icon={<Check />}>Connecté</Badge> : <Button size="sm" onClick={() => { setSel(w); setPw(""); }}>Se connecter</Button>}
            {w.known && !w.active && <Button size="sm" variant="ghost" onClick={async () => { if (await confirm({ title: `Oublier « ${w.ssid} » ?`, confirm: "Oublier", body: "Le mot de passe enregistré sera supprimé." })) { await api.post("/api/network/wifi/forget", { ssid: w.ssid }).catch((e) => toast(e.message, "bad")); scan.reload(); } }}>Oublier</Button>}
          </div>
        ))
      )}
      {sel && (
        <Modal title={`Se connecter à « ${sel.ssid} »`} onClose={() => setSel(null)} footer={<><Button variant="ghost" onClick={() => setSel(null)}>Annuler</Button><Button variant="primary" loading={busy} onClick={connect}>Se connecter</Button></>}>
          {sel.security && sel.security !== "--" ? <Field label="Mot de passe du Wi-Fi"><PasswordInput value={pw} onChange={(e) => setPw(e.target.value)} autoFocus /></Field> : <p>Ce réseau est ouvert (sans mot de passe).</p>}
          <Callout>Si tu es connecté à l'interface via ce même Wi-Fi, la page peut se couper quelques secondes.</Callout>
        </Modal>
      )}
    </Card>
  );
}

function IPDialog({ iface, onClose }: { iface: NetInterface; onClose: () => void }) {
  const { toast } = useUI();
  const [method, setMethod] = useState(iface.method === "manual" ? "manual" : "auto");
  const [address, setAddress] = useState(iface.ips[0] ?? "");
  const [gateway, setGateway] = useState(iface.gateway);
  const [dns, setDns] = useState(iface.dns.join(", ") || "1.1.1.1, 9.9.9.9");
  const [busy, setBusy] = useState(false);
  const save = async () => {
    setBusy(true);
    try {
      await api.post("/api/network/ip", { connection: iface.connection, method, address, gateway, dns: dns.split(/[,\s]+/).filter(Boolean) });
      toast(method === "manual" ? `Nouvelle adresse : ${address.split("/")[0]}. Recharge la page à cette adresse si besoin.` : "Retour en mode automatique", "good");
      onClose();
    } catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <Modal title="Adresse IP" sub={iface.connection} onClose={onClose} footer={<><Button variant="ghost" onClick={onClose}>Annuler</Button><Button variant="primary" loading={busy} onClick={save}>Appliquer</Button></>}>
      <div className="grid grid-2" style={{ gap: 10, marginBottom: 16 }}>
        {([["auto", "Automatique", "Ta box choisit l'adresse (recommandé)."], ["manual", "IP fixe", "L'adresse ne change jamais. Pratique pour un serveur."]] as const).map(([id, t, d]) => (
          <button type="button" key={id} className="card flat" onClick={() => setMethod(id)} style={{ textAlign: "left", cursor: "pointer", padding: 14, borderColor: method === id ? "var(--brand)" : undefined }}>
            <b>{t}</b><div className="small muted">{d}</div>
          </button>
        ))}
      </div>
      {method === "manual" && (
        <>
          <Field label="Adresse et masque" hint="Exemple : 192.168.1.50/24"><Input value={address} onChange={(e) => setAddress(e.target.value)} className="input mono" /></Field>
          <Field label="Passerelle (ta box)"><Input value={gateway} onChange={(e) => setGateway(e.target.value)} placeholder="192.168.1.1" /></Field>
          <Field label="Serveurs DNS"><Input value={dns} onChange={(e) => setDns(e.target.value)} /></Field>
          <Callout mood="warn">Une adresse incorrecte peut rendre ton Pi injoignable. Choisis une adresse libre sur ton réseau.</Callout>
        </>
      )}
    </Modal>
  );
}
