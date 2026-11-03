import { useEffect, useState } from "react";
import { KeyRound, Lock, LockOpen, Download, RefreshCw, TerminalSquare, Copy, ShieldCheck, ExternalLink } from "lucide-react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { HTTPSInfo, TerminalInfo } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useSession } from "../session";
import { useUI, Modal } from "../components/overlay";
import { Badge, Button, Callout, Card, Field, Input, Loading, PasswordInput, Tabs, ToggleRow } from "../components/ui";
import { dateTime } from "../lib/format";
import { t } from "../lib/i18n";

/* ---------- Double authentification ---------- */

export function TwoFactorCard() {
  const { status, refresh } = useSession();
  const user = status.user!;
  const [setup, setSetup] = useState(false);
  const [disable, setDisable] = useState(false);
  return (
    <Card title={t("Double authentification")} icon={<KeyRound />} right={user.totp ? <Badge mood="good" icon={<ShieldCheck />}>{t("Activée")}</Badge> : <Badge mood="warn">{t("Désactivée")}</Badge>}>
      <p className="muted small" style={{ marginBottom: 14 }}>
        {t("En plus du mot de passe, un code à 6 chiffres de ton téléphone sera demandé à chaque connexion. Même si quelqu'un devine ton mot de passe, il ne pourra pas entrer.")}
      </p>
      {user.totp ? (
        <>
          <p className="small" style={{ marginBottom: 12 }}>{t("Il te reste")} <b>{user.recoveryLeft}</b> {t("code(s) de secours.")}</p>
          <div className="row"><Button size="sm" onClick={() => setSetup(true)}>{t("Changer de téléphone")}</Button><Button size="sm" variant="danger" onClick={() => setDisable(true)}>{t("Désactiver")}</Button></div>
        </>
      ) : <Button variant="primary" icon={<KeyRound />} onClick={() => setSetup(true)}>{t("Activer la double authentification")}</Button>}
      {setup && <SetupDialog onClose={() => setSetup(false)} onDone={refresh} />}
      {disable && <DisableDialog onClose={() => setDisable(false)} onDone={refresh} />}
    </Card>
  );
}

function SetupDialog({ onClose, onDone }: { onClose: () => void; onDone: () => Promise<void> }) {
  const { toast } = useUI();
  const [secret, setSecret] = useState("");
  const [svg, setSvg] = useState("");
  const [code, setCode] = useState("");
  const [codes, setCodes] = useState<string[] | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    api.post<{ secret: string; uri: string }>("/api/me/2fa/begin").then(async (r) => {
      setSecret(r.secret);
      const QRCode = (await import("qrcode")).default; // chargé à la demande
      setSvg(await QRCode.toString(r.uri, { type: "svg", margin: 0, errorCorrectionLevel: "M" }));
    }).catch((e) => toast(e.message, "bad"));
  }, [toast]);

  const confirm = async () => {
    setBusy(true);
    try { const r = await api.post<{ recovery: string[] }>("/api/me/2fa/confirm", { code }); setCodes(r.recovery); await onDone(); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };
  const download = () => {
    const txt = t("Azmo OS — codes de secours ({0})\n\nChaque code ne sert qu'une fois, si tu perds ton téléphone.\n\n{1}\n", location.hostname, codes!.join("\n"));
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([txt], { type: "text/plain" }));
    a.download = t("azmoos-codes-de-secours-{0}.txt", location.hostname);
    a.click();
  };

  if (codes) return (
    <Modal title={t("C'est activé ! 🎉")} onClose={onClose} footer={<><Button onClick={download} icon={<Download />}>{t("Télécharger")}</Button><Button variant="primary" onClick={onClose}>{t("J'ai mis mes codes à l'abri")}</Button></>}>
      <Callout mood="warn" title={t("Garde ces codes de secours précieusement")}>{t("Si tu perds ton téléphone, chacun de ces codes te permettra de te connecter une fois. Note-les sur papier ou dans ton gestionnaire de mots de passe.")}</Callout>
      <div className="codes" style={{ marginTop: 16 }}>{codes.map((c) => <span key={c}>{c}</span>)}</div>
    </Modal>
  );
  return (
    <Modal title={t("Activer la double authentification")} onClose={onClose}
      footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="primary" loading={busy} disabled={code.length !== 6} onClick={confirm}>{t("Activer")}</Button></>}>
      <ol className="small" style={{ paddingLeft: 18, margin: "0 0 16px", display: "grid", gap: 6, color: "var(--text-2)" }}>
        <li>{t("Installe une application d'authentification : Google Authenticator, Authy, Microsoft Authenticator, 1Password ou Bitwarden.")}</li>
        <li>{t("Dans l'app, ajoute un compte et scanne ce QR code.")}</li>
        <li>{t("Saisis le code à 6 chiffres affiché par l'app.")}</li>
      </ol>
      {svg ? <div className="qr" dangerouslySetInnerHTML={{ __html: svg }} /> : <Loading />}
      {secret && (
        <p className="small muted" style={{ textAlign: "center", margin: "10px 0 16px" }}>
          {t("Impossible de scanner ? Saisis cette clé :")} <code style={{ userSelect: "all" }}>{secret.match(/.{1,4}/g)?.join(" ")}</code>
        </p>
      )}
      <Field label={t("Code à 6 chiffres")}>
        <input className="input otp-input" inputMode="numeric" autoComplete="one-time-code" maxLength={6} value={code} onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))} autoFocus onKeyDown={(e) => e.key === "Enter" && code.length === 6 && confirm()} />
      </Field>
    </Modal>
  );
}

