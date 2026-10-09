export function KpiCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="card">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-2xl font-bold">{value}</p>
    </div>
  );
}

export function BarChart({ data }: { data: { label: string; value: number }[] }) {
  const max = Math.max(1, ...data.map((d) => d.value));
  return (
    <div className="flex h-44 items-end gap-3" role="img" aria-label="Gráfico de barras">
      {data.map((d) => (
        <div key={d.label} className="flex flex-1 flex-col items-center gap-1">
          <span className="text-xs font-semibold">{d.value}</span>
          <div
            className="w-full rounded-t bg-brand-500"
            style={{ height: `${Math.max(6, (d.value / max) * 120)}px` }}
          />
          <span className="text-xs text-slate-500">{d.label}</span>
        </div>
      ))}
    </div>
  );
}

export function DonutChart({ data }: { data: { label: string; value: number }[] }) {
  const total = data.reduce((acc, d) => acc + d.value, 0) || 1;
  const colors = ["#2f6bff", "#22c55e", "#f59e0b", "#ef4444", "#8b5cf6", "#06b6d4"];
  let offset = 25;
  const segments = data.map((d, i) => {
    const frac = (d.value / total) * 100;
    const seg = { ...d, dash: `${frac} ${100 - frac}`, offset: -offset, color: colors[i % colors.length] };
    offset += frac;
    return seg;
  });
  return (
    <div className="flex items-center gap-4">
      <svg viewBox="0 0 42 42" className="h-36 w-36" role="img" aria-label="Gráfico de rosca">
        <circle cx="21" cy="21" r="15.9" fill="transparent" strokeWidth="6" className="stroke-slate-200 dark:stroke-slate-800" />
        {segments.map((s) => (
          <circle
            key={s.label}
            cx="21"
            cy="21"
            r="15.9"
            fill="transparent"
            stroke={s.color}
            strokeWidth="6"
            strokeDasharray={s.dash}
            strokeDashoffset={s.offset}
          />
        ))}
      </svg>
      <ul className="flex flex-col gap-1 text-xs">
        {segments.map((s) => (
          <li key={s.label} className="flex items-center gap-2">
            <span className="inline-block h-3 w-3 rounded" style={{ background: s.color }} />
            {s.label}: <strong>{s.value}</strong>
          </li>
        ))}
      </ul>
    </div>
  );
}
