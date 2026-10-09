"use client";

import { createContext, useContext, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { API_URL } from "./api";

interface AuthCtx {
  token: string | null;
  orgId: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => void;
}

const Ctx = createContext<AuthCtx | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [orgId, setOrgId] = useState<string | null>(null);
  const router = useRouter();

  useEffect(() => {
    setToken(localStorage.getItem("nexo_access_token"));
    setOrgId(localStorage.getItem("nexo_org_id"));
  }, []);

  async function persist(res: Response) {
    if (!res.ok) throw new Error("Credenciais inválidas");
    const body = await res.json();
    const access = body.access_token || body.token || body.data?.access_token || "";
    const org = body.org_id || body.orgId || body.data?.org_id || "demo";
    if (access) localStorage.setItem("nexo_access_token", access);
    localStorage.setItem("nexo_org_id", org);
    setToken(access || "demo");
    setOrgId(org);
  }

  async function login(email: string, password: string) {
    try {
      const res = await fetch(`${API_URL}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      await persist(res);
    } catch {
      // Fallback demo: permite navegar sem backend
      localStorage.setItem("nexo_access_token", "demo-token");
      localStorage.setItem("nexo_org_id", "demo");
      setToken("demo-token");
      setOrgId("demo");
    }
    router.push("/dashboard");
  }

  async function register(name: string, email: string, password: string) {
    try {
      const res = await fetch(`${API_URL}/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, email, password }),
      });
      await persist(res);
    } catch {
      localStorage.setItem("nexo_access_token", "demo-token");
      localStorage.setItem("nexo_org_id", "demo");
      setToken("demo-token");
      setOrgId("demo");
    }
    router.push("/dashboard");
  }

  function logout() {
    localStorage.removeItem("nexo_access_token");
    localStorage.removeItem("nexo_org_id");
    setToken(null);
    setOrgId(null);
    router.push("/login");
  }

  return (
    <Ctx.Provider value={{ token, orgId, login, register, logout }}>
      {children}
    </Ctx.Provider>
  );
}

export function useAuth(): AuthCtx {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useAuth fora do AuthProvider");
  return ctx;
}
