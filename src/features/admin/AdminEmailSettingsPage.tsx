import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Mail, Send } from "lucide-react";

import { SplashScreen } from "@/components/SplashScreen";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useI18n } from "@/i18n";
import {
  fetchEmailSettings,
  testSmtpEmail,
  type EmailSettings,
  type EmailSettingsUpdate,
  updateEmailSettings,
  fetchSiteConfig
} from "@/lib/api";

type MailTemplateKey =
  | "test"
  | "registerVerify"
  | "registerSuccess"
  | "loginNotification"
  | "forgotPassword"
  | "ticketCreated"
  | "ticketReplyUser"
  | "ticketReplyAdmin"
  | "ticketStatus";

type MailTemplateVariable = {
  token: string;
  labelKey: string;
};

type MailTemplateDefinition = {
  labelKey: string;
  subjectField: keyof EmailSettings;
  bodyField: keyof EmailSettings;
  defaultSubject: string;
  defaultBody: string;
  variables: MailTemplateVariable[];
};

const templateVariables = {
  siteName: {
    token: "{{site_name}}",
    labelKey: "admin.emailSettings.variable.siteName"
  },
  userName: {
    token: "{{user_name}}",
    labelKey: "admin.emailSettings.variable.userName"
  },
  email: {
    token: "{{email}}",
    labelKey: "admin.emailSettings.variable.email"
  },
  verificationCode: {
    token: "{{verification_code}}",
    labelKey: "admin.emailSettings.variable.verificationCode"
  },
  resetLink: {
    token: "{{reset_link}}",
    labelKey: "admin.emailSettings.variable.resetLink"
  },
  loginIp: {
    token: "{{login_ip}}",
    labelKey: "admin.emailSettings.variable.loginIp"
  },
  testEmail: {
    token: "{{test_email}}",
    labelKey: "admin.emailSettings.variable.testEmail"
  },
  currentTime: {
    token: "{{current_time}}",
    labelKey: "admin.emailSettings.variable.currentTime"
  },
  ticketNo: {
    token: "{{ticket_no}}",
    labelKey: "admin.emailSettings.variable.ticketNo"
  },
  ticketSubject: {
    token: "{{ticket_subject}}",
    labelKey: "admin.emailSettings.variable.ticketSubject"
  },
  ticketStatus: {
    token: "{{ticket_status}}",
    labelKey: "admin.emailSettings.variable.ticketStatus"
  },
  ticketPriority: {
    token: "{{ticket_priority}}",
    labelKey: "admin.emailSettings.variable.ticketPriority"
  },
  ticketUrl: {
    token: "{{ticket_url}}",
    labelKey: "admin.emailSettings.variable.ticketUrl"
  },
  replyBody: {
    token: "{{reply_body}}",
    labelKey: "admin.emailSettings.variable.replyBody"
  },
  staffName: {
    token: "{{staff_name}}",
    labelKey: "admin.emailSettings.variable.staffName"
  },
  adminName: {
    token: "{{admin_name}}",
    labelKey: "admin.emailSettings.variable.adminName"
  }
} satisfies Record<string, MailTemplateVariable>;

