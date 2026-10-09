export const API_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

function headers(): HeadersInit {
  const h: Record<string, string> = { "Content-Type": "application/json" };
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("nexo_access_token");
    const orgId = localStorage.getItem("nexo_org_id");
    if (token) h["Authorization"] = `Bearer ${token}`;
    if (orgId) h["X-Org-ID"] = orgId;
  }
  return h;
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: { ...headers(), ...(init?.headers || {}) },
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`API ${res.status}: ${body || res.statusText}`);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

function isEnvelope(value: unknown): value is { data: unknown } {
  return (
    typeof value === "object" &&
    value !== null &&
    "data" in value
  );
}

/** Unwrap {data: ...} envelopes or plain payloads. */
export function unwrap<T>(payload: unknown): T {
  if (isEnvelope(payload)) return payload.data as T;
  return payload as T;
}

export function formatBRL(value: number): string {
  return new Intl.NumberFormat("pt-BR", {
    style: "currency",
    currency: "BRL",
    maximumFractionDigits: 0,
  }).format(value || 0);
}
