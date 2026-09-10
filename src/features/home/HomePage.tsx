import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  Code2,
  FolderOpen,
  Globe,
  Images,
  Link2,
  Lock,
  ShieldCheck,
  Sparkles,
  UploadCloud,
  Users,
  Zap
} from "lucide-react";
import { Link } from "react-router-dom";

import { PublicTopNav } from "@/components/PublicTopNav";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { fetchRegistrationStatus, type SiteConfig } from "@/lib/api";
import { useAuthStore } from "@/state/auth";
import { useI18n } from "@/i18n";

type Step = {
  icon: typeof UploadCloud;
  title: string;
  desc: string;
};

type Scene = {
  icon: typeof Users;
  text: string;
};

export function HomePage({ siteConfig }: { siteConfig?: SiteConfig }) {
  const { t } = useI18n();
  const token = useAuthStore((state) => state.token);
  const { data: registrationStatus } = useQuery({
    queryKey: ["registration-status"],
    queryFn: fetchRegistrationStatus,
    enabled: !token,
    staleTime: 2 * 60 * 1000
  });

  const title = siteConfig?.title ?? "";
  const description = siteConfig?.description ?? "";
  const homePageMode = siteConfig?.homePageMode ?? "default";
  const homeCustomHtml = siteConfig?.homeCustomHtml ?? "";

  if (homePageMode === "custom_html" && homeCustomHtml.trim() !== "") {
    return (
      <div className="relative min-h-screen">
        <PublicTopNav title={title} description={description} compact floating />
        <main className="min-h-screen" dangerouslySetInnerHTML={{ __html: homeCustomHtml }} />
      </div>
    );
  }

  const displaySlogan = siteConfig?.slogan?.trim() || t("home.defaultSlogan");
  const primaryCtaText = t("home.defaultPrimaryCta");
  const dashboardCtaText = t("home.defaultDashboardCta");
  const secondaryCtaText = t("home.defaultSecondaryCta");

  const steps: Step[] = [
    {
      icon: UploadCloud,
      title: t("home.defaultStep1Title"),
      desc: t("home.defaultStep1Desc")
    },
    {
      icon: Link2,
      title: t("home.defaultStep2Title"),
      desc: t("home.defaultStep2Desc")
    },
    {
      icon: FolderOpen,
      title: t("home.defaultStep3Title"),
      desc: t("home.defaultStep3Desc")
    }
  ];

  const features = [
    {
      icon: Images,
      title: t("home.defaultFeature1Title"),
      desc: t("home.defaultFeature1Desc")
    },
    {
      icon: Lock,
      title: t("home.defaultFeature2Title"),
      desc: t("home.defaultFeature2Desc")
    },
    {
      icon: Zap,
      title: t("home.defaultFeature3Title"),
      desc: t("home.defaultFeature3Desc")
    },
    {
      icon: ShieldCheck,
      title: t("home.defaultFeature4Title"),
      desc: t("home.defaultFeature4Desc")
    }
  ];

  const scenes: Scene[] = [
    { icon: Users, text: t("home.defaultScene1") },
    { icon: Code2, text: t("home.defaultScene2") },
    { icon: Globe, text: t("home.defaultScene3") }
  ];

  const ctaHref = token ? "/dashboard" : "/login";

  return (
    <div className="flex min-h-screen flex-col bg-muted">
      <PublicTopNav title={title} description={description} compact />

      <main className="mx-auto flex w-full max-w-6xl flex-1 flex-col px-4 pb-20 sm:px-8">
        {/* Hero */}
        <section className="animate-enter animate-enter-1 relative flex flex-col items-center gap-6 py-16 text-center sm:py-24">
          <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-b from-primary/5 via-transparent to-transparent" />
          <Badge variant="secondary" className="w-fit">
            <Sparkles className="mr-1 h-3.5 w-3.5" />
            {t("home.defaultBadge")}
          </Badge>

          <h1 className="max-w-3xl text-4xl font-semibold leading-tight tracking-tight sm:text-5xl lg:text-6xl">
            {displaySlogan}
          </h1>

          <p className="max-w-2xl text-base text-muted-foreground sm:text-lg">
            {t("home.defaultIntro")}
          </p>

          <div className="mt-2 flex flex-wrap items-center justify-center gap-3">
            <Button asChild size="lg" className="gap-2">
              <Link to={ctaHref}>
                {token ? dashboardCtaText : primaryCtaText}
                <ArrowRight className="h-4 w-4" />
              </Link>
            </Button>
            {!token && registrationStatus?.allowed && (
              <Button asChild size="lg" variant="outline">
                <Link to="/register">{secondaryCtaText}</Link>
              </Button>
            )}
          </div>
        </section>

        {/* Steps */}
        <section className="animate-enter animate-enter-2 space-y-6">
          <div className="flex flex-col items-center gap-2 text-center">
            <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
              {t("home.defaultStepsTitle")}
            </h2>
            <p className="text-muted-foreground">{t("home.defaultStepsDesc")}</p>
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            {steps.map((step, index) => (
              <Card key={step.title} className="transition-shadow duration-300 hover:shadow-lg">
                <CardContent className="space-y-3 p-6">
                  <div className="flex items-center justify-between">
                    <span className="inline-flex h-10 w-10 items-center justify-center rounded-lg border bg-secondary text-secondary-foreground">
                      <step.icon className="h-5 w-5" />
                    </span>
                    <span className="text-sm font-semibold text-muted-foreground">
                      0{index + 1}
                    </span>
                  </div>
                  <p className="text-base font-medium">{step.title}</p>
                  <p className="text-sm text-muted-foreground">{step.desc}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        {/* Features */}
        <section className="animate-enter animate-enter-3 space-y-6 pt-14">
          <div className="flex flex-col items-center gap-2 text-center">
            <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
              {t("home.defaultHighlightTitle")}
            </h2>
            <p className="max-w-2xl text-muted-foreground">
              {t("home.defaultHighlightDesc")}
            </p>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {features.map((feature) => (
              <Card key={feature.title} className="transition-shadow duration-300 hover:shadow-lg">
                <CardContent className="flex h-full flex-col items-center gap-3 p-6 text-center">
                  <span className="inline-flex h-12 w-12 items-center justify-center rounded-xl border bg-secondary text-secondary-foreground">
                    <feature.icon className="h-6 w-6" />
                  </span>
                  <p className="text-sm font-semibold">{feature.title}</p>
                  <p className="text-sm text-muted-foreground">{feature.desc}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        {/* Scenarios */}
        <section className="animate-enter animate-enter-3 space-y-6 pt-14">
          <div className="flex flex-col items-center gap-2 text-center">
            <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
              {t("home.defaultSceneTitle")}
            </h2>
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            {scenes.map((scene) => (
              <Card key={scene.text} className="transition-shadow duration-300 hover:shadow-lg">
                <CardContent className="flex items-center gap-4 p-6">
                  <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-lg border bg-secondary text-secondary-foreground">
                    <scene.icon className="h-5 w-5" />
                  </span>
                  <p className="text-sm text-muted-foreground">{scene.text}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        {/* CTA band */}
        <section className="animate-enter animate-enter-4 pt-14">
          <Card className="relative overflow-hidden">
            <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-r from-primary/5 to-transparent" />
            <CardContent className="flex flex-col items-center gap-4 px-6 py-12 text-center sm:py-16">
              <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
                {t("home.defaultCtaTitle")}
              </h2>
              <p className="max-w-xl text-muted-foreground">{t("home.defaultCtaDesc")}</p>
              <Button asChild size="lg" className="mt-2 gap-2">
                <Link to={ctaHref}>
                  {token ? dashboardCtaText : primaryCtaText}
                  <ArrowRight className="h-4 w-4" />
                </Link>
              </Button>
            </CardContent>
          </Card>
        </section>
      </main>

      <footer className="border-t bg-card/60">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-6 sm:px-8">
          <div className="flex flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
            <p className="font-medium text-foreground">{title.trim() || "SkyImage"}</p>
            <p>{description.trim() || "Image hosting platform"}</p>
          </div>
          <div className="flex flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
            <p>© {new Date().getFullYear()} {title.trim() || "SkyImage"}</p>
            <div className="flex gap-4">
              <a
                href="https://github.com/nxtcorex/skyImage"
                target="_blank"
                rel="noreferrer"
                className="hover:text-foreground transition-colors"
              >
                GitHub
              </a>
              <Link to="/privacy" className="hover:text-foreground transition-colors">
                {t("footer.privacy")}
              </Link>
              <Link to="/terms" className="hover:text-foreground transition-colors">
                {t("footer.terms")}
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
