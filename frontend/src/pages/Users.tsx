import { useState } from "react";
import { UserPlus, Pencil, KeyRound, Trash2, Crown, ShieldCheck, ShieldOff } from "lucide-react";
import { api, q } from "../api/client";
import type { Perm, User } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useSession } from "../session";
import { useUI, Modal } from "../components/overlay";
import { Avatar, Badge, Button, Card, ErrorBox, Field, Input, Loading, PageHead, PasswordInput, PasswordMeter, ToggleRow } from "../components/ui";
import { ago, slug } from "../lib/format";
import { t } from "../lib/i18n";

const permInfo: Record<string, { title: string; desc: string }> = {
  "apps.manage": { title: t("Gérer les applications"), desc: t("Installer, démarrer, arrêter et configurer les apps.") },
  "files.read": { title: t("Voir les fichiers"), desc: t("Parcourir et télécharger les fichiers de /DATA.") },
  "files.write": { title: t("Modifier les fichiers"), desc: t("Envoyer, renommer et supprimer des fichiers.") },
};

export function Users() {
  const { status } = useSession();
  const { toast, confirm } = useUI();
  const data = usePoll(() => api.get<{ users: User[]; perms: Perm[] }>("/api/users"));
  const [edit, setEdit] = useState<User | "new" | null>(null);
  const [reset, setReset] = useState<User | null>(null);

  if (data.error && !data.data) return <ErrorBox error={data.error} onRetry={data.reload} />;
  return (
    <>
      <PageHead title={t("Utilisateurs")} sub={t("Invite ta famille ou tes colocs, et choisis précisément ce que chacun peut faire.")}
        actions={<Button variant="primary" icon={<UserPlus />} onClick={() => setEdit("new")}>{t("Ajouter quelqu'un")}</Button>} />
      {!data.data ? <Loading /> : (
        <div className="grid grid-3">
          {data.data.users.map((u) => (
            <Card key={u.username}>
              <div className="row" style={{ gap: 14, flexWrap: "nowrap" }}>
                <Avatar name={u.display} size={48} />
                <div style={{ minWidth: 0, flex: 1 }}>
                  <div className="row" style={{ gap: 8 }}><b style={{ fontSize: 16 }}>{u.display}</b>{u.username === status.user?.username && <Badge mood="brand">{t("Toi")}</Badge>}</div>
                  <div className="muted small">@{u.username} · {u.lastSeen ? `vu ${ago(u.lastSeen)}` : t("jamais connecté")}</div>
                </div>
              </div>
              <div className="row" style={{ gap: 6, margin: "16px 0" }}>
                {u.role === "admin" ? <Badge mood="warn" icon={<Crown />}>{t("Administrateur")}</Badge> : (
                  u.perms.length ? u.perms.map((p) => <Badge key={p}>{permInfo[p]?.title ?? p}</Badge>) : <Badge>{t("Accès au tableau de bord")}</Badge>
                )}
                {u.totp && <Badge mood="good" icon={<ShieldCheck />}>{t("Double authentification")}</Badge>}
              </div>
              <div className="row" style={{ gap: 6 }}>
                <Button size="sm" icon={<Pencil />} onClick={() => setEdit(u)}>{t("Modifier")}</Button>
                <Button size="sm" variant="ghost" icon={<KeyRound />} onClick={() => setReset(u)}>{t("Mot de passe")}</Button>
                <div className="spacer" />
                {u.totp && u.username !== status.user?.username && (
                  <button className="icon-btn" aria-label={t("Réinitialiser la double authentification")} title={t("Réinitialiser la double authentification")} onClick={async () => {
                    if (await confirm({ title: t("Réinitialiser la double authentification de {0} ?", u.display), confirm: t("Réinitialiser"), body: t("À faire si cette personne a perdu son téléphone et ses codes de secours. Elle pourra se connecter avec son seul mot de passe, puis réactiver la protection.") })) {
                      try { await api.del(`/api/users/${q(u.username)}/2fa`); toast(t("Double authentification réinitialisée"), "good"); data.reload(); } catch (e) { toast((e as Error).message, "bad"); }
                    }
                  }}><ShieldOff /></button>
                )}
                {u.username !== status.user?.username && (
                  <button className="icon-btn" aria-label={t("Supprimer")} onClick={async () => {
                    if (await confirm({ title: t("Supprimer le compte de {0} ?", u.display), danger: true, confirm: t("Supprimer"), body: t("Cette personne ne pourra plus se connecter. Ses fichiers dans /DATA ne sont pas supprimés.") })) {
                      try { await api.del(`/api/users/${q(u.username)}`); toast(t("Compte supprimé"), "good"); data.reload(); } catch (e) { toast((e as Error).message, "bad"); }
                    }
                  }}><Trash2 /></button>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}
      {edit && <UserDialog user={edit === "new" ? null : edit} onClose={() => setEdit(null)} onSaved={data.reload} />}
      {reset && <ResetDialog user={reset} onClose={() => setReset(null)} />}
    </>
  );
}

function UserDialog({ user, onClose, onSaved }: { user: User | null; onClose: () => void; onSaved: () => void }) {
  const { toast } = useUI();
  const [display, setDisplay] = useState(user?.display ?? "");
  const [username, setUsername] = useState(user?.username ?? "");
  const [password, setPassword] = useState("");
  const [admin, setAdmin] = useState(user?.role === "admin");
  const [perms, setPerms] = useState<string[]>(user?.role === "member" ? user.perms : ["files.read"]);
  const [busy, setBusy] = useState(false);
  const toggle = (p: string, on: boolean) => setPerms(on ? [...perms, p] : perms.filter((x) => x !== p));

  const save = async () => {
    setBusy(true);
    const role = admin ? "admin" : "member";
    try {
      if (user) await api.put(`/api/users/${q(user.username)}`, { display, role, perms });
      else await api.post("/api/users", { username, display, password, role, perms });
      toast(user ? t("Modifications enregistrées") : t("Bienvenue à {0} ! 🎉", display), "good");
      onSaved();
      onClose();
    } catch (e) { toast((e as Error).message, "bad"); } finally { setBusy(false); }
  };

  return (
    <Modal title={user ? t("Modifier {0}", user.display) : t("Ajouter quelqu'un")} onClose={onClose}
      footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="primary" loading={busy} disabled={!display || (!user && (!username || password.length < 8))} onClick={save}>{user ? t("Enregistrer") : t("Créer le compte")}</Button></>}>
      <Field label={t("Prénom ou surnom")}><Input value={display} onChange={(e) => { setDisplay(e.target.value); if (!user) setUsername(slug(e.target.value).replace(/-/g, "")); }} autoFocus placeholder={"Camille"} /></Field>
      {!user && (
        <>
          <Field label={t("Nom d'utilisateur")}><Input value={username} onChange={(e) => setUsername(e.target.value.toLowerCase())} autoCapitalize="none" /></Field>
          <Field label={t("Mot de passe")} hint={<PasswordMeter value={password} />}><PasswordInput value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" /></Field>
        </>
      )}
      <ToggleRow title={t("Administrateur")} desc={t("Accès complet : système, réseau, utilisateurs, disques…")} checked={admin} onChange={setAdmin} />
      {!admin && Object.entries(permInfo).map(([p, i]) => (
        <ToggleRow key={p} title={i.title} desc={i.desc} checked={perms.includes(p)} onChange={(v) => toggle(p, v)} />
      ))}
    </Modal>
  );
}

function ResetDialog({ user, onClose }: { user: User; onClose: () => void }) {
  const { toast } = useUI();
  const [pw, setPw] = useState("");
  return (
    <Modal title={t("Nouveau mot de passe pour {0}", user.display)} onClose={onClose}
      footer={<><Button variant="ghost" onClick={onClose}>{t("Annuler")}</Button><Button variant="primary" disabled={pw.length < 8} onClick={async () => {
        try { await api.post(`/api/users/${q(user.username)}/password`, { password: pw }); toast(t("Mot de passe changé. Pense à le lui transmettre !"), "good"); onClose(); } catch (e) { toast((e as Error).message, "bad"); }
      }}>{t("Changer")}</Button></>}>
      <Field label={t("Nouveau mot de passe")} hint={<PasswordMeter value={pw} />}><PasswordInput value={pw} onChange={(e) => setPw(e.target.value)} autoFocus /></Field>
      <p className="small muted">{t("Ses sessions ouvertes seront fermées.")}</p>
    </Modal>
  );
}
