/* AzmoOS — interface web — made by Azmo */
"use strict";

// ---------- Utilitaires ----------
const $ = (s, el = document) => el.querySelector(s);
const esc = s => String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const fmtBytes = (b, d = 1) => { if (!b) return "0 o"; const u = ["o", "Ko", "Mo", "Go", "To"]; const i = Math.min(u.length - 1, Math.floor(Math.log(b) / Math.log(1024))); return (b / 1024 ** i).toFixed(i ? d : 0) + " " + u[i]; };
const fmtUptime = s => { const d = Math.floor(s / 86400), h = Math.floor(s % 86400 / 3600), m = Math.floor(s % 3600 / 60); return (d ? d + " j " : "") + (h ? h + " h " : "") + m + " min"; };
const fmtDate = t => new Date(t).toLocaleString("fr-FR", { dateStyle: "short", timeStyle: "short" });
const pct = (a, b) => b ? Math.round(a / b * 100) : 0;
const host = location.hostname;

async function api(path, opts = {}) {
  const o = { headers: { "X-Requested-With": "AzmoOS" }, credentials: "same-origin", ...opts };
  if (o.json !== undefined) { o.method = o.method || "POST"; o.body = JSON.stringify(o.json); o.headers["Content-Type"] = "application/json"; }
  const r = await fetch(path, o);
  if (r.status === 401 && !path.startsWith("/api/login")) { state.status = null; boot(); throw new Error("Session expirée"); }
  const data = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(data.error || r.statusText);
  return data;
}
const post = (path, json = {}) => api(path, { json });

function toast(msg, type = "") {
  const t = document.createElement("div");
  t.className = "toast " + type; t.textContent = msg;
  $("#toasts").append(t); setTimeout(() => t.remove(), 4200);
}

// ---------- Icônes (SVG inline) ----------
const P = {
  home: '<path d="M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z"/>',
  apps: '<rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/>',
  store: '<path d="M4 7h16l-1.5 12.5a1 1 0 0 1-1 .9h-11a1 1 0 0 1-1-.9z"/><path d="M9 10V6a3 3 0 0 1 6 0v4"/>',
  folder: '<path d="M3 6a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
  file: '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6"/>',
  box: '<path d="M21 8 12 3 3 8v8l9 5 9-5z"/><path d="m3 8 9 5 9-5M12 13v8"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>',
  cpu: '<rect x="5" y="5" width="14" height="14" rx="2"/><rect x="9" y="9" width="6" height="6"/><path d="M9 1v4M15 1v4M9 19v4M15 19v4M1 9h4M1 15h4M19 9h4M19 15h4"/>',
  mem: '<rect x="2" y="7" width="20" height="10" rx="2"/><path d="M6 11v2M10 11v2M14 11v2M18 11v2"/>',
  temp: '<path d="M14 14.8V5a2 2 0 0 0-4 0v9.8a4 4 0 1 0 4 0z"/>',
  disk: '<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.7 4 3 9 3s9-1.3 9-3V5"/><path d="M3 12c0 1.7 4 3 9 3s9-1.3 9-3"/>',
  net: '<path d="M12 3v18M5 10l7-7 7 7M19 14l-7 7-7-7"/>',
  more: '<circle cx="12" cy="5" r="1.5"/><circle cx="12" cy="12" r="1.5"/><circle cx="12" cy="19" r="1.5"/>',
  open: '<path d="M14 3h7v7M10 14 21 3M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5"/>',
  play: '<path d="m7 4 13 8-13 8z"/>',
  stop: '<rect x="6" y="6" width="12" height="12" rx="2"/>',
  restart: '<path d="M3 12a9 9 0 1 0 3-6.7L3 8"/><path d="M3 3v5h5"/>',
  trash: '<path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14"/>',
  logs: '<path d="M4 6h16M4 12h16M4 18h10"/>',
  update: '<path d="M12 3v12M7 10l5 5 5-5M4 21h16"/>',
  upload: '<path d="M12 21V9M7 14l5-5 5 5M4 3h16"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  close: '<path d="M6 6l12 12M18 6 6 18"/>',
  edit: '<path d="M4 20h4L19 9l-4-4L4 16z"/>',
  power: '<path d="M12 3v9M6.3 6.3a8 8 0 1 0 11.4 0"/>',
  logout: '<path d="M15 4h4a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-4M10 17l-5-5 5-5M5 12h11"/>',
  menu: '<path d="M4 6h16M4 12h16M4 18h16"/>',
  moon: '<path d="M21 12.8A9 9 0 1 1 11.2 3 7 7 0 0 0 21 12.8z"/>',
  code: '<path d="m8 6-6 6 6 6M16 6l6 6-6 6"/>',
  pi: '<path d="M7 3c1.5 0 3 1 5 3 2-2 3.5-3 5-3M12 6v1"/><circle cx="12" cy="14" r="7"/><circle cx="9.5" cy="12.5" r="1.3"/><circle cx="14.5" cy="12.5" r="1.3"/><circle cx="12" cy="16.5" r="1.3"/>',
};
const ic = (n, cls = "") => `<svg class="${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${P[n]}</svg>`;

// ---------- État ----------
const state = { status: null, timers: [], history: { cpu: [], temp: [], rx: [], tx: [] }, filter: "Tout", query: "" };
const theme = localStorage.getItem("azmo-theme"); if (theme) document.documentElement.dataset.theme = theme;

