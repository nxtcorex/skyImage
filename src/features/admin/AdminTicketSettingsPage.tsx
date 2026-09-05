import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import {
  fetchTicketSettings,
  updateTicketSettings,
  fetchStrategies,
  fetchUsers,
  type TicketSettings,
  type TicketSettingsUpdate,
  type StrategyRecord
} from "@/lib/api";
import { SplashScreen } from "@/components/SplashScreen";
import { useI18n } from "@/i18n";

const defaultForm: TicketSettings = {
  attachmentStrategyId: 0,
  emailNotifyEnabled: false,
  emailNotifyMode: "all_admins",
  emailNotifyAdminIds: []
};

export function AdminTicketSettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin", "ticket-settings"],
    queryFn: fetchTicketSettings
  });
  const { data: strategies } = useQuery<StrategyRecord[]>({
    queryKey: ["admin", "strategies"],
    queryFn: fetchStrategies
  });
  const { data: users } = useQuery({
    queryKey: ["admin", "users"],
    queryFn: fetchUsers
  });
  const [form, setForm] = useState<TicketSettings>(defaultForm);
  const [initialForm, setInitialForm] = useState<TicketSettings | null>(null);

  const adminUsers = useMemo(
    () =>
      (users || []).filter(
        (u: { isAdmin?: boolean; isSuperAdmin?: boolean }) => u.isAdmin || u.isSuperAdmin
      ) as Array<{
        id: string | number;
        name?: string;
        email?: string;
        isAdmin?: boolean;
        isSuperAdmin?: boolean;
      }>,
    [users]
  );

  const isFormDirty = useMemo(() => {
    if (!initialForm) return false;
    return (
      initialForm.attachmentStrategyId !== form.attachmentStrategyId ||
      initialForm.emailNotifyEnabled !== form.emailNotifyEnabled ||
      initialForm.emailNotifyMode !== form.emailNotifyMode ||
      initialForm.emailNotifyAdminIds.join(",") !== form.emailNotifyAdminIds.join(",")
    );
  }, [initialForm, form]);

  useEffect(() => {
    if (!data) return;
    const normalized = {
      ...defaultForm,
      ...data,
      emailNotifyAdminIds: data.emailNotifyAdminIds || []
    };
    setForm(normalized);
    setInitialForm(normalized);
  }, [data]);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!initialForm) return;
      const patch: TicketSettingsUpdate = {};
      if (initialForm.attachmentStrategyId !== form.attachmentStrategyId) {
        patch.attachmentStrategyId = form.attachmentStrategyId;
      }
      if (initialForm.emailNotifyEnabled !== form.emailNotifyEnabled) {
        patch.emailNotifyEnabled = form.emailNotifyEnabled;
      }
      if (initialForm.emailNotifyMode !== form.emailNotifyMode) {
        patch.emailNotifyMode = form.emailNotifyMode;
      }
      if (initialForm.emailNotifyAdminIds.join(",") !== form.emailNotifyAdminIds.join(",")) {
        patch.emailNotifyAdminIds = form.emailNotifyAdminIds;
      }
      if (Object.keys(patch).length > 0) {
        await updateTicketSettings(patch);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "ticket-settings"] });
      queryClient.invalidateQueries({ queryKey: ["tickets", "attachment-strategy"] });
      toast.success(t("admin.ticketSettings.saved"));
    },
    onError: (err) => toast.error(err.message)
  });

  if (isLoading) return <SplashScreen message={t("admin.ticketSettings.loading")} />;
  if (error && !data) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.ticketSettings.loadFailed")}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-destructive">{error.message}</p>
        </CardContent>
      </Card>
    );
  }

  const toggleAdmin = (id: string, checked: boolean) => {
    setForm((prev) => {
      const set = new Set(prev.emailNotifyAdminIds);
      if (checked) set.add(id);
      else set.delete(id);
      return { ...prev, emailNotifyAdminIds: Array.from(set) };
    });
  };

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <h1 className="text-2xl font-semibold">{t("admin.ticketSettings.title")}</h1>
        <p className="text-muted-foreground">{t("admin.ticketSettings.description")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.ticketSettings.attachment")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <Label>{t("admin.ticketSettings.attachmentStrategy")}</Label>
          <Select
            value={String(form.attachmentStrategyId || 0)}
            onValueChange={(value) =>
              setForm((prev) => ({
                ...prev,
                attachmentStrategyId: Number.parseInt(value, 10) || 0
              }))
            }
          >
            <SelectTrigger>
              <SelectValue placeholder={t("admin.ticketSettings.attachmentNone")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="0">{t("admin.ticketSettings.attachmentNone")}</SelectItem>
              {(strategies || []).map((strategy) => (
                <SelectItem key={strategy.id} value={String(strategy.id)}>
                  {strategy.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground">
            {t("admin.ticketSettings.attachmentHint")}
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.ticketSettings.email")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between rounded-md border p-3">
            <div className="space-y-0.5">
              <Label>{t("admin.ticketSettings.emailEnabled")}</Label>
              <p className="text-xs text-muted-foreground">
                {t("admin.ticketSettings.emailEnabledHint")}
              </p>
            </div>
            <Switch
              checked={form.emailNotifyEnabled}
              onCheckedChange={(checked) =>
                setForm((prev) => ({ ...prev, emailNotifyEnabled: checked }))
              }
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.ticketSettings.emailMode")}</Label>
            <Select
              value={form.emailNotifyMode}
              onValueChange={(value) =>
                setForm((prev) => ({
                  ...prev,
                  emailNotifyMode: value as TicketSettings["emailNotifyMode"]
                }))
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all_admins">{t("admin.ticketSettings.modeAll")}</SelectItem>
                <SelectItem value="selected">{t("admin.ticketSettings.modeSelected")}</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">{t("admin.ticketSettings.emailModeHint")}</p>
          </div>
          {form.emailNotifyMode === "selected" && (
            <div className="space-y-2 rounded-md border p-3">
              <Label>{t("admin.ticketSettings.selectAdmins")}</Label>
              {!adminUsers.length ? (
                <p className="text-sm text-muted-foreground">{t("admin.ticketSettings.noAdmins")}</p>
              ) : (
                <div className="space-y-2">
                  {adminUsers.map((user) => {
                    const id = String(user.id);
                    const checked = form.emailNotifyAdminIds.includes(id);
                    return (
                      <label key={id} className="flex items-center gap-2 text-sm">
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(v) => toggleAdmin(id, v === true)}
                        />
                        <span>
                          {user.name || user.email}
                          {user.isSuperAdmin ? ` (${t("admin.ticketSettings.superAdmin")})` : ""}
                        </span>
                      </label>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-muted-foreground">
          {isFormDirty ? t("admin.ticketSettings.unsaved") : t("admin.ticketSettings.clean")}
        </p>
        <Button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending || !isFormDirty}
        >
          {mutation.isPending ? t("common.saving") : t("admin.ticketSettings.save")}
        </Button>
      </div>
    </div>
  );
}
