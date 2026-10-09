"use client";

import { useEffect, useState } from "react";
import AppShell from "@/components/AppShell";
import { EmptyState, ErrorState, LoadingState } from "@/components/ui";
import { BarChart, DonutChart, KpiCard } from "@/components/widgets";
import { api, formatBRL, unwrap } from "@/lib/api";
import { MOCK_SUMMARY } from "@/lib/mock";
import type { DashboardSummary } from "@/lib/types";

export default function DashboardPage() {
  const [data, setData] = useState<DashboardSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [fallback, setFallback] = useState(false);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const raw = await api<unknown>("/dashboard/summary");
      setData(unwrap<DashboardSummary>(raw));
      setFallback(false);
    } catch (err) {
      setData(MOCK_SUMMARY);
      setFallback(true);
      if (err instanceof Error && err.message.includes("Failed to fetch")) {
        setError("");
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  if (loading) {
    return (
      <AppShell title="Dashboard">
        <LoadingState />
      </AppShell>
    );
  }

  if (!data) {
    return (
      <AppShell title="Dashboard">
        <ErrorState message={error || "Sem dados"} onRetry={load} />
      </AppShell>
    );
  }

  const stages = data.deals_by_stage || [];
  const months = data.contacts_per_month || [];

  return (
    <AppShell title="Dashboard">
      {fallback ? (
        <p className="mb-4 rounded-lg bg-amber-100 px-3 py-2 text-sm text-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
          API indisponível — exibindo dados de demonstração.
        </p>
      ) : null}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard label="Contatos" value={String(data.total_contacts)} />
        <KpiCard label="Deals abertos" value={String(data.open_deals)} />
        <KpiCard label="Pipeline" value={formatBRL(data.pipeline_value)} />
        <KpiCard label="Tarefas vencendo" value={String(data.tasks_due)} />
      </div>
      <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="card">
          <h2 className="mb-3 font-semibold">Contatos por mês</h2>
          {months.length ? (
            <BarChart data={months.map((m) => ({ label: m.month, value: m.count }))} />
          ) : (
            <EmptyState message="Sem dados de contatos." />
          )}
        </div>
        <div className="card">
          <h2 className="mb-3 font-semibold">Deals por etapa</h2>
          {stages.length ? (
            <DonutChart data={stages.map((s) => ({ label: s.stage, value: s.count }))} />
          ) : (
            <EmptyState message="Sem deals no funil." />
          )}
        </div>
      </div>
    </AppShell>
  );
}
