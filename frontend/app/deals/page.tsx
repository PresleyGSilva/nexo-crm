"use client";

import { useEffect, useState } from "react";
import AppShell from "@/components/AppShell";
import { EmptyState, LoadingState } from "@/components/ui";
import { api, formatBRL, unwrap } from "@/lib/api";
import { MOCK_DEALS } from "@/lib/mock";
import { DEAL_STAGES, type Deal } from "@/lib/types";

export default function DealsPage() {
  const [deals, setDeals] = useState<Deal[]>([]);
  const [loading, setLoading] = useState(true);
  const [fallback, setFallback] = useState(false);
  const [dragId, setDragId] = useState<string | null>(null);
  const [title, setTitle] = useState("");
  const [value, setValue] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const raw = await api<unknown>("/deals");
        const list = unwrap<Deal[] | { items: Deal[] }>(raw);
        setDeals(Array.isArray(list) ? list : list.items);
      } catch {
        setDeals(MOCK_DEALS);
        setFallback(true);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  async function moveStage(id: string, stage: string) {
    setDeals((prev) => prev.map((d) => (d.id === id ? { ...d, stage } : d)));
    try {
      await api(`/deals/${id}/stage`, {
        method: "PATCH",
        body: JSON.stringify({ stage }),
      });
    } catch {
      setFallback(true);
    }
  }

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!title) return;
    const payload = { title, value: Number(value) || 0, stage: "lead" };
    try {
      const raw = await api<unknown>("/deals", { method: "POST", body: JSON.stringify(payload) });
      setDeals((prev) => [unwrap<Deal>(raw), ...prev]);
    } catch {
      setDeals((prev) => [{ id: `local-${Date.now()}`, ...payload }, ...prev]);
      setFallback(true);
    }
    setTitle("");
    setValue("");
  }

  return (
    <AppShell title="Deals — Kanban">
      {fallback ? (
        <p className="mb-4 rounded-lg bg-amber-100 px-3 py-2 text-sm text-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
          API indisponível — alterações ficam apenas locais.
        </p>
      ) : null}
      <form onSubmit={onCreate} className="card mb-4 flex flex-col gap-2 sm:flex-row">
        <input className="input" placeholder="Título do deal" value={title} onChange={(e) => setTitle(e.target.value)} />
        <input className="input" placeholder="Valor (R$)" inputMode="numeric" value={value} onChange={(e) => setValue(e.target.value)} />
        <button className="btn" type="submit">Novo deal</button>
      </form>
      {loading ? (
        <div className="card"><LoadingState label="Carregando funil..." /></div>
      ) : deals.length === 0 ? (
        <div className="card"><EmptyState message="Nenhum deal no funil." /></div>
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-3 xl:grid-cols-6">
          {DEAL_STAGES.map((stage) => (
            <div
              key={stage}
              className="card min-h-40"
              onDragOver={(e) => e.preventDefault()}
              onDrop={() => {
                if (dragId) {
                  moveStage(dragId, stage);
                  setDragId(null);
                }
              }}
            >
              <h2 className="mb-2 text-xs font-bold uppercase tracking-wide text-slate-500">{stage}</h2>
              <div className="flex flex-col gap-2">
                {deals
                  .filter((d) => d.stage === stage)
                  .map((d) => (
                    <div
                      key={d.id}
                      draggable
                      onDragStart={() => setDragId(d.id)}
                      className="cursor-grab rounded-lg border border-slate-200 bg-slate-50 p-2 text-sm active:cursor-grabbing dark:border-slate-700 dark:bg-slate-800"
                    >
                      <p className="font-medium">{d.title}</p>
                      <p className="text-xs text-slate-500">{formatBRL(d.value)}</p>
                      {d.contact_name ? <p className="text-xs text-slate-400">{d.contact_name}</p> : null}
                    </div>
                  ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </AppShell>
  );
}