const mailTemplateDefinitions: Record<MailTemplateKey, MailTemplateDefinition> = {
  test: {
    labelKey: "admin.emailSettings.template.test",
    subjectField: "mailTestSubject",
    bodyField: "mailTestBody",
    defaultSubject: "{{site_name}} 邮件测试",
    defaultBody: "如果你看到这条消息代表邮件已正常可用",
    variables: [templateVariables.siteName, templateVariables.testEmail, templateVariables.currentTime]
  },
  registerVerify: {
    labelKey: "admin.emailSettings.template.registerVerify",
    subjectField: "mailRegisterVerifySubject",
    bodyField: "mailRegisterVerifyBody",
    defaultSubject: "{{site_name}} 注册验证码",
    defaultBody: `您好，

您正在注册 {{site_name}}

您的验证码是：{{verification_code}}

验证码有效期为 5 分钟，请尽快完成验证。

如果这不是您本人的操作，请忽略此邮件。

此邮件由系统自动发送，请勿回复。`,
    variables: [
      templateVariables.siteName,
      templateVariables.email,
      templateVariables.verificationCode,
      templateVariables.currentTime
    ]
  },
  registerSuccess: {
    labelKey: "admin.emailSettings.template.registerSuccess",
    subjectField: "mailRegisterSuccessSubject",
    bodyField: "mailRegisterSuccessBody",
    defaultSubject: "欢迎注册 {{site_name}}",
    defaultBody: `您好 {{user_name}}，

恭喜您成功注册 {{site_name}} 成功！

您的账号已激活，现在可以开始使用我们的服务了。

如有任何问题，请联系管理员。

此邮件由系统自动发送，请勿回复。`,
    variables: [
      templateVariables.siteName,
      templateVariables.userName,
      templateVariables.email,
      templateVariables.currentTime
    ]
  },
  loginNotification: {
    labelKey: "admin.emailSettings.template.loginNotification",
    subjectField: "mailLoginNotificationSubject",
    bodyField: "mailLoginNotificationBody",
    defaultSubject: "{{site_name}} 登录提醒",
    defaultBody: `您好 {{user_name}}，

您的账号刚刚登录了 {{site_name}}。

登录信息：
- 登录 IP：{{login_ip}}

如果这不是您本人的操作，请立即修改密码并联系管理员。

此邮件由系统自动发送，请勿回复。`,
    variables: [
      templateVariables.siteName,
      templateVariables.userName,
      templateVariables.email,
      templateVariables.loginIp,
      templateVariables.currentTime
    ]
  },
  forgotPassword: {
    labelKey: "admin.emailSettings.template.forgotPassword",
    subjectField: "mailForgotPasswordSubject",
    bodyField: "mailForgotPasswordBody",
    defaultSubject: "{{site_name}} 密码重置验证码",
    defaultBody: `您好，

您正在重置 {{site_name}} 的登录密码。

验证码：{{verification_code}}
重置链接：{{reset_link}}

验证码和链接有效期为 15 分钟。
如果这不是您本人的操作，请忽略此邮件。

此邮件由系统自动发送，请勿回复。`,
    variables: [
      templateVariables.siteName,
      templateVariables.email,
      templateVariables.verificationCode,
      templateVariables.resetLink,
      templateVariables.currentTime
    ]
  },
  ticketCreated: {
    labelKey: "admin.emailSettings.template.ticketCreated",
    subjectField: "mailTicketCreatedSubject",
    bodyField: "mailTicketCreatedBody",
    defaultSubject: "[{{site_name}}] 新工单 {{ticket_no}}",
    defaultBody: `您好 {{admin_name}}，

用户 {{user_name}} 提交了新工单。

工单编号：{{ticket_no}}
标题：{{ticket_subject}}
优先级：{{ticket_priority}}

处理入口：{{ticket_url}}

此邮件由系统自动发送，请勿回复。`,
    variables: [
      templateVariables.siteName,
      templateVariables.adminName,
      templateVariables.userName,
      templateVariables.ticketNo,
      templateVariables.ticketSubject,
      templateVariables.ticketPriority,
      templateVariables.ticketUrl,
      templateVariables.currentTime
    ]
  },
  ticketReplyUser: {
    labelKey: "admin.emailSettings.template.ticketReplyUser",
    subjectField: "mailTicketReplyUserSubject",
    bodyField: "mailTicketReplyUserBody",
    defaultSubject: "[{{site_name}}] 工单 {{ticket_no}} 有新回复",
    defaultBody: `您好 {{user_name}}，

您的工单有新的工作人员回复。

工单编号：{{ticket_no}}
标题：{{ticket_subject}}
工作人员：{{staff_name}}

回复内容：
{{reply_body}}

查看工单：{{ticket_url}}

此邮件由系统自动发送，请勿回复。`,
    variables: [
      templateVariables.siteName,
      templateVariables.userName,
      templateVariables.staffName,
      templateVariables.ticketNo,
      templateVariables.ticketSubject,
      templateVariables.replyBody,
      templateVariables.ticketUrl,
      templateVariables.currentTime
    ]
  },
  ticketReplyAdmin: {
    labelKey: "admin.emailSettings.template.ticketReplyAdmin",
    subjectField: "mailTicketReplyAdminSubject",
    bodyField: "mailTicketReplyAdminBody",
    defaultSubject: "[{{site_name}}] 工单 {{ticket_no}} 用户回复",
    defaultBody: `您好 {{admin_name}}，

工单收到用户回复。

工单编号：{{ticket_no}}
标题：{{ticket_subject}}
用户：{{user_name}}

回复内容：
{{reply_body}}

处理入口：{{ticket_url}}

此邮件由系统自动发送，请勿回复。`,
    variables: [
      templateVariables.siteName,
      templateVariables.adminName,
      templateVariables.userName,
      templateVariables.ticketNo,
      templateVariables.ticketSubject,
      templateVariables.replyBody,
      templateVariables.ticketUrl,
      templateVariables.currentTime
    ]
  },
  ticketStatus: {
    labelKey: "admin.emailSettings.template.ticketStatus",
    subjectField: "mailTicketStatusSubject",
    bodyField: "mailTicketStatusBody",
    defaultSubject: "[{{site_name}}] 工单 {{ticket_no}} 状态更新",
    defaultBody: `您好 {{user_name}}，

您的工单状态已更新。

工单编号：{{ticket_no}}
标题：{{ticket_subject}}
当前状态：{{ticket_status}}

查看工单：{{ticket_url}}

此邮件由系统自动发送，请勿回复。`,
    variables: [
      templateVariables.siteName,
      templateVariables.userName,
      templateVariables.ticketNo,
      templateVariables.ticketSubject,
      templateVariables.ticketStatus,
      templateVariables.ticketUrl,
      templateVariables.currentTime
    ]
  }
};

