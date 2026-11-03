import { useEffect, useState } from "react";
import { Bell, Send, Thermometer, HardDrive, MemoryStick, Package, Gift, ShieldAlert, Archive, Smartphone, CheckCheck, Trash2 } from "lucide-react";
import { api } from "../api/client";
import type { AzmoNotification, NotifySettings } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useUI } from "../components/overlay";
import { notifIcon } from "../components/Notifications";
import { Button, Callout, Card, Empty, ErrorBox, Field, Input, Loading, PageHead, PasswordInput, Tabs, ToggleRow } from "../components/ui";
import { ago } from "../lib/format";
import { t } from "../lib/i18n";

export function NotificationsPage() {
  const [tab, setTab] = useState<"list" | "alerts" | "channels">("list");
  return (
    <>
      <PageHead title={t("Notifications")} sub={t("Azmo OS veille sur ton Pi et te prévient quand quelque chose mérite ton attention, même sur ton téléphone.")} />
      <Tabs value={tab} onChange={setTab} tabs={[
        { id: "list", label: t("Historique"), icon: <Bell /> },
        { id: "alerts", label: t("Alertes"), icon: <ShieldAlert /> },
        { id: "channels", label: t("Sur mon téléphone"), icon: <Smartphone /> },
      ]} />
      {tab === "list" ? <History /> : <SettingsForm tab={tab} />}
    </>
  );
}

