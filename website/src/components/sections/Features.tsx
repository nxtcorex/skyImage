import { FEATURE_ICONS } from "@/data/site";
import { useI18n } from "@/i18n";
import { Card, CardContent } from "@/components/ui/card";

export function Features() {
  const { copy } = useI18n();

  return (
    <section id="features" className="container mx-auto max-w-6xl scroll-mt-20 px-4 pt-16 sm:px-8">
      <div className="animate-enter animate-enter-3 space-y-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{copy.features.title}</h2>
          <p className="max-w-2xl text-muted-foreground">{copy.features.desc}</p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {copy.features.items.map((feature, index) => {
            const Icon = FEATURE_ICONS[index];
            return (
              <Card key={feature.title} className="transition-shadow duration-300 hover:shadow-lg">
                <CardContent className="flex h-full flex-col items-center gap-3 p-6 text-center">
                  <span className="inline-flex h-12 w-12 items-center justify-center rounded-xl border bg-secondary text-secondary-foreground">
                    <Icon className="h-6 w-6" />
                  </span>
                  <p className="text-sm font-semibold">{feature.title}</p>
                  <p className="text-sm text-muted-foreground">{feature.desc}</p>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>
    </section>
  );
}