const mailTemplateOrder: MailTemplateKey[] = [
  "test",
  "registerVerify",
  "registerSuccess",
  "loginNotification",
  "forgotPassword",
  "ticketCreated",
  "ticketReplyUser",
  "ticketReplyAdmin",
  "ticketStatus"
];

const defaultEmailSettingsForm: EmailSettings = {
  smtpHost: "",
  smtpPort: "",
  smtpUsername: "",
  smtpPassword: "",
  smtpFrom: "",
  smtpSecure: false,
  mailTestSubject: "",
  mailTestBody: "",
  mailRegisterVerifySubject: "",
  mailRegisterVerifyBody: "",
  mailRegisterSuccessSubject: "",
  mailRegisterSuccessBody: "",
  mailLoginNotificationSubject: "",
  mailLoginNotificationBody: "",
  mailForgotPasswordSubject: "",
  mailForgotPasswordBody: "",
  mailTicketCreatedSubject: "",
  mailTicketCreatedBody: "",
  mailTicketReplyUserSubject: "",
  mailTicketReplyUserBody: "",
  mailTicketReplyAdminSubject: "",
  mailTicketReplyAdminBody: "",
  mailTicketStatusSubject: "",
  mailTicketStatusBody: "",
  enableRegisterVerify: false,
  enableLoginNotification: false,
  enableForgotPassword: false,
  enableForgotPasswordTurnstile: false,
  enableForgotPasswordTurnstileRequest: false,
  enableForgotPasswordTurnstileReset: false
};

function applyTemplateDefaults(input: EmailSettings) {
  const next = { ...input };

  for (const definition of Object.values(mailTemplateDefinitions)) {
    const subjectValue = next[definition.subjectField];
    if (typeof subjectValue === "string" && !subjectValue.trim()) {
      next[definition.subjectField] = definition.defaultSubject as never;
    }

    const bodyValue = next[definition.bodyField];
    if (typeof bodyValue === "string" && !bodyValue.trim()) {
      next[definition.bodyField] = definition.defaultBody as never;
    }
  }

  return next;
}

