import { useState } from "react";
import { User as UserIcon, KeyRound, Cpu, Power, RotateCw, Clock, Heart, Languages } from "lucide-react";
import { lang, languages, setLang } from "../lib/i18n";
import { api } from "../api/client";
import type { SystemStats } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useSession } from "../session";
import { useUI } from "../components/overlay";
import { Avatar, Badge, Button, Card, Field, Input, PageHead, PasswordInput, PasswordMeter } from "../components/ui";
import { profileLabel, machineName } from "../lib/human";
import { uptime } from "../lib/format";
import { t } from "../lib/i18n";

export function Settings() {
  const { status, refresh, can } = useSession();
  const { toast, confirm } = useUI();
  const user = status.user!;
  const admin = can("admin");
  const sys = usePoll(() => api.get<SystemStats>("/api/system"));
  const settings = usePoll(() => (admin ? api.get<{ timezone: string }>("/api/system/settings") : Promise.resolve({ timezone: "" })));

  const [display, setDisplay] = useState(user.display);
  const [cur, setCur] = useState("");
  const [pw, setPw] = useState("");
  const [host, setHost] = useState(status.hostname);
  const [tz, setTz] = useState("");
  const hw = status.hardware;

  const run = (fn: () => Promise<unknown>, ok: string) => async () => {
    try { await fn(); toast(ok, "good"); } catch (e) { toast((e as Error).message, "bad"); }
  };
  const power = (action: "reboot" | "poweroff") => async () => {
    const label = action === "reboot" ? t("Redémarrer") : t("Éteindre");
    if (!(await confirm({ title: t("{0} ton Pi ?", label), danger: action === "poweroff", confirm: label, body: action === "reboot" ? t("Ton Pi sera de retour dans environ une minute. Cette page se reconnectera toute seule.") : t("Pour le rallumer, il faudra débrancher puis rebrancher l'alimentation (ou appuyer sur le bouton du Pi 5).") }))) return;
    try { await api.post("/api/system/power", { action }); toast(action === "reboot" ? t("À tout de suite ! 👋") : t("Bonne nuit ! 🌙")); } catch (e) { toast((e as Error).message, "bad"); }
  };

  return (
    <>
      <PageHead title={t("Réglages")} sub={t("Ton profil et les réglages généraux de ton Pi.")} />
      <div className="grid grid-2">
        <Card title={t("Mon profil")} icon={<UserIcon />}>
          <div className="row" style={{ marginBottom: 18, gap: 14 }}><Avatar name={display || user.display} size={54} /><div><b>{user.display}</b><div className="muted small">@{user.username} · {user.role === "admin" ? t("Administrateur") : t("Membre")}</div></div></div>
          <Field label={t("Nom affiché")}><Input value={display} onChange={(e) => setDisplay(e.target.value)} /></Field>
          <Button onClick={run(async () => { await api.post("/api/me/profile", { display }); await refresh(); }, "Profil mis à jour")}>{t("Enregistrer")}</Button>
        </Card>
        <Card title={t("Mot de passe")} icon={<KeyRound />}>
          <Field label={t("Mot de passe actuel")}><PasswordInput value={cur} onChange={(e) => setCur(e.target.value)} autoComplete="current-password" /></Field>
          <Field label={t("Nouveau mot de passe")} hint={<PasswordMeter value={pw} />}><PasswordInput value={pw} onChange={(e) => setPw(e.target.value)} autoComplete="new-password" /></Field>
          <Button disabled={!cur || pw.length < 8} onClick={run(async () => { await api.post("/api/me/password", { current: cur, password: pw }); setCur(""); setPw(""); }, "Mot de passe changé 🔐 Tes autres appareils ont été déconnectés.")}>{t("Changer le mot de passe")}</Button>
        </Card>
        <Card title={t("Langue")} icon={<Languages />}>
          <p className="muted small" style={{ marginBottom: 14 }}>{t("Choisis la langue de l'interface. Elle est enregistrée dans ton compte et sur cet appareil.")}</p>
          <div className="row">
            {languages.map((l) => (
              <Button key={l.id} variant={lang() === l.id ? "primary" : undefined} onClick={async () => { await api.post("/api/me/lang", { lang: l.id }).catch(() => undefined); setLang(l.id); }}>
                {l.flag} {l.label}
              </Button>
            ))}
          </div>
        </Card>
        {admin && (
          <Card title={t("Ton Pi")} icon={<Cpu />}>
            <Field label={t("Nom sur le réseau")} hint={<>{t("Accessible via")} <b>http://{host || "…"}.local</b></>}><Input value={host} onChange={(e) => setHost(e.target.value.replace(/[^a-zA-Z0-9-]/g, ""))} /></Field>
            <Field label={t("Fuseau horaire")} hint={t("Exemple : Europe/Paris")}><Input value={tz || settings.data?.timezone || ""} onChange={(e) => setTz(e.target.value)} /></Field>
            <div className="row">
              <Button onClick={run(async () => { if (host !== status.hostname) await api.post("/api/system/hostname", { hostname: host }); if (tz) await api.post("/api/system/timezone", { timezone: tz }); await refresh(); }, "Réglages enregistrés")}>{t("Enregistrer")}</Button>
            </div>
          </Card>
        )}
        {admin && (
          <Card title={t("Alimentation")} icon={<Power />}>
            <p className="muted" style={{ marginBottom: 16 }}>{t("Redémarre ton Pi après une mise à jour importante, ou éteins-le proprement avant de le débrancher.")}</p>
            <div className="row"><Button icon={<RotateCw />} onClick={power("reboot")}>{t("Redémarrer")}</Button><Button variant="danger" icon={<Power />} onClick={power("poweroff")}>{t("Éteindre")}</Button></div>
          </Card>
        )}
        <Card title={t("À propos")} icon={<Heart />}>
          <div className="grid grid-2" style={{ gap: 14 }}>
            <div><div className="tiny muted">{t("Version")}</div><b>{t("Azmo OS")} {status.version}</b></div>
            <div><div className="tiny muted">{t("Matériel")}</div><b>{machineName(hw?.model)}</b></div>
            {hw && <div><div className="tiny muted">{t("Profil")}</div><Badge mood="brand">{profileLabel[hw.profile]}</Badge></div>}
            {hw && <div><div className="tiny muted">{t("Mémoire · cœurs")}</div><b>{(hw.ramMB / 1024).toFixed(1).replace(".", ",")} {t("Go ·")} {hw.cores}</b></div>}
            <div><div className="tiny muted">{t("Système")}</div><b>{sys.data?.os ?? "—"}</b></div>
            <div><div className="tiny muted"><Clock size={11} /> {t("Allumé depuis")}</div><b>{sys.data ? uptime(sys.data.uptime) : "—"}</b></div>
          </div>
          <p className="made-by" style={{ textAlign: "left" }}>{t("Conçu avec ❤️ — Made by")} <b>Azmo</b></p>
        </Card>
      </div>
    </>
  );
}
