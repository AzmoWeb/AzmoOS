// Exporte les SVG d'Azmo OS en PNG (fond transparent). Usage : node branding/export.js branding
// Exporte les SVG en PNG (fond transparent) à la taille demandée.
const { chromium } = require('playwright');
const fs = require('fs');
const [,, B] = process.argv;
const jobs = [
  ...[16, 32, 48, 64, 128, 192, 256, 512, 1024].map((s) => ['logo.svg', `logo-${s}.png`, s, s]),
  ['favicon.svg', 'favicon-16.png', 16, 16], ['favicon.svg', 'favicon-32.png', 32, 32],
  ['logo.svg', 'apple-touch-icon.png', 180, 180],
  ['logotype-sombre.svg', 'logotype-sombre.png', null, 240 * 2],
  ['logotype-clair.svg', 'logotype-clair.png', null, 240 * 2],
  ['symbole-degrade.svg', 'symbole-degrade-512.png', 512, 512],
  ['symbole-blanc.svg', 'symbole-blanc-512.png', 512, 512],
  ['symbole-noir.svg', 'symbole-noir-512.png', 512, 512],
  ['social-preview.svg', 'social-preview.png', 1280, 640],
  ['fond-ecran.svg', 'fond-ecran-1920x1080.png', 1920, 1080],
];
(async () => {
  const b = await chromium.launch();
  const p = await b.newPage();
  for (const [src, out, w0, h] of jobs) {
    const svg = fs.readFileSync(`${B}/${src}`, 'utf8');
    const vb = svg.match(/viewBox="([^"]+)"/)[1].split(/\s+/).map(Number);
    const w = w0 ?? Math.round(h * vb[2] / vb[3]);
    await p.setViewportSize({ width: w, height: h });
    await p.setContent(`<html><body style="margin:0;background:transparent">${svg.replace('<svg ', `<svg width="${w}" height="${h}" `)}</body></html>`);
    await p.screenshot({ path: `${B}/png/${out}`, omitBackground: true, clip: { x: 0, y: 0, width: w, height: h } });
  }
  await b.close();
  console.log('exporté :', jobs.length);
})();
