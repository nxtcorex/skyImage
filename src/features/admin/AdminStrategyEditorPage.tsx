import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ListPlus, Plus, Trash2 } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import {
  fetchAuditProfiles,
  fetchGroups,
  fetchStrategies,
  saveStrategy,
  type AuditProfileRecord,
  type GroupRecord,
  type StrategyRecord
} from "@/lib/api";
import { useI18n } from "@/i18n";

const splitPublicUrls = (raw?: string) =>
  String(raw || "")
    .replace(/；/g, ";")
    .split(";")
    .map((item) => item.trim())
    .filter(Boolean)
    .filter((item, index, arr) => arr.findIndex((v) => v.toLowerCase() === item.toLowerCase()) === index);

const joinPublicUrls = (items: string[]) =>
  items
    .map((item) => item.trim())
    .filter(Boolean)
    .filter((item, index, arr) => arr.findIndex((v) => v.toLowerCase() === item.toLowerCase()) === index)
    .join(";");

export function AdminStrategyEditorPage() {
  const { t } = useI18n();
  const { id } = useParams();
  const isEditing = Boolean(id);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isS3CompatibleDriver = (driver?: string) => driver === "s3" || driver === "minio";
  const normalizePublicUrls = (raw?: string) => joinPublicUrls(splitPublicUrls(raw));
  const driverOptions = [
    { key: "local", label: t("admin.strategyEditor.driver.local") },
    { key: "webdav", label: t("admin.strategyEditor.driver.webdav") },
    { key: "ftp", label: t("admin.strategyEditor.driver.ftp") },
    { key: "sftp", label: t("admin.strategyEditor.driver.sftp") },
    { key: "s3", label: t("admin.strategyEditor.driver.s3") },
    { key: "minio", label: t("admin.strategyEditor.driver.minio") }
  ];

  const { data: strategies } = useQuery({
    queryKey: ["admin", "strategies"],
    queryFn: fetchStrategies
  });
  const { data: groups } = useQuery({
    queryKey: ["admin", "groups"],
    queryFn: fetchGroups
  });
  const { data: auditProfiles } = useQuery({
    queryKey: ["admin", "audits"],
    queryFn: fetchAuditProfiles
  });

  const [form, setForm] = useState<Partial<StrategyRecord>>({
    key: 1,
    name: "",
    intro: "",
      configs: {
        driver: "local",
        root: "storage/uploads",
        url: "",
        image_only_domain: false,
        webdav_endpoint: "",
        webdav_username: "",
        webdav_password: "",
        webdav_base_path: "",
        webdav_skip_tls_verify: false,
        ftp_host: "",
        ftp_port: 21,
        ftp_username: "",
        ftp_password: "",
        ftp_base_path: "",
        ftp_tls: false,
        ftp_skip_tls_verify: false,
        ftp_timeout: 15,
        sftp_host: "",
        sftp_port: 22,
        sftp_username: "",
        sftp_password: "",
        sftp_private_key: "",
        sftp_base_path: "",
        sftp_timeout: 15,
        s3_endpoint: "",
        s3_region: "",
        s3_bucket: "",
        s3_access_key: "",
      s3_secret_key: "",
      s3_session_token: "",
        s3_force_path_style: false,
      proxy: false,
      path_template: "{year}/{month}/{day}/{uuid}",
      image_audit_profile_id: "",
      image_audit_block_action: "delete",
      image_audit_error_action: "keep",
      enable_thumbnail: false,
      thumbnail_max_size: 400,
      thumbnail_quality: 25,
      thumbnail_format: "jpeg",
      thumbnail_strategy_id: ""
    }
  });
  const [selectedGroups, setSelectedGroups] = useState<number[]>([]);
  const [domainDialogOpen, setDomainDialogOpen] = useState(false);
  const [domainDrafts, setDomainDrafts] = useState<string[]>([""]);

  const openDomainDialog = () => {
    const items = splitPublicUrls(form.configs?.url);
    setDomainDrafts(items.length ? items : [""]);
    setDomainDialogOpen(true);
  };

  const applyDomainDialog = () => {
    const next = joinPublicUrls(domainDrafts);
    setForm((prev) => ({
      ...prev,
      configs: { ...prev.configs, url: next }
    }));
    setDomainDialogOpen(false);
  };

  useEffect(() => {
    if (isEditing && strategies) {
      const target = strategies.find((item) => item.id === Number(id));
      if (target) {
        const allowedExtensions =
          target.configs?.allowed_extensions ||
          target.configs?.allowed_exts ||
          target.configs?.extensions ||
          target.configs?.allowedExtensions ||
          "";
        const pathTemplate =
          target.configs?.path_template ||
          target.configs?.pattern ||
          "{year}/{month}/{day}/{uuid}";
        setForm({
          ...target,
          configs: {
            driver: target.configs?.driver || "local",
            root: target.configs?.root || "storage/uploads",
            url: normalizePublicUrls(
              target.configs?.url ||
                target.configs?.base_url ||
                target.configs?.baseUrl ||
                ""
            ),
            image_only_domain: Boolean(target.configs?.image_only_domain),
            webdav_endpoint:
              target.configs?.webdav_endpoint ||
              target.configs?.webdav_url ||
              target.configs?.webdavUrl ||
              "",
            webdav_username:
              target.configs?.webdav_username ||
              target.configs?.webdav_user ||
              target.configs?.webdavUsername ||
              "",
            webdav_password:
              target.configs?.webdav_password ||
              target.configs?.webdav_pass ||
              target.configs?.webdavPassword ||
              "",
            webdav_base_path:
              target.configs?.webdav_base_path ||
              target.configs?.webdav_path ||
              target.configs?.webdavBasePath ||
              "",
            webdav_skip_tls_verify:
              target.configs?.webdav_skip_tls_verify ||
              target.configs?.webdavSkipTLSVerify ||
              false,
            ftp_host:
              target.configs?.ftp_host ||
              target.configs?.ftp_endpoint ||
              "",
            ftp_port: target.configs?.ftp_port || 21,
            ftp_username:
              target.configs?.ftp_username ||
              target.configs?.ftp_user ||
              "",
            ftp_password:
              target.configs?.ftp_password ||
              target.configs?.ftp_pass ||
              "",
            ftp_base_path:
              target.configs?.ftp_base_path ||
              target.configs?.ftp_path ||
              "",
            ftp_tls: Boolean(target.configs?.ftp_tls || false),
            ftp_skip_tls_verify: Boolean(
              target.configs?.ftp_skip_tls_verify || target.configs?.ftpSkipTLSVerify || false
            ),
            ftp_timeout: target.configs?.ftp_timeout || 15,
            sftp_host:
              target.configs?.sftp_host ||
              target.configs?.sftp_endpoint ||
              "",
            sftp_port: target.configs?.sftp_port || 22,
            sftp_username:
              target.configs?.sftp_username ||
              target.configs?.sftp_user ||
              "",
            sftp_password:
              target.configs?.sftp_password ||
              target.configs?.sftp_pass ||
              "",
            sftp_private_key:
              target.configs?.sftp_private_key ||
              target.configs?.sftpPrivateKey ||
              "",
            sftp_base_path:
              target.configs?.sftp_base_path ||
              target.configs?.sftp_path ||
              "",
            sftp_timeout: target.configs?.sftp_timeout || 15,
            s3_endpoint: target.configs?.s3_endpoint || "",
            s3_region: target.configs?.s3_region || "",
            s3_bucket: target.configs?.s3_bucket || "",
            s3_access_key: target.configs?.s3_access_key || "",
            s3_secret_key: target.configs?.s3_secret_key || "",
            s3_session_token: target.configs?.s3_session_token || "",
            s3_force_path_style: Boolean(target.configs?.s3_force_path_style || false),
            proxy: Boolean(target.configs?.proxy || false),
            allowed_extensions: allowedExtensions,
            path_template: pathTemplate,
            enable_compression: target.configs?.enable_compression || false,
            compression_quality: target.configs?.compression_quality || 85,
            target_format: target.configs?.target_format || "",
            process_formats: target.configs?.process_formats || "",
            image_audit_profile_id:
              target.configs?.image_audit_profile_id
                ? String(target.configs?.image_audit_profile_id)
                : "",
            image_audit_block_action: target.configs?.image_audit_block_action || "delete",
            image_audit_error_action: target.configs?.image_audit_error_action || "keep",
            enable_thumbnail: Boolean(target.configs?.enable_thumbnail),
            thumbnail_max_size: target.configs?.thumbnail_max_size || 400,
            thumbnail_quality: target.configs?.thumbnail_quality || 25,
            thumbnail_format: target.configs?.thumbnail_format || "jpeg",
            thumbnail_strategy_id: target.configs?.thumbnail_strategy_id
              ? String(target.configs?.thumbnail_strategy_id)
              : ""
          }
        });
        setSelectedGroups(target.groups?.map((group) => group.id) || []);
      }
    } else if (!isEditing) {
      setForm({
        key: 1,
        name: "",
        intro: "",
        configs: {
          driver: "local",
          root: "storage/uploads",
          url: "",
          image_only_domain: false,
          webdav_endpoint: "",
          webdav_username: "",
          webdav_password: "",
          webdav_base_path: "",
          webdav_skip_tls_verify: false,
          ftp_host: "",
          ftp_port: 21,
          ftp_username: "",
          ftp_password: "",
          ftp_base_path: "",
          ftp_tls: false,
          ftp_skip_tls_verify: false,
          ftp_timeout: 15,
          sftp_host: "",
          sftp_port: 22,
          sftp_username: "",
          sftp_password: "",
          sftp_private_key: "",
          sftp_base_path: "",
          sftp_timeout: 15,
          s3_endpoint: "",
          s3_region: "",
          s3_bucket: "",
          s3_access_key: "",
          s3_secret_key: "",
          s3_session_token: "",
          s3_force_path_style: false,
          proxy: false,
          allowed_extensions: "",
          path_template: "{year}/{month}/{day}/{uuid}",
          image_audit_profile_id: "",
          image_audit_block_action: "delete",
          image_audit_error_action: "keep",
          enable_thumbnail: false,
          thumbnail_max_size: 400,
          thumbnail_quality: 25,
          thumbnail_format: "jpeg",
          thumbnail_strategy_id: ""
        }
      });
      setSelectedGroups([]);
    }
  }, [id, isEditing, strategies]);

  const saveMutation = useMutation({
    mutationFn: saveStrategy,
    onSuccess: () => {
      toast.success(t("admin.strategyEditor.saved"));
      queryClient.invalidateQueries({ queryKey: ["admin", "strategies"] });
      navigate("/dashboard/admin/strategies");
    },
    onError: (error) => toast.error(error.message)
  });

  const handleSave = () => {
    if (!form.name) return;
    const template = (form.configs as any)?.path_template || "{year}/{month}/{day}/{uuid}";
    if (template && !String(template).includes("{uuid}")) {
      toast.error(t("admin.strategyEditor.pathTemplateMustContainUuid"));
      return;
    }
    if (template && String(template).toLowerCase().includes("_thumb.")) {
      toast.error(t("admin.strategyEditor.pathTemplateCannotContainThumb"));
      return;
    }
    const lowerTemplate = String(template).toLowerCase();
    if (template && (lowerTemplate.includes("tickets/") || lowerTemplate.includes("_ticket."))) {
      toast.error(t("admin.strategyEditor.pathTemplateCannotContainTicket"));
      return;
    }
    const publicUrls = normalizePublicUrls(
      form.configs?.url || form.configs?.base_url || form.configs?.baseUrl || ""
    );
    if (Boolean((form.configs as any)?.image_only_domain) && !publicUrls) {
      toast.error(t("admin.strategyEditor.imageOnlyDomainRequiresUrl"));
      return;
    }
    saveMutation.mutate({
      ...form,
      groupIds: selectedGroups,
      configs: {
        ...form.configs,
        url: publicUrls,
        base_url: publicUrls,
        image_only_domain: Boolean((form.configs as any)?.image_only_domain),
        webdav_endpoint:
          form.configs?.webdav_endpoint ||
          form.configs?.webdav_url ||
          form.configs?.webdavUrl ||
          "",
        webdav_username:
          form.configs?.webdav_username ||
          form.configs?.webdav_user ||
          form.configs?.webdavUsername ||
          "",
        webdav_password:
          form.configs?.webdav_password ||
          form.configs?.webdav_pass ||
          form.configs?.webdavPassword ||
          "",
        webdav_base_path:
          form.configs?.webdav_base_path ||
          form.configs?.webdav_path ||
          form.configs?.webdavBasePath ||
          "",
        webdav_skip_tls_verify:
          Boolean(form.configs?.webdav_skip_tls_verify || form.configs?.webdavSkipTLSVerify),
        ftp_host: (form.configs as any)?.ftp_host || (form.configs as any)?.ftp_endpoint || "",
        ftp_port: (form.configs as any)?.ftp_port || 21,
        ftp_username: (form.configs as any)?.ftp_username || (form.configs as any)?.ftp_user || "",
        ftp_password: (form.configs as any)?.ftp_password || (form.configs as any)?.ftp_pass || "",
        ftp_base_path: (form.configs as any)?.ftp_base_path || (form.configs as any)?.ftp_path || "",
        ftp_tls: Boolean((form.configs as any)?.ftp_tls || false),
        ftp_skip_tls_verify: Boolean(
          (form.configs as any)?.ftp_skip_tls_verify || (form.configs as any)?.ftpSkipTLSVerify
        ),
        ftp_timeout: (form.configs as any)?.ftp_timeout || 15,
        sftp_host: (form.configs as any)?.sftp_host || (form.configs as any)?.sftp_endpoint || "",
        sftp_port: (form.configs as any)?.sftp_port || 22,
        sftp_username: (form.configs as any)?.sftp_username || (form.configs as any)?.sftp_user || "",
        sftp_password: (form.configs as any)?.sftp_password || (form.configs as any)?.sftp_pass || "",
        sftp_private_key: (form.configs as any)?.sftp_private_key || (form.configs as any)?.sftpPrivateKey || "",
        sftp_base_path: (form.configs as any)?.sftp_base_path || (form.configs as any)?.sftp_path || "",
        sftp_timeout: (form.configs as any)?.sftp_timeout || 15,
        s3_endpoint: (form.configs as any)?.s3_endpoint || "",
        s3_region: (form.configs as any)?.s3_region || "",
        s3_bucket: (form.configs as any)?.s3_bucket || "",
        s3_access_key: (form.configs as any)?.s3_access_key || "",
        s3_secret_key: (form.configs as any)?.s3_secret_key || "",
        s3_session_token: (form.configs as any)?.s3_session_token || "",
        s3_force_path_style: Boolean((form.configs as any)?.s3_force_path_style || false),
        proxy: Boolean((form.configs as any)?.proxy || false),
        allowed_extensions: form.configs?.allowed_extensions || "",
        path_template: form.configs?.path_template || "{year}/{month}/{day}/{uuid}",
        pattern: form.configs?.path_template || "{year}/{month}/{day}/{uuid}",
        enable_compression: (form.configs as any)?.enable_compression || false,
        compression_quality: (form.configs as any)?.compression_quality || 85,
        target_format: (form.configs as any)?.target_format || "",
        process_formats: (form.configs as any)?.process_formats || "",
        enable_thumbnail: Boolean((form.configs as any)?.enable_thumbnail),
        thumbnail_max_size: Number((form.configs as any)?.thumbnail_max_size) || 400,
        thumbnail_quality: Number((form.configs as any)?.thumbnail_quality) || 25,
        thumbnail_format: (form.configs as any)?.thumbnail_format || "jpeg",
        thumbnail_strategy_id: (form.configs as any)?.thumbnail_strategy_id
          ? Number((form.configs as any)?.thumbnail_strategy_id)
          : null,
        image_audit_profile_id: (form.configs as any)?.image_audit_profile_id
          ? Number((form.configs as any)?.image_audit_profile_id)
          : null,
        image_audit_block_action: (form.configs as any)?.image_audit_block_action || "delete",
        image_audit_error_action: (form.configs as any)?.image_audit_error_action || "keep"
      }
    } as StrategyRecord);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2">
        <p className="text-sm text-muted-foreground">
          <Link className="text-primary" to="/dashboard/admin/strategies">
            {t("admin.strategies.title")}
          </Link>{" "}
          / {isEditing ? t("admin.strategyEditor.edit") : t("admin.strategyEditor.new")}
        </p>
        <h1 className="text-2xl font-semibold">{isEditing ? form.name : t("admin.strategyEditor.newTitle")}</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.strategyEditor.config")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("admin.strategyEditor.name")}</Label>
              <Input
                value={form.name || ""}
                onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("admin.strategyEditor.driver")}</Label>
              <Select
                value={form.configs?.driver || "local"}
                onValueChange={(value) =>
                  setForm((prev) => ({ ...prev, configs: { ...prev.configs, driver: value } }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.strategyEditor.driverPlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  {driverOptions.map((option) => (
                    <SelectItem key={option.key} value={option.key}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-2">
            <Label>{t("admin.strategyEditor.intro")}</Label>
            <Input
              value={form.intro || ""}
              onChange={(e) => setForm((prev) => ({ ...prev, intro: e.target.value }))}
            />
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            {form.configs?.driver === "webdav" ? (
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.webdavEndpoint")}</Label>
                <Input
                  value={(form.configs as any)?.webdav_endpoint || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, webdav_endpoint: e.target.value }
                    }))
                  }
                  placeholder="https://dav.example.com/remote.php/dav/files/user"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.strategyEditor.webdavEndpointHint")}
                </p>
              </div>
            ) : form.configs?.driver === "ftp" ? (
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.ftpHost")}</Label>
                <Input
                  value={(form.configs as any)?.ftp_host || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, ftp_host: e.target.value }
                    }))
                  }
                  placeholder="ftp.example.com:21 或 ftp://user:pass@host:21"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.strategyEditor.ftpHostHint")}
                </p>
              </div>
            ) : form.configs?.driver === "sftp" ? (
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.sftpHost")}</Label>
                <Input
                  value={(form.configs as any)?.sftp_host || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, sftp_host: e.target.value }
                    }))
                  }
                  placeholder="sftp.example.com:22 或 ssh://user@host:22"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.strategyEditor.sftpHostHint")}
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                <Label>{isS3CompatibleDriver(form.configs?.driver) ? t("admin.strategyEditor.objectPrefix") : t("admin.strategyEditor.storageRoot")}</Label>
                <Input
                  value={form.configs?.root || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, root: e.target.value }
                    }))
                  }
                  placeholder={isS3CompatibleDriver(form.configs?.driver) ? "uploads" : ""}
                />
                <p className="text-xs text-muted-foreground">
                  {isS3CompatibleDriver(form.configs?.driver)
                    ? t("admin.strategyEditor.objectPrefixHint")
                    : t("admin.strategyEditor.storageRootHint")}
                </p>
              </div>
            )}
            <div className="space-y-2">
              <Label>{t("admin.strategyEditor.publicUrl")}</Label>
              <div className="flex gap-2">
                <Input
                  value={form.configs?.url || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, url: e.target.value }
                    }))
                  }
                  placeholder="https://cdn.example.com;https://img.example.com"
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="h-9 w-9 shrink-0"
                  onClick={openDomainDialog}
                  aria-label={t("admin.strategyEditor.publicUrlManage")}
                  title={t("admin.strategyEditor.publicUrlManage")}
                >
                  <ListPlus className="h-4 w-4" />
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                {t("admin.strategyEditor.publicUrlHint")}
              </p>
              <div className="flex items-center gap-2 pt-1">
                <Checkbox
                  id="image-only-domain"
                  checked={Boolean((form.configs as any)?.image_only_domain)}
                  onCheckedChange={(checked) => {
                    const actualValue = checked === "indeterminate" ? false : checked;
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, image_only_domain: actualValue }
                    }));
                  }}
                />
                <Label htmlFor="image-only-domain" className="cursor-pointer font-normal">
                  {t("admin.strategyEditor.imageOnlyDomain")}
                </Label>
              </div>
              <p className="text-xs text-muted-foreground">
                {t("admin.strategyEditor.imageOnlyDomainHint")}
              </p>
            </div>
          </div>
          {isS3CompatibleDriver(form.configs?.driver) && (
            <div className="space-y-4 rounded-lg border p-4">
              <h3 className="text-sm font-medium">{t("admin.strategyEditor.s3Config")}</h3>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>{`S3 Endpoint${t("admin.strategyEditor.optional")}`}</Label>
                  <Input
                    value={(form.configs as any)?.s3_endpoint || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, s3_endpoint: e.target.value }
                      }))
                    }
                    placeholder="https://s3.amazonaws.com"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Region</Label>
                  <Input
                    value={(form.configs as any)?.s3_region || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, s3_region: e.target.value }
                      }))
                    }
                    placeholder="us-east-1"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Bucket</Label>
                  <Input
                    value={(form.configs as any)?.s3_bucket || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, s3_bucket: e.target.value }
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>Access Key</Label>
                  <Input
                    value={(form.configs as any)?.s3_access_key || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, s3_access_key: e.target.value }
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>Secret Key</Label>
                  <Input
                    type="password"
                    value={(form.configs as any)?.s3_secret_key || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, s3_secret_key: e.target.value }
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>{`Session Token${t("admin.strategyEditor.optional")}`}</Label>
                  <Input
                    value={(form.configs as any)?.s3_session_token || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, s3_session_token: e.target.value }
                      }))
                    }
                  />
                </div>
              </div>
              <div className="flex flex-col gap-3">
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="s3-force-path-style"
                    checked={Boolean((form.configs as any)?.s3_force_path_style)}
                    onCheckedChange={(checked) => {
                      const actualValue = checked === "indeterminate" ? false : checked;
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, s3_force_path_style: actualValue }
                      }));
                    }}
                  />
                  <Label htmlFor="s3-force-path-style" className="cursor-pointer">
                    {t("admin.strategyEditor.s3ForcePathStyle")}
                  </Label>
                </div>
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="s3-proxy"
                    checked={Boolean((form.configs as any)?.proxy)}
                    onCheckedChange={(checked) => {
                      const actualValue = checked === "indeterminate" ? false : checked;
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, proxy: actualValue }
                      }));
                    }}
                  />
                  <Label htmlFor="s3-proxy" className="cursor-pointer">
                    {t("admin.strategyEditor.s3Proxy")}
                  </Label>
                </div>
                <p className="text-xs text-muted-foreground">
                  {t("admin.strategyEditor.s3ProxyHint")}
                </p>
              </div>
            </div>
          )}
          {form.configs?.driver === "ftp" && (
            <div className="space-y-4 rounded-lg border p-4">
              <h3 className="text-sm font-medium">{t("admin.strategyEditor.ftpConfig")}</h3>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>{t("admin.strategyEditor.ftpPort")}</Label>
                  <Input
                    type="number"
                    min="1"
                    value={(form.configs as any)?.ftp_port || 21}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, ftp_port: parseInt(e.target.value, 10) || 21 }
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("admin.strategyEditor.basePath")}</Label>
                  <Input
                    value={(form.configs as any)?.ftp_base_path || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, ftp_base_path: e.target.value }
                      }))
                    }
                    placeholder="skyimage"
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("admin.strategyEditor.username")}</Label>
                  <Input
                    value={(form.configs as any)?.ftp_username || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, ftp_username: e.target.value }
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("admin.strategyEditor.password")}</Label>
                  <Input
                    type="password"
                    value={(form.configs as any)?.ftp_password || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, ftp_password: e.target.value }
                      }))
                    }
                  />
                </div>
              </div>
              <div className="flex flex-col gap-3">
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="ftp-tls"
                    checked={Boolean((form.configs as any)?.ftp_tls)}
                    onCheckedChange={(checked) => {
                      const actualValue = checked === "indeterminate" ? false : checked;
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, ftp_tls: actualValue }
                      }));
                    }}
                  />
                  <Label htmlFor="ftp-tls" className="cursor-pointer">
                    {t("admin.strategyEditor.enableTls")}
                  </Label>
                </div>
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="ftp-skip-tls-verify"
                    checked={Boolean((form.configs as any)?.ftp_skip_tls_verify)}
                    onCheckedChange={(checked) => {
                      const actualValue = checked === "indeterminate" ? false : checked;
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, ftp_skip_tls_verify: actualValue }
                      }));
                    }}
                  />
                  <Label htmlFor="ftp-skip-tls-verify" className="cursor-pointer">
                    {t("admin.strategyEditor.skipTlsVerify")}
                  </Label>
                </div>
              </div>
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.timeout")}</Label>
                <Input
                  type="number"
                  min="1"
                  value={(form.configs as any)?.ftp_timeout || 15}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, ftp_timeout: parseInt(e.target.value, 10) || 15 }
                    }))
                  }
                />
              </div>
            </div>
          )}
          {form.configs?.driver === "sftp" && (
            <div className="space-y-4 rounded-lg border p-4">
              <h3 className="text-sm font-medium">{t("admin.strategyEditor.sftpConfig")}</h3>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>{t("admin.strategyEditor.sftpPort")}</Label>
                  <Input
                    type="number"
                    min="1"
                    value={(form.configs as any)?.sftp_port || 22}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, sftp_port: parseInt(e.target.value, 10) || 22 }
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("admin.strategyEditor.basePath")}</Label>
                  <Input
                    value={(form.configs as any)?.sftp_base_path || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, sftp_base_path: e.target.value }
                      }))
                    }
                    placeholder="skyimage"
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("admin.strategyEditor.sftpUsername")}</Label>
                  <Input
                    value={(form.configs as any)?.sftp_username || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, sftp_username: e.target.value }
                      }))
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("admin.strategyEditor.sftpPassword")}</Label>
                  <Input
                    type="password"
                    value={(form.configs as any)?.sftp_password || ""}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        configs: { ...prev.configs, sftp_password: e.target.value }
                      }))
                    }
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.sftpPrivateKey")}</Label>
                <textarea
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  value={(form.configs as any)?.sftp_private_key || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, sftp_private_key: e.target.value }
                    }))
                  }
                  placeholder={t("admin.strategyEditor.sftpPrivateKeyHint")}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.timeout")}</Label>
                <Input
                  type="number"
                  min="1"
                  value={(form.configs as any)?.sftp_timeout || 15}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, sftp_timeout: parseInt(e.target.value, 10) || 15 }
                    }))
                  }
                />
              </div>
            </div>
          )}
          {form.configs?.driver === "webdav" && (
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.webdavUsername")}</Label>
                <Input
                  value={(form.configs as any)?.webdav_username || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, webdav_username: e.target.value }
                    }))
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.webdavPassword")}</Label>
                <Input
                  type="password"
                  value={(form.configs as any)?.webdav_password || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, webdav_password: e.target.value }
                    }))
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.webdavBasePath")}</Label>
                <Input
                  value={(form.configs as any)?.webdav_base_path || ""}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, webdav_base_path: e.target.value }
                    }))
                  }
                  placeholder="skyimage"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.strategyEditor.webdavBasePathHint")}
                </p>
              </div>
              <div className="flex items-center gap-2 pt-8">
                <Checkbox
                  id="webdav-skip-tls-verify"
                  checked={Boolean(
                    (form.configs as any)?.webdav_skip_tls_verify ||
                      (form.configs as any)?.webdavSkipTLSVerify
                  )}
                  onCheckedChange={(checked) => {
                    const actualValue = checked === "indeterminate" ? false : checked;
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, webdav_skip_tls_verify: actualValue }
                    }));
                  }}
                />
                <Label htmlFor="webdav-skip-tls-verify" className="cursor-pointer">
                  {t("admin.strategyEditor.skipTlsVerify")}
                </Label>
              </div>
            </div>
          )}
          <div className="space-y-2">
            <Label>{t("admin.strategyEditor.allowedExtensions")}</Label>
            <Input
              value={(form.configs as any)?.allowed_extensions || ""}
              onChange={(e) =>
                setForm((prev) => ({
                  ...prev,
                  configs: { ...prev.configs, allowed_extensions: e.target.value }
                }))
              }
              placeholder="jpg,png,webp,mp4"
            />
            <p className="text-xs text-muted-foreground">{t("admin.strategyEditor.allowedExtensionsHint")}</p>
          </div>
          <div className="space-y-4 rounded-lg border p-4">
            <h3 className="text-sm font-medium">{t("admin.strategyEditor.imageProcessing")}</h3>
            <div className="flex items-center gap-2">
              <Checkbox
                id="enable-compression"
                checked={Boolean((form.configs as any)?.enable_compression)}
                onCheckedChange={(checked) => {
                  const actualValue = checked === "indeterminate" ? false : checked;
                  setForm((prev) => ({
                    ...prev,
                    configs: { ...prev.configs, enable_compression: actualValue }
                  }));
                }}
              />
              <Label htmlFor="enable-compression" className="cursor-pointer">
                {t("admin.strategyEditor.enableCompression")}
              </Label>
            </div>
            {(form.configs as any)?.enable_compression && (
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.compressionQuality")}</Label>
                <Input
                  type="number"
                  min="1"
                  max="100"
                  value={(form.configs as any)?.compression_quality || 85}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, compression_quality: parseInt(e.target.value) || 85 }
                    }))
                  }
                  placeholder="85"
                />
                <p className="text-xs text-muted-foreground">{t("admin.strategyEditor.compressionQualityHint")}</p>
              </div>
            )}
            <div className="space-y-2">
              <Label>{t("admin.strategyEditor.targetFormat")}</Label>
              <Select
                value={(form.configs as any)?.target_format || ""}
                onValueChange={(value) =>
                  setForm((prev) => ({
                    ...prev,
                    configs: { ...prev.configs, target_format: value === "none" ? "" : value }
                  }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.strategyEditor.noConversion")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t("admin.strategyEditor.noConversion")}</SelectItem>
                  <SelectItem value="webp">WebP</SelectItem>
                  <SelectItem value="jpeg">JPEG</SelectItem>
                  <SelectItem value="png">PNG</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {t("admin.strategyEditor.targetFormatHint")}
              </p>
            </div>
            <div className="space-y-2">
              <Label>{t("admin.strategyEditor.processFormats")}</Label>
              <Input
                value={(form.configs as any)?.process_formats || ""}
                onChange={(e) =>
                  setForm((prev) => ({
                    ...prev,
                    configs: { ...prev.configs, process_formats: e.target.value }
                  }))
                }
                placeholder="jpg,jpeg,png,webp"
              />
              <p className="text-xs text-muted-foreground">
                {t("admin.strategyEditor.processFormatsHint")}
              </p>
            </div>
          </div>
          <div className="space-y-4 rounded-lg border p-4">
            <h3 className="text-sm font-medium">{t("admin.strategyEditor.thumbnail")}</h3>
            <div className="flex items-center gap-2">
              <Checkbox
                id="enable-thumbnail"
                checked={Boolean((form.configs as any)?.enable_thumbnail)}
                onCheckedChange={(checked) => {
                  const actualValue = checked === "indeterminate" ? false : checked;
                  setForm((prev) => ({
                    ...prev,
                    configs: { ...prev.configs, enable_thumbnail: actualValue }
                  }));
                }}
              />
              <Label htmlFor="enable-thumbnail" className="cursor-pointer">
                {t("admin.strategyEditor.enableThumbnail")}
              </Label>
            </div>
            <p className="text-xs text-muted-foreground">
              {t("admin.strategyEditor.enableThumbnailHint")}
            </p>
            {(form.configs as any)?.enable_thumbnail && (
              <>
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label>{t("admin.strategyEditor.thumbnailMaxSize")}</Label>
                    <Input
                      type="number"
                      min="64"
                      max="1200"
                      value={(form.configs as any)?.thumbnail_max_size || 400}
                      onChange={(e) =>
                        setForm((prev) => ({
                          ...prev,
                          configs: {
                            ...prev.configs,
                            thumbnail_max_size: parseInt(e.target.value) || 400
                          }
                        }))
                      }
                      placeholder="400"
                    />
                    <p className="text-xs text-muted-foreground">
                      {t("admin.strategyEditor.thumbnailMaxSizeHint")}
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label>{t("admin.strategyEditor.thumbnailQuality")}</Label>
                    <Input
                      type="number"
                      min="1"
                      max="100"
                      value={(form.configs as any)?.thumbnail_quality || 25}
                      onChange={(e) =>
                        setForm((prev) => ({
                          ...prev,
                          configs: {
                            ...prev.configs,
                            thumbnail_quality: parseInt(e.target.value) || 25
                          }
                        }))
                      }
                      placeholder="25"
                    />
                    <p className="text-xs text-muted-foreground">
                      {t("admin.strategyEditor.thumbnailQualityHint")}
                    </p>
                  </div>
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label>{t("admin.strategyEditor.thumbnailFormat")}</Label>
                    <Select
                      value={(form.configs as any)?.thumbnail_format || "jpeg"}
                      onValueChange={(value) =>
                        setForm((prev) => ({
                          ...prev,
                          configs: { ...prev.configs, thumbnail_format: value }
                        }))
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="jpeg">JPEG</SelectItem>
                        <SelectItem value="webp">WebP</SelectItem>
                        <SelectItem value="png">PNG</SelectItem>
                      </SelectContent>
                    </Select>
                    <p className="text-xs text-muted-foreground">
                      {t("admin.strategyEditor.thumbnailFormatHint")}
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label>{t("admin.strategyEditor.thumbnailStrategy")}</Label>
                    <Select
                      value={String((form.configs as any)?.thumbnail_strategy_id || "same")}
                      onValueChange={(value) =>
                        setForm((prev) => ({
                          ...prev,
                          configs: {
                            ...prev.configs,
                            thumbnail_strategy_id: value === "same" ? "" : value
                          }
                        }))
                      }
                    >
                      <SelectTrigger>
                        <SelectValue placeholder={t("admin.strategyEditor.thumbnailStrategySame")} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="same">
                          {t("admin.strategyEditor.thumbnailStrategySame")}
                        </SelectItem>
                        {(strategies || [])
                          .filter((item) => !isEditing || item.id !== Number(id))
                          .map((item) => (
                            <SelectItem key={item.id} value={String(item.id)}>
                              {item.name}
                            </SelectItem>
                          ))}
                        {isEditing && form.id ? (
                          <SelectItem value={String(form.id)}>
                            {form.name || t("admin.strategyEditor.thumbnailStrategySame")}
                          </SelectItem>
                        ) : null}
                      </SelectContent>
                    </Select>
                    <p className="text-xs text-muted-foreground">
                      {t("admin.strategyEditor.thumbnailStrategyHint")}
                    </p>
                  </div>
                </div>
              </>
            )}
          </div>
          <div className="space-y-4 rounded-lg border p-4">
            <h3 className="text-sm font-medium">{t("admin.strategyEditor.imageAudit")}</h3>
            <div className="space-y-2">
              <Label>{t("admin.strategyEditor.imageAuditProfile")}</Label>
              <Select
                value={String((form.configs as any)?.image_audit_profile_id || "none")}
                onValueChange={(value) =>
                  setForm((prev) => ({
                    ...prev,
                    configs: {
                      ...prev.configs,
                      image_audit_profile_id: value === "none" ? "" : value
                    }
                  }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.strategyEditor.imageAuditProfilePlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t("admin.strategyEditor.imageAuditDisabled")}</SelectItem>
                  {auditProfiles?.map((profile: AuditProfileRecord) => (
                    <SelectItem key={profile.id} value={profile.id.toString()}>
                      {profile.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.imageAuditBlockAction")}</Label>
                <Select
                  value={(form.configs as any)?.image_audit_block_action || "delete"}
                  onValueChange={(value) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, image_audit_block_action: value }
                    }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="delete">{t("admin.strategyEditor.auditActionDelete")}</SelectItem>
                    <SelectItem value="keep">{t("admin.strategyEditor.auditActionKeep")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>{t("admin.strategyEditor.imageAuditErrorAction")}</Label>
                <Select
                  value={(form.configs as any)?.image_audit_error_action || "keep"}
                  onValueChange={(value) =>
                    setForm((prev) => ({
                      ...prev,
                      configs: { ...prev.configs, image_audit_error_action: value }
                    }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="delete">{t("admin.strategyEditor.auditActionDelete")}</SelectItem>
                    <SelectItem value="keep">{t("admin.strategyEditor.auditActionKeep")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
          <div className="space-y-2">
            <Label>{t("admin.strategyEditor.pathTemplate")}</Label>
            <Input
              value={(form.configs as any)?.path_template || ""}
              onChange={(e) =>
                setForm((prev) => ({
                  ...prev,
                  configs: { ...prev.configs, path_template: e.target.value }
                }))
              }
              placeholder="{year}/{month}/{day}/{uuid}"
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.strategyEditor.pathTemplateHint")}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t("admin.strategyEditor.authorizedGroups")}</Label>
            <div className="space-y-3">
              {groups?.map((group) => (
                <div
                  key={group.id}
                  className="flex items-center justify-between rounded-md border p-3"
                >
                  <div>
                    <p className="text-sm font-medium">
                      {group.name}
                      {group.isDefault && (
                        <span className="ml-2 text-xs text-muted-foreground">· {t("admin.strategyEditor.defaultGroup")}</span>
                      )}
                    </p>
                  </div>
                  <Checkbox
                    id={`group-${group.id}`}
                    checked={selectedGroups.includes(group.id)}
                    onCheckedChange={(checked) => {
                      const actualValue = checked === 'indeterminate' ? false : checked;
                      if (actualValue) {
                        setSelectedGroups((prev) => [...prev, group.id]);
                      } else {
                        setSelectedGroups((prev) => prev.filter((id) => id !== group.id));
                      }
                    }}
                  />
                </div>
              ))}
              {!groups?.length && (
                <p className="text-sm text-muted-foreground">{t("admin.strategyEditor.noGroups")}</p>
              )}
            </div>
          </div>
          <div className="flex gap-3">
            <Button onClick={handleSave} disabled={!form.name || saveMutation.isPending}>
              {saveMutation.isPending ? t("common.saving") : t("admin.strategyEditor.save")}
            </Button>
            <Button
              variant="ghost"
              onClick={() => navigate("/dashboard/admin/strategies")}
              disabled={saveMutation.isPending}
            >
              {t("common.cancel")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog open={domainDialogOpen} onOpenChange={setDomainDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("admin.strategyEditor.publicUrlManage")}</DialogTitle>
            <DialogDescription>{t("admin.strategyEditor.publicUrlManageHint")}</DialogDescription>
          </DialogHeader>
          <div className="max-h-[50vh] space-y-2 overflow-y-auto py-1">
            {domainDrafts.map((domain, index) => (
              <div key={index} className="flex gap-2">
                <Input
                  value={domain}
                  onChange={(e) =>
                    setDomainDrafts((prev) => prev.map((item, i) => (i === index ? e.target.value : item)))
                  }
                  placeholder="https://cdn.example.com"
                  className="flex-1 border-0 bg-muted/50 shadow-none focus-visible:ring-1 focus-visible:ring-ring"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9 shrink-0 text-muted-foreground hover:text-destructive"
                  disabled={domainDrafts.length <= 1}
                  onClick={() => setDomainDrafts((prev) => prev.filter((_, i) => i !== index))}
                  aria-label={t("admin.delete")}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}
          </div>
          <Button
            type="button"
            variant="outline"
            className="w-full gap-1.5"
            onClick={() => setDomainDrafts((prev) => [...prev, ""])}
          >
            <Plus className="h-4 w-4" />
            {t("admin.strategyEditor.publicUrlAdd")}
          </Button>
          <div className="space-y-2">
            <Label>{t("admin.strategyEditor.publicUrlPrimary")}</Label>
            <Input
              readOnly
              value={domainDrafts.map((item) => item.trim()).find(Boolean) || ""}
              placeholder={t("admin.strategyEditor.publicUrlPrimaryEmpty")}
              className="border-0 bg-muted/50 shadow-none focus-visible:ring-0"
            />
            <p className="text-xs text-muted-foreground">{t("admin.strategyEditor.publicUrlPrimaryHint")}</p>
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setDomainDialogOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button type="button" onClick={applyDomainDialog}>
              {t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