function DisableDialog({ onClose, onDone }: { onClose: () => void; onDone: () => Promise<void> }) {
  const { toast } = useUI();
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const go = async () => {
    try { await api.post("/api/me/2fa/disable", { password, code }); toast(t("Double authentification désactivée"), "info"); await onDone(); onClose(); }
    catch (e) { toast((e as Error).message, "bad"); }
  };
  return (
    <Modal title={t("Désactiver la double authentification ?")} onClose={onClose}
      footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="danger-solid" disabled={!password || !code} onClick={go}>{t("Désactiver")}</Button></>}>
      <p className="muted small" style={{ marginBottom: 14 }}>{t("Ton compte ne sera plus protégé que par ton mot de passe. Pour confirmer que c'est bien toi :")}</p>
      <Field label={t("Mot de passe")}><PasswordInput value={password} onChange={(e) => setPassword(e.target.value)} autoFocus /></Field>
      <Field label={t("Code à 6 chiffres (ou code de secours)")}><Input value={code} onChange={(e) => setCode(e.target.value)} autoComplete="one-time-code" /></Field>
    </Modal>
  );
}

/* ---------- HTTPS local ---------- */

const guides: Record<string, { label: string; steps: string[] }> = {
  ios: { label: t("iPhone / iPad"), steps: [
    t("Ouvre cette page dans Safari et touche « Télécharger le certificat », puis « Autoriser »."),
    t("Va dans Réglages → Profil téléchargé → Installer."),
    t("Puis Réglages → Général → Informations → Réglages des certificats : active « Azmo OS — autorité locale »."),
  ] },
  android: { label: "Android", steps: [
    t("Télécharge le certificat."),
    t("Va dans Paramètres → Sécurité → Chiffrement et identifiants → Installer un certificat → Certificat CA."),
    t("Choisis le fichier téléchargé et confirme."),
  ] },
  windows: { label: "Windows", steps: [
    t("Télécharge le certificat et double-clique dessus."),
    t("Installer le certificat → Ordinateur local → « Placer tous les certificats dans le magasin suivant » → Autorités de certification racines de confiance."),
    t("Redémarre ton navigateur."),
  ] },
  mac: { label: "Mac", steps: [
    t("Télécharge le certificat et double-clique dessus : il s'ouvre dans Trousseaux d'accès."),
    t("Double-clique sur « Azmo OS — autorité locale » → Se fier → « Toujours approuver »."),
    t("Redémarre ton navigateur."),
  ] },
  linux: { label: "Linux", steps: [
    t("Firefox : Paramètres → Vie privée et sécurité → Certificats → Afficher les certificats → Autorités → Importer."),
    t("Chrome / système : sudo cp azmoos-*.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates"),
  ] },
};