function clearTimers() { state.timers.forEach(clearInterval); state.timers = []; }
function every(ms, fn) { fn(); state.timers.push(setInterval(fn, ms)); }

// ---------- Démarrage / auth ----------
async function boot() {
  try { state.status = await api("/api/status"); }
  catch (e) { $("#app").innerHTML = `<div class="auth"><div class="card"><h1>Serveur injoignable</h1><p class="c">${esc(e.message)}</p></div></div>`; return; }
  if (!state.status.initialized) return renderAuth(true);
  if (!state.status.authenticated) return renderAuth(false);
  renderShell(); route();
}

function renderAuth(setup) {
  clearTimers();
  $("#app").innerHTML = `
  <div class="auth"><form class="card" id="authf">
    <img class="logo" src="logo.svg" alt="">
    <h1>${setup ? "Bienvenue sur AzmoOS" : "Connexion"}</h1>
    <p class="c">${setup ? "Créez votre compte administrateur pour commencer." : "Accédez à votre serveur personnel."}</p>
    <label class="f"><span>Nom d'utilisateur</span><input class="in" name="username" autocomplete="username" required value="${setup ? "azmo" : ""}"></label>
    <label class="f"><span>Mot de passe</span><input class="in" type="password" name="password" autocomplete="${setup ? "new-password" : "current-password"}" required minlength="${setup ? 6 : 1}"></label>
    ${setup ? `<label class="f"><span>Confirmer le mot de passe</span><input class="in" type="password" name="confirm" autocomplete="new-password" required></label>` : ""}
    <div class="err" id="autherr"></div>
    <button class="btn primary">${setup ? "Créer mon compte" : "Se connecter"}</button>
    <div class="made">AzmoOS ${esc(state.status.version)} — made by <b>Azmo</b></div>
  </form></div>`;
  $("#authf").onsubmit = async e => {
    e.preventDefault();
    const f = Object.fromEntries(new FormData(e.target));
    if (setup && f.password !== f.confirm) return ($("#autherr").textContent = "Les mots de passe ne correspondent pas");
    const b = e.target.querySelector("button"); b.disabled = true;
    try { await post(setup ? "/api/setup" : "/api/login", { username: f.username, password: f.password }); boot(); }
    catch (err) { $("#autherr").textContent = err.message; b.disabled = false; }
  };
}

// ---------- Coquille ----------
const routes = [
  ["dashboard", "Tableau de bord", "home"],
  ["apps", "Mes applications", "apps"],
  ["store", "App Store", "store"],
  ["files", "Fichiers", "folder"],
  ["docker", "Conteneurs", "box"],
  ["settings", "Paramètres", "gear"],
];

function renderShell() {
  const u = state.status.username || "?";
  $("#app").innerHTML = `
  <div class="mobile-top"><button class="icon-btn" id="burger">${ic("menu")}</button><img src="logo.svg" alt="">AzmoOS</div>
  <div class="shell">
    <aside class="side" id="side">
      <div class="brand"><img src="logo.svg" alt=""><div>AzmoOS<small>MADE BY AZMO</small></div></div>
      <nav class="nav">${routes.map(([id, label, i]) => `<a href="#/${id}" data-r="${id}">${ic(i)}<span>${label}</span></a>`).join("")}</nav>
      <div class="spacer"></div>
      <div class="user"><div class="avatar">${esc(u[0].toUpperCase())}</div><div style="flex:1;min-width:0"><b>${esc(u)}</b><div class="meta">Administrateur</div></div>
        <button class="icon-btn" id="themeb" title="Thème">${ic("moon")}</button>
        <button class="icon-btn" id="logout" title="Déconnexion">${ic("logout")}</button></div>
    </aside>
    <main class="main" id="view"></main>
  </div>`;
  $("#burger").onclick = () => $("#side").classList.toggle("open");
  $("#logout").onclick = async () => { await post("/api/logout"); state.status = null; boot(); };
  $("#themeb").onclick = () => {
    const t = document.documentElement.dataset.theme === "light" ? "dark" : "light";
    document.documentElement.dataset.theme = t; localStorage.setItem("azmo-theme", t);
  };
}

function route() {
  if (!state.status?.authenticated) return;
  const r = (location.hash.slice(2) || "dashboard").split("?")[0];
  document.querySelectorAll(".nav a").forEach(a => a.classList.toggle("active", a.dataset.r === r));
  $("#side")?.classList.remove("open");
  clearTimers(); closeMenu();
  (views[r] || views.dashboard)($("#view"));
}
window.addEventListener("hashchange", route);

function header(title, sub = "", right = "") {
  return `<div class="head"><div><h1>${title}</h1>${sub ? `<div class="sub">${sub}</div>` : ""}</div><div class="right">${right}</div></div>`;
}

