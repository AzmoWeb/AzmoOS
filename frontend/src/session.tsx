import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { api, onUnauthorized } from "./api/client";
import type { Job, Perm, Status } from "./api/types";
import { useUI } from "./components/overlay";
import { actionLabel } from "./lib/human";

interface SessionCtx {
  status: Status;
  refresh: () => Promise<void>;
  can: (p: Perm) => boolean;
}

const Ctx = createContext<SessionCtx>(null as unknown as SessionCtx);
export const useSession = () => useContext(Ctx);

export function SessionProvider({ status, refresh, children }: { status: Status; refresh: () => Promise<void>; children: ReactNode }) {
  const can = useCallback((p: Perm) => !!status.user && (status.user.role === "admin" || status.user.perms.includes(p)), [status.user]);
  useEffect(() => onUnauthorized(() => void refresh()), [refresh]);
  return <Ctx.Provider value={{ status, refresh, can }}>{children}</Ctx.Provider>;
}

/* ---------- Suivi global des tâches (installations, mises à jour…) ---------- */

interface JobsCtx {
  jobs: Job[];
  running: (kind: string, target: string) => Job | undefined;
  /** Force un rafraîchissement rapide après avoir lancé une tâche. */
  kick: () => void;
  onDone: (fn: (j: Job) => void) => () => void;
}

const JCtx = createContext<JobsCtx>(null as unknown as JobsCtx);
export const useJobs = () => useContext(JCtx);

const targetName = (j: Job) => (j.kind === "app" ? `« ${j.target} »` : j.kind === "storage" ? `du disque ${j.target}` : "");

export function JobsProvider({ children }: { children: ReactNode }) {
  const { toast } = useUI();
  const [jobs, setJobs] = useState<Job[]>([]);
  const prev = useRef<Map<string, string>>(new Map());
  const listeners = useRef(new Set<(j: Job) => void>());
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);

  const poll = useCallback(async () => {
    clearTimeout(timer.current);
    let list: Job[] = [];
    try {
      list = await api.get<Job[]>("/api/jobs");
      setJobs(list);
      for (const j of list) {
        const before = prev.current.get(j.id + j.start);
        if (before === "running" && j.status !== "running") {
          const what = `${actionLabel[j.action] ?? j.action} ${targetName(j)}`.trim();
          toast(j.status === "done" ? `${what} : c'est fait ✨` : `${what} : ça a échoué. Regarde le détail dans le journal.`, j.status === "done" ? "good" : "bad");
          listeners.current.forEach((l) => l(j));
        }
      }
      prev.current = new Map(list.map((j) => [j.id + j.start, j.status]));
    } catch { /* hors ligne : on réessaiera */ }
    const busy = list.some((j) => j.status === "running");
    timer.current = setTimeout(poll, busy ? 2000 : 8000);
  }, [toast]);

  useEffect(() => { poll(); return () => clearTimeout(timer.current); }, [poll]);

  const running = useCallback((kind: string, target: string) => jobs.find((j) => j.kind === kind && j.target === target && j.status === "running"), [jobs]);
  const kick = useCallback(() => { setTimeout(poll, 300); }, [poll]);
  const onDone = useCallback((fn: (j: Job) => void) => { listeners.current.add(fn); return () => { listeners.current.delete(fn); }; }, []);

  return <JCtx.Provider value={{ jobs, running, kick, onDone }}>{children}</JCtx.Provider>;
}
