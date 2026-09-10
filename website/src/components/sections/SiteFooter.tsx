import { FILINGS, SITE } from "@/data/site";
import { format, useI18n } from "@/i18n";

export function SiteFooter() {
  const { copy } = useI18n();

  return (
    <footer className="mt-16 border-t bg-card/60">
      <div className="container mx-auto max-w-6xl px-4 py-6 sm:px-8">
        <div className="flex flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
          <p className="font-medium text-foreground">{SITE.name}</p>
          <p>{copy.tagline}</p>
        </div>

        <div className="my-4 h-px bg-border" />

        <div className="flex flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
          <p>{format(copy.footer.copyright, { year: new Date().getFullYear(), license: SITE.license })}</p>
          <div className="flex flex-wrap gap-4">
            <a
              href={SITE.github}
              target="_blank"
              rel="noreferrer"
              className="press -mx-1.5 inline-block rounded px-1.5 py-0.5 hover:text-foreground"
            >
              {copy.footer.github}
            </a>
            <a
              href={`${SITE.github}/blob/main/LICENSE`}
              target="_blank"
              rel="noreferrer"
              className="press -mx-1.5 inline-block rounded px-1.5 py-0.5 hover:text-foreground"
            >
              {copy.footer.license}
            </a>
            <a
              href={SITE.demo}
              target="_blank"
              rel="noreferrer"
              className="press -mx-1.5 inline-block rounded px-1.5 py-0.5 hover:text-foreground"
            >
              {copy.footer.demo}
            </a>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
          {FILINGS.map((filing) => (
            <a
              key={filing.label}
              href={filing.href}
              target="_blank"
              rel="noreferrer"
              className="press -mx-1.5 inline-block rounded px-1.5 py-0.5 hover:text-foreground"
            >
              {filing.label}
            </a>
          ))}
        </div>
      </div>
    </footer>
  );
}