// ---------- Modales & menus ----------
function modal(html, { wide = false } = {}) {
  const root = $("#modal-root");
  const ov = document.createElement("div");
  ov.className = "overlay";
  ov.innerHTML = `<div class="modal ${wide ? "wide" : ""}">${html}</div>`;
  const close = () => ov.remove();
  ov.addEventListener("mousedown", e => { if (e.target === ov) close(); });
  ov.querySelectorAll("[data-close]").forEach(b => b.onclick = close);
  root.append(ov);
  return { el: ov, close };
}
function confirmBox(title, text, okLabel = "Confirmer", danger = true) {
  return new Promise(res => {
    const m = modal(`<header><h2>${esc(title)}</h2></header><div class="body">${text}</div>
      <footer><button class="btn" data-close>Annuler</button><button class="btn ${danger ? "danger solid" : "primary"}" id="cok">${esc(okLabel)}</button></footer>`);
    m.el.querySelector("[data-close]").onclick = () => { m.close(); res(false); };
    $("#cok", m.el).onclick = () => { m.close(); res(true); };
  });
}
function promptBox(title, label, value = "") {
  return new Promise(res => {
    const m = modal(`<header><h2>${esc(title)}</h2></header><form class="body" id="pf"><label class="f"><span>${esc(label)}</span><input class="in" name="v" value="${esc(value)}" required></label>
      <div class="row" style="justify-content:flex-end"><button type="button" class="btn" data-close>Annuler</button><button class="btn primary">Valider</button></div></form>`);
    const inp = m.el.querySelector("input"); inp.focus(); inp.select();
    m.el.querySelector("[data-close]").onclick = () => { m.close(); res(null); };
    $("#pf", m.el).onsubmit = e => { e.preventDefault(); m.close(); res(inp.value.trim()); };
  });
}
let menuEl = null;
function closeMenu() { menuEl?.remove(); menuEl = null; }
function openMenu(anchor, items) {
  closeMenu();
  menuEl = document.createElement("div");
  menuEl.className = "menu";
  items.forEach(it => {
    if (it === "-") return menuEl.append(Object.assign(document.createElement("hr")));
    const b = document.createElement("button");
    b.innerHTML = ic(it.icon) + esc(it.label);
    if (it.danger) b.className = "danger";
    b.onclick = e => { e.stopPropagation(); closeMenu(); it.action(); };
    menuEl.append(b);
  });
  document.body.append(menuEl);
  const r = anchor.getBoundingClientRect(), mw = menuEl.offsetWidth, mh = menuEl.offsetHeight;
  menuEl.style.left = Math.max(8, Math.min(r.right - mw, innerWidth - mw - 8)) + "px";
  menuEl.style.top = (r.bottom + mh + 8 > innerHeight ? r.top - mh - 4 : r.bottom + 4) + "px";
}
document.addEventListener("click", e => { if (menuEl && !menuEl.contains(e.target)) closeMenu(); });
document.addEventListener("keydown", e => { if (e.key === "Escape") { closeMenu(); $("#modal-root .overlay:last-child")?.remove(); } });

// ---------- Rendu commun ----------
function appIcon(a, size = "") {
  const letter = `<div class="letter" ${size}>${esc((a.name || a.id || "?")[0].toUpperCase())}</div>`;
  if (!a.icon) return letter;
  return `<img class="ico" src="${esc(a.icon)}" alt="" loading="lazy" onerror="this.outerHTML=this.dataset.f" data-f="${esc(letter)}">`;
}
function appURL(a) {
  if (!a.port) return null;
  return `${a.scheme || "http"}://${host}:${a.port}${a.path || ""}`;
}
function sparkline(values, max) {
  if (values.length < 2) return `<svg class="spark"></svg>`;
  const m = max || Math.max(...values, 1), w = 200, h = 42;
  const pts = values.map((v, i) => [i / (values.length - 1) * w, h - 2 - v / m * (h - 6)]);
  const d = pts.map((p, i) => (i ? "L" : "M") + p[0].toFixed(1) + " " + p[1].toFixed(1)).join("");
  return `<svg class="spark" viewBox="0 0 ${w} ${h}" preserveAspectRatio="none"><defs><linearGradient id="sg" x1="0" x2="0" y1="0" y2="1"><stop offset="0" stop-color="#8b5cf6" stop-opacity=".45"/><stop offset="1" stop-color="#8b5cf6" stop-opacity="0"/></linearGradient></defs>
    <path d="${d}L${w} ${h}L0 ${h}Z" fill="url(#sg)"/><path d="${d}" fill="none" stroke="url(#sl)" stroke-width="2" vector-effect="non-scaling-stroke"/>
    <linearGradient id="sl"><stop offset="0" stop-color="#8b5cf6"/><stop offset="1" stop-color="#22d3ee"/></linearGradient></svg>`;
}
const level = p => p >= 90 ? "bad" : p >= 75 ? "warn" : "";
const push = (arr, v) => { arr.push(v); if (arr.length > 40) arr.shift(); };

// ---------- Vues ----------
const views = {};

