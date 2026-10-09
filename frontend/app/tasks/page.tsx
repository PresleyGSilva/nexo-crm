"use client";

import { useEffect, useState } from "react";
import AppShell from "@/components/AppShell";
import { EmptyState, LoadingState } from "@/components/ui";
import { api, unwrap } from "@/lib/api";
import { MOCK_TASKS } from "@/lib/mock";
import type { Task } from "@/lib/types";

export default function TasksPage() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [fallback, setFallback] = useState(false);
  const [title, setTitle] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const raw = await api<unknown>("/tasks");
        const list = unwrap<Task[] | { items: Task[] }>(raw);
        setTasks(Array.isArray(list) ? list : list.items);
      } catch {
        setTasks(MOCK_TASKS);
        setFallback(true);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  async function toggle(id: string, done: boolean) {
    setTasks((prev) => prev.map((t) => (t.id === id ? { ...t, done } : t)));
    try {
      await api(`/tasks/${id}`, { method: "PATCH", body: JSON.stringify({ done }) });
    } catch {
      setFallback(true);
    }
  }

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    try {
      const raw = await api<unknown>("/tasks", { method: "POST", body: JSON.stringify({ title }) });
      setTasks((prev) => [unwrap<Task>(raw), ...prev]);
    } catch {
      setTasks((prev) => [{ id: `local-${Date.now()}`, title, done: false }, ...prev]);
      setFallback(true);
    }
    setTitle("");
  }

  return (
    <AppShell title="Tarefas">
      {fallback ? (
        <p className="mb-4 rounded-lg bg-amber-100 px-3 py-2 text-sm text-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
          API indisponível — alterações ficam apenas locais.
        </p>
      ) : null}
      <form onSubmit={onCreate} className="card mb-4 flex gap-2">
        <input className="input" placeholder="Nova tarefa..." value={title} onChange={(e) => setTitle(e.target.value)} />
        <button className="btn" type="submit">Adicionar</button>
      </form>
      <div className="card">
        {loading ? (
          <LoadingState label="Carregando tarefas..." />
        ) : tasks.length === 0 ? (
          <EmptyState message="Nenhuma tarefa. Crie a primeira acima." />
        ) : (
          <ul className="flex flex-col divide-y divide-slate-100 dark:divide-slate-800">
            {tasks.map((t) => (
              <li key={t.id} className="flex items-center gap-3 py-2">
                <input type="checkbox" checked={t.done} onChange={(e) => toggle(t.id, e.target.checked)} className="h-4 w-4" />
                <span className={`flex-1 text-sm ${t.done ? "text-slate-400 line-through" : ""}`}>{t.title}</span>
                {t.due_date ? <span className="text-xs text-slate-400">{new Date(t.due_date).toLocaleDateString("pt-BR")}</span> : null}
              </li>
            ))}
          </ul>
        )}
      </div>
    </AppShell>
  );
}
