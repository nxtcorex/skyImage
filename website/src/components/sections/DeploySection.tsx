import { useState } from "react";
import { Check, Terminal } from "lucide-react";

import { DEPLOY_POINT_ICONS } from "@/data/site";
import { useI18n } from "@/i18n";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";

type TabKey = "docker" | "binary";

export function DeploySection() {
  const { copy } = useI18n();
  const [tab, setTab] = useState<TabKey>("docker");
  const lines = copy.deploy.commands[tab];

  return (
    <section id="deploy" className="container mx-auto max-w-6xl scroll-mt-20 px-4 pt-16 sm:px-8">
      <div className="animate-enter animate-enter-4 space-y-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{copy.deploy.title}</h2>
          <p className="max-w-2xl text-muted-foreground">{copy.deploy.desc}</p>
        </div>

        <div className="grid gap-4 lg:grid-cols-[1.15fr_0.85fr]">
          <Card className="overflow-hidden">
            <div className="flex items-center justify-between gap-3 border-b bg-muted px-4 py-3">
              <span className="flex items-center gap-2 text-[13px] font-semibold">
                <Terminal className="h-4 w-4 text-muted-foreground" />
                {copy.deploy.quickStart}
              </span>
              <div className="flex gap-1">
                {(["docker", "binary"] as TabKey[]).map((key) => (
                  <button
                    key={key}
                    type="button"
                    onClick={() => setTab(key)}
                    className={cn(
                      "press rounded-md px-2.5 py-1 text-xs",
                      tab === key
                        ? "bg-background font-semibold text-foreground"
                        : "text-muted-foreground hover:bg-foreground/5 hover:text-foreground"
                    )}
                  >
                    {copy.deploy[key]}
                  </button>
                ))}
              </div>
            </div>

            <pre className="overflow-x-auto p-4 font-mono text-[13px] leading-relaxed">
              <code>
                {lines.map((line, i) =>
                  line.type === "cmt" ? (
                    <span key={i} className="block text-muted-foreground">
                      {line.text || "\u00A0"}
                    </span>
                  ) : (
                    <span key={i} className="block text-foreground/75">
                      {line.text || "\u00A0"}
                    </span>
                  )
                )}
              </code>
            </pre>
          </Card>

          <Card>
            <CardContent className="flex h-full flex-col gap-4 p-6">
              {copy.deploy.points.map((point, index) => {
                const Icon = DEPLOY_POINT_ICONS[index];
                return (
                  <div key={point.title} className="flex items-start gap-3">
                    <Icon className="mt-0.5 h-[18px] w-[18px] shrink-0 text-muted-foreground" />
                    <div className="text-sm leading-relaxed">
                      <p className="font-semibold">{point.title}</p>
                      <p className="text-muted-foreground">{point.desc}</p>
                    </div>
                  </div>
                );
              })}
              <div className="mt-auto flex items-center gap-2 border-t pt-4 text-xs text-muted-foreground">
                <Check className="h-3.5 w-3.5" />
                {copy.deploy.note}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </section>
  );
}