views.dashboard = el => {
  el.innerHTML = header("Tableau de bord", "Vue d'ensemble de votre serveur") + `<div id="sys"><div class="spin"></div></div>
    <div class="section-title">Mes applications <a class="btn sm" href="#/store" style="margin-left:auto">${ic("plus")}Ajouter</a></div><div id="dapps"></div>`;
  every(2500, async () => {
    let s; try { s = await api("/api/system"); } catch { return; }
    const h = state.history;
    push(h.cpu, s.cpu); push(h.temp, s.temp); push(h.rx, s.netRx); push(h.tx, s.netTx);
    const mem = pct(s.memUsed, s.memTotal);
    const tempP = Math.min(100, Math.round(s.temp / 85 * 100));
    const ips = s.interfaces.flatMap(i => i.ips).join(", ") || "—";
    const box = $("#sys"); if (!box) return;
    box.innerHTML = `
    <div class="card hero" style="margin-bottom:16px"><div class="pi">${ic("pi")}</div>
      <div style="min-width:200px"><div style="font-size:20px;font-weight:750">${esc(s.hostname)}</div><div class="meta">${esc(s.model)}</div></div>
      <div class="kv"><div><span>Système</span><b>${esc(s.os)}</b></div><div><span>Noyau</span><b>${esc(s.kernel)} (${esc(s.arch)})</b></div>
      <div><span>Adresse IP</span><b>${esc(ips)}</b></div><div><span>Allumé depuis</span><b>${fmtUptime(s.uptime)}</b></div></div></div>
    <div class="grid stats">
      <div class="card"><h3>${ic("cpu")}Processeur</h3><div class="big">${s.cpu.toFixed(0)}<small> %</small></div>
        <div class="meta">${s.cores} cœurs${s.cpuFreq ? " · " + s.cpuFreq + " MHz" : ""} · charge ${(s.load || []).map(x => x.toFixed(2)).join(" ")}</div>${sparkline(h.cpu, 100)}</div>
      <div class="card"><h3>${ic("mem")}Mémoire</h3><div class="big">${mem}<small> %</small></div>
        <div class="bar ${level(mem)}"><i style="width:${mem}%"></i></div><div class="meta">${fmtBytes(s.memUsed)} / ${fmtBytes(s.memTotal)}${s.swapTotal ? " · swap " + fmtBytes(s.swapUsed) : ""}</div></div>
      <div class="card"><h3>${ic("temp")}Température</h3><div class="big">${s.temp ? s.temp.toFixed(1) : "—"}<small> °C</small></div>
        <div class="bar ${level(tempP)}"><i style="width:${tempP}%"></i></div><div class="meta">${s.throttled && s.throttled !== "0x0" ? "⚠️ Bridage détecté (" + esc(s.throttled) + ")" : "Aucun bridage"}</div></div>
      <div class="card"><h3>${ic("net")}Réseau</h3><div class="big" style="font-size:22px">↓ ${fmtBytes(s.netRx)}/s</div>
        <div class="meta">↑ ${fmtBytes(s.netTx)}/s</div>${sparkline(h.rx)}</div>
      ${s.disks.map(d => { const p = pct(d.used, d.total); return `<div class="card"><h3>${ic("disk")}Stockage ${esc(d.path)}</h3><div class="big">${p}<small> %</small></div>
        <div class="bar ${level(p)}"><i style="width:${p}%"></i></div><div class="meta">${fmtBytes(d.free)} libres sur ${fmtBytes(d.total)}</div></div>`; }).join("")}
    </div>`;
  });
  every(5000, () => renderApps($("#dapps"), true));
};

async function renderApps(box, compact) {
  if (!box) return;
  let data, jobs;
  try { [data, jobs] = await Promise.all([api("/api/apps"), api("/api/jobs")]); } catch (e) { box.innerHTML = `<div class="card empty">${esc(e.message)}</div>`; return; }
  if (!document.body.contains(box)) return;
  const running = Object.fromEntries(jobs.filter(j => j.status === "running").map(j => [j.app, j]));
  if (!data.apps.length) {
    box.innerHTML = `<div class="card empty">${ic("store")}<div>Aucune application installée.</div><p><a class="btn primary" href="#/store">Découvrir l'App Store</a></p></div>`;
  } else {
    box.innerHTML = `<div class="grid apps">${data.apps.map(a => `
      <div class="card app" data-id="${esc(a.id)}">
        <span class="dot ${a.state}" title="${a.state}"></span>
        <button class="icon-btn menu-btn">${ic("more")}</button>
        ${appIcon(a)}<div class="name">${esc(a.name)}</div>
        ${running[a.id] ? `<div class="busy"><div class="spin"></div>${actionLabel(running[a.id].action)}…</div>` : ""}
      </div>`).join("")}</div>`;
    box.querySelectorAll(".app").forEach(card => {
      const a = data.apps.find(x => x.id === card.dataset.id);
      card.onclick = e => {
        if (e.target.closest(".menu-btn")) return appMenu(e.target.closest(".menu-btn"), a);
        const u = appURL(a);
        if (u && a.state === "running") window.open(u, "_blank", "noopener");
        else appMenu(card.querySelector(".menu-btn"), a);
      };
    });
  }
  if (!compact) renderOthers($("#others"), data.others);
}
const actionLabel = a => ({ install: "Installation", update: "Mise à jour", uninstall: "Désinstallation", start: "Démarrage", stop: "Arrêt", restart: "Redémarrage" }[a] || a);

