// Messages « humains » : on parle à la personne, pas à un administrateur système.
import type { Hardware, SystemStats } from "../api/types";

export function greeting(name: string): string {
  const h = new Date().getHours();
  const first = name.split(" ")[0];
  if (h < 5) return `Encore debout, ${first} ? 🌙`;
  if (h < 12) return `Bonjour ${first} ☀️`;
  if (h < 18) return `Bon après-midi ${first} 👋`;
  return `Bonsoir ${first} 🌆`;
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
    return { mood: "bad", title: "Ton Pi a trop chaud 🥵", detail: "Il ralentit pour se protéger. Un ventilateur ou un boîtier mieux aéré l'aiderait beaucoup." };
  if (disk >= 92)
    return { mood: "bad", title: "Le stockage est presque plein", detail: "Libère un peu de place ou branche un disque externe pour éviter les soucis." };
  if (mem >= 92)
    return { mood: "warn", title: "La mémoire est bien remplie", detail: "Ça fonctionne, mais arrêter une application inutilisée rendrait ton Pi plus réactif." };
  if (s.temp >= 70)
    return { mood: "warn", title: "Il commence à faire chaud", detail: `${s.temp.toFixed(0)} °C : rien de grave, mais garde un œil sur la ventilation.` };
  if (s.cpu >= 85)
    return { mood: "warn", title: "Ton Pi travaille dur 💪", detail: "Le processeur est très sollicité en ce moment." };
  const temp = s.temp ? `, au frais à ${s.temp.toFixed(0)} °C` : "";
  return { mood: "good", title: "Tout roule ✨", detail: `Ton serveur est en pleine forme${temp}.` };
}

export function tempMood(t: number): Mood {
  return t >= 80 ? "bad" : t >= 70 ? "warn" : "good";
}

export function usageMood(p: number): Mood {
  return p >= 90 ? "bad" : p >= 75 ? "warn" : "good";
}

export function deviceLabel(h?: Hardware): string {
  if (!h) return "ton serveur";
  return { pi5: "ton Raspberry Pi 5", pi4: "ton Raspberry Pi 4", pi3: "ton Raspberry Pi 3", pizero2: "ton Pi Zero 2", pi: "ton Raspberry Pi", generic: "ton serveur" }[h.family];
}

export const profileLabel: Record<Hardware["profile"], string> = {
  leger: "Mode léger",
  standard: "Mode standard",
  performance: "Mode performance",
};

export const actionLabel: Record<string, string> = {
  install: "Installation",
  update: "Mise à jour",
  uninstall: "Désinstallation",
  start: "Démarrage",
  stop: "Arrêt",
  restart: "Redémarrage",
  format: "Formatage",
  upgrade: "Mise à jour du système",
  refresh: "Recherche de mises à jour",
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
    return "Docker Hub limite temporairement les téléchargements depuis ta connexion. Patiente une heure ou deux, puis réessaie.";
  if (l.includes("port is already allocated") || l.includes("address already in use"))
    return "Le port utilisé par cette application est déjà pris par une autre. Change le port dans la configuration (onglet Configuration).";
  if (l.includes("no matching manifest") || l.includes("exec format error"))
    return "Cette application n'existe pas pour le processeur de ton Pi. Un système 64 bits est peut-être nécessaire.";
  if (l.includes("no space left"))
    return "Il n'y a plus assez de place sur le stockage. Libère de l'espace ou fais le ménage dans Conteneurs → Images.";
  if (l.includes("dial tcp") || l.includes("temporary failure in name resolution") || l.includes("i/o timeout") || l.includes("tls handshake"))
    return "Ton Pi n'arrive pas à joindre internet pour télécharger l'application. Vérifie sa connexion réseau.";
  if (l.includes("yaml") || l.includes("services must be a mapping") || l.includes("additional property"))
    return "Le fichier de configuration contient une erreur de syntaxe. Vérifie l'indentation dans l'onglet Configuration.";
  if (l.includes("cannot connect to the docker daemon"))
    return "Docker ne répond pas. Redémarre ton Pi pour le relancer.";
  return "Quelque chose s'est mal passé. Le détail technique ci-dessous peut t'aider (ou aider quelqu'un à t'aider).";
}
