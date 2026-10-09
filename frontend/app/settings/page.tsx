"use client";

import { useEffect, useState } from "react";
import AppShell from "@/components/AppShell";
import { LoadingState } from "@/components/ui";
import { useTheme } from "@/components/ThemeProvider";
import { api } from "@/lib/api";

export default function SettingsPage() {
  const { dark, toggle } = useTheme();
  const [orgName, setOrgName] = useState("Organização Demo");
  const [loading, setLoading] = useState(true);
  const [saved, setSaved] = useState("");
  const [fallback, setFallback] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const s = await api<{ name?: string }>("/settings");
        if (typeof s.name === "string" && s.name) setOrgName(s.name);
      } catch {
        setFallback(true);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  async function onSave(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api("/settings", { method: "PATCH", body: JSON.stringify({ name: orgName }) });
      setSaved("Configurações salvas.");
    } catch {
      setFallback(true);
      setSaved("API indisponível — salvo apenas localmente.");
    }
  }

  return (
    <AppShell title="Configurações">
      <div className="card max-w-lg">
        {loading ? (
          <LoadingState label="Carregando configurações..." />
        ) : (
          <form onSubmit={onSave} className="flex flex-col gap-3">
            {fallback ? (
              <p className="rounded-lg bg-amber-100 px-3 py-2 text-sm text-amber-800 dark:bg-amber-900/30 dark:text-amber-200">
                API indisponível — modo demonstração.
              </p>
            ) : null}
            <label className="flex flex-col gap-1 text-sm font-medium">
              Nome da organização
              <input className="input" value={orgName} onChange={(e) => setOrgName(e.target.value)} />
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={dark} onChange={toggle} className="h-4 w-4" />
              Modo escuro
            </label>
            <button className="btn w-fit" type="submit">Salvar</button>
            {saved ? <p className="text-sm text-green-600">{saved}</p> : null}
          </form>
        )}
      </div>
    </AppShell>
  );
}
