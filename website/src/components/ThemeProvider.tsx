import { createContext, useContext, useEffect, useMemo, useState } from "react";

import {
  defaultThemePalette,
  isThemePalette,
  type ThemePalette
} from "@/lib/theme-palettes";

export type Theme = "light" | "dark" | "system";

type ThemeContextValue = {
  theme: Theme;
  resolvedTheme: "light" | "dark";
  palette: ThemePalette;
  setTheme: (theme: Theme) => void;
  setPalette: (palette: ThemePalette) => void;
};

const ThemeContext = createContext<ThemeContextValue>({
  theme: "system",
  resolvedTheme: "light",
  palette: defaultThemePalette,
  setTheme: () => {},
  setPalette: () => {}
});

// 与主项目共用同一组 localStorage key，便于官网与后台保持一致
const storageKey = "skyimage-theme";
const paletteStorageKey = "skyimage-theme-palette";

function readStoredTheme(): Theme {
  if (typeof window === "undefined") return "system";
  const value = window.localStorage.getItem(storageKey);
  if (value === "light" || value === "dark" || value === "system") {
    return value;
  }
  return "system";
}

function readStoredPalette(): ThemePalette {
  if (typeof window === "undefined") return defaultThemePalette;
  return isThemePalette(window.localStorage.getItem(paletteStorageKey))
    ? (window.localStorage.getItem(paletteStorageKey) as ThemePalette)
    : defaultThemePalette;
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(readStoredTheme);
  const [palette, setPaletteState] = useState<ThemePalette>(readStoredPalette);

  const [systemTheme, setSystemTheme] = useState<"light" | "dark">(() => {
    if (typeof window === "undefined") return "light";
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const listener = (event: MediaQueryListEvent) => {
      setSystemTheme(event.matches ? "dark" : "light");
    };
    if (typeof media.addEventListener === "function") {
      media.addEventListener("change", listener);
      return () => media.removeEventListener("change", listener);
    }
    media.addListener(listener);
    return () => media.removeListener(listener);
  }, []);

  const resolvedTheme = theme === "system" ? systemTheme : theme;

  useEffect(() => {
    if (typeof window === "undefined") return;
    const root = document.documentElement;
    root.classList.toggle("dark", resolvedTheme === "dark");
    root.style.colorScheme = resolvedTheme;
  }, [resolvedTheme]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    document.documentElement.dataset.palette = palette;
  }, [palette]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      theme,
      resolvedTheme,
      palette,
      setTheme: (next) => {
        setThemeState(next);
        if (typeof window !== "undefined") {
          window.localStorage.setItem(storageKey, next);
        }
      },
      setPalette: (next) => {
        setPaletteState(next);
        if (typeof window !== "undefined") {
          window.localStorage.setItem(paletteStorageKey, next);
        }
      }
    }),
    [theme, resolvedTheme, palette]
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  return useContext(ThemeContext);
}
