import { SCENE_ICONS, STEP_ICONS } from "@/data/site";
import { useI18n } from "@/i18n";
import { Card, CardContent } from "@/components/ui/card";

export function Steps() {
  const { copy } = useI18n();

  return (
    <section className="container mx-auto max-w-6xl px-4 pt-16 sm:px-8">
      <div className="animate-enter animate-enter-3 space-y-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{copy.steps.title}</h2>
          <p className="text-muted-foreground">{copy.steps.desc}</p>
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          {copy.steps.items.map((step, index) => {
            const Icon = STEP_ICONS[index];
            return (
              <Card key={step.title} className="transition-shadow duration-300 hover:shadow-lg">
                <CardContent className="space-y-3 p-6">
                  <div className="flex items-center justify-between">
                    <span className="inline-flex h-10 w-10 items-center justify-center rounded-lg border bg-secondary text-secondary-foreground">
                      <Icon className="h-5 w-5" />
                    </span>
                    <span className="text-sm font-semibold text-muted-foreground">
                      0{index + 1}
                    </span>
                  </div>
                  <p className="text-base font-medium">{step.title}</p>
                  <p className="text-sm text-muted-foreground">{step.desc}</p>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>
    </section>
  );
}

export function Scenes() {
  const { copy } = useI18n();

  return (
    <section className="container mx-auto max-w-6xl px-4 pt-16 sm:px-8">
      <div className="animate-enter animate-enter-3 space-y-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{copy.scenes.title}</h2>
          <p className="text-muted-foreground">{copy.scenes.desc}</p>
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          {copy.scenes.items.map((scene, index) => {
            const Icon = SCENE_ICONS[index];
            return (
              <Card key={scene.text} className="transition-shadow duration-300 hover:shadow-lg">
                <CardContent className="flex items-center gap-4 p-6">
                  <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-lg border bg-secondary text-secondary-foreground">
                    <Icon className="h-5 w-5" />
                  </span>
                  <p className="text-sm text-muted-foreground">{scene.text}</p>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>
    </section>
  );
}