export function HTTPSCard() {
  const { toast, confirm } = useUI();
  const info = usePoll(() => api.get<HTTPSInfo>("/api/https"));
  const [guide, setGuide] = useState("ios");
  const [busy, setBusy] = useState(false);
  if (!info.data) return <Card title={t("HTTPS local")} icon={<Lock />}><Loading /></Card>;
  const d = info.data;
  const httpsURL = `https://${location.hostname}${d.port !== 443 ? ":" + d.port : ""}${location.pathname}`;

  const apply = async (enabled: boolean, redirect: boolean) => {
    if (redirect && !d.secure && !(await confirm({ title: t("Passer tout en HTTPS ?"), confirm: t("Activer la redirection"), body: t("Toutes les connexions seront redirigées vers https://. Installe d'abord le certificat sur tes appareils pour éviter l'avertissement du navigateur.") }))) return;
    setBusy(true);
    try { await api.post("/api/https", { enabled, redirect }); toast(enabled ? t("HTTPS activé 🔒") : t("HTTPS désactivé"), "good"); setTimeout(info.reload, 400); }
    catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };

  return (
    <Card title={t("HTTPS local")} icon={d.secure ? <Lock /> : <LockOpen />} right={d.cert.running ? <Badge mood="good" icon={<Lock />}>{t("Actif")}</Badge> : <Badge>{t("Inactif")}</Badge>}>
      <p className="muted small" style={{ marginBottom: 10 }}>
        {t("Chiffre la connexion entre ton navigateur et ton Pi : personne sur ton Wi-Fi ne peut lire tes mots de passe ou tes fichiers. Azmo OS crée sa propre autorité de certification,")} <b>{t("limitée à ton réseau local")}</b>.
      </p>
      <ToggleRow title={t("Activer le HTTPS")} desc={t("Interface disponible sur https://…{0}", d.port !== 443 ? ":" + d.port : "")} checked={d.enabled} disabled={busy} onChange={(v) => apply(v, false)} />
      {d.enabled && <ToggleRow title={t("Toujours utiliser HTTPS")} desc={t("Redirige automatiquement http:// vers https://.")} checked={d.redirect} disabled={busy} onChange={(v) => apply(true, v)} />}
      {d.enabled && !d.secure && d.cert.running && (
        <a className="btn sm primary" href={httpsURL} style={{ marginTop: 12 }}><ExternalLink /> {t("Ouvrir en HTTPS")}</a>
      )}
      <div style={{ marginTop: 18 }}>
        <b style={{ fontSize: 14.5 }}>{t("Faire confiance au certificat sur tes appareils")}</b>
        <p className="small muted" style={{ margin: "4px 0 12px" }}>{t("À faire une fois par appareil, pour ne plus voir l'avertissement « connexion non sécurisée ».")}</p>
        <a className="btn sm" href="/api/https/ca.crt" download><Download /> {t("Télécharger le certificat")}</a>
        <div style={{ marginTop: 14 }}>
          <Tabs value={guide} onChange={setGuide} tabs={Object.entries(guides).map(([id, g]) => ({ id, label: g.label }))} />
          <ol className="small" style={{ paddingLeft: 18, margin: 0, display: "grid", gap: 6, color: "var(--text-2)" }}>{guides[guide].steps.map((s) => <li key={s}>{s}</li>)}</ol>
        </div>
        {d.cert.caFingerprint && (
          <details style={{ marginTop: 14 }}>
            <summary>{t("Détails techniques")}</summary>
            <div className="small" style={{ display: "grid", gap: 6, marginTop: 8 }}>
              <div><span className="muted">{t("Empreinte SHA-256 de l'autorité :")}</span> <code style={{ wordBreak: "break-all" }}>{d.cert.caFingerprint}</code> <button className="icon-btn" style={{ width: 26, height: 26 }} onClick={() => navigator.clipboard?.writeText(d.cert.caFingerprint)} aria-label={t("Copier")}><Copy size={14} /></button></div>
              {d.cert.names.length > 0 && <div><span className="muted">{t("Valable pour :")}</span> {d.cert.names.join(", ")}</div>}
              {d.cert.expires > 0 && <div><span className="muted">{t("Renouvellement automatique avant le :")}</span> {dateTime(d.cert.expires)}</div>}
              <Button size="sm" variant="ghost" icon={<RefreshCw />} onClick={async () => { await api.post("/api/https/regenerate"); toast(t("Certificat régénéré"), "good"); info.reload(); }}>{t("Régénérer le certificat")}</Button>
            </div>
          </details>
        )}
      </div>
    </Card>
  );
}

/* ---------- Terminal ---------- */

export function TerminalToggle() {
  const { toast } = useUI();
  const info = usePoll(() => api.get<TerminalInfo>("/api/terminal/info"));
  if (!info.data) return null;
  return (
    <>
      <ToggleRow title={<><TerminalSquare size={15} style={{ verticalAlign: -2 }} /> {t("Terminal web")}</>}
        desc={t("La ligne de commande dans le navigateur, réservée aux administrateurs (utilisateur système : {0}).", info.data.account.name)}
        checked={info.data.enabled} onChange={async (v) => {
          try { await api.post("/api/terminal/settings", { enabled: v }); toast(v ? t("Terminal web activé") : t("Terminal web désactivé"), "good"); info.reload(); }
          catch (e) { toast((e as Error).message, "bad"); }
        }} />
      {info.data.enabled && <Link to="/terminal" className="btn sm" style={{ marginTop: 6 }}><TerminalSquare /> {t("Ouvrir le terminal")}</Link>}
    </>
  );
}
