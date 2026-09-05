import { useEffect, useState, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Shield, CheckCircle2, AlertTriangle, Loader2, Info } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  fetchCaptchaSettings,
  updateCaptchaSettings,
  testCaptchaConfig,
  type CaptchaSettings,
  type CaptchaSettingsUpdate
} from "@/lib/api";
import { SplashScreen } from "@/components/SplashScreen";
import { Turnstile, type TurnstileRef } from "@/components/Turnstile";
import { Geetest, type GeetestRef } from "@/components/Geetest";
import { CapWidget, type CapWidgetRef } from "@/components/CapWidget";
import { loadTurnstileScript } from "@/lib/turnstile";
import { loadGeetestScript } from "@/lib/geetest";
import { buildCapApiEndpoint, loadCapWidget } from "@/lib/cap";
import { useI18n } from "@/i18n";

type CaptchaProvider = "cloudflare" | "geetest" | "cap" | "";

interface ProviderStatus {
  verified: boolean;
  lastVerifiedAt: string | null;
  canUse: boolean;
}

function providerLabel(provider: CaptchaProvider): string {
  if (provider === "cloudflare") return "Cloudflare Turnstile";
  if (provider === "geetest") return "Geetest";
  if (provider === "cap") return "Cap";
  return "";
}

export function AdminCaptchaSettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery<CaptchaSettings>({
    queryKey: ["admin", "captcha-settings"],
    queryFn: fetchCaptchaSettings
  });

  const [form, setForm] = useState({
    enableCaptcha: false,
    captchaProvider: "" as CaptchaProvider,

    cloudflareSiteKey: "",
    cloudflareSecretKey: "",

    geetestCaptchaId: "",
    geetestCaptchaKey: "",

    capInstanceUrl: "",
    capSiteKey: "",
    capSecretKey: "",

    enableLoginCaptcha: false,
    enableRegisterCaptcha: false,
    enableRegisterVerifyCaptcha: false,
    enableForgotPasswordRequestCaptcha: false,
    enableForgotPasswordResetCaptcha: false,
    enableRedeemCaptcha: false,
    enableTicketCaptcha: false,
  });

  const [providerStatus, setProviderStatus] = useState<{
    cloudflare: ProviderStatus;
    geetest: ProviderStatus;
    cap: ProviderStatus;
  }>({
    cloudflare: { verified: false, lastVerifiedAt: null, canUse: false },
    geetest: { verified: false, lastVerifiedAt: null, canUse: false },
    cap: { verified: false, lastVerifiedAt: null, canUse: false },
  });

  const [showTester, setShowTester] = useState<{
    cloudflare: boolean;
    geetest: boolean;
    cap: boolean;
  }>({
    cloudflare: false,
    geetest: false,
    cap: false,
  });

  const [turnstileReady, setTurnstileReady] = useState(false);
  const [geetestReady, setGeetestReady] = useState(false);
  const [capReady, setCapReady] = useState(false);
  const [initialForm, setInitialForm] = useState<typeof form | null>(null);
  const turnstileRef = useRef<TurnstileRef>(null);
  const geetestRef = useRef<GeetestRef>(null);
  const capRef = useRef<CapWidgetRef>(null);

  useEffect(() => {
    if (data) {
      const normalized = {
        enableCaptcha: data.enableCaptcha ?? false,
        captchaProvider: (data.captchaProvider || "") as CaptchaProvider,

        cloudflareSiteKey: data.cloudflareSiteKey || "",
        cloudflareSecretKey: data.cloudflareSecretKey || "",

        geetestCaptchaId: data.geetestCaptchaId || "",
        geetestCaptchaKey: data.geetestCaptchaKey || "",

        capInstanceUrl: data.capInstanceUrl || "",
        capSiteKey: data.capSiteKey || "",
        capSecretKey: data.capSecretKey || "",

        enableLoginCaptcha: data.enableLoginCaptcha ?? false,
        enableRegisterCaptcha: data.enableRegisterCaptcha ?? false,
        enableRegisterVerifyCaptcha: data.enableRegisterVerifyCaptcha ?? false,
        enableForgotPasswordRequestCaptcha: data.enableForgotPasswordRequestCaptcha ?? false,
        enableForgotPasswordResetCaptcha: data.enableForgotPasswordResetCaptcha ?? false,
        enableRedeemCaptcha: data.enableRedeemCaptcha ?? false,
        enableTicketCaptcha: data.enableTicketCaptcha ?? false,
      };
      setForm(normalized);
      setInitialForm(normalized);

      setProviderStatus({
        cloudflare: {
          verified: data.cloudflareVerified || false,
          lastVerifiedAt: data.cloudflareLastVerifiedAt || null,
          canUse: !!(data.cloudflareSiteKey && data.cloudflareSecretKey && data.cloudflareVerified),
        },
        geetest: {
          verified: data.geetestVerified || false,
          lastVerifiedAt: data.geetestLastVerifiedAt || null,
          canUse: !!(data.geetestCaptchaId && data.geetestCaptchaKey && data.geetestVerified),
        },
        cap: {
          verified: data.capVerified || false,
          lastVerifiedAt: data.capLastVerifiedAt || null,
          canUse: !!(data.capInstanceUrl && data.capSiteKey && data.capSecretKey && data.capVerified),
        },
      });
    }
  }, [data]);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!initialForm) return;
      const patch: CaptchaSettingsUpdate = {};
      (Object.keys(form) as (keyof typeof form)[]).forEach((key) => {
        if (initialForm[key] !== form[key]) {
          (patch as Record<string, unknown>)[key] = form[key];
        }
      });
      if (Object.keys(patch).length > 0) {
        await updateCaptchaSettings(patch);
      }
    },
    onSuccess: () => {
      setInitialForm({ ...form });
      queryClient.invalidateQueries({ queryKey: ["site-config"] });
      queryClient.invalidateQueries({ queryKey: ["admin", "captcha-settings"] });
      toast.success(t("admin.captchaSettings.saved"));
    },
    onError: (error) => toast.error(error.message)
  });

  const testCloudflareMutation = useMutation({
    mutationFn: testCaptchaConfig,
    onSuccess: (result) => {
      if (result.success) {
        toast.success(t("admin.captchaSettings.cloudflare.verifiedSuccess"));
        setProviderStatus(prev => ({
          ...prev,
          cloudflare: {
            verified: true,
            lastVerifiedAt: result.verifiedAt || new Date().toISOString(),
            canUse: true,
          }
        }));
        setShowTester(prev => ({ ...prev, cloudflare: false }));
      } else {
        setProviderStatus(prev => ({
          ...prev,
          cloudflare: { ...prev.cloudflare, verified: false, canUse: false }
        }));
        toast.error(result.message || t("admin.captchaSettings.cloudflare.verifyFailed"));
      }
    },
    onError: (error) => {
      setProviderStatus(prev => ({
        ...prev,
        cloudflare: { ...prev.cloudflare, verified: false, canUse: false }
      }));
      toast.error(error.message);
    }
  });

  const testGeetestMutation = useMutation({
    mutationFn: testCaptchaConfig,
    onSuccess: (result) => {
      if (result.success) {
        toast.success(t("admin.captchaSettings.geetest.verifiedSuccess"));
        setProviderStatus(prev => ({
          ...prev,
          geetest: {
            verified: true,
            lastVerifiedAt: result.verifiedAt || new Date().toISOString(),
            canUse: true,
          }
        }));
        setShowTester(prev => ({ ...prev, geetest: false }));
      } else {
        setProviderStatus(prev => ({
          ...prev,
          geetest: { ...prev.geetest, verified: false, canUse: false }
        }));
        toast.error(result.message || t("admin.captchaSettings.geetest.verifyFailed"));
      }
    },
    onError: (error) => {
      setProviderStatus(prev => ({
        ...prev,
        geetest: { ...prev.geetest, verified: false, canUse: false }
      }));
      toast.error(error.message);
    }
  });

  const testCapMutation = useMutation({
    mutationFn: testCaptchaConfig,
    onSuccess: (result) => {
      if (result.success) {
        toast.success(t("admin.captchaSettings.cap.verifiedSuccess"));
        setProviderStatus(prev => ({
          ...prev,
          cap: {
            verified: true,
            lastVerifiedAt: result.verifiedAt || new Date().toISOString(),
            canUse: true,
          }
        }));
        setShowTester(prev => ({ ...prev, cap: false }));
      } else {
        setProviderStatus(prev => ({
          ...prev,
          cap: { ...prev.cap, verified: false, canUse: false }
        }));
        toast.error(result.message || t("admin.captchaSettings.cap.verifyFailed"));
      }
    },
    onError: (error) => {
      setProviderStatus(prev => ({
        ...prev,
        cap: { ...prev.cap, verified: false, canUse: false }
      }));
      toast.error(error.message);
    }
  });

  if (isLoading) {
    return <SplashScreen message={t("admin.captchaSettings.loading")} />;
  }

  if (error && !data) {
    return (
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle>{t("admin.captchaSettings.loadFailed")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-destructive">{error.message}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const handleChange = (field: keyof typeof form, value: any) => {
    const actualValue = value === "indeterminate" ? false : value;

    if (field === "cloudflareSiteKey" || field === "cloudflareSecretKey") {
      setProviderStatus(prev => ({
        ...prev,
        cloudflare: { verified: false, lastVerifiedAt: null, canUse: false }
      }));
    }
    if (field === "geetestCaptchaId" || field === "geetestCaptchaKey") {
      setProviderStatus(prev => ({
        ...prev,
        geetest: { verified: false, lastVerifiedAt: null, canUse: false }
      }));
    }
    if (field === "capInstanceUrl" || field === "capSiteKey" || field === "capSecretKey") {
      setProviderStatus(prev => ({
        ...prev,
        cap: { verified: false, lastVerifiedAt: null, canUse: false }
      }));
    }

    if (field === "enableCaptcha" && actualValue === true) {
      const provider = form.captchaProvider;
      if (!provider) {
        toast.error(t("admin.captchaSettings.selectProviderFirst"));
        return;
      }
      const status = providerStatus[provider as "cloudflare" | "geetest" | "cap"];
      if (!status?.canUse) {
        toast.error(t("admin.captchaSettings.testProviderFirst"));
        return;
      }
    }

    if (field === "captchaProvider" && actualValue && form.enableCaptcha) {
      if (actualValue === "cloudflare" || actualValue === "geetest" || actualValue === "cap") {
        const status = providerStatus[actualValue as "cloudflare" | "geetest" | "cap"];
        if (!status?.canUse) {
          toast.error(t("admin.captchaSettings.providerNotTested"));
          return;
        }
      }
    }

    setForm((prev) => ({ ...prev, [field]: actualValue }));
  };

  const startCloudflareTest = () => {
    if (!form.cloudflareSiteKey || !form.cloudflareSecretKey) {
      toast.error(t("admin.captchaSettings.cloudflare.keysRequired"));
      return;
    }
    setShowTester(prev => ({ ...prev, cloudflare: true }));
    setTurnstileReady(false);
    loadTurnstileScript()
      .then(() => setTurnstileReady(true))
      .catch(() => {
        toast.error(t("admin.captchaSettings.cloudflare.loadScriptFailed"));
      });
  };

  const startGeetestTest = () => {
    if (!form.geetestCaptchaId || !form.geetestCaptchaKey) {
      toast.error(t("admin.captchaSettings.geetest.keysRequired"));
      return;
    }

    const trimmedId = form.geetestCaptchaId.trim();
    if (trimmedId.length !== 32) {
      toast.error(t("admin.captchaSettings.geetest.captchaIdLengthWrong", { length: trimmedId.length }));
      return;
    }

    if (!/^[a-f0-9]{32}$/i.test(trimmedId)) {
      toast.error(t("admin.captchaSettings.geetest.captchaIdFormatError"));
      return;
    }

    setShowTester(prev => ({ ...prev, geetest: true }));
    setGeetestReady(false);
    loadGeetestScript()
      .then(() => setGeetestReady(true))
      .catch(() => {
        toast.error(t("admin.captchaSettings.geetest.loadScriptFailed"));
      });
  };

  const handleGeetestSuccess = (result: { lot_number: string; pass_token: string; gen_time: string; captcha_output: string }) => {
    testGeetestMutation.mutate({
      provider: "geetest" as const,
      captchaId: form.geetestCaptchaId,
      captchaKey: form.geetestCaptchaKey,
      token: result.lot_number,
      extraData: {
        challenge: result.lot_number,
        validate: result.pass_token,
        seccode: result.gen_time,
        captcha_output: result.captcha_output,
      }
    });
  };

  const handleGeetestError = (error?: string) => {
    toast.error(error || t("admin.captchaSettings.geetest.initFailed"));
    setShowTester(prev => ({ ...prev, geetest: false }));
  };

  const handleCloudflareVerify = (token: string) => {
    testCloudflareMutation.mutate({
      provider: "cloudflare",
      siteKey: form.cloudflareSiteKey,
      secretKey: form.cloudflareSecretKey,
      token
    });
  };

  const startCapTest = () => {
    if (!form.capInstanceUrl || !form.capSiteKey || !form.capSecretKey) {
      toast.error(t("admin.captchaSettings.cap.keysRequired"));
      return;
    }
    setShowTester(prev => ({ ...prev, cap: true }));
    setCapReady(false);
    loadCapWidget()
      .then(() => setCapReady(true))
      .catch(() => {
        toast.error(t("admin.captchaSettings.cap.loadScriptFailed"));
      });
  };

  const handleCapVerify = (token: string) => {
    testCapMutation.mutate({
      provider: "cap",
      instanceUrl: form.capInstanceUrl,
      siteKey: form.capSiteKey,
      secretKey: form.capSecretKey,
      token
    });
  };

  const isFormDirty = initialForm
    ? Object.keys(form).some((key) => initialForm[key as keyof typeof form] !== form[key as keyof typeof form])
    : false;

  const availableProviders: { value: CaptchaProvider; label: string; disabled: boolean }[] = [
    {
      value: "",
      label: t("admin.captchaSettings.selectProvider"),
      disabled: false
    },
    {
      value: "cloudflare",
      label: providerStatus.cloudflare.canUse ? "Cloudflare Turnstile" : t("admin.captchaSettings.cloudflare.unverified"),
      disabled: !providerStatus.cloudflare.canUse
    },
    {
      value: "geetest",
      label: providerStatus.geetest.canUse ? "Geetest" : t("admin.captchaSettings.geetest.unverified"),
      disabled: !providerStatus.geetest.canUse
    },
    {
      value: "cap",
      label: providerStatus.cap.canUse ? "Cap" : t("admin.captchaSettings.cap.unverified"),
      disabled: !providerStatus.cap.canUse
    }
  ];

  const capApiEndpoint = buildCapApiEndpoint(form.capInstanceUrl, form.capSiteKey);

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <h1 className="text-2xl font-semibold">{t("admin.captchaSettings.title")}</h1>
        <p className="text-muted-foreground">
          {t("admin.captchaSettings.description")}
        </p>
      </div>

      {/* Global Settings */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            {t("admin.captchaSettings.globalConfig")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center space-x-2">
            <Checkbox
              id="enableCaptcha"
              checked={form.enableCaptcha}
              onCheckedChange={(checked) => handleChange("enableCaptcha", checked)}
            />
            <Label htmlFor="enableCaptcha" className="font-medium">
              {t("admin.captchaSettings.enableCaptcha")}
            </Label>
          </div>
          <p className="text-sm text-muted-foreground ml-6">
            {t("admin.captchaSettings.enableCaptchaHint")}
          </p>

          <div className="space-y-2">
            <Label>{t("admin.captchaSettings.captchaProvider")}</Label>
            <Select
              value={form.captchaProvider || "none"}
              onValueChange={(value) => handleChange("captchaProvider", value === "none" ? "" : value as CaptchaProvider)}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("admin.captchaSettings.providerPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {availableProviders.map((provider) => (
                  <SelectItem
                    key={provider.value || "none"}
                    value={provider.value || "none"}
                    disabled={provider.disabled}
                  >
                    {provider.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {t("admin.captchaSettings.providerHint")}
            </p>
          </div>

          {form.enableCaptcha && form.captchaProvider && (
            <div className="flex items-center gap-2 rounded-md border border-blue-200 bg-blue-50 p-3 text-sm text-blue-900 dark:border-blue-900 dark:bg-blue-950 dark:text-blue-100">
              <Info className="h-4 w-4 flex-shrink-0" />
              <div>
                {t("admin.captchaSettings.currentProvider")}: <strong>{providerLabel(form.captchaProvider)}</strong>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Scenarios */}
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.captchaSettings.scenes")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-muted-foreground mb-4">
            {t("admin.captchaSettings.scenesHint")}
          </p>

          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="enableLoginCaptcha"
                checked={form.enableLoginCaptcha}
                disabled={!form.enableCaptcha}
                onCheckedChange={(checked) => handleChange("enableLoginCaptcha", checked)}
              />
              <Label htmlFor="enableLoginCaptcha" className={!form.enableCaptcha ? "text-muted-foreground" : ""}>
                {t("admin.captchaSettings.scene.login")}
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="enableRegisterCaptcha"
                checked={form.enableRegisterCaptcha}
                disabled={!form.enableCaptcha}
                onCheckedChange={(checked) => handleChange("enableRegisterCaptcha", checked)}
              />
              <Label htmlFor="enableRegisterCaptcha" className={!form.enableCaptcha ? "text-muted-foreground" : ""}>
                {t("admin.captchaSettings.scene.register")}
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="enableRegisterVerifyCaptcha"
                checked={form.enableRegisterVerifyCaptcha}
                disabled={!form.enableCaptcha}
                onCheckedChange={(checked) => handleChange("enableRegisterVerifyCaptcha", checked)}
              />
              <Label htmlFor="enableRegisterVerifyCaptcha" className={!form.enableCaptcha ? "text-muted-foreground" : ""}>
                {t("admin.captchaSettings.scene.registerVerify")}
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="enableForgotPasswordRequestCaptcha"
                checked={form.enableForgotPasswordRequestCaptcha}
                disabled={!form.enableCaptcha}
                onCheckedChange={(checked) => handleChange("enableForgotPasswordRequestCaptcha", checked)}
              />
              <Label htmlFor="enableForgotPasswordRequestCaptcha" className={!form.enableCaptcha ? "text-muted-foreground" : ""}>
                {t("admin.captchaSettings.scene.forgotPasswordRequest")}
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="enableForgotPasswordResetCaptcha"
                checked={form.enableForgotPasswordResetCaptcha}
                disabled={!form.enableCaptcha}
                onCheckedChange={(checked) => handleChange("enableForgotPasswordResetCaptcha", checked)}
              />
              <Label htmlFor="enableForgotPasswordResetCaptcha" className={!form.enableCaptcha ? "text-muted-foreground" : ""}>
                {t("admin.captchaSettings.scene.forgotPasswordReset")}
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="enableRedeemCaptcha"
                checked={form.enableRedeemCaptcha}
                disabled={!form.enableCaptcha}
                onCheckedChange={(checked) => handleChange("enableRedeemCaptcha", checked)}
              />
              <Label htmlFor="enableRedeemCaptcha" className={!form.enableCaptcha ? "text-muted-foreground" : ""}>
                {t("admin.captchaSettings.scene.redeem")}
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="enableTicketCaptcha"
                checked={form.enableTicketCaptcha}
                disabled={!form.enableCaptcha}
                onCheckedChange={(checked) => handleChange("enableTicketCaptcha", checked)}
              />
              <Label htmlFor="enableTicketCaptcha" className={!form.enableCaptcha ? "text-muted-foreground" : ""}>
                {t("admin.captchaSettings.scene.ticket")}
              </Label>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Cloudflare Turnstile Settings */}
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.captchaSettings.cloudflare.title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>{t("admin.captchaSettings.cloudflare.siteKey")}</Label>
            <Input
              value={form.cloudflareSiteKey}
              onChange={(e) => handleChange("cloudflareSiteKey", e.target.value)}
              placeholder="0x4AAAAAAA..."
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.captchaSettings.cloudflare.secretKey")}</Label>
            <Input
              type="password"
              value={form.cloudflareSecretKey}
              onChange={(e) => handleChange("cloudflareSecretKey", e.target.value)}
              placeholder="0x4AAAAAAA..."
            />
          </div>

          <div className="space-y-3 rounded-md border border-dashed p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium">{t("admin.captchaSettings.testStatus")}</p>
                <p className="text-xs text-muted-foreground">
                  {providerStatus.cloudflare.verified
                    ? `${t("admin.captchaSettings.verified")} (${new Date(providerStatus.cloudflare.lastVerifiedAt!).toLocaleString()})`
                    : t("admin.captchaSettings.testRequired")
                  }
                </p>
              </div>
              {providerStatus.cloudflare.verified ? (
                <CheckCircle2 className="h-5 w-5 text-green-500" />
              ) : (
                <AlertTriangle className="h-5 w-5 text-amber-500" />
              )}
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={startCloudflareTest}
              disabled={!form.cloudflareSiteKey || !form.cloudflareSecretKey || testCloudflareMutation.isPending}
              className="w-full"
            >
              {testCloudflareMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {t("admin.captchaSettings.verifying")}
                </>
              ) : showTester.cloudflare ? (
                t("admin.captchaSettings.retest")
              ) : providerStatus.cloudflare.verified ? (
                t("admin.captchaSettings.retest")
              ) : (
                t("admin.captchaSettings.startTest")
              )}
            </Button>
            {showTester.cloudflare && (
              <div className="rounded-md border p-4 text-center space-y-3">
                {!turnstileReady && (
                  <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    {t("admin.captchaSettings.loadingWidget")}
                  </div>
                )}
                {turnstileReady && (
                  <div className="flex justify-center">
                    <Turnstile
                      ref={turnstileRef}
                      siteKey={form.cloudflareSiteKey}
                      onVerify={handleCloudflareVerify}
                      onError={() => toast.error(t("admin.captchaSettings.widgetError"))}
                      onExpire={() => {}}
                    />
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="rounded-md bg-muted p-3 text-sm">
            <p className="font-medium mb-1">{t("admin.captchaSettings.getKeys")}</p>
            <p className="text-muted-foreground">
              <a
                href="https://dash.cloudflare.com/?to=/:account/turnstile"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                Cloudflare Turnstile
              </a>{" "}
              {t("admin.captchaSettings.cloudflare.guide")}
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Cap Settings */}
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.captchaSettings.cap.title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>{t("admin.captchaSettings.cap.instanceUrl")}</Label>
            <Input
              value={form.capInstanceUrl}
              onChange={(e) => handleChange("capInstanceUrl", e.target.value.trim())}
              placeholder="https://cap.example.com"
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.captchaSettings.cap.siteKey")}</Label>
            <Input
              value={form.capSiteKey}
              onChange={(e) => handleChange("capSiteKey", e.target.value.trim())}
              placeholder="d9256640cb53"
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.captchaSettings.cap.secretKey")}</Label>
            <Input
              type="password"
              value={form.capSecretKey}
              onChange={(e) => handleChange("capSecretKey", e.target.value)}
              placeholder={t("admin.captchaSettings.cap.secretKeyPlaceholder")}
            />
          </div>

          <div className="space-y-3 rounded-md border border-dashed p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium">{t("admin.captchaSettings.testStatus")}</p>
                <p className="text-xs text-muted-foreground">
                  {providerStatus.cap.verified
                    ? `${t("admin.captchaSettings.verified")} (${new Date(providerStatus.cap.lastVerifiedAt!).toLocaleString()})`
                    : t("admin.captchaSettings.testRequired")
                  }
                </p>
              </div>
              {providerStatus.cap.verified ? (
                <CheckCircle2 className="h-5 w-5 text-green-500" />
              ) : (
                <AlertTriangle className="h-5 w-5 text-amber-500" />
              )}
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={startCapTest}
              disabled={!form.capInstanceUrl || !form.capSiteKey || !form.capSecretKey || testCapMutation.isPending}
              className="w-full"
            >
              {testCapMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {t("admin.captchaSettings.verifying")}
                </>
              ) : showTester.cap || providerStatus.cap.verified ? (
                t("admin.captchaSettings.retest")
              ) : (
                t("admin.captchaSettings.startTest")
              )}
            </Button>
            {showTester.cap && (
              <div className="rounded-md border p-4 text-center space-y-3">
                {!capReady && (
                  <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    {t("admin.captchaSettings.loadingWidget")}
                  </div>
                )}
                {capReady && capApiEndpoint && (
                  <div className="flex justify-center">
                    <CapWidget
                      ref={capRef}
                      apiEndpoint={capApiEndpoint}
                      onVerify={handleCapVerify}
                      onError={() => toast.error(t("admin.captchaSettings.widgetError"))}
                      onExpire={() => {}}
                    />
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="rounded-md bg-muted p-3 text-sm">
            <p className="font-medium mb-1">{t("admin.captchaSettings.getKeys")}</p>
            <p className="text-muted-foreground">
              <a
                href="https://trycap.dev/guide/standalone/"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                Cap Standalone
              </a>{" "}
              {t("admin.captchaSettings.cap.guide")}
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Geetest Settings */}
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.captchaSettings.geetest.title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>{t("admin.captchaSettings.geetest.captchaId")}</Label>
            <Input
              value={form.geetestCaptchaId}
              onChange={(e) => handleChange("geetestCaptchaId", e.target.value.trim())}
              placeholder={t("admin.captchaSettings.geetest.captchaIdPlaceholder")}
              className={form.geetestCaptchaId && form.geetestCaptchaId.length !== 32 ? "border-amber-500" : ""}
            />
            {form.geetestCaptchaId && form.geetestCaptchaId.length !== 32 && (
              <p className="text-xs text-amber-600">
                {t("admin.captchaSettings.geetest.captchaIdLengthError", { length: form.geetestCaptchaId.length })}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label>{t("admin.captchaSettings.geetest.captchaKey")}</Label>
            <Input
              type="password"
              value={form.geetestCaptchaKey}
              onChange={(e) => handleChange("geetestCaptchaKey", e.target.value)}
              placeholder={t("admin.captchaSettings.geetest.captchaKeyPlaceholder")}
            />
          </div>

          <div className="space-y-3 rounded-md border border-dashed p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium">{t("admin.captchaSettings.testStatus")}</p>
                <p className="text-xs text-muted-foreground">
                  {providerStatus.geetest.verified
                    ? `${t("admin.captchaSettings.verified")} (${new Date(providerStatus.geetest.lastVerifiedAt!).toLocaleString()})`
                    : t("admin.captchaSettings.testRequired")
                  }
                </p>
              </div>
              {providerStatus.geetest.verified ? (
                <CheckCircle2 className="h-5 w-5 text-green-500" />
              ) : (
                <AlertTriangle className="h-5 w-5 text-amber-500" />
              )}
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={startGeetestTest}
              disabled={!form.geetestCaptchaId || !form.geetestCaptchaKey || testGeetestMutation.isPending}
              className="w-full"
            >
              {testGeetestMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {t("admin.captchaSettings.verifying")}
                </>
              ) : (
                t("admin.captchaSettings.startTest")
              )}
            </Button>
            {showTester.geetest && (
              <div className="rounded-md border p-4 text-center space-y-3">
                {!geetestReady && (
                  <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    {t("admin.captchaSettings.loadingWidget")}
                  </div>
                )}
                {geetestReady && (
                  <div className="flex justify-center">
                    <Geetest
                      ref={geetestRef}
                      captchaId={form.geetestCaptchaId}
                      onSuccess={handleGeetestSuccess}
                      onError={handleGeetestError}
                      onReady={() => {}}
                    />
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="rounded-md bg-muted p-3 text-sm">
            <p className="font-medium mb-1">{t("admin.captchaSettings.getKeys")}</p>
            <p className="text-muted-foreground">
              <a
                href="https://www.geetest.com"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                {t("admin.captchaSettings.geetest.website")}
              </a>{" "}
              {t("admin.captchaSettings.geetest.guide")}
            </p>
          </div>
        </CardContent>
      </Card>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-muted-foreground">
          {isFormDirty ? t("admin.captchaSettings.unsavedChanges") : t("admin.captchaSettings.allChangesSaved")}
        </p>
        <Button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending || !isFormDirty}
        >
          {mutation.isPending ? t("admin.captchaSettings.saving") : t("admin.captchaSettings.saveConfig")}
        </Button>
      </div>
    </div>
  );
}
