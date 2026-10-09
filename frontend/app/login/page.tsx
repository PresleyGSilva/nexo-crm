"use client";

import { useState } from "react";
import Link from "next/link";
import { useAuth } from "@/lib/auth";

export default function LoginPage() {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await login(email, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha no login");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <form onSubmit={onSubmit} className="card w-full max-w-sm">
        <h1 className="text-xl font-bold text-brand-500">Nexo CRM</h1>
        <p className="mt-1 text-sm text-slate-500">Entre na sua conta</p>
        <div className="mt-4 flex flex-col gap-3">
          <input className="input" placeholder="E-mail" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
          <input className="input" placeholder="Senha" type="password" required value={password} onChange={(e) => setPassword(e.target.value)} />
          {error ? <p className="text-sm text-red-600">{error}</p> : null}
          <button className="btn" disabled={busy} type="submit">
            {busy ? "Entrando..." : "Entrar"}
          </button>
          <p className="text-center text-sm text-slate-500">
            Sem conta? <Link href="/register" className="font-semibold text-brand-500">Cadastre-se</Link>
          </p>
        </div>
      </form>
    </div>
  );
}