function History() {
  const data = usePoll(() => api.get<{ items: AzmoNotification[]; unread: number }>("/api/notifications"), 15000);
  if (data.error && !data.data) return <ErrorBox error={data.error} onRetry={data.reload} />;
  if (!data.data) return <Loading />;
  const { items, unread } = data.data;
  return (
    <>
      <div className="row" style={{ marginBottom: 14 }}>
        <span className="muted small">{unread ? t("{0} non lue(s)", unread) : t("Tout est lu")}</span>
        <div className="spacer" />
        {unread > 0 && <Button size="sm" icon={<CheckCheck />} onClick={async () => { await api.post("/api/notifications/read", { id: 0 }); data.reload(); }}>{t("Tout marquer comme lu")}</Button>}
        {items.some((n) => n.read) && <Button size="sm" variant="ghost" icon={<Trash2 />} onClick={async () => { await api.post("/api/notifications/clear"); data.reload(); }}>{t("Effacer les notifications lues")}</Button>}
      </div>
      {items.length === 0 ? <Card><Empty emoji="🔕" title={t("Rien à signaler")}>{t("Tout va bien ! Les alertes apparaîtront ici : température, stockage, applications, mises à jour, sécurité…")}</Empty></Card> : (
        <div className="card pad-0">
          <ul className="timeline">
            {items.map((n) => (
              <li key={n.id} style={n.read ? undefined : { background: "color-mix(in srgb, var(--brand-soft) 50%, transparent)" }}>
                <div className={`tl-icon ${n.level}`}>{notifIcon(n.kind)}</div>
                <div className="tl-body"><b>{n.title}</b><div className="small" style={{ color: "var(--text-2)", marginTop: 2 }}>{n.message}</div><div className="tl-meta">{ago(n.time)}</div></div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  );
}

function SettingsForm({ tab }: { tab: "alerts" | "channels" }) {
  const { toast } = useUI();
  const data = usePoll(() => api.get<NotifySettings>("/api/notifications/settings"));
  const [s, setS] = useState<NotifySettings | null>(null);
  const [busy, setBusy] = useState(false);
  const [test, setTest] = useState<Record<string, string> | null>(null);
  useEffect(() => { if (data.data) setS(data.data); }, [data.data]);
  if (data.error && !data.data) return <ErrorBox error={data.error} onRetry={data.reload} />;
  if (!s) return <Loading />;
  const set = <K extends keyof NotifySettings>(k: K, v: NotifySettings[K]) => setS({ ...s, [k]: v });

  const save = async (quiet = false) => {
    setBusy(true);
    try { await api.post("/api/notifications/settings", s); if (!quiet) toast(t("Réglages enregistrés"), "good"); return true; }
    catch (e) { toast((e as Error).message, "bad"); return false; } finally { setBusy(false); }
  };
  const sendTest = async () => {
    if (!(await save(true))) return;
    try { setTest(await api.post<Record<string, string>>("/api/notifications/test")); } catch (e) { toast((e as Error).message, "bad"); }
  };
  const anyChannel = s.ntfy.enabled || s.discord.enabled || s.telegram.enabled;

  return (
    <div className="stack">
      {tab === "alerts" && (
        <>
          <Card title={t("Ce que je surveille")} icon={<ShieldAlert />}>
            <ToggleRow title={<><Thermometer size={15} style={{ verticalAlign: -2 }} /> {t("Température et alimentation")}</>} desc={t("Alerte au-dessus de {0} °C, ou si l'alimentation est trop faible.", s.tempWarn)} checked={s.temperature} onChange={(v) => set("temperature", v)} />
            <ToggleRow title={<><HardDrive size={15} style={{ verticalAlign: -2 }} /> {t("Stockage")}</>} desc={t("Alerte quand un disque est rempli à plus de {0} %.", s.diskWarn)} checked={s.disk} onChange={(v) => set("disk", v)} />
            <ToggleRow title={<><MemoryStick size={15} style={{ verticalAlign: -2 }} /> {t("Mémoire")}</>} desc={t("Alerte si la mémoire reste au-dessus de {0} % plusieurs minutes.", s.memWarn)} checked={s.memory} onChange={(v) => set("memory", v)} />
            <ToggleRow title={<><Package size={15} style={{ verticalAlign: -2 }} /> {t("Applications")}</>} desc={t("Une application s'arrête toute seule ou plante en boucle.")} checked={s.apps} onChange={(v) => set("apps", v)} />
            <ToggleRow title={<><Gift size={15} style={{ verticalAlign: -2 }} /> {t("Mises à jour")}</>} desc={t("Une nouvelle version d'Azmo OS est disponible.")} checked={s.updates} onChange={(v) => set("updates", v)} />
            <ToggleRow title={<><ShieldAlert size={15} style={{ verticalAlign: -2 }} /> {t("Sécurité")}</>} desc={t("Tentatives de connexion suspectes, utilisation d'un code de secours.")} checked={s.security} onChange={(v) => set("security", v)} />
            <ToggleRow title={<><Archive size={15} style={{ verticalAlign: -2 }} /> {t("Sauvegardes")}</>} desc={t("Résultat des sauvegardes automatiques.")} checked={s.backups} onChange={(v) => set("backups", v)} />
          </Card>
          <Card title={t("Seuils d'alerte")}>
            <div className="grid grid-3" style={{ gap: 14 }}>
              <Field label={t("Température (°C)")} hint={t("Le Pi ralentit vers 80-85 °C.")}><Input type="number" min={50} max={90} value={s.tempWarn} onChange={(e) => set("tempWarn", +e.target.value)} /></Field>
              <Field label={t("Stockage (%)")}><Input type="number" min={50} max={99} value={s.diskWarn} onChange={(e) => set("diskWarn", +e.target.value)} /></Field>
              <Field label={t("Mémoire (%)")}><Input type="number" min={50} max={99} value={s.memWarn} onChange={(e) => set("memWarn", +e.target.value)} /></Field>
            </div>
          </Card>
        </>
      )}

      {tab === "channels" && (
        <>
          <Callout title={t("Recevoir les alertes sur ton téléphone")}>{t("Choisis un ou plusieurs services. Le plus simple est")} <b>{"ntfy"}</b> {t(": gratuit, sans compte, une app Android/iOS et c'est prêt.")}</Callout>
          <Card title="ntfy" icon={<Bell />} sub={t("Installe l'app ntfy, abonne-toi à un sujet secret (inventes-en un long et unique), puis renseigne-le ici.")} right={<SwitchInline on={s.ntfy.enabled} set={(v) => set("ntfy", { ...s.ntfy, enabled: v })} />}>
            {s.ntfy.enabled && (
              <div className="grid grid-2" style={{ gap: 14 }}>
                <Field label={t("Sujet (topic)")} hint={t("Exemple : azmoos-maison-7f3k9q")}><Input value={s.ntfy.topic} onChange={(e) => set("ntfy", { ...s.ntfy, topic: e.target.value.trim() })} placeholder="azmoos-…" /></Field>
                <Field label={t("Serveur")} hint={t("Laisse https://ntfy.sh sauf si tu as ton propre serveur.")}><Input value={s.ntfy.server} onChange={(e) => set("ntfy", { ...s.ntfy, server: e.target.value })} /></Field>
                <Field label={t("Jeton d'accès (optionnel)")}><PasswordInput value={s.ntfy.token ?? ""} onChange={(e) => set("ntfy", { ...s.ntfy, token: e.target.value })} /></Field>
              </div>
            )}
          </Card>
          <Card title={"Discord"} icon={<Send />} sub={t("Dans ton serveur Discord : Paramètres du salon → Intégrations → Webhooks → Nouveau webhook → Copier l'URL.")} right={<SwitchInline on={s.discord.enabled} set={(v) => set("discord", { ...s.discord, enabled: v })} />}>
            {s.discord.enabled && <Field label={t("URL du webhook")}><PasswordInput value={s.discord.webhook} onChange={(e) => set("discord", { ...s.discord, webhook: e.target.value.trim() })} placeholder="https://discord.com/api/webhooks/…" /></Field>}
          </Card>
          <Card title={"Telegram"} icon={<Send />} sub={t("Crée un bot avec @BotFather, envoie-lui un message, puis récupère ton identifiant de discussion (par exemple avec @userinfobot).")} right={<SwitchInline on={s.telegram.enabled} set={(v) => set("telegram", { ...s.telegram, enabled: v })} />}>
            {s.telegram.enabled && (
              <div className="grid grid-2" style={{ gap: 14 }}>
                <Field label={t("Jeton du bot")}><PasswordInput value={s.telegram.token} onChange={(e) => set("telegram", { ...s.telegram, token: e.target.value.trim() })} placeholder="123456:ABC…" /></Field>
                <Field label={t("Identifiant de discussion")}><Input value={s.telegram.chatId} onChange={(e) => set("telegram", { ...s.telegram, chatId: e.target.value.trim() })} placeholder="123456789" /></Field>
              </div>
            )}
          </Card>
          <Card title={t("Préférences d'envoi")}>
            <div className="grid grid-2" style={{ gap: 14 }}>
              <Field label={t("Envoyer sur le téléphone à partir de")}>
                <select className="select" value={s.pushLevel} onChange={(e) => set("pushLevel", e.target.value as NotifySettings["pushLevel"])}>
                  <option value="info">{t("Toutes les notifications")}</option>
                  <option value="warning">{t("Avertissements et alertes graves (conseillé)")}</option>
                  <option value="critical">{t("Alertes graves uniquement")}</option>
                </select>
              </Field>
              <Field label={t("Langue des messages envoyés")}>
                <select className="select" value={s.lang} onChange={(e) => set("lang", e.target.value as "fr" | "en")}>
                  <option value="fr">{"Français"}</option><option value="en">{"English"}</option>
                </select>
              </Field>
            </div>
            {test && (
              <div className="stack" style={{ gap: 8, marginTop: 4 }}>
                {Object.keys(test).length === 0 && <Callout mood="warn">{t("Active au moins un service ci-dessus pour tester.")}</Callout>}
                {Object.entries(test).map(([k, v]) => <Callout key={k} mood={v === "ok" ? "good" : "bad"} title={k}>{v === "ok" ? t("Message envoyé, vérifie ton téléphone 📱") : v}</Callout>)}
              </div>
            )}
          </Card>
        </>
      )}
      <div className="row" style={{ justifyContent: "flex-end" }}>
        {tab === "channels" && <Button icon={<Send />} disabled={!anyChannel} onClick={sendTest}>{t("Envoyer un test")}</Button>}
        <Button variant="primary" loading={busy} onClick={() => save()}>{t("Enregistrer")}</Button>
      </div>
    </div>
  );
}

function SwitchInline({ on, set }: { on: boolean; set: (v: boolean) => void }) {
  return (
    <label className="switch" aria-label={t("Activer")}>
      <input type="checkbox" checked={on} onChange={(e) => set(e.target.checked)} /><i />
    </label>
  );
}
