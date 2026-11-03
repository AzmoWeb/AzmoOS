import { useEffect, useState, type FormEvent } from "react";
import { ArrowRight, ArrowLeft, Wifi, Lock, Check, Sparkles } from "lucide-react";
import { api } from "../api/client";
import type { Hardware, Status, WiFi } from "../api/types";
import { Button, Field, Input, PasswordInput, PasswordMeter, Spinner, Callout, Badge } from "../components/ui";
import { slug } from "../lib/format";

export function Login({ status, onDone }: { status: Status; onDone: () => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.post("/api/login", { username, password });
      onDone();
    } catch (err) {
      setError((err as Error).message);
      setBusy(false);
    }
  };

  return (
    <div className="auth">
      <form className="card auth-card" onSubmit={submit} style={{ padding: 34 }}>
        <img className="auth-logo" src="/logo.svg" alt="" />
        <h1>Content de te revoir <span className="wave">👋</span></h1>
        <p className="lead">Connecte-toi à <b>{status.hostname}</b> pour retrouver ton serveur.</p>
        <Field label="Nom d'utilisateur">
          <Input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" autoCapitalize="none" autoFocus required />
        </Field>
        <Field label="Mot de passe">
          <PasswordInput value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required />
        </Field>
        <div className="form-error" role="alert">{error}</div>
        <Button variant="primary" size="lg" block loading={busy}>Se connecter</Button>
        <div className="made-by">Azmo OS {status.version} · Made by <b>Azmo</b></div>
      </form>
    </div>
  );
}

const timezones = ["Europe/Paris", "Europe/Brussels", "Europe/Zurich", "Europe/Luxembourg", "America/Montreal", "Africa/Casablanca", "Africa/Algiers", "Africa/Tunis", "Africa/Dakar", "Africa/Abidjan", "Indian/Reunion", "America/Martinique", "America/Guadeloupe", "Pacific/Tahiti", "Pacific/Noumea", "UTC"];

