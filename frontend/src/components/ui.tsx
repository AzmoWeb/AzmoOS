import { type ReactNode, type ButtonHTMLAttributes, type InputHTMLAttributes, forwardRef, useState } from "react";
import { Info, AlertTriangle, XCircle, CheckCircle2, Search as SearchIcon } from "lucide-react";
import type { Mood } from "../lib/human";
import { avatarHue } from "../lib/human";
import { t } from "../lib/i18n";

type BtnProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "ghost" | "danger" | "danger-solid";
  size?: "sm" | "lg";
  block?: boolean;
  loading?: boolean;
  icon?: ReactNode;
};

export function Button({ variant, size, block, loading, icon, className = "", children, disabled, ...rest }: BtnProps) {
  const v = variant === "danger-solid" ? "danger solid" : variant ?? "";
  return (
    <button className={`btn ${v} ${size ?? ""} ${block ? "block" : ""} ${className}`} disabled={disabled || loading} {...rest}>
      {loading ? <span className="spinner sm" /> : icon}
      {children}
    </button>
  );
}

export const IconButton = forwardRef<HTMLButtonElement, ButtonHTMLAttributes<HTMLButtonElement> & { label: string }>(function IconButton({ label, children, ...rest }, ref) {
  return (
    <button ref={ref} className="icon-btn" aria-label={label} title={label} {...rest}>
      {children}
    </button>
  );
});

export function Card({ title, icon, right, sub, children, className = "", flat, pad0 }: {
  title?: ReactNode; icon?: ReactNode; right?: ReactNode; sub?: ReactNode; children?: ReactNode; className?: string; flat?: boolean; pad0?: boolean;
}) {
  return (
    <section className={`card ${flat ? "flat" : ""} ${pad0 ? "pad-0" : ""} ${className}`}>
      {title && (
        <div className="card-title" style={pad0 ? { padding: "20px 20px 0" } : undefined}>
          {icon}
          {title}
          {right && <div className="right">{right}</div>}
        </div>
      )}
      {sub && <div className="card-sub" style={pad0 ? { padding: "0 20px" } : undefined}>{sub}</div>}
      {children}
    </section>
  );
}

export function PageHead({ title, sub, actions }: { title: ReactNode; sub?: ReactNode; actions?: ReactNode }) {
  return (
    <header className="page-head">
      <div>
        <h1>{title}</h1>
        {sub && <p>{sub}</p>}
      </div>
      {actions && <div className="actions">{actions}</div>}
    </header>
  );
}

export function Badge({ mood, children, icon }: { mood?: Mood | "brand"; children: ReactNode; icon?: ReactNode }) {
  return <span className={`badge ${mood ?? ""}`}>{icon}{children}</span>;
}

export function Dot({ mood, pulse }: { mood?: Mood; pulse?: boolean }) {
  return <span className={`dot ${mood ?? ""} ${pulse ? "pulse" : ""}`} />;
}

export function Progress({ value, mood, thin }: { value: number; mood?: Mood; thin?: boolean }) {
  return (
    <div className={`progress ${mood && mood !== "good" ? mood : ""} ${thin ? "thin" : ""}`} role="progressbar" aria-valuenow={Math.round(value)} aria-valuemin={0} aria-valuemax={100}>
      <i style={{ width: `${Math.max(0, Math.min(100, value))}%` }} />
    </div>
  );
}

export function Field({ label, hint, children }: { label: ReactNode; hint?: ReactNode; children: ReactNode }) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
      {hint && <small>{hint}</small>}
    </label>
  );
}

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input className="input" {...props} />;
}

export function PasswordInput(props: InputHTMLAttributes<HTMLInputElement>) {
  const [show, setShow] = useState(false);
  return (
    <div style={{ position: "relative" }}>
      <input className="input" {...props} type={show ? "text" : "password"} style={{ paddingRight: 84 }} />
      <button type="button" className="btn ghost sm" style={{ position: "absolute", right: 6, top: 6 }} onClick={() => setShow(!show)}>
        {show ? t("Masquer") : t("Afficher")}
      </button>
    </div>
  );
}

