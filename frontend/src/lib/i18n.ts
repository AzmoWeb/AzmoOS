// Traductions d'Azmo OS. Le français est la langue source : le texte français sert de clé,
// ce qui garde le code lisible. Les traductions sont dans src/i18n/<langue>.ts.
// Les variables s'écrivent {0}, {1}… : t("{0} applications", n).
import { en } from "../i18n/en";

export type Lang = "fr" | "en";
export const languages: { id: Lang; label: string; flag: string }[] = [
  { id: "fr", label: "Français", flag: "🇫🇷" },
  { id: "en", label: "English", flag: "🇬🇧" },
];

const dictionaries: Record<Lang, Record<string, string> | null> = { fr: null, en };

function detect(): Lang {
  try {
    const saved = localStorage.getItem("azmo-lang");
    if (saved === "fr" || saved === "en") return saved;
  } catch { /* stockage indisponible */ }
  const nav = (navigator.languages?.[0] ?? navigator.language ?? "fr").toLowerCase();
  return nav.startsWith("fr") ? "fr" : "en";
}

let current: Lang = detect();
document.documentElement.lang = current;

export const lang = () => current;
/** Locale BCP 47 pour Intl (dates, nombres). */
export const locale = () => (current === "fr" ? "fr-FR" : "en-GB");

/** Change de langue et recharge l'interface. */
export function setLang(l: Lang) {
  try { localStorage.setItem("azmo-lang", l); } catch { /* navigation privée */ }
  if (l !== current) window.location.reload();
}

/** Applique la langue enregistrée dans le compte, si l'utilisateur n'en a pas choisi une sur cet appareil. */
export function adoptAccountLang(l?: string) {
  if (l !== "fr" && l !== "en") return;
  let saved: string | null = null;
  try { saved = localStorage.getItem("azmo-lang"); } catch { /* ignore */ }
  if (!saved && l !== current) setLang(l);
}

/** Traduit un texte français. */
export function t(key: string, ...args: unknown[]): string {
  const dict = dictionaries[current];
  let s = (dict && dict[key]) || key;
  if (args.length) s = s.replace(/\{(\d+)\}/g, (m, i) => (args[+i] !== undefined ? String(args[+i]) : m));
  return s;
}

/** Traduit un message venant du serveur (erreurs), s'il est connu. */
export function tServer(msg: string): string {
  const dict = dictionaries[current];
  if (!dict) return msg;
  return dict[msg] ?? dict[msg.charAt(0).toLowerCase() + msg.slice(1)] ?? msg;
}

/** Texte d'une application du catalogue dans la langue courante (repli : français). */
export function appText<T extends { i18n?: Record<string, Partial<Record<string, string>>> }>(app: T, field: "tagline" | "description" | "notes", fallback?: string): string {
  return app.i18n?.[current]?.[field] ?? fallback ?? "";
}
