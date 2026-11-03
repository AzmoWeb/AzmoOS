// Messages « humains » : on parle à la personne, pas à un administrateur système.
import type { Hardware, SystemStats } from "../api/types";
import { t } from "./i18n";

export function greeting(name: string): string {
  const h = new Date().getHours();
  const first = name.split(" ")[0];
  if (h < 5) return t("Encore debout, {0} ? 🌙", first);
  if (h < 12) return t("Bonjour {0} ☀️", first);
  if (h < 18) return t("Bon après-midi {0} 👋", first);
  return t("Bonsoir {0} 🌆", first);
}

export type Mood = "good" | "warn" | "bad";

export interface Health {
  mood: Mood;
  title: string;
  detail: string;
}

/** Résume l'état du Pi en une phrase. */
export function health(s: SystemStats): Health {
  const mem = s.memTotal ? (s.memUsed / s.memTotal) * 100 : 0;
  const disk = s.disks[0] ? (s.disks[0].used / s.disks[0].total) * 100 : 0;
  const throttled = s.throttled && s.throttled !== "0x0";
  if (s.temp >= 80 || throttled)
    return { mood: "bad", title: t("Ton Pi a trop chaud 🥵"), detail: t("Il ralentit pour se protéger. Un ventilateur ou un boîtier mieux aéré l'aiderait beaucoup.") };
  if (disk >= 92)
    return { mood: "bad", title: t("Le stockage est presque plein"), detail: t("Libère un peu de place ou branche un disque externe pour éviter les soucis.") };
  if (mem >= 92)
    return { mood: "warn", title: t("La mémoire est bien remplie"), detail: t("Ça fonctionne, mais arrêter une application inutilisée rendrait ton Pi plus réactif.") };
  if (s.temp >= 70)
    return { mood: "warn", title: t("Il commence à faire chaud"), detail: t("{0} °C : rien de grave, mais garde un œil sur la ventilation.", s.temp.toFixed(0)) };
  if (s.cpu >= 85)
    return { mood: "warn", title: t("Ton Pi travaille dur 💪"), detail: t("Le processeur est très sollicité en ce moment.") };
  const temp = s.temp ? t(", au frais à {0} °C", s.temp.toFixed(0)) : "";
  return { mood: "good", title: t("Tout roule ✨"), detail: t("Ton serveur est en pleine forme{0}.", temp) };
}

export function tempMood(t: number): Mood {
  return t >= 80 ? "bad" : t >= 70 ? "warn" : "good";
}

export function usageMood(p: number): Mood {
  return p >= 90 ? "bad" : p >= 75 ? "warn" : "good";
}

export function deviceLabel(h?: Hardware): string {
  if (!h) return t("ton serveur");
  return { pi5: t("ton Raspberry Pi 5"), pi4: t("ton Raspberry Pi 4"), pi3: t("ton Raspberry Pi 3"), pizero2: t("ton Pi Zero 2"), pi: t("ton Raspberry Pi"), generic: t("ton serveur") }[h.family];
}

export const profileLabel: Record<Hardware["profile"], string> = {
  leger: t("Mode léger"),
  standard: t("Mode standard"),
  performance: t("Mode performance"),
};

export const actionLabel: Record<string, string> = {
  install: t("Installation"),
  update: t("Mise à jour"),
  uninstall: t("Désinstallation"),
  start: t("Démarrage"),
  stop: t("Arrêt"),
  restart: t("Redémarrage"),
  format: t("Formatage"),
  upgrade: t("Mise à jour du système"),
  refresh: t("Recherche de mises à jour"),
};

/** Couleur d'avatar stable dérivée du nom. */
export function avatarHue(name: string): number {
  let h = 0;
  for (const c of name) h = (h * 31 + c.charCodeAt(0)) % 360;
  return h;
}

/** Traduit les erreurs techniques fréquentes (Docker, réseau…) en explication compréhensible. */
export function explainError(log: string): string {
  const l = log.toLowerCase();
  if (l.includes("429 too many requests") || l.includes("toomanyrequests"))
    return t("Docker Hub limite temporairement les téléchargements depuis ta connexion. Patiente une heure ou deux, puis réessaie.");
  if (l.includes("port is already allocated") || l.includes("address already in use"))
    return t("Le port utilisé par cette application est déjà pris par une autre. Change le port dans la configuration (onglet Configuration).");
  if (l.includes("no matching manifest") || l.includes("exec format error"))
    return t("Cette application n'existe pas pour le processeur de ton Pi. Un système 64 bits est peut-être nécessaire.");
  if (l.includes("no space left"))
    return t("Il n'y a plus assez de place sur le stockage. Libère de l'espace ou fais le ménage dans Conteneurs → Images.");
  if (l.includes("dial tcp") || l.includes("temporary failure in name resolution") || l.includes("i/o timeout") || l.includes("tls handshake"))
    return t("Ton Pi n'arrive pas à joindre internet pour télécharger l'application. Vérifie sa connexion réseau.");
  if (l.includes("yaml") || l.includes("services must be a mapping") || l.includes("additional property"))
    return t("Le fichier de configuration contient une erreur de syntaxe. Vérifie l'indentation dans l'onglet Configuration.");
  if (l.includes("cannot connect to the docker daemon"))
    return t("Docker ne répond pas. Redémarre ton Pi pour le relancer.");
  return t("Quelque chose s'est mal passé. Le détail technique ci-dessous peut t'aider (ou aider quelqu'un à t'aider).");
}

/** Explication du profil matériel, dans la langue de l'interface. */
export function profileHint(p?: Hardware["profile"]): string {
  if (p === "leger") return t("Mode léger : les applications gourmandes sont déconseillées pour garder ton Pi réactif.");
  if (p === "standard") return t("Mode standard : la plupart des applications fonctionnent, évite d'en lancer trop de grosses en même temps.");
  return t("Mode performance : ton matériel peut faire tourner toutes les applications du catalogue.");
}

const ramText = (mb: number) => (mb >= 1024 ? t("{0} Go", (mb / 1024).toFixed(1).replace(/\.0$/, "")) : t("{0} Mo", mb));

/** Avertissement de compatibilité d'une app du catalogue. */
export function appWarning(a: { warnKind?: string; cpuArch?: string; minRam?: number; warning?: string }, h?: Hardware): string {
  if (a.warnKind === "arch") return t("Cette application n'existe pas pour ton processeur ({0}). Un système 64 bits est nécessaire.", a.cpuArch ?? "");
  if (a.warnKind === "ram" && a.minRam && h) return t("Cette application demande environ {0} de RAM, ton Pi en a {1}. Elle risque d'être lente.", ramText(a.minRam), ramText(h.ramMB));
  return a.warning ?? "";
}

/** Nom de la machine affiché (le serveur renvoie « Ordinateur générique (arch) » hors Raspberry Pi). */
export function machineName(model?: string): string {
  if (!model) return "—";
  const m = model.match(/^Ordinateur générique \((.+)\)$/);
  return m ? t("Ordinateur générique ({0})", m[1]) : model;
}
