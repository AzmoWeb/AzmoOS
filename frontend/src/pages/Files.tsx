import { useEffect, useRef, useState, type DragEvent } from "react";
import { useSearchParams } from "react-router-dom";
import { Folder, File, FileImage, FileVideo, FileAudio, FileText, FileArchive, Home, ChevronRight, MoreVertical, FolderPlus, Upload, Download, Eye, Pencil, Trash2, FolderInput } from "lucide-react";
import { api, q } from "../api/client";
import type { FileEntry } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useSession } from "../session";
import { useMenu, useUI } from "../components/overlay";
import { Button, Card, Empty, Loading, PageHead, Progress } from "../components/ui";
import { ago, bytes } from "../lib/format";

const ext = (n: string) => n.split(".").pop()?.toLowerCase() ?? "";
function FileIcon({ f }: { f: FileEntry }) {
  if (f.dir) return <Folder style={{ color: "#f6b44b" }} fill="rgba(246,180,75,.25)" />;
  const e = ext(f.name);
  if (["jpg", "jpeg", "png", "gif", "webp", "heic", "svg"].includes(e)) return <FileImage style={{ color: "#5fd39a" }} />;
  if (["mp4", "mkv", "avi", "mov", "webm"].includes(e)) return <FileVideo style={{ color: "#ff8a65" }} />;
  if (["mp3", "flac", "wav", "ogg", "m4a"].includes(e)) return <FileAudio style={{ color: "#c07cf0" }} />;
  if (["zip", "gz", "tar", "7z", "rar", "xz"].includes(e)) return <FileArchive style={{ color: "#958ca8" }} />;
  if (["txt", "md", "pdf", "doc", "docx", "odt", "yml", "yaml", "json", "conf"].includes(e)) return <FileText style={{ color: "#8b7bff" }} />;
  return <File style={{ color: "var(--muted)" }} />;
}

const shortcuts = [
  { emoji: "🎬", label: "Films", path: "/Media/Movies" },
  { emoji: "📺", label: "Séries", path: "/Media/TV" },
  { emoji: "🎵", label: "Musique", path: "/Media/Music" },
  { emoji: "📷", label: "Photos", path: "/Photos" },
  { emoji: "📄", label: "Documents", path: "/Documents" },
  { emoji: "⬇️", label: "Téléchargements", path: "/Downloads" },
  { emoji: "💽", label: "Disques", path: "/Disks" },
];

