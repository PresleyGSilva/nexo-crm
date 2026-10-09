"use client";

import { useAuth } from "@/lib/auth";

export function Topbar({ title }: { title: string }) {
  const { orgId } = useAuth();
  return (
    <header className="flex items-center justify-between border-b border-slate-200 bg-white px-6 py-3 dark:border-slate-800 dark:bg-slate-900">
      <h1 className="text-lg font-semibold">{title}</h1>
      <span className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-300">
        Org: {orgId || "—"}
      </span>
    </header>
  );
}

export function LoadingState({ label = "Carregando..." }: { label?: string }) {
  return (
    <div className="flex flex-col gap-2 p-6" aria-busy="true">
      <div className="skeleton h-8 w-1/3" />
      <div className="skeleton h-24 w-full" />
      <div className="skeleton h-24 w-full" />
      <span className="text-sm text-slate-500">{label}</span>
    </div>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="card m-6 border-red-200 text-center">
      <p className="font-semibold text-red-600">Algo deu errado</p>
      <p className="mt-1 text-sm text-slate-500">{message}</p>
      {onRetry ? (
        <button className="btn mt-3" onClick={onRetry} type="button">
          Tentar novamente
        </button>
      ) : null}
    </div>
  );
}

export function EmptyState({ message }: { message: string }) {
  return (
    <div className="card m-6 text-center text-sm text-slate-500">{message}</div>
  );
}
