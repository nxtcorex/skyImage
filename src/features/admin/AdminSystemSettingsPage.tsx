import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import {
  fetchGeneralSettings,
  updateGeneralSettings,
  type GeneralSettings,
  type GeneralSettingsUpdate
} from "@/lib/api";
import { SplashScreen } from "@/components/SplashScreen";
import { useI18n } from "@/i18n";
import {
  HIDEABLE_SIDEBAR_ITEMS,
  type SidebarItemConfig
} from "@/lib/navigation";

const defaultAdminImageDeleteReasonText = "图片已被管理员删除";
const defaultSystemAutoDeleteReasonText = "图片已被系统自动删除";

const defaultGeneralSettingsForm: GeneralSettings = {
  imageLoadRows: 4,
  userNotificationLimit: 50,
  adminImageDeleteDefaultReason: defaultAdminImageDeleteReasonText,
  systemAutoDeleteDefaultReason: defaultSystemAutoDeleteReasonText,
  enableCDN: false,
  enableGallery: true,
  enableHome: true,
  enableApi: true,
  enablePasskey: true,
  allowRegistration: true,
  registrationMode: "open",
  hiddenSidebarItems: []
};

export function AdminSystemSettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery<GeneralSettings>({
    queryKey: ["admin", "general-settings"],
    queryFn: fetchGeneralSettings
  });
  const [form, setForm] = useState<GeneralSettings>(defaultGeneralSettingsForm);
  const [initialForm, setInitialForm] = useState<GeneralSettings | null>(null);

  // Calculate if form is dirty - must be before any conditional returns
  const isFormDirty = useMemo(() => {
    if (!initialForm) {
      return false;
    }
    const keys = Object.keys(defaultGeneralSettingsForm) as (keyof GeneralSettings)[];
    return keys.some((key) => initialForm[key] !== form[key]);
  }, [initialForm, form]);

  // 侧边栏项按分组归类，用于渲染“侧边栏配置”开关；关键页面不可隐藏，不展示开关。
  const sidebarGroups = useMemo(() => {
    const grouped = new Map<string, SidebarItemConfig[]>();
    for (const item of HIDEABLE_SIDEBAR_ITEMS) {
      if (item.critical) {
        continue;
      }
      const list = grouped.get(item.groupKey) ?? [];
      list.push(item);
      grouped.set(item.groupKey, list);
    }
    return Array.from(grouped.entries()).map(([key, items]) => ({ key, items }));
  }, []);

  useEffect(() => {
    if (data) {
      const normalized = {
        ...defaultGeneralSettingsForm,
        ...data
      };
      setForm(normalized);
      setInitialForm(normalized);
    }
  }, [data]);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!initialForm) {
        return;
      }
      const patch: GeneralSettingsUpdate = {};
      (Object.keys(initialForm) as (keyof GeneralSettings)[]).forEach((key) => {
        if (initialForm[key] !== form[key]) {
          (patch as Record<string, unknown>)[key] = form[key];
        }
      });
      if (Object.keys(patch).length > 0) {
        await updateGeneralSettings(patch);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["site-config"] });
      queryClient.invalidateQueries({ queryKey: ["site-meta"] });
      queryClient.invalidateQueries({ queryKey: ["admin", "general-settings"] });
      toast.success(t("admin.systemSettings.saved"));
    },
    onError: (error) => toast.error(error.message)
  });


  if (isLoading) {
    return <SplashScreen message={t("admin.systemSettings.loading")} />;
  }
  if (error && !data) {
    const message =
      error.message === "account disabled"
        ? t("admin.systemSettings.disabled")
        : error.message;
    return (
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle>{t("admin.systemSettings.loadFailed")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-destructive">{message}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const handleChange = (field: keyof GeneralSettings, value: any) => {
    const actualValue = value === "indeterminate" ? false : value;
    setForm((prev) => ({ ...prev, [field]: actualValue }));
  };

  const handleSidebarToggle = (url: string, checked: boolean) => {
    setForm((prev) => {
      const hidden = new Set(prev.hiddenSidebarItems);
      if (checked) {
        hidden.add(url);
      } else {
        hidden.delete(url);
      }
      return { ...prev, hiddenSidebarItems: Array.from(hidden) };
    });
  };

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <h1 className="text-2xl font-semibold">{t("admin.systemSettings.title")}</h1>
        <p className="text-muted-foreground">{t("admin.systemSettings.description")}</p>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.systemSettings.imageLoad")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <Label>{t("admin.systemSettings.imageLoadRows")}</Label>
          <Input
            type="number"
            min={1}
            max={20}
            value={form.imageLoadRows}
            onChange={(e) => {
              const value = Number.parseInt(e.target.value, 10);
              handleChange("imageLoadRows", Number.isNaN(value) ? 1 : value);
            }}
          />
          <p className="text-xs text-muted-foreground">
            {t("admin.systemSettings.imageLoadRowsHint")}
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.systemSettings.notifications")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>{t("admin.systemSettings.userNotificationLimit")}</Label>
            <Input
              type="number"
              min={1}
              max={500}
              value={form.userNotificationLimit}
              onChange={(e) => {
                const value = Number.parseInt(e.target.value, 10);
                handleChange("userNotificationLimit", Number.isNaN(value) ? 50 : value);
              }}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.systemSettings.userNotificationLimitHint")}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t("admin.systemSettings.adminDeleteReason")}</Label>
            <Textarea
              rows={3}
              value={form.adminImageDeleteDefaultReason}
              onChange={(e) =>
                handleChange("adminImageDeleteDefaultReason", e.target.value)
              }
              placeholder={t("admin.systemSettings.adminDeleteReasonPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.systemSettings.adminDeleteReasonHint")}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t("admin.systemSettings.systemAutoDeleteReason")}</Label>
            <Textarea
              rows={3}
              value={form.systemAutoDeleteDefaultReason}
              onChange={(e) =>
                handleChange("systemAutoDeleteDefaultReason", e.target.value)
              }
              placeholder={t("admin.systemSettings.systemAutoDeleteReasonPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.systemSettings.systemAutoDeleteReasonHint")}
            </p>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.systemSettings.network")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between space-x-2 rounded-md border p-3">
            <div className="space-y-0.5">
              <Label>{t("admin.systemSettings.enableCDN")}</Label>
              <p className="text-xs text-muted-foreground">
                {t("admin.systemSettings.enableCDNHint")}
              </p>
            </div>
            <Switch
              checked={form.enableCDN}
              onCheckedChange={(checked) => handleChange("enableCDN", checked)}
            />
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.systemSettings.sidebarFeatures")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between space-x-2 rounded-md border p-3">
            <div className="space-y-0.5">
              <Label>{t("admin.systemSettings.enableGallery")}</Label>
            </div>
            <Switch
              checked={form.enableGallery}
              onCheckedChange={(checked) => handleChange("enableGallery", checked)}
            />
          </div>
          <div className="flex items-center justify-between space-x-2 rounded-md border p-3">
            <div className="space-y-0.5">
              <Label>{t("admin.systemSettings.enableHome")}</Label>
            </div>
            <Switch
              checked={form.enableHome}
              onCheckedChange={(checked) => handleChange("enableHome", checked)}
            />
          </div>
          <div className="flex items-center justify-between space-x-2 rounded-md border p-3">
            <div className="space-y-0.5">
              <Label>{t("admin.systemSettings.enableApi")}</Label>
            </div>
            <Switch
              checked={form.enableApi}
              onCheckedChange={(checked) => handleChange("enableApi", checked)}
            />
          </div>
          <div className="flex items-center justify-between space-x-2 rounded-md border p-3">
            <div className="space-y-0.5">
              <Label>{t("admin.systemSettings.enablePasskey")}</Label>
            </div>
            <Switch
              checked={form.enablePasskey}
              onCheckedChange={(checked) => handleChange("enablePasskey", checked)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.systemSettings.registrationMode")}</Label>
            <Select
              value={form.registrationMode || "open"}
              onValueChange={(value) => {
                handleChange("registrationMode", value);
                handleChange("allowRegistration", value !== "closed");
              }}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="open">{t("admin.systemSettings.registrationMode.open")}</SelectItem>
                <SelectItem value="oauth_only">{t("admin.systemSettings.registrationMode.oauthOnly")}</SelectItem>
                <SelectItem value="closed">{t("admin.systemSettings.registrationMode.closed")}</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {t("admin.systemSettings.registrationModeHint")}
            </p>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.systemSettings.sidebar")}</CardTitle>
          <p className="text-xs text-muted-foreground">
            {t("admin.systemSettings.sidebarHint")}
          </p>
        </CardHeader>
        <CardContent className="space-y-5">
          {sidebarGroups.map((group) => (
            <div key={group.key} className="space-y-2">
              <Label className="text-muted-foreground">
                {t(group.key)}
              </Label>
              <div className="space-y-2">
                {group.items.map((item) => {
                  const isHidden = form.hiddenSidebarItems.includes(item.url);
                  return (
                    <div
                      key={item.url}
                      className="flex items-center justify-between space-x-2 rounded-md border p-3"
                    >
                      <div className="space-y-0.5">
                        <Label>{t(item.nameKey)}</Label>
                      </div>
                      <Switch
                        checked={!isHidden}
                        onCheckedChange={(checked) =>
                          handleSidebarToggle(item.url, !checked)
                        }
                      />
                    </div>
                  );
                })}
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-muted-foreground">
          {isFormDirty ? t("admin.systemSettings.unsaved") : t("admin.systemSettings.clean")}
        </p>
        <Button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending || !isFormDirty}
        >
          {mutation.isPending ? t("common.saving") : t("admin.systemSettings.saveAll")}
        </Button>
      </div>
    </div>
  );
}
