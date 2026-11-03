import { useCallback, useEffect, useState, type ReactElement } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { api } from "./api/client";
import type { Perm, Status } from "./api/types";
import { JobsProvider, SessionProvider, useSession } from "./session";
import { Layout } from "./components/Layout";
import { Loading, ErrorBox } from "./components/ui";
import { Login, Setup } from "./pages/Auth";
import { Dashboard } from "./pages/Dashboard";
import { Apps } from "./pages/Apps";
import { StorePage } from "./pages/Store";
import { Containers } from "./pages/Containers";
import { Files } from "./pages/Files";
import { Storage } from "./pages/Storage";
import { Network } from "./pages/Network";
import { Security } from "./pages/Security";
import { Users } from "./pages/Users";
import { Backups } from "./pages/Backups";
import { Updates } from "./pages/Updates";
import { Activity } from "./pages/Activity";
import { Settings } from "./pages/Settings";

function Guard({ perm, children }: { perm: Perm; children: ReactElement }) {
  const { can } = useSession();
  return can(perm) ? children : <Navigate to="/" replace />;
}

export function App() {
  const [status, setStatus] = useState<Status | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [setupMode, setSetupMode] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const s = await api.get<Status>("/api/status");
      setStatus(s);
      setError(null);
      if (!s.initialized) setSetupMode(true);
    } catch (e) {
      setError((e as Error).message);
    }
  }, []);
  useEffect(() => { refresh(); }, [refresh]);

  if (error && !status) return <div className="auth"><div style={{ width: "min(460px,100%)" }}><ErrorBox error={error} onRetry={refresh} /></div></div>;
  if (!status) return <div className="auth"><Loading label="Connexion à ton Pi…" /></div>;
  if (setupMode) return <Setup status={status} onDone={async () => { setSetupMode(false); await refresh(); }} />;
  if (!status.authenticated) return <Login status={status} onDone={refresh} />;

  return (
    <SessionProvider status={status} refresh={refresh}>
      <JobsProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<Dashboard />} />
              <Route path="apps" element={<Apps />} />
              <Route path="store" element={<Guard perm="apps.manage"><StorePage /></Guard>} />
              <Route path="containers" element={<Guard perm="apps.manage"><Containers /></Guard>} />
              <Route path="files" element={<Guard perm="files.read"><Files /></Guard>} />
              <Route path="storage" element={<Guard perm="admin"><Storage /></Guard>} />
              <Route path="network" element={<Guard perm="admin"><Network /></Guard>} />
              <Route path="security" element={<Guard perm="admin"><Security /></Guard>} />
              <Route path="users" element={<Guard perm="admin"><Users /></Guard>} />
              <Route path="backups" element={<Guard perm="admin"><Backups /></Guard>} />
              <Route path="updates" element={<Guard perm="admin"><Updates /></Guard>} />
              <Route path="activity" element={<Guard perm="admin"><Activity /></Guard>} />
              <Route path="settings" element={<Settings />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </JobsProvider>
    </SessionProvider>
  );
}
