// Formatage lisible par un humain (en français).

export function bytes(b: number, digits = 1): string {
  if (!b || b < 0) return "0 o";
  const units = ["o", "Ko", "Mo", "Go", "To"];
  const i = Math.min(units.length - 1, Math.floor(Math.log(b) / Math.log(1024)));
  return `${(b / 1024 ** i).toFixed(i ? digits : 0).replace(".", ",")} ${units[i]}`;
}

export function uptime(s: number): string {
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d} jour${d > 1 ? "s" : ""} et ${h} h`;
  if (h > 0) return `${h} h ${m} min`;
  return `${m} min`;
}

export function pct(a: number, b: number): number {
  return b ? Math.round((a / b) * 100) : 0;
}

const rtf = new Intl.RelativeTimeFormat("fr", { numeric: "auto" });

/** « il y a 5 minutes », « hier »… */
export function ago(unixSeconds: number): string {
  const diff = unixSeconds - Date.now() / 1000;
  const abs = Math.abs(diff);
  if (abs < 45) return "à l'instant";
  if (abs < 3600) return rtf.format(Math.round(diff / 60), "minute");
  if (abs < 86400) return rtf.format(Math.round(diff / 3600), "hour");
  if (abs < 86400 * 30) return rtf.format(Math.round(diff / 86400), "day");
  return new Date(unixSeconds * 1000).toLocaleDateString("fr-FR", { day: "numeric", month: "long", year: "numeric" });
}

export function dateTime(v: string | number): string {
  const d = typeof v === "number" ? new Date(v * 1000) : new Date(v);
  return d.toLocaleString("fr-FR", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
}

export function plural(n: number, one: string, many = one + "s"): string {
  return `${n} ${n > 1 ? many : one}`;
}

export function slug(s: string): string {
  return s.toLowerCase().normalize("NFD").replace(/[̀-ͯ]/g, "").replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 40);
}
