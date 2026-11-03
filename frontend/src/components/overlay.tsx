import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X, CheckCircle2, XCircle, Info } from "lucide-react";
import { Button, Field, Input, IconButton } from "./ui";

/* ---------- Modale ---------- */

export function Modal({ title, sub, icon, onClose, children, footer, wide }: {
  title: ReactNode; sub?: ReactNode; icon?: ReactNode; onClose: () => void; children?: ReactNode; footer?: ReactNode; wide?: boolean;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => { window.removeEventListener("keydown", onKey); document.body.style.overflow = prev; };
  }, [onClose]);
  return createPortal(
    <div className="overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className={`modal ${wide ? "wide" : ""}`} role="dialog" aria-modal="true">
        <div className="modal-head">
          {icon}
          <div><h2>{title}</h2>{sub && <p>{sub}</p>}</div>
          <IconButton label="Fermer" onClick={onClose}><X /></IconButton>
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-foot">{footer}</div>}
      </div>
    </div>,
    document.body,
  );
}

/* ---------- Menu contextuel ---------- */

export type MenuItem = { label: string; icon?: ReactNode; danger?: boolean; onClick: () => void } | "-";

export function Menu({ anchor, items, onClose }: { anchor: HTMLElement; items: MenuItem[]; onClose: () => void }) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ left: -9999, top: -9999 });
  useLayoutEffect(() => {
    const r = anchor.getBoundingClientRect();
    const m = ref.current!;
    const w = m.offsetWidth, h = m.offsetHeight;
    setPos({
      left: Math.max(8, Math.min(r.right - w, window.innerWidth - w - 8)),
      top: r.bottom + h + 8 > window.innerHeight ? Math.max(8, r.top - h - 6) : r.bottom + 6,
    });
  }, [anchor]);
  useEffect(() => {
    const close = (e: Event) => { if (!ref.current?.contains(e.target as Node)) onClose(); };
    const key = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    setTimeout(() => document.addEventListener("mousedown", close), 0);
    window.addEventListener("keydown", key);
    window.addEventListener("scroll", onClose, true);
    return () => { document.removeEventListener("mousedown", close); window.removeEventListener("keydown", key); window.removeEventListener("scroll", onClose, true); };
  }, [onClose]);
  return createPortal(
    <div className="menu" ref={ref} style={pos} role="menu">
      {items.map((it, i) =>
        it === "-" ? <hr key={i} /> : (
          <button key={i} role="menuitem" className={it.danger ? "danger" : ""} onClick={() => { onClose(); it.onClick(); }}>
            {it.icon}{it.label}
          </button>
        ),
      )}
    </div>,
    document.body,
  );
}

/** Hook pratique : const menu = useMenu(); menu.open(e.currentTarget, items); {menu.node} */
export function useMenu() {
  const [state, setState] = useState<{ anchor: HTMLElement; items: MenuItem[] } | null>(null);
  const close = useCallback(() => setState(null), []);
  return {
    open: (anchor: HTMLElement, items: MenuItem[]) => setState({ anchor, items }),
    node: state ? <Menu anchor={state.anchor} items={state.items} onClose={close} /> : null,
  };
}

/* ---------- Notifications & dialogues ---------- */

type ToastKind = "good" | "bad" | "info";
interface Toast { id: number; kind: ToastKind; text: string }

interface ConfirmOpts { title: string; body?: ReactNode; confirm?: string; danger?: boolean; typeToConfirm?: string }
interface PromptOpts { title: string; label: string; value?: string; confirm?: string; placeholder?: string }

interface UICtx {
  toast: (text: string, kind?: ToastKind) => void;
  confirm: (o: ConfirmOpts) => Promise<boolean>;
  prompt: (o: PromptOpts) => Promise<string | null>;
}

const Ctx = createContext<UICtx>(null as unknown as UICtx);
export const useUI = () => useContext(Ctx);

let nextId = 1;

export function UIProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [dialog, setDialog] = useState<ReactNode>(null);

  const toast = useCallback((text: string, kind: ToastKind = "info") => {
    const id = nextId++;
    setToasts((t) => [...t, { id, kind, text }]);
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), kind === "bad" ? 6500 : 4200);
  }, []);

  const confirm = useCallback((o: ConfirmOpts) => new Promise<boolean>((resolve) => {
    const done = (v: boolean) => { setDialog(null); resolve(v); };
    setDialog(<ConfirmDialog o={o} done={done} />);
  }), []);

  const prompt = useCallback((o: PromptOpts) => new Promise<string | null>((resolve) => {
    const done = (v: string | null) => { setDialog(null); resolve(v); };
    setDialog(<PromptDialog o={o} done={done} />);
  }), []);

  return (
    <Ctx.Provider value={{ toast, confirm, prompt }}>
      {children}
      {dialog}
      {createPortal(
        <div className="toasts" aria-live="polite">
          {toasts.map((t) => (
            <div key={t.id} className={`toast ${t.kind}`}>
              {t.kind === "good" ? <CheckCircle2 /> : t.kind === "bad" ? <XCircle /> : <Info />}
              <div>{t.text}</div>
            </div>
          ))}
        </div>,
        document.body,
      )}
    </Ctx.Provider>
  );
}

function ConfirmDialog({ o, done }: { o: ConfirmOpts; done: (v: boolean) => void }) {
  const [typed, setTyped] = useState("");
  const blocked = !!o.typeToConfirm && typed !== o.typeToConfirm;
  return (
    <Modal title={o.title} onClose={() => done(false)} footer={<>
      <Button variant="ghost" onClick={() => done(false)}>Annuler</Button>
      <Button variant={o.danger ? "danger-solid" : "primary"} disabled={blocked} onClick={() => done(true)} autoFocus={!o.typeToConfirm}>{o.confirm ?? "Confirmer"}</Button>
    </>}>
      <div className="muted" style={{ color: "var(--text-2)" }}>{o.body}</div>
      {o.typeToConfirm && (
        <div style={{ marginTop: 16 }}>
          <Field label={<>Pour confirmer, tape <code>{o.typeToConfirm}</code></>}>
            <Input value={typed} onChange={(e) => setTyped(e.target.value)} autoFocus />
          </Field>
        </div>
      )}
    </Modal>
  );
}

function PromptDialog({ o, done }: { o: PromptOpts; done: (v: string | null) => void }) {
  const [v, setV] = useState(o.value ?? "");
  return (
    <Modal title={o.title} onClose={() => done(null)}>
      <form onSubmit={(e) => { e.preventDefault(); if (v.trim()) done(v.trim()); }}>
        <Field label={o.label}>
          <Input value={v} onChange={(e) => setV(e.target.value)} placeholder={o.placeholder} autoFocus onFocus={(e) => e.target.select()} />
        </Field>
        <div className="row" style={{ justifyContent: "flex-end" }}>
          <Button type="button" variant="ghost" onClick={() => done(null)}>Annuler</Button>
          <Button variant="primary" disabled={!v.trim()}>{o.confirm ?? "Valider"}</Button>
        </div>
      </form>
    </Modal>
  );
}
