// Client HTTP minimal pour l'API Azmo OS.
import { t, tServer } from "../lib/i18n";

export class ApiError extends Error {
  constructor(message: string, public status: number) {
    super(message);
  }
}

type Listener = () => void;
const unauthorizedListeners = new Set<Listener>();

/** Appelé quand la session expire (401) : l'application revient à l'écran de connexion. */
export function onUnauthorized(fn: Listener) {
  unauthorizedListeners.add(fn);
  return () => { unauthorizedListeners.delete(fn); };
}

async function request<T>(method: string, path: string, body?: unknown, raw?: BodyInit): Promise<T> {
  const headers: Record<string, string> = { "X-Requested-With": "AzmoOS" };
  let payload: BodyInit | undefined = raw;
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  let res: Response;
  try {
    res = await fetch(path, { method, headers, body: payload, credentials: "same-origin" });
  } catch {
    throw new ApiError(t("Impossible de joindre ton Pi. Est-il bien allumé et connecté ?"), 0);
  }
  if (res.status === 401 && !path.startsWith("/api/login") && !path.startsWith("/api/status")) {
    unauthorizedListeners.forEach((l) => l());
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = tServer((data as { error?: string }).error ?? res.statusText);
    throw new ApiError(msg.charAt(0).toUpperCase() + msg.slice(1), res.status);
  }
  return data as T;
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T = { ok: boolean }>(path: string, body: unknown = {}) => request<T>("POST", path, body),
  put: <T = { ok: boolean }>(path: string, body: unknown = {}) => request<T>("PUT", path, body),
  del: <T = { ok: boolean }>(path: string) => request<T>("DELETE", path),
  upload: <T = { ok: boolean }>(path: string, form: FormData) => request<T>("POST", path, undefined, form),
};

export const q = (v: string) => encodeURIComponent(v);
