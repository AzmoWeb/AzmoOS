import { useId } from "react";

/** Mini-graphique en aire, sans dépendance. */
export function Sparkline({ values, max, color = "var(--brand)" }: { values: number[]; max?: number; color?: string }) {
  const id = useId();
  if (values.length < 2) return <svg className="spark" aria-hidden />;
  const w = 200, h = 46;
  const m = max ?? Math.max(...values, 1);
  const pts = values.map((v, i) => [(i / (values.length - 1)) * w, h - 3 - (Math.min(v, m) / m) * (h - 8)]);
  const d = pts.map((p, i) => `${i ? "L" : "M"}${p[0].toFixed(1)} ${p[1].toFixed(1)}`).join("");
  return (
    <svg className="spark" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id={id} x1="0" x2="0" y1="0" y2="1">
          <stop offset="0" stopColor={color} stopOpacity=".32" />
          <stop offset="1" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={`${d}L${w} ${h}L0 ${h}Z`} fill={`url(#${id})`} />
      <path d={d} fill="none" stroke={color} strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}