export function Files() {
  const { can } = useSession();
  const { toast, confirm, prompt } = useUI();
  const menu = useMenu();
  const [params, setParams] = useSearchParams();
  const cur = params.get("p") || "/";
  const go = (p: string) => setParams(p === "/" ? {} : { p });
  const list = usePoll(() => api.get<FileEntry[]>(`/api/files?path=${q(cur)}`), 0, [cur]);
  const [drag, setDrag] = useState(false);
  const [upload, setUpload] = useState<{ name: string; pct: number } | null>(null);
  const input = useRef<HTMLInputElement>(null);
  const writable = can("files.write");
  const parts = cur.split("/").filter(Boolean);

  useEffect(() => { setDrag(false); }, [cur]);

  const doUpload = (files: FileList | File[]) => {
    const arr = [...files];
    if (!arr.length || !writable) return;
    const fd = new FormData();
    arr.forEach((f) => fd.append("files", f));
    const xhr = new XMLHttpRequest();
    xhr.open("POST", `/api/files/upload?path=${q(cur)}`);
    xhr.setRequestHeader("X-Requested-With", "AzmoOS");
    const label = arr.length > 1 ? `${arr.length} fichiers` : arr[0].name;
    setUpload({ name: label, pct: 0 });
    xhr.upload.onprogress = (e) => e.lengthComputable && setUpload({ name: label, pct: (e.loaded / e.total) * 100 });
    xhr.onload = () => {
      setUpload(null);
      if (xhr.status < 300) { toast(`${label} : envoyé ✨`, "good"); list.reload(); }
      else toast(JSON.parse(xhr.responseText || "{}").error ?? "L'envoi a échoué", "bad");
    };
    xhr.onerror = () => { setUpload(null); toast("L'envoi a été interrompu", "bad"); };
    xhr.send(fd);
  };

  const op = async (name: string, body: object) => {
    try { await api.post(`/api/files/${name}`, body); list.reload(); } catch (e) { toast((e as Error).message, "bad"); }
  };

  const openMenu = (el: HTMLElement, f: FileEntry) => {
    const dl = (inline: boolean) => `/api/files/download?path=${q(f.path)}${inline ? "&inline=1" : ""}`;
    menu.open(el, [
      ...(f.dir ? [{ label: "Ouvrir", icon: <Folder />, onClick: () => go(f.path) }] : [
        { label: "Aperçu", icon: <Eye />, onClick: () => window.open(dl(true), "_blank", "noopener") },
        { label: "Télécharger", icon: <Download />, onClick: () => { const a = document.createElement("a"); a.href = dl(false); a.click(); } },
      ]),
      ...(writable ? [
        { label: "Renommer", icon: <Pencil />, onClick: async () => { const n = await prompt({ title: "Renommer", label: "Nouveau nom", value: f.name, confirm: "Renommer" }); if (n && n !== f.name) op("rename", { path: f.path, name: n }); } },
        { label: "Déplacer vers…", icon: <FolderInput />, onClick: async () => { const to = await prompt({ title: `Déplacer « ${f.name} »`, label: "Dossier de destination", value: cur, confirm: "Déplacer", placeholder: "/Documents" }); if (to) op("move", { path: f.path, to }); } },
        "-" as const,
        { label: "Supprimer", icon: <Trash2 />, danger: true, onClick: async () => {
          if (await confirm({ title: `Supprimer « ${f.name} » ?`, danger: true, confirm: "Supprimer", body: f.dir ? "Le dossier et tout ce qu'il contient seront supprimés définitivement." : "Le fichier sera supprimé définitivement." })) op("delete", { path: f.path });
        } },
      ] : []),
    ]);
  };

  const onDrop = (e: DragEvent) => { e.preventDefault(); setDrag(false); doUpload(e.dataTransfer.files); };

  return (
    <>
      <PageHead title="Fichiers" sub="Tout ce qui est stocké sur ton Pi, dans /DATA. Glisse-dépose des fichiers ici pour les envoyer."
        actions={writable && <>
          <Button icon={<FolderPlus />} onClick={async () => { const n = await prompt({ title: "Nouveau dossier", label: "Nom du dossier", confirm: "Créer" }); if (n) op("mkdir", { path: cur, name: n }); }}>Nouveau dossier</Button>
          <Button variant="primary" icon={<Upload />} onClick={() => input.current?.click()}>Envoyer</Button>
          <input ref={input} type="file" multiple hidden onChange={(e) => { if (e.target.files) doUpload(e.target.files); e.target.value = ""; }} />
        </>} />
      {cur === "/" && (
        <div className="quick">
          {shortcuts.map((s) => <button key={s.path} onClick={() => go(s.path)}><span>{s.emoji}</span>{s.label}</button>)}
        </div>
      )}
      {upload && <Card flat className="" ><div className="row small" style={{ marginBottom: 8 }}><Upload size={16} /> Envoi de <b>{upload.name}</b> <span className="spacer" />{upload.pct.toFixed(0)}%</div><Progress value={upload.pct} /></Card>}
      <div className={`card pad-0 ${drag ? "dropzone" : ""}`} style={{ marginTop: upload ? 14 : 0 }}
        onDragOver={(e) => { e.preventDefault(); if (writable) setDrag(true); }} onDragLeave={() => setDrag(false)} onDrop={onDrop}>
        <div style={{ padding: "12px 14px", borderBottom: "1px solid var(--border)" }} className="crumbs">
          <button onClick={() => go("/")}><Home /> DATA</button>
          {parts.map((p, i) => (
            <span key={i} className="row" style={{ gap: 2 }}><ChevronRight size={14} className="muted" /><button onClick={() => go("/" + parts.slice(0, i + 1).join("/"))}>{p}</button></span>
          ))}
        </div>
        {list.error ? <Empty emoji="🚧" title="Impossible d'ouvrir ce dossier">{list.error}</Empty> : !list.data ? <Loading /> : list.data.length === 0 ? (
          <Empty emoji="📭" title="Ce dossier est vide">{writable ? "Glisse des fichiers ici, ou utilise le bouton Envoyer." : "Il n'y a rien ici pour le moment."}</Empty>
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr><th>Nom</th><th className="hide-sm">Taille</th><th className="hide-sm">Modifié</th><th /></tr></thead>
              <tbody>
                {list.data.map((f) => (
                  <tr key={f.path} className="clickable" onClick={(e) => {
                    const btn = (e.target as HTMLElement).closest(".icon-btn") as HTMLElement | null;
                    if (!btn && f.dir) return go(f.path);
                    openMenu(btn ?? (e.currentTarget as HTMLElement), f);
                  }}>
                    <td><div className="file-name"><FileIcon f={f} />{f.name}</div></td>
                    <td className="hide-sm muted">{f.dir ? "—" : bytes(f.size)}</td>
                    <td className="hide-sm muted">{ago(new Date(f.modTime).getTime() / 1000)}</td>
                    <td className="actions"><button className="icon-btn" aria-label="Actions"><MoreVertical /></button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
      {menu.node}
    </>
  );
}
