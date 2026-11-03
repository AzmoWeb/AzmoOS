// Vérifie que chaque texte de l'interface (t("…")) et chaque message d'erreur du serveur
// a sa traduction anglaise. Usage : npm run i18n:check   (--list pour tout afficher)
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "@babel/parser";
import traverseMod from "@babel/traverse";

const traverse = traverseMod.default ?? traverseMod;
const here = path.dirname(fileURLToPath(import.meta.url));
const src = path.join(here, "..", "src");
const repo = path.join(here, "..", "..");

function walk(dir, ext, out = []) {
  for (const f of fs.readdirSync(dir)) {
    const p = path.join(dir, f);
    if (fs.statSync(p).isDirectory()) walk(p, ext, out);
    else if (ext.test(f)) out.push(p);
  }
  return out;
}

// 1. Textes de l'interface
const keys = new Map();
for (const file of walk(src, /\.(tsx?|jsx?)$/)) {
  const ast = parse(fs.readFileSync(file, "utf8"), { sourceType: "module", plugins: ["typescript", "jsx"] });
  traverse(ast, {
    CallExpression(p) {
      const c = p.node.callee;
      if (c.type === "Identifier" && c.name === "t" && p.node.arguments[0]?.type === "StringLiteral") {
        const k = p.node.arguments[0].value;
        if (!keys.has(k)) keys.set(k, path.relative(src, file));
      }
    },
  });
}

// 2. Messages d'erreur du serveur (errString("…"), errors.New("…"))
const server = new Set();
for (const file of walk(path.join(repo, "internal"), /\.go$/).concat(walk(path.join(repo, "cmd"), /\.go$/))) {
  if (file.endsWith("_test.go")) continue;
  const code = fs.readFileSync(file, "utf8");
  for (const m of code.matchAll(/(?:errString|errors\.New)\("((?:[^"\\]|\\.)+)"\)/g)) server.add(JSON.parse(`"${m[1]}"`));
  // Messages fixes du journal (s.log / Events.Add avec un texte littéral)
  for (const m of code.matchAll(/(?:s\.log\(r, events\.\w+, "\w+", |(?:Events|evlog)\.Add\(events\.\w+, "\w+", [^,]+, [^,]+, )"((?:[^"\\]|\\.)+)"\)/g)) server.add(JSON.parse(`"${m[1]}"`));
}

const { en } = await import(path.join(src, "i18n", "en.ts").replace(/\.ts$/, ".ts")).catch(() => ({ en: null }));
const dict = en ?? (() => {
  // Lecture sans compilation : en.ts est un simple objet littéral.
  const txt = fs.readFileSync(path.join(src, "i18n", "en.ts"), "utf8");
  return Function(`"use strict"; return (${txt.slice(txt.indexOf("{"), txt.lastIndexOf("}") + 1)});`)();
})();

const missingUI = [...keys.keys()].filter((k) => !(k in dict));
const missingServer = [...server].filter((k) => !(k in dict));
const unused = Object.keys(dict).filter((k) => !keys.has(k) && !server.has(k));

console.log(`Interface : ${keys.size} textes, ${missingUI.length} sans traduction`);
console.log(`Serveur   : ${server.size} messages, ${missingServer.length} sans traduction`);
if (unused.length) console.log(`Inutilisées : ${unused.length} traductions qui ne servent plus`);
const list = process.argv.includes("--list") || process.argv.includes("--json");
if (process.argv.includes("--json")) {
  fs.writeFileSync(process.argv[process.argv.indexOf("--json") + 1], JSON.stringify({ ui: missingUI, server: missingServer }, null, 1));
} else {
  for (const k of missingUI.slice(0, list ? Infinity : 30)) console.log(`  ✘ ${JSON.stringify(k)}   (${keys.get(k)})`);
  for (const k of missingServer.slice(0, list ? Infinity : 30)) console.log(`  ✘ [serveur] ${JSON.stringify(k)}`);
}
process.exit(missingUI.length || missingServer.length ? 1 : 0);