function appMenu(btn, a) {
  const u = appURL(a);
  const act = (action, msg) => async () => {
    try { await post(`/api/apps/${a.id}/${action}`); toast(`${actionLabel(action)} de ${a.name} lancé(e)`); watchJob(a.id, a.name); route(); }
    catch (e) { toast(e.message, "error"); }
  };
  openMenu(btn, [
    ...(u ? [{ icon: "open", label: "Ouvrir", action: () => window.open(u, "_blank", "noopener") }] : []),
    a.state === "running" ? { icon: "stop", label: "Arrêter", action: act("stop") } : { icon: "play", label: "Démarrer", action: act("start") },
    { icon: "restart", label: "Redémarrer", action: act("restart") },
    { icon: "update", label: "Mettre à jour", action: act("update") },
    { icon: "logs", label: "Logs", action: () => a.containers[0] ? showLogs(a.containers[0].id, a.name) : toast("Aucun conteneur", "error") },
    { icon: "code", label: "Voir / modifier le compose", action: () => editCompose(a) },
    "-",
    { icon: "trash", label: "Désinstaller", danger: true, action: async () => {
      if (await confirmBox("Désinstaller " + a.name + " ?", `<p>Les conteneurs seront supprimés. Vos données dans <code>/DATA/AppData/${esc(a.id)}</code> sont <b>conservées</b>.</p>`, "Désinstaller")) act("uninstall")();
    } },
  ]);
}

function watchJob(id, name) {
  const t = setInterval(async () => {
    try {
      const j = (await api("/api/jobs")).find(x => x.app === id);
      if (!j || j.status === "running") return;
      clearInterval(t);
      if (j.status === "done") toast(`${name} : ${actionLabel(j.action).toLowerCase()} terminé(e) ✓`, "ok");
      else { toast(`${name} : échec — voir le détail`, "error"); showJobLog(j, name); }
      if (["dashboard", "apps", "store"].includes((location.hash.slice(2) || "dashboard"))) route();
    } catch { clearInterval(t); }
  }, 2000);
}
function showJobLog(j, name) {
  modal(`<header><h2>${esc(name)} — ${esc(actionLabel(j.action))}</h2><button class="icon-btn" data-close>${ic("close")}</button></header>
    <div class="body"><pre class="logs">${esc(j.log)}</pre></div>`, { wide: true });
}

async function showLogs(id, name) {
  const m = modal(`<header><h2>Logs — ${esc(name)}</h2><button class="icon-btn" id="lr" title="Rafraîchir">${ic("restart")}</button><button class="icon-btn" data-close style="margin-left:0">${ic("close")}</button></header>
    <div class="body"><pre class="logs" id="lg">Chargement…</pre></div>`, { wide: true });
  const load = async () => {
    try { const r = await api(`/api/docker/containers/${encodeURIComponent(id)}/logs?tail=500`); const pre = $("#lg", m.el); pre.textContent = r.logs || "(vide)"; pre.scrollTop = pre.scrollHeight; }
    catch (e) { $("#lg", m.el).textContent = e.message; }
  };
  $("#lr", m.el).onclick = load; load();
}

async function editCompose(a) {
  let compose = "";
  try { compose = (await api(`/api/apps/${a.id}/compose`)).compose; } catch (e) { return toast(e.message, "error"); }
  const m = modal(`<header>${appIcon(a)}<h2>${esc(a.name)} — docker-compose.yml</h2><button class="icon-btn" data-close>${ic("close")}</button></header>
    <div class="body"><textarea class="in" id="cmp" spellcheck="false">${esc(compose)}</textarea></div>
    <footer><button class="btn" data-close>Fermer</button><button class="btn primary" id="sv">Enregistrer et redéployer</button></footer>`, { wide: true });
  m.el.querySelectorAll("[data-close]").forEach(b => b.onclick = m.close);
  $("#sv", m.el).onclick = async () => {
    try { await post(`/api/apps/${a.id}/install`, { compose: $("#cmp", m.el).value, name: a.name, icon: a.icon, port: a.port, path: a.path }); m.close(); toast("Redéploiement lancé"); watchJob(a.id, a.name); route(); }
    catch (e) { toast(e.message, "error"); }
  };
}

views.apps = el => {
  el.innerHTML = header("Mes applications", "Applications installées via AzmoOS",
    `<button class="btn" id="custom">${ic("plus")}App personnalisée</button><a class="btn primary" href="#/store">${ic("store")}App Store</a>`) +
    `<div id="iapps"><div class="spin"></div></div><div id="others"></div>`;
  $("#custom").onclick = customApp;
  every(5000, () => renderApps($("#iapps"), false));
};

function renderOthers(box, others) {
  if (!box) return;
  if (!others?.length) { box.innerHTML = ""; return; }
  box.innerHTML = `<div class="section-title">Autres conteneurs Docker <span class="count">${others.length}</span></div>
    <div class="grid apps">${others.map(c => {
      const p = c.ports[0]?.publicPort;
      return `<div class="card app" data-id="${esc(c.id)}"><span class="dot ${c.state === "running" ? "running" : "stopped"}"></span>
        <div class="letter">${esc(c.name[0]?.toUpperCase() || "?")}</div><div class="name">${esc(c.name)}</div><div class="meta">${p ? ":" + p : esc(c.state)}</div></div>`;
    }).join("")}</div>`;
  box.querySelectorAll(".app").forEach(card => card.onclick = () => containerMenu(card, others.find(c => c.id === card.dataset.id)));
}

