"use client";

import { useEffect, useState } from "react";
import AppShell from "@/components/AppShell";
import { EmptyState, LoadingState } from "@/components/ui";
import { api, unwrap } from "@/lib/api";
import { MOCK_AUDIT } from "@/lib/mock";
import type { AuditLog } from "@/lib/types";

export default function AuditPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [fallback, setFallback] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const raw = await api<unknown>("/audit-logs");
        const list = unwrap<AuditLog[] | { items: AuditLog[] }>(raw);
        setLogs(Array.isArray(list) ? list : list.items);
      } catch {
        setLogs(MOCK_AUDIT);
        setFallback(true);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return (
    <AppShell title="Auditoria">
      {fallback ? (
        <p className="mb-4 rounded-lg bg-amber-100 px-3 py-2 text-sm text-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
          API indisponível — exibindo logs de demonstração.
        </p>
      ) : null}
      <div className="card">
        {loading ? (
          <LoadingState label="Carregando logs..." />
        ) : logs.length === 0 ? (
          <EmptyState message="Nenhum evento registrado." />
        ) : (
          <table className="table w-full">
            <thead>
              <tr>
                <th>Ação</th>
                <th>Entidade</th>
                <th>Data</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((l) => (
                <tr key={l.id}>
                  <td className="font-medium">{l.action}</td>
                  <td>{l.entity}{l.entity_id ? ` #${l.entity_id}` : ""}</td>
                  <td>{l.created_at ? new Date(l.created_at).toLocaleString("pt-BR") : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </AppShell>
  );
}
