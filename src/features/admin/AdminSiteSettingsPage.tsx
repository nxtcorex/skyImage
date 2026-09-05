import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { RotateCcw } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  fetchSiteSettings,
  updateSiteSettings,
  fetchLegalDefaults,
  type SiteSettings
} from "@/lib/api";
import { SplashScreen } from "@/components/SplashScreen";
import { useI18n } from "@/i18n";

const defaultSiteSettingsForm: SiteSettings = {
  siteTitle: "",
  consoleUrl: "http://localhost:8080",
  siteDescription: "",
  siteSlogan: "",
  siteLogo: "",
  about: "",
  aboutTitle: "",
  notFoundMode: "template",
  notFoundHeading: "",
  notFoundText: "",
  notFoundHtml: "",
  termsOfService: "",
  privacyPolicy: "",
  homePageMode: "default",
  homeCustomHtml: "",
  accountDisabledNotice: "",
};

export function AdminSiteSettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery<SiteSettings>({
    queryKey: ["admin", "site-settings"],
    queryFn: fetchSiteSettings
  });
  const [form, setForm] = useState<SiteSettings>(defaultSiteSettingsForm);
  const [initialForm, setInitialForm] = useState<SiteSettings | null>(null);

  const isFormDirty = useMemo(() => {
    if (!initialForm) {
      return false;
    }
    return (Object.keys(initialForm) as (keyof SiteSettings)[]).some(
      (key) => initialForm[key] !== form[key]
    );
  }, [initialForm, form]);

  useEffect(() => {
    if (!data) return;
    const normalized = {
      ...defaultSiteSettingsForm,
      ...data
    };
    setForm(normalized);
    setInitialForm(normalized);
  }, [data]);

  const mutation = useMutation({
    mutationFn: (input: SiteSettings) => updateSiteSettings(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["site-config"] });
      queryClient.invalidateQueries({ queryKey: ["site-meta"] });
      queryClient.invalidateQueries({ queryKey: ["admin", "site-settings"] });
      toast.success(t("admin.siteSettings.saved"));
    },
    onError: (mutationError) => toast.error(mutationError.message)
  });

  if (isLoading) {
    return <SplashScreen message={t("admin.siteSettings.loading")} />;
  }

  if (error && !data) {
    const message =
      error.message === "account disabled"
        ? t("admin.siteSettings.disabled")
        : error.message;
    return (
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle>{t("admin.siteSettings.loadFailed")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-destructive">{message}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const handleChange = (field: keyof SiteSettings, value: unknown) => {
    const actualValue = value === "indeterminate" ? false : value;
    setForm((prev) => ({ ...prev, [field]: actualValue as never }));
  };

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <h1 className="text-2xl font-semibold">{t("nav.systemSettings")}</h1>
        <p className="text-muted-foreground">{t("admin.siteSettings.description")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.siteSettings.card")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>{t("admin.siteSettings.siteTitle")}</Label>
            <Input value={form.siteTitle} onChange={(e) => handleChange("siteTitle", e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.siteSettings.siteDescription")}</Label>
            <Input
              value={form.siteDescription}
              onChange={(e) => handleChange("siteDescription", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.siteSettings.consoleUrl")}</Label>
            <Input
              value={form.consoleUrl}
              onChange={(e) => handleChange("consoleUrl", e.target.value)}
              placeholder="http://localhost:8080"
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.siteSettings.consoleUrlHint")}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t("admin.siteSettings.siteSlogan")}</Label>
            <Input
              value={form.siteSlogan}
              onChange={(e) => handleChange("siteSlogan", e.target.value)}
              placeholder={t("admin.siteSettings.siteSloganPlaceholder")}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.siteSettings.siteLogo")}</Label>
            <Input
              value={form.siteLogo}
              onChange={(e) => handleChange("siteLogo", e.target.value)}
              placeholder={t("admin.siteSettings.siteLogoPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.siteSettings.siteLogoHint")}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t("admin.siteSettings.aboutTitle")}</Label>
            <Input
              value={form.aboutTitle}
              onChange={(e) => handleChange("aboutTitle", e.target.value)}
              placeholder={t("admin.siteSettings.aboutTitlePlaceholder")}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.siteSettings.aboutTitleHint")}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t("admin.siteSettings.about")}</Label>
            <Textarea
              rows={4}
              value={form.about}
              onChange={(e) => handleChange("about", e.target.value)}
            />
          </div>
          
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>{t("admin.siteSettings.terms")}</Label>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={async () => {
                  try {
                    const defaults = await fetchLegalDefaults();
                    handleChange("termsOfService", defaults.termsOfService);
                    toast.success(t("admin.siteSettings.resetTermsSuccess"));
                  } catch (error) {
                    toast.error(t("admin.siteSettings.resetDefaultFailed"));
                  }
                }}
              >
                <RotateCcw className="h-4 w-4 mr-1" />
                {t("admin.siteSettings.resetDefault")}
              </Button>
            </div>
            <Textarea
              rows={6}
              value={form.termsOfService}
              onChange={(e) => handleChange("termsOfService", e.target.value)}
              placeholder={t("admin.siteSettings.termsPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.siteSettings.legalHint")}
            </p>
          </div>
          
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>{t("admin.siteSettings.privacy")}</Label>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={async () => {
                  try {
                    const defaults = await fetchLegalDefaults();
                    handleChange("privacyPolicy", defaults.privacyPolicy);
                    toast.success(t("admin.siteSettings.resetPrivacySuccess"));
                  } catch (error) {
                    toast.error(t("admin.siteSettings.resetDefaultFailed"));
                  }
                }}
              >
                <RotateCcw className="h-4 w-4 mr-1" />
                {t("admin.siteSettings.resetDefault")}
              </Button>
            </div>
            <Textarea
              rows={6}
              value={form.privacyPolicy}
              onChange={(e) => handleChange("privacyPolicy", e.target.value)}
              placeholder={t("admin.siteSettings.privacyPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.siteSettings.legalHint")}
            </p>
          </div>
          
          <div className="space-y-4 rounded-lg border p-4">
            <div className="space-y-2">
              <Label>{t("admin.siteSettings.homePageMode")}</Label>
              <Select
                value={form.homePageMode}
                onValueChange={(value) => handleChange("homePageMode", value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.siteSettings.homePageModePlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="default">{t("admin.siteSettings.homePageModeDefault")}</SelectItem>
                  <SelectItem value="custom_html">{t("admin.siteSettings.homePageModeCustomHtml")}</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {t("admin.siteSettings.homePageModeHint")}
              </p>
            </div>

            {form.homePageMode === "custom_html" && (
              <div className="space-y-2">
                <Label>{t("admin.siteSettings.homeCustomHtml")}</Label>
                <Textarea
                  rows={10}
                  value={form.homeCustomHtml}
                  onChange={(e) => handleChange("homeCustomHtml", e.target.value)}
                  placeholder={t("admin.siteSettings.homeCustomHtmlPlaceholder")}
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.siteSettings.homeCustomHtmlHint")}
                </p>
              </div>
            )}
          </div>
          
          <div className="space-y-4 rounded-lg border p-4">
            <div className="space-y-2">
              <Label>{t("admin.siteSettings.notFoundMode")}</Label>
              <Select
                value={form.notFoundMode}
                onValueChange={(value) => handleChange("notFoundMode", value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.siteSettings.notFoundModePlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="template">{t("admin.siteSettings.notFoundModeTemplate")}</SelectItem>
                  <SelectItem value="html">{t("admin.siteSettings.notFoundModeHtml")}</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {t("admin.siteSettings.notFoundModeHint")}
              </p>
            </div>

            {form.notFoundMode === "template" ? (
              <>
                <div className="space-y-2">
                  <Label>{t("admin.siteSettings.notFoundHeading")}</Label>
                  <Input
                    value={form.notFoundHeading}
                    onChange={(e) => handleChange("notFoundHeading", e.target.value)}
                    placeholder="404"
                  />
                  <p className="text-xs text-muted-foreground">
                    {t("admin.siteSettings.notFoundHeadingHint")}
                  </p>
                </div>
                <div className="space-y-2">
                  <Label>{t("admin.siteSettings.notFoundText")}</Label>
                  <Textarea
                    rows={3}
                    value={form.notFoundText}
                    onChange={(e) => handleChange("notFoundText", e.target.value)}
                    placeholder={t("admin.siteSettings.notFoundTextPlaceholder")}
                  />
                  <p className="text-xs text-muted-foreground">
                    {t("admin.siteSettings.notFoundTextHint")}
                  </p>
                </div>
              </>
            ) : (
              <div className="space-y-2">
                <Label>{t("admin.siteSettings.notFoundHtml")}</Label>
                <Textarea
                  rows={8}
                  value={form.notFoundHtml}
                  onChange={(e) => handleChange("notFoundHtml", e.target.value)}
                  placeholder={t("admin.siteSettings.notFoundHtmlPlaceholder")}
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.siteSettings.notFoundHtmlHint")}
                </p>
              </div>
            )}
          </div>
          <div className="space-y-2">
            <Label>{t("admin.siteSettings.accountDisabledNotice")}</Label>
            <Textarea
              value={form.accountDisabledNotice}
              onChange={(e) => handleChange("accountDisabledNotice", e.target.value)}
              minLength={4}
              maxLength={200}
              rows={3}
            />
          </div>
        </CardContent>
      </Card>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-muted-foreground">
          {isFormDirty ? t("admin.systemSettings.unsaved") : t("admin.systemSettings.clean")}
        </p>
        <Button onClick={() => mutation.mutate(form)} disabled={mutation.isPending || !isFormDirty}>
          {mutation.isPending ? t("common.saving") : t("admin.siteSettings.save")}
        </Button>
      </div>
    </div>
  );
}