function containerMenu(anchor, c) {
  const act = action => async () => {
    try { await post(`/api/docker/containers/${encodeURIComponent(c.id)}/${action}`); toast(`${c.name} : ${action} ✓`, "ok"); route(); } catch (e) { toast(e.message, "error"); }
  };
  const p = c.ports[0]?.publicPort;
  openMenu(anchor, [
    ...(p ? [{ icon: "open", label: "Ouvrir :" + p, action: () => window.open(`http://${host}:${p}`, "_blank", "noopener") }] : []),
    c.state === "running" ? { icon: "stop", label: "Arrêter", action: act("stop") } : { icon: "play", label: "Démarrer", action: act("start") },
    { icon: "restart", label: "Redémarrer", action: act("restart") },
    { icon: "logs", label: "Logs", action: () => showLogs(c.id, c.name) },
    "-",
    { icon: "trash", label: "Supprimer", danger: true, action: async () => { if (await confirmBox("Supprimer " + c.name + " ?", "<p>Le conteneur sera supprimé définitivement.</p>", "Supprimer")) act("remove")(); } },
  ]);
}

function customApp() {
  const m = modal(`<header><h2>Application personnalisée</h2><button class="icon-btn" data-close>${ic("close")}</button></header>
  <form class="body" id="cf">
    <div class="row" style="align-items:flex-start"><label class="f" style="flex:2;min-width:180px"><span>Nom</span><input class="in" name="name" required placeholder="Mon app"></label>
    <label class="f" style="flex:1;min-width:120px"><span>Port web</span><input class="in" name="port" type="number" placeholder="8080"></label></div>
    <label class="f"><span>URL de l'icône (optionnel)</span><input class="in" name="icon" placeholder="https://…/icon.png"></label>
    <label class="f"><span>docker-compose.yml</span><textarea class="in" name="compose" spellcheck="false" required>services:
  monapp:
    image: nginx:alpine
    container_name: monapp
    restart: unless-stopped
    ports:
      - "8080:80"
    volumes:
      - /DATA/AppData/monapp:/usr/share/nginx/html
</textarea></label>
    <div class="row" style="justify-content:flex-end"><button type="button" class="btn" data-close>Annuler</button><button class="btn primary">Installer</button></div>
  </form>`, { wide: true });
  m.el.querySelectorAll("[data-close]").forEach(b => b.onclick = m.close);
  $("#cf", m.el).onsubmit = async e => {
    e.preventDefault();
    const f = Object.fromEntries(new FormData(e.target));
    const id = f.name.toLowerCase().normalize("NFD").replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 40) || "app";
    try { await post(`/api/apps/${id}/install`, { compose: f.compose, name: f.name, icon: f.icon, port: +f.port || 0 }); m.close(); toast("Installation lancée"); watchJob(id, f.name); route(); }
    catch (err) { toast(err.message, "error"); }
  };
}

views.store = async el => {
  el.innerHTML = header("App Store", "Installez des applications en un clic", `<input class="in search" id="q" placeholder="Rechercher…" value="${esc(state.query)}">`) + `<div id="cats"></div><div id="sl"><div class="spin"></div></div>`;
  let cat;
  try { cat = await api("/api/store/catalog"); } catch (e) { $("#sl").innerHTML = `<div class="card empty">${esc(e.message)}</div>`; return; }
  const cats = ["Tout", ...new Set(cat.map(a => a.category))];
  const draw = () => {
    $("#cats").innerHTML = `<div class="chips">${cats.map(c => `<button class="chip ${c === state.filter ? "active" : ""}">${esc(c)}</button>`).join("")}</div>`;
    $("#cats").querySelectorAll(".chip").forEach(b => b.onclick = () => { state.filter = b.textContent; draw(); });
    const q = state.query.toLowerCase();
    const list = cat.filter(a => (state.filter === "Tout" || a.category === state.filter) && (!q || (a.name + a.tagline + a.description).toLowerCase().includes(q)));
    $("#sl").innerHTML = list.length ? `<div class="grid store">${list.map(a => `
      <div class="card sapp" data-id="${a.id}">${appIcon(a)}<div><h4>${esc(a.name)}</h4><p>${esc(a.tagline)}</p>
        <span class="tag">${esc(a.category)}</span>${a.installed ? '<span class="tag ok">Installée</span>' : ""}</div></div>`).join("")}</div>`
      : `<div class="card empty">Aucune application trouvée.</div>`;
    $("#sl").querySelectorAll(".sapp").forEach(c => c.onclick = () => storeDetail(cat.find(a => a.id === c.dataset.id)));
  };
  $("#q").oninput = e => { state.query = e.target.value; draw(); };
  draw();
};

function storeDetail(a) {
  const m = modal(`<header>${appIcon(a)}<div><h2>${esc(a.name)}</h2><div class="meta">${esc(a.category)}${a.port ? " · port " + a.port : ""}</div></div><button class="icon-btn" data-close>${ic("close")}</button></header>
  <div class="body"><p style="margin-top:0">${esc(a.description)}</p>
    <div>${(a.arch || []).map(x => `<span class="tag">${esc(x)}</span>`).join("")}</div>
    ${a.notes ? `<div class="note">💡 ${esc(a.notes)}</div>` : ""}
    <details><summary>Configuration avancée (docker-compose.yml)</summary><textarea class="in" id="cmp" spellcheck="false">${esc(a.compose)}</textarea></details>
  </div>
  <footer><button class="btn" data-close>Fermer</button>
    ${a.installed ? `<a class="btn primary" href="#/apps" data-close>Déjà installée — voir</a>` : `<button class="btn primary" id="inst">${ic("update")}Installer</button>`}</footer>`, { wide: true });
  m.el.querySelectorAll("[data-close]").forEach(b => b.onclick = m.close);
  const ib = $("#inst", m.el);
  if (ib) ib.onclick = async () => {
    ib.disabled = true;
    try { await post(`/api/apps/${a.id}/install`, { compose: $("#cmp", m.el).value }); m.close(); toast(`Installation de ${a.name} lancée — téléchargement des images…`); watchJob(a.id, a.name); location.hash = "#/apps"; }
    catch (e) { toast(e.message, "error"); ib.disabled = false; }
  };
}

