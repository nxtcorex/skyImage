import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { PublicTopNav } from "@/components/PublicTopNav";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SplashScreen } from "@/components/SplashScreen";
import { fetchShopProducts, fetchSiteConfig, formatPriceCents } from "@/lib/api";
import { useI18n } from "@/i18n";
import { useAuthStore } from "@/state/auth";

export function PublicShopPage() {
  const { t } = useI18n();
  const user = useAuthStore((s) => s.user);
  const { data: siteConfig } = useQuery({
    queryKey: ["site-config"],
    queryFn: fetchSiteConfig,
    staleTime: 5 * 60 * 1000
  });
  const { data: products, isLoading, error } = useQuery({
    queryKey: ["shop", "products"],
    queryFn: fetchShopProducts
  });

  if (isLoading) return <SplashScreen />;

  return (
    <div className="flex min-h-screen flex-col bg-muted">
      <PublicTopNav
        title={siteConfig?.title}
        description={siteConfig?.description}
        compact
      />

      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-10 sm:px-8">
        <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">{t("shop.title")}</h1>
            <p className="mt-2 text-muted-foreground">{t("shop.description")}</p>
          </div>
          <div className="flex gap-2">
            {user ? (
              <>
                <Button asChild variant="outline">
                  <Link to="/dashboard/shop">{t("shop.buyInConsole")}</Link>
                </Button>
                <Button asChild variant="outline">
                  <Link to="/dashboard/orders">{t("shop.myOrders")}</Link>
                </Button>
              </>
            ) : (
              <Button asChild>
                <Link to="/login?redirect=/dashboard/shop">{t("shop.loginToBuy")}</Link>
              </Button>
            )}
          </div>
        </div>

        {error && (
          <p className="mb-4 text-sm text-destructive">{t("shop.loadFailed")}</p>
        )}

        {!products?.length ? (
          <Card>
            <CardContent className="py-10 text-center text-muted-foreground">
              {t("shop.empty")}
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {products.map((p) => (
              <Card key={p.id} className="flex flex-col">
                <CardHeader>
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-lg">{p.name}</CardTitle>
                    <Badge variant="secondary">
                      {formatPriceCents(p.priceCents, p.currency)}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="flex flex-1 flex-col gap-3">
                  <p className="text-sm text-muted-foreground">
                    {p.description || t("shop.noDescription")}
                  </p>
                  <ul className="space-y-1 text-sm">
                    <li>
                      {t("shop.duration")}: {t("shop.days", { count: p.durationDays })}
                    </li>
                    <li>
                      {t("shop.group")}: {p.group?.name ?? `#${p.groupId}`}
                    </li>
                  </ul>
                  <div className="mt-auto pt-2">
                    {user ? (
                      <Button asChild className="w-full">
                        <Link to={`/dashboard/shop?product=${p.id}`}>{t("shop.buy")}</Link>
                      </Button>
                    ) : (
                      <Button asChild className="w-full">
                        <Link
                          to={`/login?redirect=${encodeURIComponent(`/dashboard/shop?product=${p.id}`)}`}
                        >
                          {t("shop.loginToBuy")}
                        </Link>
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </main>

      <footer className="border-t bg-card/60">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-6 sm:px-8">
          <div className="flex flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
            <p className="font-medium text-foreground">
              {(siteConfig?.title || "").trim() || "SkyImage"}
            </p>
            <p>{(siteConfig?.description || "").trim() || "Image hosting platform"}</p>
          </div>
          <div className="flex flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
            <p>
              © {new Date().getFullYear()} {(siteConfig?.title || "").trim() || "SkyImage"}
            </p>
            <div className="flex gap-4">
              <a
                href="https://github.com/nxtcorex/skyImage"
                target="_blank"
                rel="noreferrer"
                className="transition-colors hover:text-foreground"
              >
                GitHub
              </a>
              <Link to="/privacy" className="transition-colors hover:text-foreground">
                {t("footer.privacy")}
              </Link>
              <Link to="/terms" className="transition-colors hover:text-foreground">
                {t("footer.terms")}
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
