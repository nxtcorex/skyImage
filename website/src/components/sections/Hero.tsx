import { Github } from "lucide-react";

import { SITE } from "@/data/site";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { DemoDialogButton } from "@/components/DemoAccessDialog";

export function Hero() {
  const { copy } = useI18n();

  return (
    <section id="top" className="relative py-16 sm:py-24">
      {/* 纯灰阶柔和光晕，不使用任何蓝紫渐变 */}
      <div className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(ellipse_60%_50%_at_50%_0%,hsl(var(--foreground)/0.05),transparent_70%)]" />

      <div className="container mx-auto flex max-w-6xl flex-col items-center gap-6 px-4 text-center sm:px-8">
        <div className="animate-enter animate-enter-1 flex flex-col items-center gap-6">
          <h1 className="max-w-3xl text-4xl font-semibold leading-[1.1] tracking-tight sm:text-5xl lg:text-6xl">
            {copy.hero.slogan}
          </h1>

          <p className="max-w-2xl text-base text-muted-foreground sm:text-lg">{copy.hero.intro}</p>

          <div className="flex flex-wrap items-center justify-center gap-3">
            <DemoDialogButton label={copy.hero.demo} />
            <Button asChild size="lg" variant="outline" className="gap-2">
              <a href={SITE.github} target="_blank" rel="noreferrer">
                <Github className="h-4 w-4" />
                {copy.hero.source}
              </a>
            </Button>
          </div>
        </div>

        <dl className="animate-enter animate-enter-2 mt-8 grid w-full max-w-2xl grid-cols-2 gap-px overflow-hidden rounded-xl border bg-border sm:grid-cols-4">
          {copy.hero.stats.map((stat) => (
            <div key={stat.label} className="bg-card px-4 py-4 text-left">
              <dt className="text-xl font-semibold tracking-tight">{stat.value}</dt>
              <dd className="text-xs text-muted-foreground">{stat.label}</dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}