// ---------- Fichiers ----------
views.files = el => {
  const cur = decodeURIComponent((location.hash.split("?p=")[1] || "/"));
  el.innerHTML = header("Fichiers", "Contenu de /DATA",
    `<button class="btn" id="mk">${ic("plus")}Dossier</button><label class="btn primary">${ic("upload")}Envoyer<input type="file" id="up" multiple hidden></label>`) +
    `<div class="card" id="fc" style="padding:0"><div style="padding:14px 16px;border-bottom:1px solid var(--border)" class="crumbs" id="cr"></div><div class="scroll" id="fl"><div class="empty"><div class="spin" style="margin:auto"></div></div></div></div>
    <div class="meta" style="margin-top:10px">Astuce : glissez-déposez des fichiers dans la liste pour les envoyer.</div>`;
  const go = p => { location.hash = "#/files?p=" + encodeURIComponent(p); };
  const parts = cur.split("/").filter(Boolean);
  $("#cr").innerHTML = `<a href="#/files?p=%2F">${ic("folder")}</a>` + parts.map((p, i) => `<span class="meta">/</span><a href="#/files?p=${encodeURIComponent("/" + parts.slice(0, i + 1).join("/"))}">${esc(p)}</a>`).join("");
  const load = async () => {
    let list;
    try { list = await api("/api/files?path=" + encodeURIComponent(cur)); } catch (e) { $("#fl").innerHTML = `<div class="empty">${esc(e.message)}</div>`; return; }
    $("#fl").innerHTML = list.length ? `<table class="table"><thead><tr><th>Nom</th><th class="hide-sm">Taille</th><th class="hide-sm">Modifié</th><th></th></tr></thead><tbody>
      ${list.map((f, i) => `<tr class="click" data-i="${i}"><td><div class="fname">${f.dir ? ic("folder", "folder") : ic("file")}${esc(f.name)}</div></td>
        <td class="hide-sm meta">${f.dir ? "—" : fmtBytes(f.size)}</td><td class="hide-sm meta">${fmtDate(f.modTime)}</td>
        <td class="actions"><button class="icon-btn">${ic("more")}</button></td></tr>`).join("")}</tbody></table>`
      : `<div class="empty">${ic("folder")}<div>Dossier vide</div></div>`;
    $("#fl").querySelectorAll("tr.click").forEach(tr => {
      const f = list[+tr.dataset.i];
      const dl = (inline) => `/api/files/download?path=${encodeURIComponent(f.path)}${inline ? "&inline=1" : ""}`;
      tr.onclick = e => {
        const btn = e.target.closest(".icon-btn");
        if (!btn && f.dir) return go(f.path);
        openMenu(btn || tr, [
          ...(f.dir ? [{ icon: "folder", label: "Ouvrir", action: () => go(f.path) }] : [
            { icon: "open", label: "Aperçu", action: () => window.open(dl(true), "_blank", "noopener") },
            { icon: "update", label: "Télécharger", action: () => { const a = document.createElement("a"); a.href = dl(false); a.click(); } }]),
          { icon: "edit", label: "Renommer", action: async () => { const n = await promptBox("Renommer", "Nouveau nom", f.name); if (n && n !== f.name) try { await post("/api/files/rename", { path: f.path, name: n }); load(); } catch (e) { toast(e.message, "error"); } } },
          "-",
          { icon: "trash", label: "Supprimer", danger: true, action: async () => { if (await confirmBox("Supprimer « " + f.name + " » ?", f.dir ? "<p>Le dossier et <b>tout son contenu</b> seront supprimés.</p>" : "<p>Cette action est irréversible.</p>", "Supprimer")) try { await post("/api/files/delete", { path: f.path }); load(); } catch (e) { toast(e.message, "error"); } } },
        ]);
      };
    });
  };
  const upload = async files => {
    if (!files.length) return;
    const fd = new FormData(); [...files].forEach(f => fd.append("files", f));
    toast(`Envoi de ${files.length} fichier(s)…`);
    try { await api("/api/files/upload?path=" + encodeURIComponent(cur), { method: "POST", body: fd }); toast("Envoi terminé ✓", "ok"); load(); }
    catch (e) { toast(e.message, "error"); }
  };
  $("#up").onchange = e => upload(e.target.files);
  $("#mk").onclick = async () => { const n = await promptBox("Nouveau dossier", "Nom du dossier"); if (n) try { await post("/api/files/mkdir", { path: cur, name: n }); load(); } catch (e) { toast(e.message, "error"); } };
  const fc = $("#fc");
  fc.ondragover = e => { e.preventDefault(); fc.classList.add("drop"); };
  fc.ondragleave = () => fc.classList.remove("drop");
  fc.ondrop = e => { e.preventDefault(); fc.classList.remove("drop"); upload(e.dataTransfer.files); };
  load();
};

