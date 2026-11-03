"""Génère le logo Azmo OS et ses déclinaisons en SVG. Made by Azmo.

Le texte est converti en tracés (Plus Jakarta Sans, installée par « npm ci » dans frontend/).
Usage : python generate.py <dossier de sortie>
"""

import os
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen

FONT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "frontend", "node_modules", "@fontsource-variable", "plus-jakarta-sans", "files", "plus-jakarta-sans-latin-wght-normal.woff2")
_cache = {}

def font(weight):
    if weight not in _cache:
        f = TTFont(FONT)
        _cache[weight] = instancer.instantiateVariableFont(f, {"wght": weight})
    return _cache[weight]

def text_path(text, weight, size, x=0, y=0, tracking=0):
    """Renvoie (d, largeur) : tracé SVG du texte, ligne de base en y."""
    f = font(weight)
    gs = f.getGlyphSet()
    cmap = f.getBestCmap()
    upm = f["head"].unitsPerEm
    scale = size / upm
    pen = SVGPathPen(gs)
    cx = 0.0
    hmtx = f["hmtx"]
    for ch in text:
        g = cmap[ord(ch)]
        tp = TransformPen(pen, (scale, 0, 0, -scale, x + cx, y))
        gs[g].draw(tp)
        cx += hmtx[g][0] * scale + tracking
    return pen.getCommands(), cx - tracking


import os, sys
OUT = sys.argv[1]
os.makedirs(OUT, exist_ok=True)

STOPS = '<stop offset="0" stop-color="#8b7bff"/><stop offset=".55" stop-color="#c07cf0"/><stop offset="1" stop-color="#ff8a65"/>'
DEFS = (f'<linearGradient id="g" x1="0" y1="0" x2="1" y2="1">{STOPS}</linearGradient>'
        f'<linearGradient id="gu" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="512" y2="512">{STOPS}</linearGradient>'
        '<radialGradient id="shine" cx=".3" cy=".15" r=".9"><stop offset="0" stop-color="#fff" stop-opacity=".28"/>'
        '<stop offset=".6" stop-color="#fff" stop-opacity="0"/></radialGradient>')
TILE = '<rect width="512" height="512" rx="120" fill="url(#gu)"/><rect width="512" height="512" rx="120" fill="url(#shine)"/>'

def svg(vb, body):
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{vb}"><defs>{DEFS}</defs>{body}</svg>'

# ---- Concepts de comparaison ----
A_INNER = ('<path d="M256 96 112 416h68l28-66h96l28 66h68L256 96Zm0 128 28 66h-56l28-66Z" fill="#fff"/>'
           '<circle cx="392" cy="122" r="26" fill="#fff" opacity=".92"/>')
C_HEX = ('<path d="M256 22c14 0 28 4 40 11l150 87c25 14 40 41 40 70v172c0 29-15 56-40 70L296 519c-25 14-55 14-80 0L66 432c-25-14-40-41-40-70V190c0-29 15-56 40-70L216 33c12-7 26-11 40-11z" transform="translate(0 -15)" fill="url(#gu)"/>')
C_INNER = ('<path d="M256 120 140 384h62l22-52h64l22 52h62L256 120Zm0 112 20 50h-40l20-50Z" fill="#fff"/>'
           '<path d="M256 88c-6-22-26-36-48-34 2 22 20 38 42 40M262 94c8-20 30-32 52-28-4 22-24 36-46 36" fill="#fff" opacity=".9"/>')

# ---- Concept retenu : le A-serveur — une unité de rack traverse le A ----
def server_mark(color="#fff", cut="url(#gu)", led1="#ff8a65", led2="#8b7bff", slot="#d9d2ff"):
    cut_attr = f'stroke="{cut}" stroke-width="16" paint-order="stroke"' if cut != "none" else ""
    return (f'<path d="M150 396 256 134 362 396" fill="none" stroke="{color}" stroke-width="62" stroke-linecap="round" stroke-linejoin="round"/>'
            f'<rect x="122" y="262" width="268" height="68" rx="34" fill="{color}" {cut_attr}/>'
            f'<path d="M166 296h70" stroke="{slot}" stroke-width="12" stroke-linecap="round"/>'
            f'<circle cx="320" cy="296" r="12" fill="{led2}"/><circle cx="352" cy="296" r="12" fill="{led1}"/>')

