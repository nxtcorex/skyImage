import { Github } from "lucide-react";

import { SITE } from "@/data/site";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { DemoDialogButton } from "@/components/DemoAccessDialog";

export function CtaBand() {
  const { copy } = useI18n();

  return (
    <section className="container mx-auto max-w-6xl px-4 pt-16 sm:px-8">
      <div className="animate-enter animate-enter-4">
        <Card className="relative overflow-hidden">
          <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_60%_100%_at_50%_0%,hsl(var(--foreground)/0.05),transparent_65%)]" />
          <CardContent className="relative flex flex-col items-center gap-4 px-6 py-12 text-center sm:py-16">
            <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{copy.cta.title}</h2>
            <p className="max-w-xl text-muted-foreground">{copy.cta.desc}</p>
            <div className="mt-2 flex flex-wrap items-center justify-center gap-3">
              <DemoDialogButton label={copy.cta.demo} />
              <Button asChild size="lg" variant="outline" className="gap-2">
                <a href={SITE.github} target="_blank" rel="noreferrer">
                  <Github className="h-4 w-4" />
                  {copy.cta.github}
                </a>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </section>
  );
}
