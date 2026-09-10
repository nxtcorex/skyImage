import { Github } from "lucide-react";

import { SITE } from "@/data/site";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { LanguageToggle } from "@/components/LanguageToggle";
import { PaletteToggle } from "@/components/PaletteToggle";
import { ThemeToggle } from "@/components/ThemeToggle";
import { BrandMark } from "@/components/BrandMark";

export function SiteHeader() {
  const { copy } = useI18n();

  return (
    <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/70 backdrop-blur-md">
      <div className="container mx-auto flex w-full max-w-6xl items-center justify-between py-4">
        <a href="#top" className="flex items-center gap-2.5 focus-visible:outline-none">
          <span className="inline-flex h-8 w-8 items-center justify-center rounded-md border bg-secondary text-secondary-foreground">
            <BrandMark className="h-[18px] w-[18px]" />
          </span>
          <span className="flex flex-col leading-tight">
            <span className="text-[17px] font-semibold tracking-tight">{SITE.name}</span>
            <span className="hidden text-xs text-muted-foreground sm:block">{copy.tagline}</span>
          </span>
        </a>

        <nav className="hidden items-center gap-0.5 lg:flex">
          {copy.nav.items.map((item) => (
            <a
              key={item.href}
              href={item.href}
              className="inline-flex h-8 items-center rounded-md px-3 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
            >
              {item.label}
            </a>
          ))}
        </nav>

        <div className="flex items-center gap-2">
          <Button asChild variant="ghost" size="icon" className="text-muted-foreground">
            <a href={SITE.github} target="_blank" rel="noreferrer" aria-label={copy.nav.github} title={copy.nav.github}>
              <Github className="h-4 w-4" />
            </a>
          </Button>
          <LanguageToggle iconOnly />
          <PaletteToggle iconOnly />
          <ThemeToggle iconOnly />
          <Button asChild size="sm" className="hidden sm:inline-flex">
            <a href={SITE.demo} target="_blank" rel="noreferrer">
              {copy.nav.demo}
            </a>
          </Button>
        </div>
      </div>
    </header>
  );
}
