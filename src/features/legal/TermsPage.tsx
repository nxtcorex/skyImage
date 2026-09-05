import { useQuery } from "@tanstack/react-query";
import { fetchSiteConfig, fetchLegalContent } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useI18n } from "@/i18n";
import { PublicTopNav } from "@/components/PublicTopNav";
import { MarkdownContent } from "@/components/MarkdownContent";
import { Skeleton } from "@/components/ui/skeleton";

export function TermsPage() {
  const { t } = useI18n();
  const { data: siteConfig } = useQuery({
    queryKey: ["site-config"],
    queryFn: fetchSiteConfig,
  });
  const { data: termsContent, isLoading } = useQuery({
    queryKey: ["legal-content", "terms"],
    queryFn: () => fetchLegalContent("terms"),
  });
  const siteName = siteConfig?.title;
  const content = termsContent || "";

  if (isLoading) {
    return (
      <div className="min-h-screen bg-muted">
        <PublicTopNav title={siteName} description="" compact />
        <div className="flex min-h-[calc(100svh-88px)] items-center justify-center px-4 pb-8">
        <Card className="w-full max-w-4xl mx-4">
          <CardContent className="space-y-4 pt-6">
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
            <Skeleton className="h-4 w-4/6" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/6" />
          </CardContent>
        </Card>
        </div>
      </div>
    );
  }

  if (!content.trim()) {
    return (
      <div className="min-h-screen bg-muted">
        <PublicTopNav title={siteName} description="" compact />
        <div className="flex min-h-[calc(100svh-88px)] items-center justify-center px-4 pb-8">
        <Card className="w-full max-w-4xl mx-4">
          <CardContent className="pt-6">
            <p className="text-center text-muted-foreground">{t("legal.notConfiguredTerms")}</p>
          </CardContent>
        </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-muted">
      <PublicTopNav title={siteName} description={siteConfig?.description || ""} compact />
      <div className="py-8 px-4">
      <div className="max-w-4xl mx-auto">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-2xl">{t("legal.terms")}</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <MarkdownContent content={content} />
          </CardContent>
        </Card>
      </div>
      </div>
    </div>
  );
}
