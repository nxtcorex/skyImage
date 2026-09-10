import { STACK_BACKEND, STACK_DATABASE, STACK_FRONTEND, STACK_GROUP_ICONS } from "@/data/site";
import { useI18n } from "@/i18n";
import { Card, CardContent } from "@/components/ui/card";

export function TechStack() {
  const { copy } = useI18n();
  const groups = [STACK_BACKEND, STACK_FRONTEND, STACK_DATABASE];

  return (
    <section id="stack" className="container mx-auto max-w-6xl scroll-mt-20 px-4 pt-16 sm:px-8">
      <div className="animate-enter animate-enter-3 space-y-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{copy.stack.title}</h2>
          <p className="max-w-2xl text-muted-foreground">{copy.stack.desc}</p>
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          {copy.stack.groups.map((label, index) => {
            const Icon = STACK_GROUP_ICONS[index];
            return (
              <Card key={label}>
                <CardContent className="space-y-4 p-6">
                  <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    <Icon className="h-4 w-4" />
                    {label}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {groups[index].map((item) => (
                      <span
                        key={item}
                        className="rounded-md border bg-secondary px-2.5 py-1 text-[13px] font-medium text-secondary-foreground"
                      >
                        {item}
                      </span>
                    ))}
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>
    </section>
  );
}
