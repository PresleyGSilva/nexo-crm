"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { useTheme } from "./ThemeProvider";

const NAV = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/contacts", label: "Contatos" },
  { href: "/deals", label: "Deals" },
  { href: "/tasks", label: "Tarefas" },
  { href: "/audit", label: "Auditoria" },
  { href: "/settings", label: "Configurações" },
];

export function Sidebar() {
  const pathname = usePathname();
  const { logout } = useAuth();
  const { dark, toggle } = useTheme();

  return (
    <aside className="flex w-56 shrink-0 flex-col border-r border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
      <div className="px-4 py-5 text-xl font-bold text-brand-500">Nexo CRM</div>
      <nav className="flex flex-1 flex-col gap-1 px-2">
        {NAV.map((item) => {
          const active = pathname === item.href || pathname?.startsWith(item.href + "/");
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`rounded-lg px-3 py-2 text-sm font-medium transition ${
                active
                  ? "bg-brand-50 text-brand-600 dark:bg-slate-800 dark:text-white"
                  : "text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="flex flex-col gap-2 border-t border-slate-200 p-3 dark:border-slate-800">
        <button onClick={toggle} className="btn-secondary text-xs" type="button">
          {dark ? "☀️ Modo claro" : "🌙 Modo escuro"}
        </button>
        <button onClick={logout} className="btn-secondary text-xs" type="button">
          Sair
        </button>
      </div>
    </aside>
  );
}
