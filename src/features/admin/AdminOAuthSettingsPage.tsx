import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  fetchOAuthSettings,
  updateOAuthSettings,
  type OAuthSettings,
  type OAuthProviderSettings,
  type OAuthSettingsUpdate,
  type OAuthProviderSettingsUpdate
} from "@/lib/api";
import { SplashScreen } from "@/components/SplashScreen";
import { useI18n } from "@/i18n";

const emptyProvider = (): OAuthProviderSettings => ({
  enabled: false,
  clientId: "",
  clientSecret: "",
  name: "",
  authUrl: "",
  tokenUrl: "",
  userInfoUrl: "",
  scopes: ""
});

const defaultForm: OAuthSettings = {
  enabled: false,
  autoLinkByEmail: false,
  github: emptyProvider(),
  google: emptyProvider(),
  discord: emptyProvider(),
  custom: emptyProvider()
};

function ProviderFields({
  title,
  value,
  onChange,
  custom
}: {
  title: string;
  value: OAuthProviderSettings;
  onChange: (next: OAuthProviderSettings) => void;
  custom?: boolean;
}) {
  const { t } = useI18n();
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center space-x-2">
          <Checkbox
            id={`${title}-enabled`}
            checked={value.enabled}
            onCheckedChange={(checked) =>
              onChange({ ...value, enabled: checked === true })
            }
          />
          <Label htmlFor={`${title}-enabled`}>{t("admin.oauthSettings.providerEnabled")}</Label>
        </div>
        {custom && (
          <div className="space-y-2">
            <Label>{t("admin.oauthSettings.displayName")}</Label>
            <Input
              value={value.name || ""}
              onChange={(e) => onChange({ ...value, name: e.target.value })}
            />
          </div>
        )}
        <div className="space-y-2">
          <Label>Client ID</Label>
          <Input
            value={value.clientId}
            onChange={(e) => onChange({ ...value, clientId: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label>Client Secret</Label>
          <Input
            type="password"
            value={value.clientSecret}
            onChange={(e) => onChange({ ...value, clientSecret: e.target.value })}
            placeholder={value.clientSecret === "***" ? "***" : ""}
          />
        </div>
        {custom && (
          <>
            <div className="space-y-2">
              <Label>Auth URL</Label>
              <Input
                value={value.authUrl || ""}
                onChange={(e) => onChange({ ...value, authUrl: e.target.value })}
                placeholder="https://casdoor.example.com/login/oauth/authorize"
              />
              <p className="text-xs text-muted-foreground">
                只填授权地址本身，不要带 ?client_id= 等参数。Casdoor 示例：
                https://你的Casdoor域名/login/oauth/authorize
              </p>
            </div>
            <div className="space-y-2">
              <Label>Token URL</Label>
              <Input
                value={value.tokenUrl || ""}
                onChange={(e) => onChange({ ...value, tokenUrl: e.target.value })}
                placeholder="https://casdoor.example.com/api/login/oauth/access_token"
              />
            </div>
            <div className="space-y-2">
              <Label>UserInfo URL</Label>
              <Input
                value={value.userInfoUrl || ""}
                onChange={(e) => onChange({ ...value, userInfoUrl: e.target.value })}
                placeholder="https://casdoor.example.com/api/userinfo"
              />
            </div>
            <div className="space-y-2">
              <Label>Scopes</Label>
              <Input
                value={value.scopes || ""}
                onChange={(e) => onChange({ ...value, scopes: e.target.value })}
                placeholder="openid profile email"
              />
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

export function AdminOAuthSettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin", "oauth-settings"],
    queryFn: fetchOAuthSettings
  });
  const [form, setForm] = useState<OAuthSettings>(defaultForm);
  const [initialForm, setInitialForm] = useState<OAuthSettings | null>(null);

  const isFormDirty = useMemo(() => {
    if (!initialForm) return false;
    return JSON.stringify(initialForm) !== JSON.stringify(form);
  }, [initialForm, form]);

  useEffect(() => {
    if (!data) return;
    const normalized: OAuthSettings = {
      ...defaultForm,
      ...data,
      github: { ...emptyProvider(), ...data.github },
      google: { ...emptyProvider(), ...data.google },
      discord: { ...emptyProvider(), ...data.discord },
      custom: { ...emptyProvider(), ...data.custom }
    };
    setForm(normalized);
    setInitialForm(normalized);
  }, [data]);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!initialForm) return;
      const patch: OAuthSettingsUpdate = {};
      if (initialForm.enabled !== form.enabled) {
        patch.enabled = form.enabled;
      }
      if (initialForm.autoLinkByEmail !== form.autoLinkByEmail) {
        patch.autoLinkByEmail = form.autoLinkByEmail;
      }
      const diffProvider = (name: keyof Pick<OAuthSettings, "github" | "google" | "discord" | "custom">) => {
        const initialProvider = initialForm[name];
        const formProvider = form[name];
        const providerPatch: OAuthProviderSettingsUpdate = {};
        (Object.keys(initialProvider) as (keyof OAuthProviderSettings)[]).forEach((key) => {
          if (initialProvider[key] !== formProvider[key]) {
            (providerPatch as Record<string, unknown>)[key] = formProvider[key];
          }
        });
        if (Object.keys(providerPatch).length > 0) {
          patch[name] = providerPatch;
        }
      };
      diffProvider("github");
      diffProvider("google");
      diffProvider("discord");
      diffProvider("custom");

      if (Object.keys(patch).length > 0) {
        await updateOAuthSettings(patch);
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "oauth-settings"] });
      await queryClient.invalidateQueries({ queryKey: ["oauth-providers"] });
      await queryClient.invalidateQueries({ queryKey: ["account", "oauth-bindings"] });
      toast.success(t("admin.oauthSettings.saved"));
    },
    onError: (err) => toast.error(err.message)
  });

  if (isLoading) {
    return <SplashScreen message={t("admin.oauthSettings.loading")} />;
  }

  if (error && !data) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.oauthSettings.loadFailed")}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-destructive">{(error as Error).message}</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <h1 className="text-2xl font-semibold">{t("admin.oauthSettings.title")}</h1>
        <p className="text-muted-foreground">{t("admin.oauthSettings.description")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.oauthSettings.global")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center space-x-2">
            <Checkbox
              id="oauth-enabled"
              checked={form.enabled}
              onCheckedChange={(checked) =>
                setForm((prev) => ({ ...prev, enabled: checked === true }))
              }
            />
            <Label htmlFor="oauth-enabled">{t("admin.oauthSettings.enabled")}</Label>
          </div>
          <div className="flex items-center space-x-2">
            <Checkbox
              id="oauth-auto-link"
              checked={form.autoLinkByEmail}
              onCheckedChange={(checked) =>
                setForm((prev) => ({ ...prev, autoLinkByEmail: checked === true }))
              }
            />
            <Label htmlFor="oauth-auto-link">{t("admin.oauthSettings.autoLinkByEmail")}</Label>
          </div>
          <p className="text-xs text-muted-foreground">{t("admin.oauthSettings.autoLinkByEmailHint")}</p>
          <p className="text-xs text-muted-foreground">{t("admin.oauthSettings.callbackHint")}</p>
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <ProviderFields
          title="GitHub"
          value={form.github}
          onChange={(github) => setForm((prev) => ({ ...prev, github }))}
        />
        <ProviderFields
          title="Google"
          value={form.google}
          onChange={(google) => setForm((prev) => ({ ...prev, google }))}
        />
        <ProviderFields
          title="Discord"
          value={form.discord}
          onChange={(discord) => setForm((prev) => ({ ...prev, discord }))}
        />
        <ProviderFields
          title={t("admin.oauthSettings.custom")}
          value={form.custom}
          onChange={(custom) => setForm((prev) => ({ ...prev, custom }))}
          custom
        />
      </div>

      <div className="flex justify-end">
        <Button onClick={() => mutation.mutate()} disabled={mutation.isPending || !isFormDirty}>
          {mutation.isPending ? t("common.saving") : t("admin.oauthSettings.save")}
        </Button>
      </div>
    </div>
  );
}
