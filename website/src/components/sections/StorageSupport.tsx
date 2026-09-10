import { CheckCircle2, CircleDashed } from "lucide-react";

import { useI18n } from "@/i18n";
import { Card, CardContent } from "@/components/ui/card";

export function StorageSupport() {
  const { copy } = useI18n();

  return (
    <section id="storage" className="container mx-auto max-w-6xl scroll-mt-20 px-4 pt-16 sm:px-8">
      <div className="animate-enter animate-enter-3 space-y-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{copy.storage.title}</h2>
          <p className="max-w-2xl text-muted-foreground">{copy.storage.desc}</p>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardContent className="space-y-4 p-6">
              <div className="flex items-center gap-2 text-[15px] font-semibold">
                <CheckCircle2 className="h-4 w-4 text-foreground/70" />
                {copy.storage.supported}
              </div>
              <div className="flex flex-wrap gap-2">
                {copy.storage.supportedItems.map((name) => (
                  <span
                    key={name}
                    className="inline-flex items-center gap-1.5 rounded-md border bg-background px-2.5 py-1 text-[13px]"
                  >
                    <span className="h-1.5 w-1.5 rounded-full bg-foreground/65" />
                    {name}
                  </span>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="space-y-4 p-6">
              <div className="flex items-center gap-2 text-[15px] font-semibold">
                <CircleDashed className="h-4 w-4 text-muted-foreground" />
                {copy.storage.untested}
              </div>
              <div className="flex flex-wrap gap-2">
                {copy.storage.untestedItems.map((name) => (
                  <span
                    key={name}
                    className="inline-flex items-center gap-1.5 rounded-md border border-dashed bg-background px-2.5 py-1 text-[13px] text-muted-foreground"
                  >
                    <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/45" />
                    {name}
                  </span>
                ))}
              </div>
              <p className="text-sm text-muted-foreground">{copy.storage.untestedNote}</p>
            </CardContent>
          </Card>
        </div>
      </div>
    </section>
  );
}