files = {
    "concept-A.svg": svg("0 0 512 512", TILE + A_INNER),
    "concept-B.svg": svg("0 0 512 512", TILE + server_mark()),
    "concept-C.svg": svg("0 0 512 512", C_HEX + C_INNER),
}
files["logo.svg"] = files["concept-B.svg"]
# Symbole seul (sans tuile) : monochromes et dégradé
files["symbole-blanc.svg"] = svg("96 104 320 320", server_mark("#fff", "#15121c", "#15121c", "#15121c", "#15121c"))
files["symbole-noir.svg"] = svg("96 104 320 320", server_mark("#15121c", "#fff", "#fff", "#fff", "#fff"))
files["symbole-degrade.svg"] = svg("96 104 320 320", server_mark("url(#gu)", "#fff", "#fff", "#fff", "#fff"))
# Favicon : traits épaissis pour rester lisible en 16 px
files["favicon.svg"] = svg("0 0 512 512",
    '<rect width="512" height="512" rx="128" fill="url(#gu)"/>'
    '<path d="M146 404 256 126 366 404" fill="none" stroke="#fff" stroke-width="80" stroke-linecap="round" stroke-linejoin="round"/>'
    '<rect x="106" y="252" width="300" height="84" rx="42" fill="#fff" stroke="url(#gu)" stroke-width="22" paint-order="stroke"/>'
    '<circle cx="350" cy="294" r="17" fill="#ff8a65"/>')

def tile_at(x, y, size):
    return f'<g transform="translate({x} {y}) scale({size/512})">{TILE}{server_mark()}</g>'

# Logotypes horizontaux
def wordmark(dark):
    txt, sub = ("#f5f2fb", "#a39cb5") if dark else ("#1f1a2b", "#6f6782")
    d1, w1 = text_path("Azmo", 800, 150, 0, 0, -3)
    d2, w2 = text_path("OS", 500, 150, 0, 0, -2)
    d3, _ = text_path("MADE BY AZMO", 700, 34, 0, 0, 6)
    icon, gap, base = 220, 44, 150
    x0 = icon + gap
    w = x0 + w1 + 30 + w2 + 8
    # Le dégradé du « OS » est calé sur le mot lui-même (objectBoundingBox).
    return svg(f"0 0 {w:.0f} 240",
        tile_at(0, 10, icon)
        + f'<path transform="translate({x0} {base})" d="{d1}" fill="{txt}"/>'
        + f'<path transform="translate({x0 + w1 + 30:.1f} {base})" d="{d2}" fill="url(#g)"/>'
        + f'<path transform="translate({x0 + 4} {base + 62})" d="{d3}" fill="{sub}"/>')
files["logotype-sombre.svg"] = wordmark(True)
files["logotype-clair.svg"] = wordmark(False)

HALOS = ('<radialGradient id="h1" cx=".1" cy="0" r=".7"><stop offset="0" stop-color="#8b7bff" stop-opacity=".35"/><stop offset="1" stop-color="#8b7bff" stop-opacity="0"/></radialGradient>'
         '<radialGradient id="h2" cx="1" cy=".1" r=".6"><stop offset="0" stop-color="#ff8a65" stop-opacity=".25"/><stop offset="1" stop-color="#ff8a65" stop-opacity="0"/></radialGradient>')

def centered_title(cx, base, size):
    d1, w1 = text_path("Azmo", 800, size, 0, 0, -3)
    d2, w2 = text_path("OS", 500, size, 0, 0, -2)
    gap = size * 0.2
    x = cx - (w1 + gap + w2) / 2
    return (f'<path transform="translate({x:.1f} {base})" d="{d1}" fill="#f5f2fb"/>'
            f'<path transform="translate({x + w1 + gap:.1f} {base})" d="{d2}" fill="url(#g)"/>')

def centered(text, weight, size, cx, base, color, tracking=0):
    d, w = text_path(text, weight, size, 0, 0, tracking)
    return f'<path transform="translate({cx - w/2:.1f} {base})" d="{d}" fill="{color}"/>'

# Image de partage GitHub (1280×640)
files["social-preview.svg"] = (
    f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 640"><defs>{DEFS}{HALOS}</defs>'
    '<rect width="1280" height="640" fill="#15121c"/><rect width="1280" height="640" fill="url(#h1)"/><rect width="1280" height="640" fill="url(#h2)"/>'
    + tile_at(550, 66, 180) + centered_title(640, 392, 132)
    + centered("Ton serveur personnel sur Raspberry Pi", 600, 38, 640, 470, "#c9c2d9")
    + centered("Made by Azmo", 700, 28, 640, 560, "#958ca8", 2) + '</svg>')

# Fond d'écran / écran de démarrage (1920×1080)
files["fond-ecran.svg"] = (
    f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080"><defs>{DEFS}{HALOS}'
    '<radialGradient id="h3" cx=".5" cy=".42" r=".6"><stop offset="0" stop-color="#8b7bff" stop-opacity=".22"/><stop offset="1" stop-color="#15121c" stop-opacity="0"/></radialGradient></defs>'
    '<rect width="1920" height="1080" fill="#15121c"/><rect width="1920" height="1080" fill="url(#h3)"/>'
    + tile_at(860, 300, 200) + centered_title(960, 640, 110)
    + centered("Made by Azmo", 700, 30, 960, 720, "#958ca8", 3) + '</svg>')

# Les concepts de comparaison ne sont pas publiés.
for k in ("concept-A.svg", "concept-B.svg", "concept-C.svg"):
    files.pop(k)
for n, v in files.items():
    open(os.path.join(OUT, n), "w").write(v)
print("ok", len(files))
