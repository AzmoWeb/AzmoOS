import { useState, type ReactElement } from "react";
import { LogIn, Package, Settings, HardDrive, Wifi, ShieldAlert, FileText, Info } from "lucide-react";
import { api } from "../api/client";
import type { AzmoEvent } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { Card, Empty, ErrorBox, Loading, PageHead } from "../components/ui";
import { ago, dateTime } from "../lib/format";
import { t, tServer } from "../lib/i18n";

const cats: { id: string; label: string; icon: ReactElement }[] = [
  { id: "", label: t("Tout"), icon: <Info /> },
  { id: "auth", label: t("Connexions"), icon: <LogIn /> },
  { id: "security", label: t("Sécurité"), icon: <ShieldAlert /> },
  { id: "apps", label: t("Applications"), icon: <Package /> },
  { id: "storage", label: t("Stockage"), icon: <HardDrive /> },
  { id: "network", label: t("Réseau"), icon: <Wifi /> },
  { id: "files", label: t("Fichiers"), icon: <FileText /> },
  { id: "system", label: t("Système"), icon: <Settings /> },
];
const iconFor = (c: string) => cats.find((x) => x.id === c)?.icon ?? <Info />;

export function Activity() {
  const [cat, setCat] = useState("");
  const ev = usePoll(() => api.get<AzmoEvent[]>(`/api/events?limit=400${cat ? `&category=${cat}` : ""}`), 10000, [cat]);
  return (
    <>
      <PageHead title={t("Journal")} sub={t("Tout ce qui s'est passé sur ton serveur : qui s'est connecté, ce qui a été installé, modifié…")} />
      <div className="chips">{cats.map((c) => <button key={c.id} className={`chip ${cat === c.id ? "active" : ""}`} onClick={() => setCat(c.id)}>{c.label}</button>)}</div>
      {ev.error && !ev.data ? <ErrorBox error={ev.error} onRetry={ev.reload} /> : !ev.data ? <Loading /> : ev.data.length === 0 ? (
        <Card><Empty emoji="🍃" title={t("Rien à signaler")}>{t("Aucun événement dans cette catégorie pour l'instant.")}</Empty></Card>
      ) : (
        <div className="card pad-0">
          <ul className="timeline">
            {ev.data.map((e, i) => (
              <li key={i}>
                <div className={`tl-icon ${e.level}`}>{iconFor(e.category)}</div>
                <div className="tl-body">
                  <b>{tServer(e.message)}</b>
                  <div className="tl-meta" title={dateTime(e.time)}>{ago(e.time)}{e.user ? ` · ${e.user}` : ""}{e.ip ? ` · ${e.ip}` : ""}</div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  );
}
