import { useCallback, useEffect, useRef, useState } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { RotateCw, Plus, Minus, ShieldAlert, TerminalSquare } from "lucide-react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { TerminalInfo } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { Badge, Button, Callout, Card, Dot, Empty, Loading, PageHead } from "../components/ui";
import { t } from "../lib/i18n";

const theme = {
  background: "#0d0b12", foreground: "#e8e3f3", cursor: "#ff8a65", cursorAccent: "#0d0b12", selectionBackground: "#8b7bff55",
  black: "#1f1b29", red: "#f2706b", green: "#5fd39a", yellow: "#f6b44b", blue: "#8b7bff", magenta: "#c07cf0", cyan: "#5ccfe6", white: "#e8e3f3",
  brightBlack: "#6f6782", brightRed: "#ff8a85", brightGreen: "#7ee6b0", brightYellow: "#ffd07a", brightBlue: "#a99cff", brightMagenta: "#d6a0ff", brightCyan: "#8be3f2", brightWhite: "#ffffff",
};

type State = "connecting" | "open" | "closed";

// Touches difficiles à taper sur un téléphone.
const keys: { label: string; seq: string }[] = [
  { label: t("Échap"), seq: "\x1b" }, { label: "Tab", seq: "\t" }, { label: "Ctrl+C", seq: "\x03" }, { label: "Ctrl+D", seq: "\x04" },
  { label: "↑", seq: "\x1b[A" }, { label: "↓", seq: "\x1b[B" }, { label: "←", seq: "\x1b[D" }, { label: "→", seq: "\x1b[C" },
];

export function TerminalPage() {
  const info = usePoll(() => api.get<TerminalInfo>("/api/terminal/info"));
  const box = useRef<HTMLDivElement>(null);
  const term = useRef<XTerm | null>(null);
  const ws = useRef<WebSocket | null>(null);
  const fit = useRef<FitAddon | null>(null);
  const [state, setState] = useState<State>("connecting");
  const [size, setSize] = useState(() => {
    try { return Number(localStorage.getItem("azmo-term-size")) || 14; } catch { return 14; }
  });

  const send = useCallback((data: string) => {
    if (ws.current?.readyState === WebSocket.OPEN) ws.current.send("0" + data);
  }, []);

  const connect = useCallback(() => {
    ws.current?.close();
    const xt = term.current!;
    xt.reset();
    setState("connecting");
    const sock = new WebSocket(`${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/api/terminal`);
    sock.binaryType = "arraybuffer";
    ws.current = sock;
    sock.onopen = () => {
      setState("open");
      fit.current?.fit();
      sock.send("1" + JSON.stringify({ cols: xt.cols, rows: xt.rows }));
      xt.focus();
    };
    sock.onmessage = (e) => xt.write(typeof e.data === "string" ? e.data : new Uint8Array(e.data));
    sock.onclose = () => setState("closed");
  }, []);

  useEffect(() => {
    if (!info.data?.enabled || !box.current || term.current) return;
    const xt = new XTerm({
      theme, fontFamily: 'ui-monospace, "SF Mono", "JetBrains Mono", Menlo, Consolas, monospace', fontSize: size,
      cursorBlink: true, scrollback: 5000, allowProposedApi: false,
    });
    const f = new FitAddon();
    xt.loadAddon(f);
    xt.open(box.current);
    f.fit();
    term.current = xt;
    fit.current = f;
    xt.onData((d) => send(d));
    xt.onResize(({ cols, rows }) => {
      if (ws.current?.readyState === WebSocket.OPEN) ws.current.send("1" + JSON.stringify({ cols, rows }));
    });
    const ro = new ResizeObserver(() => f.fit());
    ro.observe(box.current);
    connect();
    return () => { ro.disconnect(); ws.current?.close(); xt.dispose(); term.current = null; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [info.data?.enabled]);

  const zoom = (d: number) => {
    const n = Math.max(10, Math.min(22, size + d));
    setSize(n);
    try { localStorage.setItem("azmo-term-size", String(n)); } catch { /* ignore */ }
    if (term.current) { term.current.options.fontSize = n; fit.current?.fit(); }
  };

  if (!info.data) return <><PageHead title={t("Terminal")} /><Loading /></>;
  if (!info.data.enabled) return (
    <>
      <PageHead title={t("Terminal")} />
      <Card><Empty emoji="🔒" title={t("Le terminal web est désactivé")} action={<Link to="/security" className="btn primary">{t("Ouvrir Sécurité")}</Link>}>{t("Un administrateur peut l'activer depuis la page Sécurité.")}</Empty></Card>
    </>
  );
  const acc = info.data.account;
  return (
    <>
      <PageHead title={t("Terminal")} sub={<>{t("La ligne de commande de ton Pi, directement dans le navigateur. Tu es connecté en tant que")} <b>{acc.name}</b>{acc.root ? "" : t(" (utilise sudo pour les commandes d'administration)")}.</>}
        actions={<>
          <Button size="sm" variant="ghost" onClick={() => zoom(-1)} aria-label={t("Réduire le texte")}><Minus /></Button>
          <Button size="sm" variant="ghost" onClick={() => zoom(1)} aria-label={t("Agrandir le texte")}><Plus /></Button>
          <Button size="sm" icon={<RotateCw />} onClick={connect}>{t("Nouvelle session")}</Button>
        </>} />
      {acc.root && <div style={{ marginBottom: 14 }}><Callout mood="warn" title={t("Attention, tu es root")}>{t("Toutes les commandes ont les pleins pouvoirs sur le système. Une erreur peut tout casser.")}</Callout></div>}
      <div className="card pad-0 terminal-card">
        <div className="terminal-bar">
          <Dot mood={state === "open" ? "good" : state === "connecting" ? "warn" : "bad"} pulse={state === "connecting"} />
          <span className="small">{state === "open" ? t("Connecté") : state === "connecting" ? t("Connexion…") : t("Session terminée")}</span>
          <Badge icon={<TerminalSquare />}>{acc.name}@{location.hostname}</Badge>
          <div className="spacer" />
          {state === "closed" && <Button size="sm" variant="primary" icon={<RotateCw />} onClick={connect}>{t("Reconnecter")}</Button>}
        </div>
        <div ref={box} className="terminal-box" />
        <div className="terminal-keys">
          {keys.map((k) => <button key={k.label} onClick={() => { send(k.seq); term.current?.focus(); }}>{k.label}</button>)}
        </div>
      </div>
      <p className="small muted" style={{ marginTop: 12 }}><ShieldAlert size={13} style={{ verticalAlign: -2 }} /> {t("Chaque ouverture de terminal est enregistrée dans le Journal.")}</p>
    </>
  );
}
