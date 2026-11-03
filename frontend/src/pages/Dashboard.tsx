import { useRef } from "react";
import { Link } from "react-router-dom";
import { Cpu, MemoryStick, Thermometer, HardDrive, ArrowDownUp, Smile, AlertTriangle, Flame, Plus, Store } from "lucide-react";
import { api } from "../api/client";
import type { AppsResponse, SystemStats } from "../api/types";
import { usePoll } from "../hooks/usePoll";
import { useSession } from "../session";
import { bytes, pct, uptime } from "../lib/format";
import { greeting, health, tempMood, usageMood, deviceLabel, profileLabel } from "../lib/human";
import { Badge, Button, Card, Empty, Progress, Loading } from "../components/ui";
import { Sparkline } from "../components/Sparkline";
import { AppGrid } from "./Apps";

export function Dashboard() {
  const { status, can } = useSession();
  const hw = status.hardware;
  const hist = useRef<{ cpu: number[]; temp: number[]; rx: number[]; mem: number[] }>({ cpu: [], temp: [], rx: [], mem: [] });
  const sys = usePoll(async () => {
    const s = await api.get<SystemStats>("/api/system");
    const h = hist.current;
    const push = (a: number[], v: number) => { a.push(v); if (a.length > 40) a.shift(); };
    push(h.cpu, s.cpu); push(h.temp, s.temp); push(h.rx, s.netRx + s.netTx); push(h.mem, pct(s.memUsed, s.memTotal));
    return s;
  }, hw?.refresh ?? 3000);
  const apps = usePoll(() => api.get<AppsResponse>("/api/apps"), 6000);

  const user = status.user!;
  const s = sys.data;
  const hl = s ? health(s) : null;
  const HealthIcon = hl?.mood === "bad" ? Flame : hl?.mood === "warn" ? AlertTriangle : Smile;
  const running = apps.data?.apps.filter((a) => a.state === "running").length ?? 0;

  return (
    <>
      <div className="hello">
        <h1>{greeting(user.display)}</h1>
        <p>
          Voici comment va {deviceLabel(hw)} aujourd'hui.
          {apps.data && apps.data.apps.length > 0 && (running === 0 ? <> Aucune application ne tourne pour l'instant.</> : <> {running} application{running > 1 ? "s tournent" : " tourne"} en ce moment.</>)}
        </p>
      </div>

      {!s ? <Loading label="Je prends le pouls de ton Pi…" /> : (
        <>
          <div className={`health ${hl!.mood}`}>
            <div className="health-icon"><HealthIcon /></div>
            <div style={{ minWidth: 220 }}>
              <h2>{hl!.title}</h2>
              <p>{hl!.detail}</p>
            </div>
            <div className="facts">
              <div className="fact"><span>Machine</span><b>{s.model.replace("Raspberry Pi", "Pi").replace(" Rev ", " · rév. ")}</b></div>
              <div className="fact"><span>Allumé depuis</span><b>{uptime(s.uptime)}</b></div>
              <div className="fact"><span>Adresse</span><b>{s.interfaces.flatMap((i) => i.ips)[0] ?? "—"}</b></div>
              {hw && <div className="fact"><span>Profil</span><b>{profileLabel[hw.profile]}</b></div>}
            </div>
          </div>

          <div className="grid grid-4">
            <Card className="metric">
              <div className="metric-head"><div className="metric-icon"><Cpu /></div>Processeur</div>
              <div className="metric-value">{s.cpu.toFixed(0)}<small>%</small></div>
              <Sparkline values={hist.current.cpu} max={100} />
              <div className="metric-note">{s.cores} cœurs{s.cpuFreq ? ` · ${(s.cpuFreq / 1000).toFixed(1).replace(".", ",")} GHz` : ""}</div>
            </Card>
            <Card className="metric">
              <div className="metric-head"><div className="metric-icon"><MemoryStick /></div>Mémoire</div>
              <div className="metric-value">{pct(s.memUsed, s.memTotal)}<small>%</small></div>
              <Sparkline values={hist.current.mem} max={100} color="#c07cf0" />
              <div className="metric-note">{bytes(s.memUsed)} utilisés sur {bytes(s.memTotal)}</div>
            </Card>
            <Card className="metric">
              <div className="metric-head"><div className="metric-icon" style={{ background: "var(--warn-soft)", color: "var(--warn)" }}><Thermometer /></div>Température</div>
              <div className="metric-value">{s.temp ? s.temp.toFixed(1).replace(".", ",") : "—"}<small>°C</small></div>
              <Sparkline values={hist.current.temp} max={90} color="#ff8a65" />
              <div className="metric-note">
                {!s.temp ? "Capteur non disponible" : <Badge mood={tempMood(s.temp)}>{tempMood(s.temp) === "good" ? "Bien au frais" : tempMood(s.temp) === "warn" ? "Un peu chaud" : "Trop chaud"}</Badge>}
              </div>
            </Card>
            <Card className="metric">
              <div className="metric-head"><div className="metric-icon" style={{ background: "var(--good-soft)", color: "var(--good)" }}><ArrowDownUp /></div>Réseau</div>
              <div className="metric-value" style={{ fontSize: 24, marginTop: 20 }}>↓ {bytes(s.netRx)}/s</div>
              <Sparkline values={hist.current.rx} color="#5fd39a" />
              <div className="metric-note">↑ {bytes(s.netTx)}/s envoyés</div>
            </Card>
          </div>

          <div className="grid grid-3" style={{ marginTop: 18 }}>
            {s.disks.map((d) => {
              const p = pct(d.used, d.total);
              return (
                <Card key={d.path}>
                  <div className="metric-head"><div className="metric-icon"><HardDrive /></div>{d.path === "/" ? "Stockage système" : `Données (${d.path})`}</div>
                  <div className="row between" style={{ margin: "16px 0 10px" }}>
                    <span className="bold" style={{ fontSize: 20 }}>{bytes(d.free)} <span className="muted small" style={{ fontWeight: 500 }}>libres</span></span>
                    <span className="muted small">{p}% de {bytes(d.total)}</span>
                  </div>
                  <Progress value={p} mood={usageMood(p)} />
                </Card>
              );
            })}
          </div>
        </>
      )}

      <div className="section-title">
        Mes applications
        {apps.data && <span className="count">{apps.data.apps.length}</span>}
        {can("apps.manage") && <div className="right"><Link to="/store" className="btn sm"><Plus /> Ajouter</Link></div>}
      </div>
      {apps.data ? (
        apps.data.apps.length ? <AppGrid data={apps.data} reload={apps.reload} /> : (
          <Card>
            <Empty emoji="🧺" title="Aucune application pour l'instant" action={can("apps.manage") && <Link to="/store"><Button variant="primary" icon={<Store />}>Ouvrir l'App Store</Button></Link>}>
              Ton Pi n'attend que ça ! Installe Jellyfin pour tes films, Nextcloud pour tes fichiers ou Home Assistant pour ta maison.
            </Empty>
          </Card>
        )
      ) : <Loading />}
    </>
  );
}
