import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Download, Check, Cpu, MemoryStick, Globe, ChevronDown } from "lucide-react";
import { api } from "../api/client";
import type { CatalogApp } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useJobs, useSession } from "../session";
import { useUI, Modal } from "../components/overlay";
import { AppIcon } from "../components/AppIcon";
import { Badge, Button, Callout, Card, Empty, ErrorBox, Loading, PageHead, Search } from "../components/ui";

const categoryEmoji: Record<string, string> = {
  "Multimédia": "🎬", Cloud: "☁️", Domotique: "🏡", "Réseau": "🌐", "Sécurité": "🔒", Outils: "🧰",
  "Développement": "💻", "Téléchargement": "⬇️", Sauvegarde: "💾",
};

export function StorePage() {
  const { status } = useSession();
  const cat = usePoll(() => api.get<CatalogApp[]>("/api/store/catalog"));
  const [filter, setFilter] = useState("Tout");
  const [query, setQuery] = useState("");
  const [sel, setSel] = useState<CatalogApp | null>(null);

  const categories = useMemo(() => ["Tout", ...new Set((cat.data ?? []).map((a) => a.category))], [cat.data]);
  const list = useMemo(() => {
    const qq = query.toLowerCase().trim();
    return (cat.data ?? []).filter((a) => (filter === "Tout" || a.category === filter) && (!qq || `${a.name} ${a.tagline} ${a.description} ${a.category}`.toLowerCase().includes(qq)));
  }, [cat.data, filter, query]);

  return (
    <>
      <PageHead title="App Store" sub="Des applications prêtes à l'emploi, installées en un clic sur ton Pi." actions={<Search value={query} onChange={setQuery} placeholder="Jellyfin, photos, VPN…" />} />
      {filter === "Tout" && !query && (
        <div className="store-hero">
          <h2>Fais de ton Pi le centre de ta maison</h2>
          <p>Films, photos, domotique, mots de passe, VPN… Chaque application tourne dans son propre espace isolé : rien ne casse, et tu peux tout désinstaller sans laisser de traces.</p>
        </div>
      )}
      {status.hardware?.profile === "leger" && (
        <div style={{ marginBottom: 18 }}><Callout mood="warn" title="Ton Pi a peu de mémoire">{status.hardware.profileHint}</Callout></div>
      )}
      <div className="chips">
        {categories.map((c) => (
          <button key={c} className={`chip ${c === filter ? "active" : ""}`} onClick={() => setFilter(c)}>{categoryEmoji[c] ?? "✨"} {c}</button>
        ))}
      </div>
      {cat.error ? <ErrorBox error={cat.error} onRetry={cat.reload} /> : !cat.data ? <Loading /> : list.length === 0 ? (
        <Card><Empty emoji="🔍" title="Aucune application trouvée">Essaie un autre mot, ou ajoute ta propre application depuis « Mes applications ».</Empty></Card>
      ) : (
        <div className="store-grid">
          {list.map((a, i) => (
            <div key={a.id} className="store-card" onClick={() => setSel(a)} style={{ animationDelay: `${Math.min(i, 12) * 25}ms`, opacity: a.compatible ? 1 : 0.6 }}>
              <AppIcon name={a.name} icon={a.icon} size={56} />
              <div style={{ minWidth: 0 }}>
                <h4>{a.name}</h4>
                <p>{a.tagline}</p>
                <div className="row" style={{ gap: 6 }}>
                  <Badge>{a.category}</Badge>
                  {a.installed && <Badge mood="good" icon={<Check />}>Installée</Badge>}
                  {!a.installed && a.warning && <Badge mood={a.compatible ? "warn" : "bad"}>{a.compatible ? "Gourmande" : "Incompatible"}</Badge>}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
      {sel && <StoreDetail app={sel} onClose={() => setSel(null)} onInstalled={cat.reload} />}
    </>
  );
}

function StoreDetail({ app, onClose, onInstalled }: { app: CatalogApp; onClose: () => void; onInstalled: () => void }) {
  const { toast } = useUI();
  const { kick } = useJobs();
  const nav = useNavigate();
  const [compose, setCompose] = useState(app.compose);
  const [advanced, setAdvanced] = useState(false);
  const [force, setForce] = useState(false);
  const [busy, setBusy] = useState(false);

  const install = async () => {
    setBusy(true);
    try {
      await api.post(`/api/apps/${app.id}/install`, { compose });
      toast(`C'est parti ! ${app.name} s'installe. Le téléchargement peut prendre quelques minutes.`);
      kick();
      onInstalled();
      nav("/apps");
    } catch (e) { toast((e as Error).message, "bad"); setBusy(false); }
  };

  return (
    <Modal wide title={app.name} sub={app.tagline} icon={<AppIcon name={app.name} icon={app.icon} size={56} />} onClose={onClose}
      footer={<>
        <Button variant="ghost" onClick={onClose}>Fermer</Button>
        {app.installed ? <Button onClick={() => nav("/apps")} icon={<Check />}>Déjà installée</Button>
          : <Button variant="primary" icon={<Download />} loading={busy} disabled={!app.compatible && !force} onClick={install}>Installer {app.name}</Button>}
      </>}>
      <p style={{ fontSize: 15.5, color: "var(--text-2)", marginBottom: 18 }}>{app.description}</p>
      <div className="row" style={{ marginBottom: 18 }}>
        <Badge icon={<MemoryStick />}>{app.minRam ? (app.minRam >= 1024 ? `${app.minRam / 1024} Go de RAM conseillés` : `${app.minRam} Mo de RAM`) : "Légère"}</Badge>
        <Badge icon={<Cpu />}>{(app.arch ?? []).join(" · ") || "toutes architectures"}</Badge>
        {app.port > 0 && <Badge icon={<Globe />}>Port {app.port}</Badge>}
      </div>
      <div className="stack" style={{ gap: 12 }}>
        {app.warning && (
          <Callout mood={app.compatible ? "warn" : "bad"} title={app.compatible ? "À savoir avant d'installer" : "Probablement pas adaptée à ton Pi"}>
            {app.warning}
            {!app.compatible && (
              <label className="row small" style={{ marginTop: 8, gap: 8 }}><input type="checkbox" checked={force} onChange={(e) => setForce(e.target.checked)} /> J'ai compris, installer quand même</label>
            )}
          </Callout>
        )}
        {app.notes && <Callout title="Bon à savoir">{app.notes}</Callout>}
      </div>
      <button className="btn ghost sm" style={{ marginTop: 16 }} onClick={() => setAdvanced(!advanced)}>
        <ChevronDown style={{ transform: advanced ? "rotate(180deg)" : undefined, transition: "0.2s" }} /> Configuration avancée
      </button>
      {advanced && (
        <div style={{ marginTop: 10 }}>
          <p className="small muted" style={{ marginBottom: 8 }}>Tu peux modifier les ports, dossiers ou mots de passe avant l'installation.</p>
          <textarea className="textarea" value={compose} onChange={(e) => setCompose(e.target.value)} spellCheck={false} />
        </div>
      )}
    </Modal>
  );
}