// ---------- Conteneurs ----------
views.docker = el => {
  el.innerHTML = header("Conteneurs", "Tous les conteneurs Docker de la machine") + `<div class="card" style="padding:0"><div class="scroll" id="dl"><div class="empty"><div class="spin" style="margin:auto"></div></div></div></div>`;
  every(5000, async () => {
    let list;
    try { list = await api("/api/docker/containers"); } catch (e) { $("#dl") && ($("#dl").innerHTML = `<div class="empty">${esc(e.message)}<p class="meta">Docker est-il installé et démarré ?</p></div>`); return; }
    const box = $("#dl"); if (!box) return;
    box.innerHTML = list.length ? `<table class="table"><thead><tr><th>Nom</th><th class="hide-sm">Image</th><th>État</th><th class="hide-sm">Ports</th><th></th></tr></thead><tbody>
      ${list.map((c, i) => `<tr data-i="${i}"><td><div class="fname"><span class="dot ${c.state === "running" ? "running" : "stopped"}"></span><b>${esc(c.name)}</b></div></td>
        <td class="hide-sm meta">${esc(c.image)}</td><td class="meta">${esc(c.status)}</td>
        <td class="hide-sm">${c.ports.map(p => `<a href="http://${esc(host)}:${p.publicPort}" target="_blank" rel="noopener" class="tag">${p.publicPort}</a>`).join("")}</td>
        <td class="actions"><button class="icon-btn">${ic("more")}</button></td></tr>`).join("")}</tbody></table>`
      : `<div class="empty">${ic("box")}<div>Aucun conteneur</div></div>`;
    box.querySelectorAll(".icon-btn").forEach(b => b.onclick = () => containerMenu(b, list[+b.closest("tr").dataset.i]));
  });
};

// ---------- Paramètres ----------
views.settings = async el => {
  el.innerHTML = header("Paramètres", "Configuration du système") + `<div class="grid" style="grid-template-columns:repeat(auto-fit,minmax(300px,1fr))">
    <form class="card" id="hf"><h3>${ic("pi")}Nom de la machine</h3><label class="f"><span>Nom d'hôte</span><input class="in" name="hostname" required pattern="[a-zA-Z0-9\-]+"></label>
      <button class="btn primary">Enregistrer</button><p class="meta">Accessible ensuite via http://&lt;nom&gt;.local</p></form>
    <form class="card" id="af"><h3>${ic("gear")}Compte</h3>
      <label class="f"><span>Nom d'utilisateur</span><input class="in" name="username" value="${esc(state.status.username)}" required></label>
      <label class="f"><span>Mot de passe actuel</span><input class="in" type="password" name="current" required autocomplete="current-password"></label>
      <label class="f"><span>Nouveau mot de passe</span><input class="in" type="password" name="password" minlength="6" required autocomplete="new-password"></label>
      <button class="btn primary">Mettre à jour</button></form>
    <div class="card"><h3>${ic("power")}Alimentation</h3><p class="meta">Redémarrer ou éteindre le Raspberry Pi.</p>
      <div class="row"><button class="btn" id="rb">${ic("restart")}Redémarrer</button><button class="btn danger" id="po">${ic("power")}Éteindre</button></div></div>
    <div class="card"><h3>${ic("box")}À propos</h3><div class="kv" id="about"><div><span>Version</span><b>AzmoOS ${esc(state.status.version)}</b></div>
      <div><span>Docker</span><b>${state.status.docker ? "✅ Disponible" : "❌ Indisponible"}</b></div></div>
      <p class="made" style="text-align:left">Conçu avec ❤️ — made by <b>Azmo</b></p></div>
  </div>`;
  api("/api/system").then(s => { $("#hf [name=hostname]").value = s.hostname; $("#about").insertAdjacentHTML("beforeend", `<div><span>Matériel</span><b>${esc(s.model)}</b></div><div><span>Système</span><b>${esc(s.os)}</b></div>`); }).catch(() => {});
  $("#hf").onsubmit = async e => { e.preventDefault(); try { await post("/api/system/hostname", { hostname: e.target.hostname.value }); toast("Nom d'hôte modifié ✓", "ok"); } catch (err) { toast(err.message, "error"); } };
  $("#af").onsubmit = async e => {
    e.preventDefault(); const f = Object.fromEntries(new FormData(e.target));
    try { await post("/api/account", f); state.status.username = f.username; toast("Compte mis à jour ✓", "ok"); renderShell(); route(); } catch (err) { toast(err.message, "error"); }
  };
  const power = (action, label) => async () => {
    if (!await confirmBox(label + " ?", `<p>Le Raspberry Pi va ${action === "reboot" ? "redémarrer" : "s'éteindre"}. Les applications seront temporairement indisponibles.</p>`, label)) return;
    try { await post("/api/system/power", { action }); toast(label + " en cours…"); } catch (e) { toast(e.message, "error"); }
  };
  $("#rb").onclick = power("reboot", "Redémarrer");
  $("#po").onclick = power("poweroff", "Éteindre");
};

boot();
