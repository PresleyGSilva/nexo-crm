"use client";

import { useEffect, useState } from "react";
import AppShell from "@/components/AppShell";
import { EmptyState, ErrorState, LoadingState } from "@/components/ui";
import { api, unwrap } from "@/lib/api";
import { MOCK_CONTACTS } from "@/lib/mock";
import type { Contact } from "@/lib/types";

const PER_PAGE = 10;

export default function ContactsPage() {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [q, setQ] = useState("");
  const [debounced, setDebounced] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [fallback, setFallback] = useState(false);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");

  useEffect(() => {
    const t = setTimeout(() => {
      setDebounced(q);
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [q]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError("");
      try {
        const params = new URLSearchParams({
          q: debounced,
          page: String(page),
          per_page: String(PER_PAGE),
        });
        const raw = await api<unknown>(`/contacts?${params}`);
        const list = unwrap<Contact[] | { items: Contact[] }>(raw);
        if (!cancelled) {
          setContacts(Array.isArray(list) ? list : list.items);
          setFallback(false);
        }
      } catch {
        if (!cancelled) {
          const filtered = MOCK_CONTACTS.filter(
            (c) =>
              !debounced ||
              c.name.toLowerCase().includes(debounced.toLowerCase()) ||
              c.email.toLowerCase().includes(debounced.toLowerCase())
          );
          setContacts(filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE));
          setFallback(true);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, [debounced, page]);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name || !email) return;
    setCreating(true);
    try {
      const raw = await api<unknown>("/contacts", {
        method: "POST",
        body: JSON.stringify({ name, email }),
      });
      const created = unwrap<Contact>(raw);
      setContacts((prev) => [created, ...prev]);
      setName("");
      setEmail("");
    } catch {
      setContacts((prev) => [{ id: `local-${Date.now()}`, name, email }, ...prev]);
      setFallback(true);
      setName("");
      setEmail("");
    } finally {
      setCreating(false);
    }
  }

  return (
    <AppShell title="Contatos">
      {fallback ? (
        <p className="mb-4 rounded-lg bg-amber-100 px-3 py-2 text-sm text-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
          API indisponível — dados locais de demonstração.
        </p>
      ) : null}
      <form onSubmit={onCreate} className="card mb-4 flex flex-col gap-2 sm:flex-row">
        <input className="input" placeholder="Nome" value={name} onChange={(e) => setName(e.target.value)} />
        <input className="input" placeholder="E-mail" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        <button className="btn" disabled={creating} type="submit">
          {creating ? "Salvando..." : "Adicionar"}
        </button>
      </form>
      <div className="card">
        <input
          className="input mb-3 max-w-sm"
          placeholder="Buscar por nome ou e-mail..."
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        {loading ? (
          <LoadingState label="Carregando contatos..." />
        ) : error ? (
          <ErrorState message={error} />
        ) : contacts.length === 0 ? (
          <EmptyState message="Nenhum contato encontrado." />
        ) : (
          <>
            <table className="table w-full">
              <thead>
                <tr>
                  <th>Nome</th>
                  <th>E-mail</th>
                  <th>Empresa</th>
                  <th>Telefone</th>
                </tr>
              </thead>
              <tbody>
                {contacts.map((c) => (
                  <tr key={c.id}>
                    <td className="font-medium">{c.name}</td>
                    <td>{c.email}</td>
                    <td>{c.company || "—"}</td>
                    <td>{c.phone || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="mt-3 flex items-center gap-2">
              <button className="btn-secondary text-xs" disabled={page <= 1} onClick={() => setPage((p) => p - 1)} type="button">
                Anterior
              </button>
              <span className="text-sm">Página {page}</span>
              <button
                className="btn-secondary text-xs"
                disabled={contacts.length < PER_PAGE}
                onClick={() => setPage((p) => p + 1)}
                type="button"
              >
                Próxima
              </button>
            </div>
          </>
        )}
      </div>
    </AppShell>
  );
}
