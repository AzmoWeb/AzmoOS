// Types partagés avec l'API d'Azmo OS (internal/server).

export type Perm = "admin" | "apps.manage" | "files.read" | "files.write";

export interface User {
  username: string;
  display: string;
  role: "admin" | "member";
  perms: Perm[];
  created: number;
  lastSeen: number;
}

export interface Hardware {
  model: string;
  family: "pi5" | "pi4" | "pi3" | "pizero2" | "pi" | "generic";
  isPi: boolean;
  ramMB: number;
  cores: number;
  arch: string;
  profile: "leger" | "standard" | "performance";
  profileHint: string;
  wifi: boolean;
  bluetooth: boolean;
  pcie: boolean;
  refresh: number;
}

export interface Status {
  product: string;
  version: string;
  hostname: string;
  initialized: boolean;
  authenticated: boolean;
  user?: User;
  docker?: boolean;
  hardware?: Hardware;
}

export interface DiskUsage { path: string; total: number; used: number; free: number }
export interface Iface { name: string; mac: string; ips: string[] }

export interface SystemStats {
  hostname: string;
  model: string;
  os: string;
  kernel: string;
  arch: string;
  cores: number;
  cpu: number;
  cpuFreq: number;
  load: number[];
  memTotal: number;
  memUsed: number;
  swapTotal: number;
  swapUsed: number;
  temp: number;
  throttled?: string;
  uptime: number;
  netRx: number;
  netTx: number;
  disks: DiskUsage[];
  interfaces: Iface[];
  time: number;
}

export interface HistoryPoint { t: number; cpu: number; mem: number; temp: number; rx: number; tx: number }

export interface Port { privatePort: number; publicPort: number; type: string }

export interface Container {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  created: number;
  ports: Port[] | null;
  labels: Record<string, string>;
  project: string;
}

export interface App {
  id: string;
  name: string;
  tagline: string;
  description: string;
  category: string;
  icon: string;
  port: number;
  path?: string;
  scheme?: string;
  arch: string[] | null;
  compose: string;
  notes?: string;
  minRam?: number;
  installed: boolean;
  custom?: boolean;
}

export interface CatalogApp extends App { compatible: boolean; warning?: string }

export interface InstalledApp extends App {
  state: "running" | "stopped" | "partial" | "unknown";
  containers: Container[];
}

export interface AppsResponse { apps: InstalledApp[]; others: Container[]; docker: boolean }

export interface Job {
  id: string;
  kind: string;
  target: string;
  action: string;
  status: "running" | "done" | "error";
  log: string;
  start: string;
  end?: string;
}

export interface ContainerStats { id: string; cpu: number; memUsage: number; memLimit: number; netRx: number; netTx: number }

export interface ContainerDetails {
  id: string;
  name: string;
  image: string;
  created: string;
  started: string;
  state: string;
  restart: string;
  network: string;
  mounts: { type: string; source: string; destination: string; rw: boolean; name?: string }[];
  ports: string[];
  env: string[] | null;
  restarts: number;
}

export interface DockerImage { id: string; tags: string[]; size: number; created: number; inUse: boolean; dangling: boolean }
export interface DockerVolume { name: string; driver: string; mountpoint: string; created: string; inUse: boolean; project: string }

export interface FileEntry { name: string; path: string; dir: boolean; size: number; modTime: string }

export interface Partition {
  name: string;
  path: string;
  size: number;
  fstype: string;
  label: string;
  uuid: string;
  mountpoint: string;
  fsUsed: number;
  fsAvail: number;
  autoMount: boolean;
  system: boolean;
}

export interface Disk {
  name: string;
  path: string;
  model: string;
  size: number;
  transport: string;
  removable: boolean;
  system: boolean;
  partitions: Partition[];
}

export interface Share { name: string; path: string; readOnly: boolean; guest: boolean; users?: string[] }
export interface SharesResponse { status: { installed: boolean; running: boolean; smbUsers: string[] }; shares: Share[] }

export interface NetInterface {
  name: string;
  type: "ethernet" | "wifi" | "other";
  state: string;
  mac: string;
  ips: string[];
  gateway: string;
  dns: string[];
  connection: string;
  method: string;
  speed: number;
}

export interface WiFi { ssid: string; signal: number; security: string; active: boolean; known: boolean }

export interface FirewallRule { number: number; to: string; action: string; from: string; v6: boolean }
export interface Firewall { installed: boolean; active: boolean; rules: FirewallRule[] }

export interface ServiceState { name: string; installed: boolean; active: boolean; enabled: boolean }

export interface Release { version: string; name: string; notes: string; published: string; url: string }
export interface UpdatesResponse {
  azmoos: { current: string; latest: Release | null; available: boolean; checked: number; error?: string };
  packages: { name: string; current: string; new: string }[];
  rebootRequired: boolean;
}

export interface AzmoEvent {
  time: number;
  level: "info" | "warning" | "error";
  category: string;
  user?: string;
  ip?: string;
  message: string;
}

export interface Session { id: string; ip: string; agent: string; expires: number; current: boolean }
