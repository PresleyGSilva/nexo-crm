"use client";

import { createContext, useContext, useEffect, useState } from "react";

interface ThemeCtx {
  dark: boolean;
  toggle: () => void;
}

const Ctx = createContext<ThemeCtx>({ dark: false, toggle: () => {} });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem("nexo_theme");
    const isDark = saved === "dark";
    setDark(isDark);
    document.documentElement.classList.toggle("dark", isDark);
  }, []);

  function toggle() {
    setDark((d) => {
      localStorage.setItem("nexo_theme", d ? "light" : "dark");
      document.documentElement.classList.toggle("dark", !d);
      return !d;
    });
  }

  return <Ctx.Provider value={{ dark, toggle }}>{children}</Ctx.Provider>;
}

export function useTheme() {
  return useContext(Ctx);
}