/** Assistant de premier démarrage. */
export function Setup({ status, onDone }: { status: Status; onDone: () => void }) {
  const [step, setStep] = useState(0);
  const [hw, setHw] = useState<Hardware | null>(null);
  const [net, setNet] = useState(false);

  // Étape compte
  const [display, setDisplay] = useState("");
  const [username, setUsername] = useState("");
  const [touchedUser, setTouchedUser] = useState(false);
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  // Étape Pi
  const [hostname, setHostname] = useState(status.hostname || "azmoos");
  const [tz, setTz] = useState(() => Intl.DateTimeFormat().resolvedOptions().timeZone || "Europe/Paris");

  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => { if (!touchedUser) setUsername(slug(display).replace(/-/g, "") || ""); }, [display, touchedUser]);

  const steps = ["Bienvenue", "Ton compte", "Ton Pi", ...(net ? ["Wi-Fi"] : []), "C'est prêt"];
  const last = steps.length - 1;

  const createAccount = async (e: FormEvent) => {
    e.preventDefault();
    if (password !== confirm) return setError("Les deux mots de passe ne sont pas identiques.");
    setBusy(true);
    setError("");
    try {
      await api.post("/api/setup", { username, display, password });
      const s = await api.get<Status>("/api/status");
      setHw(s.hardware ?? null);
      const n = await api.get<{ managed: boolean }>("/api/network").catch(() => ({ managed: false }));
      setNet(!!(n.managed && s.hardware?.wifi));
      setStep(2);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const savePi = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      if (hostname !== status.hostname) await api.post("/api/system/hostname", { hostname });
      await api.post("/api/system/timezone", { timezone: tz }).catch(() => undefined);
      setStep(3);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth">
      <div className="card auth-card" style={{ padding: 34, width: "min(500px, 100%)" }}>
        <div className="steps" aria-label={`Étape ${step + 1} sur ${steps.length}`}>{steps.map((_, i) => <i key={i} className={i <= step ? "on" : ""} />)}</div>

        {step === 0 && (
          <div className="page">
            <img className="auth-logo" src="/logo.svg" alt="" />
            <h1>Bienvenue sur Azmo OS <span className="wave">👋</span></h1>
            <p className="lead">Ton serveur personnel est presque prêt. On va le configurer ensemble, ça prend moins de deux minutes.</p>
            <ul style={{ listStyle: "none", padding: 0, margin: "0 0 26px", display: "grid", gap: 12 }}>
              {[["🔐", "Créer ton compte"], ["🍓", "Donner un nom à ton Pi"], ["🚀", "Installer tes premières apps"]].map(([e, t]) => (
                <li key={t} className="row" style={{ gap: 14 }}><span style={{ fontSize: 22 }}>{e}</span><b style={{ fontWeight: 600 }}>{t}</b></li>
              ))}
            </ul>
            <Button variant="primary" size="lg" block onClick={() => setStep(1)}>C'est parti <ArrowRight /></Button>
            <div className="made-by">Made by <b>Azmo</b></div>
          </div>
        )}

        {step === 1 && (
          <form className="page" onSubmit={createAccount}>
            <h1>Faisons connaissance</h1>
            <p className="lead">Ce compte aura tous les droits sur ton serveur.</p>
            <Field label="Comment tu t'appelles ?">
              <Input value={display} onChange={(e) => setDisplay(e.target.value)} placeholder="Azmo" autoFocus required />
            </Field>
            <Field label="Nom d'utilisateur" hint="Il te servira à te connecter : minuscules et chiffres.">
              <Input value={username} onChange={(e) => { setTouchedUser(true); setUsername(e.target.value.toLowerCase()); }} autoCapitalize="none" required pattern="[a-z][a-z0-9_\-]{1,31}" />
            </Field>
            <Field label="Mot de passe" hint={<PasswordMeter value={password} />}>
              <PasswordInput value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" minLength={8} required />
            </Field>
            <Field label="Confirme le mot de passe">
              <PasswordInput value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required />
            </Field>
            <div className="form-error" role="alert">{error}</div>
            <div className="row">
              <Button type="button" variant="ghost" onClick={() => setStep(0)}><ArrowLeft /> Retour</Button>
              <div className="spacer" />
              <Button variant="primary" loading={busy}>Créer mon compte <ArrowRight /></Button>
            </div>
          </form>
        )}

        {step === 2 && (
          <form className="page" onSubmit={savePi}>
            <h1>Enchanté {display.split(" ")[0]} !</h1>
            <p className="lead">Parlons un peu de ton Pi.</p>
            {hw && (
              <div className="callout" style={{ marginBottom: 20 }}>
                <Sparkles />
                <div>
                  <b>J'ai détecté : {hw.model}</b>
                  <div className="small" style={{ marginTop: 2 }}>{Math.round(hw.ramMB / 1024 * 10) / 10} Go de RAM · {hw.cores} cœurs · <Badge mood="brand">{hw.profile === "performance" ? "Mode performance" : hw.profile === "standard" ? "Mode standard" : "Mode léger"}</Badge></div>
                  <div className="small muted" style={{ marginTop: 6 }}>{hw.profileHint}</div>
                </div>
              </div>
            )}
            <Field label="Nom de ton Pi sur le réseau" hint={<>Tu pourras l'ouvrir depuis n'importe quel appareil avec <b>http://{hostname || "…"}.local</b></>}>
              <Input value={hostname} onChange={(e) => setHostname(e.target.value.replace(/[^a-zA-Z0-9-]/g, ""))} required maxLength={63} />
            </Field>
            <Field label="Fuseau horaire">
              <select className="select" value={tz} onChange={(e) => setTz(e.target.value)}>
                {[...new Set([tz, ...timezones])].map((z) => <option key={z} value={z}>{z.replace("_", " ")}</option>)}
              </select>
            </Field>
            <div className="form-error" role="alert">{error}</div>
            <div className="row"><div className="spacer" /><Button variant="primary" loading={busy}>Continuer <ArrowRight /></Button></div>
          </form>
        )}

        {step === 3 && net && <WifiStep onNext={() => setStep(4)} />}

        {step === last && step >= 3 && (
          <div className="page" style={{ textAlign: "center" }}>
            <div style={{ fontSize: 64, lineHeight: 1, marginBottom: 14 }}>🎉</div>
            <h1>Tout est prêt !</h1>
            <p className="lead">Ton serveur Azmo OS t'attend. Commence par installer une application depuis l'App Store : Jellyfin pour tes films, Home Assistant pour ta maison…</p>
            <Button variant="primary" size="lg" block onClick={onDone}>Découvrir mon tableau de bord <ArrowRight /></Button>
            <div className="made-by">Made with ❤️ by <b>Azmo</b></div>
          </div>
        )}
      </div>
    </div>
  );
}

function WifiStep({ onNext }: { onNext: () => void }) {
  const [list, setList] = useState<WiFi[] | null>(null);
  const [sel, setSel] = useState<WiFi | null>(null);
  const [pw, setPw] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => { api.get<WiFi[]>("/api/network/wifi").then(setList).catch(() => setList([])); }, []);

  const connect = async (e: FormEvent) => {
    e.preventDefault();
    if (!sel) return;
    setBusy(true);
    setError("");
    try {
      await api.post("/api/network/wifi/connect", { ssid: sel.ssid, password: pw });
      onNext();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const active = list?.find((w) => w.active);
  return (
    <form className="page" onSubmit={connect}>
      <h1>Connexion Wi-Fi</h1>
      <p className="lead">{active ? <>Ton Pi est déjà connecté à <b>{active.ssid}</b>. Tu peux passer cette étape.</> : "Choisis ton réseau Wi-Fi (ou passe si ton Pi est branché en Ethernet)."}</p>
      {!list ? <div className="center"><Spinner /></div> : (
        <div style={{ display: "grid", gap: 8, maxHeight: 260, overflowY: "auto", marginBottom: 16 }}>
          {list.length === 0 && <Callout>Aucun réseau trouvé. Pas d'inquiétude, tu pourras le configurer plus tard dans Réseau.</Callout>}
          {list.map((w) => (
            <button type="button" key={w.ssid} onClick={() => { setSel(w); setPw(""); }} className="btn" style={{ justifyContent: "flex-start", height: 48, borderColor: sel?.ssid === w.ssid ? "var(--brand)" : undefined }}>
              <Wifi style={{ opacity: 0.3 + w.signal / 140 }} />
              <span style={{ flex: 1, textAlign: "left" }}>{w.ssid}</span>
              {w.active && <Check style={{ color: "var(--good)" }} />}
              {w.security && w.security !== "--" && <Lock style={{ opacity: 0.5 }} />}
            </button>
          ))}
        </div>
      )}
      {sel && !sel.active && sel.security && sel.security !== "--" && (
        <Field label={`Mot de passe de « ${sel.ssid} »`}><PasswordInput value={pw} onChange={(e) => setPw(e.target.value)} autoFocus required /></Field>
      )}
      <div className="form-error" role="alert">{error}</div>
      <div className="row">
        <Button type="button" variant="ghost" onClick={onNext}>Passer</Button>
        <div className="spacer" />
        <Button variant="primary" loading={busy} disabled={!sel || sel.active}>Se connecter <ArrowRight /></Button>
      </div>
    </form>
  );
}
