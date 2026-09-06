import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
  fetchInstallerStatus,
  runInstaller,
  testLegacySource,
  importLegacySource,
  type LegacyImportSummary,
  type LegacyProbeResult
} from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { useAuthStore } from "@/state/auth";
import { useI18n } from "@/i18n";
import { cn } from "@/lib/utils";
import { Database, HardDriveDownload, TriangleAlert } from "lucide-react";

type WizardStep = "mode" | "product" | "legacy" | "database" | "site" | "result";
type InstallMode = "fresh" | "import";
type LegacySourceDbType = "mysql" | "sqlite" | "postgres" | "sqlserver";

function legacyDefaultPort(dbType: string): string {
  switch (dbType) {
    case "postgres":
      return "5432";
    case "sqlserver":
      return "1433";
    case "sqlite":
      return "";
    default:
      return "3306";
  }
}

function BackupWarning({ text }: { text: string }) {
  return (
    <div className="flex gap-3 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-600 dark:text-amber-400">
      <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0" />
      <p>{text}</p>
    </div>
  );
}

export function InstallerPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const clearAuth = useAuthStore((state) => state.clear);
  const [step, setStep] = useState<WizardStep>("mode");
  const [mode, setMode] = useState<InstallMode>("fresh");
  const [product, setProduct] = useState<"lsky" | null>(null);
  const [productAck, setProductAck] = useState(false);
  const [installedLocally, setInstalledLocally] = useState(false);
  // onError 的闭包可能捕获到旧的状态值，安装是否成功用 ref 同步跟踪。
  const installedRef = useRef(false);

  const { data, isLoading } = useQuery({
    queryKey: ["installer"],
    queryFn: fetchInstallerStatus
  });

  const [form, setForm] = useState({
    databaseType: "sqlite",
    databasePath: "storage/data/skyimage.db",
    databaseHost: "localhost",
    databasePort: "3306",
    databaseName: "skyimage",
    databaseUser: "root",
    databasePassword: "",
    siteName: "skyImage",
    adminName: "Administrator",
    adminEmail: "",
    adminPassword: ""
  });

  const [legacy, setLegacy] = useState({
    databaseType: "mysql" as LegacySourceDbType,
    host: "localhost",
    port: "3306",
    database: "lsky",
    username: "root",
    password: "",
    tablePrefix: "",
    sqliteDir: "",
    imageDir: ""
  });
  const [probe, setProbe] = useState<LegacyProbeResult | null>(null);
  const [summary, setSummary] = useState<LegacyImportSummary | null>(null);
  const [importError, setImportError] = useState<string | null>(null);

  const legacyPayload = () => ({
    databaseType: legacy.databaseType,
    host: legacy.host,
    port: legacy.port,
    database: legacy.database,
    username: legacy.username,
    password: legacy.password,
    tablePrefix: legacy.tablePrefix,
    sqliteDir: legacy.sqliteDir,
    imageDir: legacy.imageDir,
    adminEmail: form.adminEmail,
    adminPassword: form.adminPassword
  });

  const installMutation = useMutation({
    mutationFn: async () => {
      // 导入模式：安装前先做源库连通性预检，避免"源库连不上也安装成功"。
      if (mode === "import") {
        const probe = await testLegacySource(legacyPayload());
        if (probe.missingTables?.length) {
          throw new Error(t("installer.legacy.notLsky"));
        }
      }
      await runInstaller(form);
      // 先于导入标记安装成功：导入失败时 onError 据此进入结果页提供重试，
      // 而不是让用户重跑一个已完成的安装。
      installedRef.current = true;
      setInstalledLocally(true);
      if (mode === "import") {
        const result = await importLegacySource(legacyPayload());
        return result;
      }
      return null;
    },
    onSuccess: (result) => {
      clearAuth();
      toast.success(t("installer.complete"));
      queryClient.invalidateQueries({ queryKey: ["installer"] });
      setInstalledLocally(true);
      setSummary(result);
      setImportError(null);
      setStep("result");
    },
    onError: (error) => {
      toast.error(error.message);
      // 安装成功但导入失败时也会走到这里：保留结果页以便重试
      if (installedRef.current) {
        setImportError(error.message);
        setStep("result");
      }
    }
  });

  // 导入失败后的重试：只重跑导入
  const retryMutation = useMutation({
    mutationFn: () => importLegacySource(legacyPayload()),
    onSuccess: (result) => {
      setSummary(result);
      setImportError(null);
      toast.success(t("installer.legacy.importOk"));
    },
    onError: (error) => setImportError(error.message)
  });

  const probeMutation = useMutation({
    mutationFn: () => testLegacySource(legacyPayload()),
    onSuccess: (result) => {
      setProbe(result);
      if (result.missingTables?.length) {
        toast.error(t("installer.legacy.notLsky"));
      } else {
        toast.success(t("installer.legacy.testOk"));
      }
    },
    onError: (error) => {
      setProbe(null);
      toast.error(error.message);
    }
  });

  if (isLoading) {
    return (
      <div className="mx-auto max-w-2xl space-y-4 p-6">
        <div className="h-8 w-48 animate-pulse rounded-md bg-muted" />
        <div className="h-4 w-72 max-w-full animate-pulse rounded-md bg-muted" />
        <div className="h-64 w-full animate-pulse rounded-md bg-muted" />
      </div>
    );
  }

  if (data?.installed && !installedLocally) {
    return (
      <Card className="max-w-xl mx-auto mt-20">
        <CardHeader>
          <CardTitle>{t("installer.installed")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p>{t("installer.version", { version: data.version ?? "" })}</p>
          <Button onClick={() => (window.location.href = "/login")}>
            {t("installer.goLogin")}
          </Button>
        </CardContent>
      </Card>
    );
  }

  const stepTitle = {
    mode: t("installer.mode.step"),
    product: t("installer.product.step"),
    legacy: t("installer.legacy.step"),
    database: t("installer.step1"),
    site: t("installer.step2"),
    result: t("installer.result.step")
  }[step];

  const goNextFromLegacy = () => {
    if (legacy.databaseType === "sqlite") {
      if (!legacy.sqliteDir) {
        toast.error(t("installer.legacy.needSqliteDir"));
        return;
      }
    } else if (!legacy.host || !legacy.port || !legacy.database || !legacy.username) {
      toast.error(t("installer.legacy.incompleteConn"));
      return;
    }
    setProbe(null);
    setStep("database");
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6 py-10">
      <div>
        <h1 className="text-2xl font-semibold">{t("installer.title")}</h1>
        <p className="text-muted-foreground">{stepTitle}</p>
      </div>

      {step === "mode" && (
        <div className="space-y-4">
          <BackupWarning text={t("installer.backupWarning")} />
          <button
            type="button"
            className={cn(
              "w-full rounded-lg border p-4 text-left transition-colors hover:border-primary/60",
              mode === "fresh" ? "border-primary bg-primary/5" : "border-border"
            )}
            onClick={() => setMode("fresh")}
          >
            <div className="flex items-center gap-3">
              <Database className="h-5 w-5" />
              <span className="font-medium">{t("installer.mode.fresh")}</span>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("installer.mode.freshDesc")}
            </p>
          </button>
          <button
            type="button"
            className={cn(
              "w-full rounded-lg border p-4 text-left transition-colors hover:border-primary/60",
              mode === "import" ? "border-primary bg-primary/5" : "border-border"
            )}
            onClick={() => setMode("import")}
          >
            <div className="flex items-center gap-3">
              <HardDriveDownload className="h-5 w-5" />
              <span className="font-medium">{t("installer.mode.import")}</span>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("installer.mode.importDesc")}
            </p>
          </button>
          <Button className="w-full" onClick={() => setStep(mode === "import" ? "product" : "database")}>
            {t("installer.next")}
          </Button>
        </div>
      )}

      {step === "product" && (
        <div className="space-y-4">
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive space-y-2">
            <p className="flex items-center gap-2 font-semibold">
              <TriangleAlert className="h-4 w-4 shrink-0" />
              {t("installer.product.warning.title")}
            </p>
            <p className="leading-relaxed">{t("installer.product.warning.body")}</p>
            <label className="flex items-start gap-2 pt-1 text-foreground cursor-pointer">
              <Checkbox
                checked={productAck}
                onCheckedChange={(v) => setProductAck(v === true)}
                className="mt-0.5"
              />
              <span>{t("installer.product.warning.ack")}</span>
            </label>
          </div>
          <button
            type="button"
            className={cn(
              "w-full rounded-lg border p-4 text-left transition-colors hover:border-primary/60",
              product === "lsky" ? "border-primary bg-primary/5" : "border-border"
            )}
            onClick={() => setProduct("lsky")}
          >
            <div className="flex items-center gap-3">
              <HardDriveDownload className="h-5 w-5" />
              <span className="font-medium">{t("installer.product.lsky")}</span>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("installer.product.lskyDesc")}
            </p>
          </button>
          <p className="text-sm text-muted-foreground">{t("installer.product.more")}</p>
          <div className="grid grid-cols-2 gap-3">
            <Button variant="outline" onClick={() => setStep("mode")}>
              {t("installer.previous")}
            </Button>
            <Button
              onClick={() => setStep("legacy")}
              disabled={!product || !productAck}
            >
              {t("installer.next")}
            </Button>
          </div>
        </div>
      )}

      {step === "legacy" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("installer.legacy.title")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <BackupWarning text={t("installer.backupWarning")} />

            <div className="space-y-2">
              <Label htmlFor="legacyType">{t("installer.legacy.sourceType")}</Label>
              <Select
                value={legacy.databaseType}
                onValueChange={(value) =>
                  setLegacy((prev) => ({
                    ...prev,
                    databaseType: value as LegacySourceDbType,
                    port: legacyDefaultPort(value)
                  }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("installer.legacy.sourceType")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="mysql">{t("installer.legacy.typeMysql")}</SelectItem>
                  <SelectItem value="postgres">{t("installer.legacy.typePostgres")}</SelectItem>
                  <SelectItem value="sqlserver">{t("installer.legacy.typeSqlserver")}</SelectItem>
                  <SelectItem value="sqlite">{t("installer.legacy.typeSqlite")}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {legacy.databaseType === "sqlite" ? (
              <div className="space-y-2">
                <Label htmlFor="legacySqliteDir">{t("installer.legacy.sqliteDir")}</Label>
                <Input
                  id="legacySqliteDir"
                  value={legacy.sqliteDir}
                  onChange={(e) =>
                    setLegacy((prev) => ({ ...prev, sqliteDir: e.target.value }))
                  }
                  placeholder="database"
                />
                <p className="text-sm text-muted-foreground">
                  {t("installer.legacy.sqliteDirHint")}
                </p>
              </div>
            ) : (
              <>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label htmlFor="legacyHost">{t("installer.databaseHost")}</Label>
                    <Input
                      id="legacyHost"
                      value={legacy.host}
                      onChange={(e) =>
                        setLegacy((prev) => ({ ...prev, host: e.target.value }))
                      }
                      placeholder="localhost"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="legacyPort">{t("installer.port")}</Label>
                    <Input
                      id="legacyPort"
                      value={legacy.port}
                      onChange={(e) =>
                        setLegacy((prev) => ({ ...prev, port: e.target.value }))
                      }
                      placeholder="3306"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="legacyDatabase">{t("installer.legacy.database")}</Label>
                  <Input
                    id="legacyDatabase"
                    value={legacy.database}
                    onChange={(e) =>
                      setLegacy((prev) => ({ ...prev, database: e.target.value }))
                    }
                    placeholder="lsky"
                  />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label htmlFor="legacyUsername">{t("installer.databaseUser")}</Label>
                    <Input
                      id="legacyUsername"
                      value={legacy.username}
                      onChange={(e) =>
                        setLegacy((prev) => ({ ...prev, username: e.target.value }))
                      }
                      placeholder="root"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="legacyPassword">{t("installer.databasePassword")}</Label>
                    <Input
                      id="legacyPassword"
                      type="password"
                      value={legacy.password}
                      onChange={(e) =>
                        setLegacy((prev) => ({ ...prev, password: e.target.value }))
                      }
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="legacyPrefix">{t("installer.legacy.prefix")}</Label>
                  <Input
                    id="legacyPrefix"
                    value={legacy.tablePrefix}
                    onChange={(e) =>
                      setLegacy((prev) => ({ ...prev, tablePrefix: e.target.value }))
                    }
                    placeholder="lsky_"
                  />
                </div>

                <p className="text-sm text-muted-foreground">
                  {t("installer.legacy.plaintextHint")}
                </p>
              </>
            )}

            <div className="space-y-2">
              <Label htmlFor="legacyImageDir">{t("installer.legacy.imageDir")}</Label>
              <Input
                id="legacyImageDir"
                value={legacy.imageDir}
                onChange={(e) =>
                  setLegacy((prev) => ({ ...prev, imageDir: e.target.value }))
                }
                placeholder="storage/app/uploads"
              />
              <p className="text-sm text-muted-foreground">
                {t("installer.legacy.imageDirHint")}
              </p>
            </div>

            {probe && !probe.missingTables?.length && (
              <div className="rounded-md border bg-muted/40 p-3 text-sm space-y-1">
                <p>
                  {t("installer.legacy.detected", {
                    name: probe.appName || "Lsky Pro",
                    version: probe.appVersion || "?"
                  })}
                </p>
                <p>
                  {t("installer.legacy.counts", {
                    users: probe.counts?.users ?? 0,
                    images: probe.counts?.images ?? 0,
                    albums: probe.counts?.albums ?? 0,
                    groups: probe.counts?.groups ?? 0,
                    strategies: probe.counts?.strategies ?? 0
                  })}
                </p>
                <p className="text-muted-foreground">
                  {t("installer.legacy.readOnly", {
                    status:
                      probe.readOnlyReady
                        ? t("installer.legacy.readOnlyOn")
                        : t("installer.legacy.readOnlyOff")
                  })}
                </p>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              <Button
                variant="outline"
                onClick={() => probeMutation.mutate()}
                disabled={probeMutation.isPending}
              >
                {probeMutation.isPending
                  ? t("installer.legacy.testing")
                  : t("installer.legacy.test")}
              </Button>
              <Button onClick={goNextFromLegacy}>{t("installer.next")}</Button>
            </div>
            <Button
              variant="ghost"
              className="w-full"
              onClick={() => setStep("product")}
            >
              {t("installer.previous")}
            </Button>
          </CardContent>
        </Card>
      )}

      {step === "database" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("installer.database")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="databaseType">{t("installer.databaseType")}</Label>
              <Select
                value={form.databaseType}
                onValueChange={(value) =>
                  setForm((prev) => ({ ...prev, databaseType: value }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("installer.selectDatabaseType")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="sqlite">{t("installer.sqliteRecommended")}</SelectItem>
                  <SelectItem value="mysql">MySQL</SelectItem>
                  <SelectItem value="postgres">PostgreSQL</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {form.databaseType === "sqlite" && (
              <div className="space-y-2">
                <Label htmlFor="databasePath">{t("installer.databasePath")}</Label>
                <Input
                  id="databasePath"
                  value={form.databasePath}
                  onChange={(e) =>
                    setForm((prev) => ({ ...prev, databasePath: e.target.value }))
                  }
                  placeholder="storage/data/skyimage.db"
                />
                <p className="text-sm text-muted-foreground">
                  {t("installer.sqliteHint")}
                </p>
              </div>
            )}

            {form.databaseType !== "sqlite" && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="databaseHost">{t("installer.databaseHost")}</Label>
                  <Input
                    id="databaseHost"
                    value={form.databaseHost}
                    onChange={(e) =>
                      setForm((prev) => ({ ...prev, databaseHost: e.target.value }))
                    }
                    placeholder="localhost"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="databasePort">{t("installer.port")}</Label>
                  <Input
                    id="databasePort"
                    value={form.databasePort}
                    onChange={(e) =>
                      setForm((prev) => ({ ...prev, databasePort: e.target.value }))
                    }
                    placeholder={form.databaseType === "postgres" ? "5432" : "3306"}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="databaseName">{t("installer.databaseName")}</Label>
                  <Input
                    id="databaseName"
                    value={form.databaseName}
                    onChange={(e) =>
                      setForm((prev) => ({ ...prev, databaseName: e.target.value }))
                    }
                    placeholder="skyimage"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="databaseUser">{t("installer.databaseUser")}</Label>
                  <Input
                    id="databaseUser"
                    value={form.databaseUser}
                    onChange={(e) =>
                      setForm((prev) => ({ ...prev, databaseUser: e.target.value }))
                    }
                    placeholder="root"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="databasePassword">{t("installer.databasePassword")}</Label>
                  <Input
                    id="databasePassword"
                    type="password"
                    value={form.databasePassword}
                    onChange={(e) =>
                      setForm((prev) => ({ ...prev, databasePassword: e.target.value }))
                    }
                  />
                </div>
              </>
            )}

            <div className="flex gap-3">
              <Button
                variant="outline"
                className="w-full"
                onClick={() => setStep(mode === "import" ? "product" : "mode")}
              >
                {t("installer.previous")}
              </Button>
              <Button className="w-full" onClick={() => setStep("site")}>
                {t("installer.next")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {step === "site" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("installer.siteInfo")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="siteName">{t("installer.siteName")}</Label>
              <Input
                id="siteName"
                value={form.siteName}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, siteName: e.target.value }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="adminName">{t("installer.adminName")}</Label>
              <Input
                id="adminName"
                value={form.adminName}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, adminName: e.target.value }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="adminEmail">{t("installer.adminEmail")}</Label>
              <Input
                id="adminEmail"
                type="email"
                value={form.adminEmail}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, adminEmail: e.target.value }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="adminPassword">{t("installer.adminPassword")}</Label>
              <Input
                id="adminPassword"
                type="password"
                value={form.adminPassword}
                onChange={(e) =>
                  setForm((prev) => ({
                    ...prev,
                    adminPassword: e.target.value
                  }))
                }
              />
            </div>
            <div className="flex gap-3">
              <Button
                variant="outline"
                className="w-full"
                onClick={() => setStep("database")}
              >
                {t("installer.previous")}
              </Button>
              <Button
                className="w-full"
                onClick={() => installMutation.mutate()}
                disabled={installMutation.isPending}
              >
                {installMutation.isPending
                  ? t("installer.installing")
                  : mode === "import"
                    ? t("installer.legacy.importAndInstall")
                    : t("installer.installNow")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {step === "result" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("installer.result.title")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-md border bg-muted/40 p-3 text-sm">
              <p>{t("installer.result.installOk")}</p>
            </div>

            {importError && (
              <div className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
                <p>{t("installer.legacy.importFailed", { error: importError })}</p>
                <p className="mt-1 text-muted-foreground">
                  {t("installer.legacy.importFailedHint")}
                </p>
              </div>
            )}

            {summary && (
              <div className="rounded-md border bg-muted/40 p-3 text-sm space-y-1">
                <p>{t("installer.legacy.summaryTitle")}</p>
                <p>
                  {t("installer.legacy.summary", {
                    users: summary.users ?? 0,
                    merged: summary.usersMerged ?? 0,
                    files: summary.files ?? 0,
                    copied: summary.filesCopied ?? 0,
                    albums: summary.albums ?? 0,
                    groups: summary.groups ?? 0,
                    strategies: summary.strategies ?? 0,
                    settings: summary.settings ?? 0
                  })}
                </p>
              </div>
            )}

            {summary && ((summary.filesMissing ?? 0) > 0 || (summary.filesForbidden ?? 0) > 0) && (
              <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-600 dark:text-amber-400 space-y-1">
                <p>
                  {t("installer.legacy.incomplete", {
                    missing: summary.filesMissing ?? 0,
                    forbidden: summary.filesForbidden ?? 0
                  })}
                </p>
                {!!summary.forbiddenPaths?.length && (
                  <p className="break-all">
                    {t("installer.legacy.forbiddenList", {
                      paths: summary.forbiddenPaths.join("、")
                    })}
                  </p>
                )}
              </div>
            )}

            {summary && (summary.usersMerged ?? 0) > 0 && (
              <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-600 dark:text-amber-400">
                <p>
                  {t("installer.legacy.merged", { count: summary.usersMerged ?? 0 })}
                </p>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              {importError && (
                <Button
                  variant="outline"
                  onClick={() => retryMutation.mutate()}
                  disabled={retryMutation.isPending}
                >
                  {retryMutation.isPending
                    ? t("installer.legacy.importing")
                    : t("installer.legacy.retry")}
                </Button>
              )}
              <Button
                className={importError ? "" : "col-span-2"}
                onClick={() => (window.location.href = "/login")}
              >
                {importError
                  ? t("installer.legacy.skipToLogin")
                  : t("installer.legacy.finish")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