/** Indicateur de robustesse d'un mot de passe (purement indicatif). */
export function PasswordMeter({ value }: { value: string }) {
  let score = 0;
  if (value.length >= 8) score++;
  if (value.length >= 12) score++;
  if (/[A-Z]/.test(value) && /[a-z]/.test(value)) score++;
  if (/\d/.test(value) && /[^A-Za-z0-9]/.test(value)) score++;
  const colors = ["var(--bad)", "var(--warn)", "var(--warn)", "var(--good)", "var(--good)"];
  const labels = [t("Trop court"), t("Faible"), t("Correct"), t("Bon"), t("Excellent")];
  return (
    <>
      <div className="pw-meter">{[0, 1, 2, 3].map((i) => <i key={i} style={{ background: i < score ? colors[score] : undefined }} />)}</div>
      {value && <small style={{ color: colors[score] }}>{labels[score]}</small>}
    </>
  );
}

export function Search({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <div className="search">
      <SearchIcon />
      <input className="input" value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder ?? t("Rechercher…")} />
    </div>
  );
}

export function Switch({ checked, onChange, disabled, label }: { checked: boolean; onChange: (v: boolean) => void; disabled?: boolean; label?: string }) {
  return (
    <label className="switch" aria-label={label}>
      <input type="checkbox" checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
      <i />
    </label>
  );
}

export function ToggleRow({ title, desc, checked, onChange, disabled }: { title: ReactNode; desc?: ReactNode; checked: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  return (
    <div className="toggle-row">
      <div className="txt"><b>{title}</b>{desc && <span>{desc}</span>}</div>
      <Switch checked={checked} onChange={onChange} disabled={disabled} label={typeof title === "string" ? title : undefined} />
    </div>
  );
}

export function Tabs<T extends string>({ tabs, value, onChange }: { tabs: { id: T; label: string; icon?: ReactNode }[]; value: T; onChange: (v: T) => void }) {
  return (
    <div className="tabs" role="tablist">
      {tabs.map((tab) => (
        <button key={tab.id} role="tab" aria-selected={value === tab.id} className={`tab ${value === tab.id ? "active" : ""}`} onClick={() => onChange(tab.id)}>
          {tab.icon}{tab.label}
        </button>
      ))}
    </div>
  );
}

export function Spinner({ small }: { small?: boolean }) {
  return <span className={`spinner ${small ? "sm" : ""}`} />;
}

export function Loading({ label = t("Chargement…") }: { label?: string }) {
  return (
    <div className="center">
      <div className="row muted"><Spinner /> {label}</div>
    </div>
  );
}

export function Empty({ emoji, title, children, action }: { emoji: string; title: string; children?: ReactNode; action?: ReactNode }) {
  return (
    <div className="empty">
      <div className="emoji">{emoji}</div>
      <h3>{title}</h3>
      {children && <p>{children}</p>}
      {action}
    </div>
  );
}

export function Callout({ mood, children, title }: { mood?: Mood | "info"; children: ReactNode; title?: ReactNode }) {
  const Icon = mood === "bad" ? XCircle : mood === "warn" ? AlertTriangle : mood === "good" ? CheckCircle2 : Info;
  return (
    <div className={`callout ${mood && mood !== "info" ? mood : ""}`}>
      <Icon />
      <div>{title && <b style={{ display: "block", marginBottom: 2 }}>{title}</b>}{children}</div>
    </div>
  );
}

export function ErrorBox({ error, onRetry }: { error: string; onRetry?: () => void }) {
  return (
    <Card>
      <Empty emoji="😕" title={t("Oups, ça n'a pas marché")} action={onRetry && <Button onClick={onRetry}>{t("Réessayer")}</Button>}>
        {error}
      </Empty>
    </Card>
  );
}

export function Avatar({ name, size = 36 }: { name: string; size?: number }) {
  const hue = avatarHue(name);
  return (
    <div className="avatar" style={{ width: size, height: size, fontSize: size * 0.42, background: `linear-gradient(135deg, hsl(${hue} 70% 62%), hsl(${(hue + 40) % 360} 75% 58%))` }}>
      {(name.trim()[0] ?? "?").toUpperCase()}
    </div>
  );
}