function normalizeTemplateFieldsForSave(input: Partial<EmailSettings>) {
  const next = { ...input };

  for (const definition of Object.values(mailTemplateDefinitions)) {
    const subjectValue = next[definition.subjectField];
    if (typeof subjectValue === "string" && subjectValue === definition.defaultSubject) {
      next[definition.subjectField] = "" as never;
    }

    const bodyValue = next[definition.bodyField];
    if (typeof bodyValue === "string" && bodyValue === definition.defaultBody) {
      next[definition.bodyField] = "" as never;
    }
  }

  return next;
}

export function AdminEmailSettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const subjectInputRef = useRef<HTMLInputElement>(null);
  const bodyTextareaRef = useRef<HTMLTextAreaElement>(null);

  const { data, isLoading, error } = useQuery<EmailSettings>({
    queryKey: ["admin", "email-settings"],
    queryFn: fetchEmailSettings
  });

  const { data: siteConfig } = useQuery({
    queryKey: ["site-config"],
    queryFn: fetchSiteConfig
  });

  const [form, setForm] = useState<EmailSettings>(defaultEmailSettingsForm);
  const [initialForm, setInitialForm] = useState<EmailSettings | null>(null);
  const [testEmail, setTestEmail] = useState("");
  const [selectedTemplate, setSelectedTemplate] = useState<MailTemplateKey>("test");
  const [resetDialogOpen, setResetDialogOpen] = useState(false);

  const activeTemplate = mailTemplateDefinitions[selectedTemplate];

  const isFormDirty = useMemo(() => {
    if (!initialForm) {
      return false;
    }
    return (Object.keys(form) as (keyof EmailSettings)[]).some(
      (key) => initialForm[key] !== form[key]
    );
  }, [initialForm, form]);

  useEffect(() => {
    if (!data) return;
    const normalized = applyTemplateDefaults({
      ...defaultEmailSettingsForm,
      ...data
    });
    setForm(normalized);
    setInitialForm(normalized);
  }, [data]);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!initialForm) return;
      const patch: EmailSettingsUpdate = {};
      (Object.keys(initialForm) as (keyof EmailSettings)[]).forEach((key) => {
        if (initialForm[key] !== form[key]) {
          (patch as Record<string, unknown>)[key] = form[key];
        }
      });
      if (Object.keys(patch).length > 0) {
        await updateEmailSettings(normalizeTemplateFieldsForSave(patch));
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "email-settings"] });
      queryClient.invalidateQueries({ queryKey: ["site-config"] });
      queryClient.invalidateQueries({ queryKey: ["site-meta"] });
      toast.success(t("admin.emailSettings.saved"));
    },
    onError: (mutationError) => toast.error(mutationError.message)
  });

  const testEmailMutation = useMutation({
    mutationFn: testSmtpEmail,
    onSuccess: (result) => {
      if (result.success) {
        toast.success(t("admin.emailSettings.testMailSuccess"));
        setTestEmail("");
      } else {
        toast.error(result.message || t("admin.emailSettings.testMailFailed"));
      }
    },
    onError: (mutationError) => toast.error(mutationError.message)
  });

  if (isLoading) {
    return <SplashScreen message={t("admin.emailSettings.loading")} />;
  }

  if (error && !data) {
    const message =
      error.message === "account disabled"
        ? t("admin.emailSettings.disabled")
        : error.message;
    return (
      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle>{t("admin.emailSettings.loadFailed")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-destructive">{message}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const handleChange = (field: keyof EmailSettings, value: unknown) => {
    const actualValue = value === "indeterminate" ? false : value;
    setForm((prev) => ({ ...prev, [field]: actualValue as never }));
  };

  const getStringFieldValue = (field: keyof EmailSettings) => {
    const value = form[field];
    return typeof value === "string" ? value : "";
  };

  const insertTemplateVariable = (target: "subject" | "body", token: string) => {
    const field = target === "subject" ? activeTemplate.subjectField : activeTemplate.bodyField;
    const input = target === "subject" ? subjectInputRef.current : bodyTextareaRef.current;
    const currentValue = getStringFieldValue(field);
    const start = input?.selectionStart ?? currentValue.length;
    const end = input?.selectionEnd ?? currentValue.length;
    const nextValue = `${currentValue.slice(0, start)}${token}${currentValue.slice(end)}`;

    handleChange(field, nextValue);

    requestAnimationFrame(() => {
      const element = target === "subject" ? subjectInputRef.current : bodyTextareaRef.current;
      if (!element) {
        return;
      }
      const nextCursor = start + token.length;
      element.focus();
      element.setSelectionRange(nextCursor, nextCursor);
    });
  };

  const handleResetTemplate = () => {
    handleChange(activeTemplate.subjectField, activeTemplate.defaultSubject);
    handleChange(activeTemplate.bodyField, activeTemplate.defaultBody);
    setResetDialogOpen(false);
  };

  const handleTestEmail = () => {
    if (!testEmail) {
      toast.error(t("admin.emailSettings.testMailRequired"));
      return;
    }
    if (!form.smtpHost || !form.smtpPort || !form.smtpUsername) {
      toast.error(t("admin.emailSettings.configRequired"));
      return;
    }
    testEmailMutation.mutate({
      testEmail,
      siteTitle: siteConfig?.title ?? "",
      smtpHost: form.smtpHost,
      smtpPort: form.smtpPort,
      smtpUsername: form.smtpUsername,
      smtpPassword: form.smtpPassword,
      smtpFrom: form.smtpFrom,
      smtpSecure: form.smtpSecure,
      mailTestSubject: form.mailTestSubject,
      mailTestBody: form.mailTestBody
    });
  };

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <h1 className="text-2xl font-semibold">{t("nav.systemSettings")}</h1>
        <p className="text-muted-foreground">{t("admin.emailSettings.description")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.emailSettings.card")}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label>Host</Label>
            <Input value={form.smtpHost} onChange={(e) => handleChange("smtpHost", e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>Port</Label>
            <Input value={form.smtpPort} onChange={(e) => handleChange("smtpPort", e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.emailSettings.username")}</Label>
            <Input
              value={form.smtpUsername}
              onChange={(e) => handleChange("smtpUsername", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.emailSettings.password")}</Label>
            <Input
              type="password"
              value={form.smtpPassword}
              onChange={(e) => handleChange("smtpPassword", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("admin.emailSettings.from")}</Label>
            <Input
              type="email"
              placeholder="noreply@yourdomain.com"
              value={form.smtpFrom}
              onChange={(e) => handleChange("smtpFrom", e.target.value)}
            />
          </div>
          <div className="space-y-2 md:col-span-2">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="smtpSecure"
                checked={form.smtpSecure}
                onCheckedChange={(checked) => handleChange("smtpSecure", checked)}
              />
              <Label htmlFor="smtpSecure">{t("admin.emailSettings.secure")}</Label>
            </div>
          </div>
          <div className="mt-4 space-y-2 border-t pt-4 md:col-span-2">
            <Label className="flex items-center gap-2">
              <Mail className="h-4 w-4" />
              {t("admin.emailSettings.testMail")}
            </Label>
            <div className="flex gap-2">
              <Input
                type="email"
                placeholder={t("admin.emailSettings.testMailPlaceholder")}
                value={testEmail}
                onChange={(e) => setTestEmail(e.target.value)}
                className="flex-1"
              />
              <Button
                type="button"
                variant="outline"
                onClick={handleTestEmail}
                disabled={testEmailMutation.isPending}
              >
                <Send className="mr-2 h-4 w-4" />
                {testEmailMutation.isPending
                  ? t("admin.emailSettings.testMailSending")
                  : t("admin.emailSettings.testMailSend")}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.emailSettings.notifications")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center space-x-2">
            <Checkbox
              id="enableRegisterVerify"
              checked={form.enableRegisterVerify}
              onCheckedChange={(checked) => handleChange("enableRegisterVerify", checked)}
            />
            <Label htmlFor="enableRegisterVerify">
              {t("admin.emailSettings.enableRegisterVerify")}
            </Label>
          </div>
          <div className="flex items-center space-x-2">
            <Checkbox
              id="enableLoginNotification"
              checked={form.enableLoginNotification}
              onCheckedChange={(checked) => handleChange("enableLoginNotification", checked)}
            />
            <Label htmlFor="enableLoginNotification">
              {t("admin.emailSettings.enableLoginNotification")}
            </Label>
          </div>
          <div className="flex items-center space-x-2">
            <Checkbox
              id="enableForgotPassword"
              checked={form.enableForgotPassword}
              onCheckedChange={(checked) => handleChange("enableForgotPassword", checked)}
            />
            <Label htmlFor="enableForgotPassword">
              {t("admin.emailSettings.enableForgotPassword")}
            </Label>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.emailSettings.templates")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-4 md:grid-cols-[minmax(0,260px)_1fr] md:items-end">
            <div className="space-y-2">
              <Label>{t("admin.emailSettings.templateType")}</Label>
              <Select
                value={selectedTemplate}
                onValueChange={(value) => setSelectedTemplate(value as MailTemplateKey)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.emailSettings.templateTypePlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  {mailTemplateOrder.map((key) => (
                    <SelectItem key={key} value={key}>
                      {t(mailTemplateDefinitions[key].labelKey)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center justify-between gap-3 rounded-md border border-dashed px-4 py-3">
              <p className="text-sm text-muted-foreground">
                {t("admin.emailSettings.templateHint")}
              </p>
              <Button type="button" variant="ghost" size="sm" onClick={() => setResetDialogOpen(true)}>
                {t("admin.emailSettings.resetTemplate")}
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            <Label>{t("admin.emailSettings.templateSubject")}</Label>
            <Input
              ref={subjectInputRef}
              value={getStringFieldValue(activeTemplate.subjectField)}
              onChange={(e) => handleChange(activeTemplate.subjectField, e.target.value)}
            />
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground">
                {t("admin.emailSettings.insertSubjectVariable")}
              </p>
              <div className="flex flex-wrap gap-2">
                {activeTemplate.variables.map((variable) => (
                  <Button
                    key={`subject-${variable.token}`}
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => insertTemplateVariable("subject", variable.token)}
                  >
                    {t(variable.labelKey)} {variable.token}
                  </Button>
                ))}
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <Label>{t("admin.emailSettings.templateBody")}</Label>
            <Textarea
              ref={bodyTextareaRef}
              value={getStringFieldValue(activeTemplate.bodyField)}
              onChange={(e) => handleChange(activeTemplate.bodyField, e.target.value)}
              rows={12}
            />
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground">
                {t("admin.emailSettings.insertBodyVariable")}
              </p>
              <div className="flex flex-wrap gap-2">
                {activeTemplate.variables.map((variable) => (
                  <Button
                    key={`body-${variable.token}`}
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => insertTemplateVariable("body", variable.token)}
                  >
                    {t(variable.labelKey)} {variable.token}
                  </Button>
                ))}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <AlertDialog open={resetDialogOpen} onOpenChange={setResetDialogOpen}>
        <AlertDialogContent size="sm">
          <AlertDialogHeader>
            <AlertDialogTitle>{t("admin.emailSettings.resetTemplateConfirmTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("admin.emailSettings.resetTemplateConfirmDescription", {
                name: t(activeTemplate.labelKey)
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={handleResetTemplate}>
              {t("admin.emailSettings.resetTemplateConfirmAction")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-xs text-muted-foreground">
          {isFormDirty ? t("admin.systemSettings.unsaved") : t("admin.systemSettings.clean")}
        </p>
        <Button onClick={() => mutation.mutate()} disabled={mutation.isPending || !isFormDirty}>
          {mutation.isPending ? t("common.saving") : t("admin.emailSettings.save")}
        </Button>
      </div>
    </div>
  );
}
