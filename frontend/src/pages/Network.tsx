import { useState } from "react";
import { Cable, Wifi, Lock, Check, RefreshCw, Settings2 } from "lucide-react";
import { api } from "../api/client";
import type { NetInterface, WiFi } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useUI, Modal } from "../components/overlay";
import { Badge, Button, Callout, Card, Empty, ErrorBox, Field, Input, Loading, PageHead, PasswordInput, Spinner } from "../components/ui";
import { t } from "../lib/i18n";

export function Network() {
  const net = usePoll(() => api.get<{ managed: boolean; interfaces: NetInterface[] }>("/api/network"), 8000);
  const [ipEdit, setIpEdit] = useState<NetInterface | null>(null);
  if (net.error && !net.data) return <><PageHead title={t("Réseau")} /><ErrorBox error={net.error} onRetry={net.reload} /></>;
  const hasWifi = net.data?.interfaces.some((i) => i.type === "wifi");
  return (
    <>
      <PageHead title={t("Réseau")} sub={t("Comment ton Pi est connecté à ta box et au reste de la maison.")} />
      {!net.data ? <Loading /> : (
        <div className="stack">
          {!net.data.managed && <Callout mood="warn" title={t("Gestion avancée indisponible")}>{t("NetworkManager n'est pas actif sur ce système : tu peux voir les connexions mais pas les modifier depuis ici.")}</Callout>}
          <div className="grid grid-2">
            {net.data.interfaces.map((i) => {
              const up = i.state.startsWith("connected") || i.state === "connecté";
              return (
                <Card key={i.name} title={i.type === "wifi" ? t("Wi-Fi") : t("Câble Ethernet")} icon={i.type === "wifi" ? <Wifi /> : <Cable />}
                  right={<Badge mood={up ? "good" : undefined}>{up ? t("Connecté") : t("Déconnecté")}</Badge>}>
                  {up ? (
                    <div className="grid grid-2" style={{ gap: 12 }}>
                      <div><div className="tiny muted">{t("Adresse IP")}</div><b className="mono">{i.ips[0]?.split("/")[0] ?? "—"}</b></div>
                      <div><div className="tiny muted">{t("Mode")}</div><b>{i.method === "manual" ? t("IP fixe") : t("Automatique (DHCP)")}</b></div>
                      {i.connection && <div><div className="tiny muted">{t("Réseau")}</div><b>{i.connection}</b></div>}
                      {i.speed > 0 && <div><div className="tiny muted">{t("Vitesse")}</div><b>{i.speed >= 1000 ? t("{0} Gb/s", i.speed / 1000) : t("{0} Mb/s", i.speed)}</b></div>}
                      <div><div className="tiny muted">{t("Passerelle")}</div><b className="mono">{i.gateway || "—"}</b></div>
                      <div><div className="tiny muted">{t("Adresse MAC")}</div><b className="mono small">{i.mac}</b></div>
                    </div>
                  ) : <p className="muted small">{i.type === "wifi" ? t("Choisis un réseau ci-dessous pour te connecter.") : t("Branche un câble Ethernet pour une connexion plus stable.")}</p>}
                  {up && net.data!.managed && i.connection && <div style={{ marginTop: 16 }}><Button size="sm" icon={<Settings2 />} onClick={() => setIpEdit(i)}>{t("Configurer l'adresse IP")}</Button></div>}
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
    try { await api.post("/api/network/wifi/connect", { ssid: sel.ssid, password: pw }); toast(t("Connecté à {0} 📶", sel.ssid), "good"); setSel(null); scan.reload(); onChanged(); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <Card title={t("Réseaux Wi-Fi à proximité")} icon={<Wifi />} right={<Button size="sm" variant="ghost" icon={scan.loading ? <Spinner small /> : <RefreshCw />} onClick={scan.reload}>{t("Actualiser")}</Button>}>
      {scan.error ? <Callout mood="warn">{scan.error}</Callout> : !scan.data ? <Loading label={t("Je scanne les ondes…")} /> : scan.data.length === 0 ? <Empty emoji="📡" title={t("Aucun réseau trouvé")} /> : (
        scan.data.map((w) => (
          <div key={w.ssid} className="toggle-row">
            <Wifi style={{ opacity: 0.35 + w.signal / 140, flex: "none" }} />
            <div className="txt"><b>{w.ssid}</b><span>{t("Signal")} {w.signal >= 70 ? "excellent" : w.signal >= 45 ? "correct" : "faible"} · {w.security && w.security !== "--" ? w.security : "ouvert"}</span></div>
            {w.security && w.security !== "--" && <Lock size={15} className="muted" />}
            {w.active ? <Badge mood="good" icon={<Check />}>{t("Connecté")}</Badge> : <Button size="sm" onClick={() => { setSel(w); setPw(""); }}>{t("Se connecter")}</Button>}
            {w.known && !w.active && <Button size="sm" variant="ghost" onClick={async () => { if (await confirm({ title: t("Oublier « {0} » ?", w.ssid), confirm: t("Oublier"), body: t("Le mot de passe enregistré sera supprimé.") })) { await api.post("/api/network/wifi/forget", { ssid: w.ssid }).catch((e) => toast(e.message, "bad")); scan.reload(); } }}>{t("Oublier")}</Button>}
          </div>
        ))
      )}
      {sel && (
        <Modal title={t("Se connecter à « {0} »", sel.ssid)} onClose={() => setSel(null)} footer={<><Button variant="ghost" onClick={() => setSel(null)}>{t("Annuler")}</Button><Button variant="primary" loading={busy} onClick={connect}>{t("Se connecter")}</Button></>}>
          {sel.security && sel.security !== "--" ? <Field label={t("Mot de passe du Wi-Fi")}><PasswordInput value={pw} onChange={(e) => setPw(e.target.value)} autoFocus /></Field> : <p>{t("Ce réseau est ouvert (sans mot de passe).")}</p>}
          <Callout>{t("Si tu es connecté à l'interface via ce même Wi-Fi, la page peut se couper quelques secondes.")}</Callout>
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
      toast(method === "manual" ? t("Nouvelle adresse : {0}. Recharge la page à cette adresse si besoin.", address.split("/")[0]) : t("Retour en mode automatique"), "good");
      onClose();
    } catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  return (
    <Modal title={t("Adresse IP")} sub={iface.connection} onClose={onClose} footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="primary" loading={busy} onClick={save}>{t("Appliquer")}</Button></>}>
      <div className="grid grid-2" style={{ gap: 10, marginBottom: 16 }}>
        {([["auto", "Automatique", "Ta box choisit l'adresse (recommandé)."], ["manual", "IP fixe", "L'adresse ne change jamais. Pratique pour un serveur."]] as const).map(([id, t, d]) => (
          <button type="button" key={id} className="card flat" onClick={() => setMethod(id)} style={{ textAlign: "left", cursor: "pointer", padding: 14, borderColor: method === id ? "var(--brand)" : undefined }}>
            <b>{t}</b><div className="small muted">{d}</div>
          </button>
        ))}
      </div>
      {method === "manual" && (
        <>
          <Field label={t("Adresse et masque")} hint={t("Exemple : 192.168.1.50/24")}><Input value={address} onChange={(e) => setAddress(e.target.value)} className="input mono" /></Field>
          <Field label={t("Passerelle (ta box)")}><Input value={gateway} onChange={(e) => setGateway(e.target.value)} placeholder="192.168.1.1" /></Field>
          <Field label={t("Serveurs DNS")}><Input value={dns} onChange={(e) => setDns(e.target.value)} /></Field>
          <Callout mood="warn">{t("Une adresse incorrecte peut rendre ton Pi injoignable. Choisis une adresse libre sur ton réseau.")}</Callout>
        </>
      )}
    </Modal>
  );
}
