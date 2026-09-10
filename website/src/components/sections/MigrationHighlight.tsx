import { ArrowRight, Check } from "lucide-react";

import { SITE } from "@/data/site";
import { useI18n } from "@/i18n";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export function MigrationHighlight() {
  const { copy } = useI18n();

  return (
    <section id="migration" className="container mx-auto max-w-6xl scroll-mt-20 px-4 pt-16 sm:px-8">
      <div className="animate-enter animate-enter-4">
        <Card className="relative overflow-hidden">
          <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_70%_120%_at_0%_0%,hsl(var(--foreground)/0.045),transparent_60%)]" />
          <CardContent className="relative grid items-center gap-6 p-8 md:p-10 lg:grid-cols-2">
            <div className="space-y-4">
              <Badge variant="secondary" className="w-fit">
                {copy.migration.badge}
              </Badge>
              <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
                {copy.migration.title}
              </h2>
              <p className="text-muted-foreground">{copy.migration.desc}</p>
              <Button asChild className="gap-2">
                <a href={`${SITE.github}#readme`} target="_blank" rel="noreferrer">
                  {copy.migration.cta}
                  <ArrowRight className="h-4 w-4" />
                </a>
              </Button>
            </div>

            <ul className="flex flex-col gap-3">
              {copy.migration.points.map((point) => (
                <li key={point} className="flex items-start gap-2.5 text-sm text-muted-foreground">
                  <Check className="mt-0.5 h-4 w-4 shrink-0 text-foreground/70" />
                  <span>{point}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </div>
    </section>
  );
}
