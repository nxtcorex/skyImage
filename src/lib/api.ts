import axios from "axios";
import { useAuthStore } from "@/state/auth";

const apiBase = import.meta.env.VITE_API_BASE_URL || "/api";
const disabledNoticeKey = "skyimage-disabled-notice";

export const apiClient = axios.create({
  baseURL: apiBase,
  withCredentials: true
});

function getCookie(name: string): string {
  if (typeof document === "undefined") {
    return "";
  }
  const encoded = encodeURIComponent(name) + "=";
  const parts = document.cookie.split(";");
  for (const part of parts) {
    const trimmed = part.trim();
    if (trimmed.startsWith(encoded)) {
      return decodeURIComponent(trimmed.slice(encoded.length));
    }
  }
  return "";
}

apiClient.interceptors.request.use((config) => {
  const method = (config.method || "get").toLowerCase();
  if (method === "post" || method === "put" || method === "patch" || method === "delete") {
    const csrf = getCookie("skyimage_csrf");
    if (csrf) {
      config.headers = config.headers || {};
      config.headers["X-CSRF-Token"] = csrf;
    }
  }
  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error.response?.status;
    const message =
      error.response?.data?.error || error.message || "Unknown error";
    const normalized = String(message).toLowerCase();

    const shouldFlagDisabled =
      status === 403 && normalized.includes("account disabled");

    if (shouldFlagDisabled) {
      useAuthStore.getState().clear();
      if (typeof window !== "undefined") {
        window.sessionStorage.setItem(disabledNoticeKey, "1");
        if (window.location.pathname !== "/login") {
          window.location.href = "/login";
        }
      }
    }

    const wrappedError: Error & { status?: number } = new Error(message);
    wrappedError.status = status;
    return Promise.reject(wrappedError);
  }
);

export type InstallerStatus = {
  installed: boolean;
  siteName?: string;
  version?: string;
};

export async function fetchInstallerStatus() {
  try {
    const res = await apiClient.get<{ data: InstallerStatus }>(
      "/installer/status"
    );
    return res.data.data;
  } catch (error) {
    const status = (error as Error & { status?: number }).status;
    if (status === 404) {
      return { installed: true };
    }
    throw error;
  }
}

export async function runInstaller(payload: {
  databaseType?: string;
  databasePath?: string;
  databaseHost?: string;
  databasePort?: string;
  databaseName?: string;
  databaseUser?: string;
  databasePassword?: string;
  siteName: string;
  adminName: string;
  adminEmail: string;
  adminPassword: string;
}) {
  const res = await apiClient.post<{ data: InstallerStatus }>(
    "/installer/run",
    payload
  );
  return res.data.data;
}

export type LegacyProbeResult = {
  appName?: string;
  appVersion?: string;
  counts?: Record<string, number>;
  missingTables?: string[];
  readOnlyReady?: boolean;
};

export type LegacyImportSummary = {
  appName?: string;
  appVersion?: string;
  sourceReadOnly?: boolean;
  groups?: number;
  strategies?: number;
  groupStrategies?: number;
  users?: number;
  usersMerged?: number;
  albums?: number;
  files?: number;
  filesCopied?: number;
  filesMissing?: number;
  filesForbidden?: number;
  forbiddenPaths?: string[];
  settings?: number;
};

export type LegacySourcePayload = {
  databaseType: "mysql" | "sqlite" | "postgres" | "sqlserver";
  host?: string;
  port?: string;
  database?: string;
  username?: string;
  password?: string;
  tablePrefix?: string;
  sqliteDir?: string;
  imageDir?: string;
  adminEmail: string;
  adminPassword: string;
};

// 只读探测 Lsky Pro 源数据库（仅 SELECT）
export async function testLegacySource(payload: LegacySourcePayload) {
  const res = await apiClient.post<{ data: LegacyProbeResult }>(
    "/installer/legacy/test",
    payload
  );
  return res.data.data;
}

// 从 Lsky Pro 源数据库导入数据到新数据库（源库只读）
export async function importLegacySource(payload: LegacySourcePayload) {
  const res = await apiClient.post<{ data: LegacyImportSummary }>(
    "/installer/legacy/import",
    payload
  );
  return res.data.data;
}

export async function login(payload: {
  email: string;
  password: string;
  turnstileToken?: string;
  captchaToken?: string;
  captchaData?: Record<string, string>;
  captchaProvider?: string;
}) {
  const res = await apiClient.post<{ data: { user: any } }>(
    "/auth/login",
    payload
  );
  return res.data.data;
}

export async function sendVerificationCode(payload: {
  email: string;
  turnstileToken?: string;
  captchaToken?: string;
  captchaData?: Record<string, string>;
  captchaProvider?: string;
}) {
  const res = await apiClient.post<{ data: { message: string } }>("/auth/send-verification-code", payload);
  return res.data.data;
}

export async function register(payload: {
  name: string;
  email: string;
  password: string;
  verificationCode: string;
  turnstileToken?: string;
  captchaToken?: string;
  captchaData?: Record<string, string>;
  captchaProvider?: string;
}) {
  const res = await apiClient.post<{ data: { user: any } }>("/auth/register", payload);
  return res.data.data;
}

export async function logout() {
  await apiClient.post("/auth/logout");
}

export type RegistrationStatus = {
  allowed: boolean;
  mode: "open" | "oauth_only" | "closed";
  passwordAllowed: boolean;
  oauthAllowed: boolean;
  emailVerifyEnabled: boolean;
  forgotPasswordEnabled: boolean;
  passkeyEnabled?: boolean;
};

export async function fetchRegistrationStatus() {
  const res = await apiClient.get<{ data: RegistrationStatus }>("/auth/registration-status");
  return res.data.data;
}

export type OAuthProviderPublic = {
  id: string;
  name: string;
  enabled: boolean;
};

export async function fetchOAuthProviders() {
  const res = await apiClient.get<{ data: { providers: OAuthProviderPublic[] } }>(
    "/auth/oauth/providers"
  );
  return res.data.data.providers ?? [];
}

export function getOAuthStartUrl(provider: string) {
  const base = apiBase.replace(/\/$/, "");
  return `${base}/auth/oauth/${provider}/start`;
}

export type OAuthBinding = {
  id: number;
  provider: string;
  providerEmail?: string;
  providerName?: string;
  avatarUrl?: string;
  createdAt: string;
};

export async function fetchOAuthBindings() {
  const res = await apiClient.get<{ data: OAuthBinding[] }>("/auth/oauth/bindings");
  return res.data.data ?? [];
}

export async function startOAuthBind(provider: string) {
  const res = await apiClient.post<{ data: { url: string } }>(
    `/auth/oauth/${provider}/bind`,
    {},
    { headers: { Accept: "application/json" } }
  );
  return res.data.data.url;
}

export async function unbindOAuth(provider: string) {
  await apiClient.delete(`/auth/oauth/${provider}`);
}

export async function requestPasswordReset(payload: {
  email: string;
  turnstileToken?: string;
  captchaToken?: string;
  captchaData?: Record<string, string>;
  captchaProvider?: string;
}) {
  const res = await apiClient.post<{ data: { message: string } }>("/auth/forgot-password", payload);
  return res.data.data;
}

export async function resetPasswordByEmail(payload: {
  token: string;
  code: string;
  password: string;
  turnstileToken?: string;
  captchaToken?: string;
  captchaData?: Record<string, string>;
  captchaProvider?: string;
}) {
  const res = await apiClient.post<{ data: { message: string } }>("/auth/reset-password", payload);
  return res.data.data;
}

export type ResetPasswordStatus = {
  valid: boolean;
  captchaEnabled: boolean;
  captchaConfig?: CaptchaConfig;
};

export async function fetchResetPasswordStatus(token: string) {
  const res = await apiClient.get<{ data: ResetPasswordStatus }>("/auth/reset-password-status", {
    params: { token }
  });
  return res.data.data;
}

export async function fetchProfile() {
  const res = await apiClient.get<{ data: any }>("/auth/me");
  return res.data.data;
}

export async function fetchHasUsers() {
  const res = await apiClient.get<{ data: { hasUsers: boolean } }>(
    "/auth/needs-setup"
  );
  return res.data.data.hasUsers;
}

export async function fetchAdminMetrics() {
  const res = await apiClient.get<{ data: any }>("/admin/metrics");
  return res.data.data;
}

export type TrendData = {
  date: string;
  uploads: number;
  registrations: number;
};

export async function fetchAdminTrends(days: number = 90): Promise<TrendData[]> {
  const res = await apiClient.get<{ data: TrendData[] }>(`/admin/trends?days=${days}`);
  return res.data.data;
}

export type UserTrendData = {
  date: string;
  uploads: number;
};

export async function fetchUserTrends(days: number = 90): Promise<UserTrendData[]> {
  const res = await apiClient.get<{ data: UserTrendData[] }>(`/files/trends?days=${days}`);
  return res.data.data;
}

export type SiteConfig = {
  title: string;
  description: string;
  slogan?: string;
  logo?: string;
  about: string;
  aboutTitle?: string;
  notFoundMode?: string;
  notFoundHeading?: string;
  notFoundText?: string;
  notFoundHtml?: string;
  homePageMode?: "default" | "custom_html";
  homeCustomHtml?: string;
  enableGallery: boolean;
  enableHome?: boolean;
  enableApi: boolean;
  hiddenSidebarItems?: string[];
  forgotPasswordEnabled?: boolean;
  forgotPasswordTurnstileRequest?: boolean;
  forgotPasswordTurnstileReset?: boolean;
  imageLoadRows?: number;
  version?: string;
  accountDisabledNotice?: string;
};

export async function fetchSiteConfig() {
  const res = await apiClient.get<{ data: SiteConfig }>("/site/config");
  const config = res.data.data;
  
  try {
    localStorage.setItem("skyimage-site-config", JSON.stringify(config));
  } catch (error) {
    console.warn("Failed to cache site config:", error);
  }
  
  return config;
}

export type LegalType = "terms" | "privacy";

export async function fetchLegalContent(type: LegalType) {
  const res = await apiClient.get<{ data: { content: string } }>(`/site/legal/${type}`);
  return res.data.data.content;
}

export async function fetchGalleryPublic(params?: {
  limit?: number;
  offset?: number;
}) {
  const res = await apiClient.get<{ data: FileRecord[] }>("/gallery/public", {
    params
  });
  return res.data.data;
}

export type PublicUserImage = {
  viewUrl: string;
  thumbnailUrl: string;
};

export type PublicUserProfile = {
  name: string;
  avatarUrl?: string;
  images: PublicUserImage[];
};

export async function fetchPublicUserProfile(
  userId: string,
  params?: { limit?: number; offset?: number }
) {
  const res = await apiClient.get<{ data: PublicUserProfile }>(
    `/users/${encodeURIComponent(userId)}/public`,
    { params }
  );
  return res.data.data;
}

export async function fetchAdminSettings() {
  const res = await apiClient.get<{ data: Record<string, string> }>(
    "/admin/settings"
  );
  return res.data.data;
}

export async function updateAdminSettings(input: Record<string, string>) {
  await apiClient.put("/admin/settings", input);
}

export async function fetchUsers() {
  const res = await apiClient.get<{ data: any[] }>("/admin/users");
  return res.data.data;
}

export async function updateUserStatus(userId: string | number, status: number) {
  await apiClient.patch(`/admin/users/${userId}/status`, { status });
}

export async function toggleUserAdmin(userId: string | number, admin: boolean) {
  await apiClient.post(`/admin/users/${userId}/admin`, { admin });
}

export type CreateUserPayload = {
  name: string;
  email: string;
  password: string;
  role: "admin" | "user";
};

export async function createUser(payload: CreateUserPayload) {
  const res = await apiClient.post<{ data: any }>("/admin/users", payload);
  return res.data.data;
}

export async function deleteUserAccount(userId: string | number) {
  await apiClient.delete(`/admin/users/${userId}`);
}

export async function fetchFiles(params?: { limit?: number; offset?: number }) {
  const res = await apiClient.get<{ data: FileRecord[] }>("/files", { params });
  return res.data.data;
}

export async function deleteFile(id: number) {
  await apiClient.delete(`/files/${id}`);
}

export async function updateFileVisibility(id: number, visibility: "public" | "private") {
  const res = await apiClient.patch<{ data: FileRecord }>(`/files/${id}/visibility`, {
    visibility
  });
  return res.data.data;
}

export async function updateFilesVisibilityBatch(
  ids: number[],
  visibility: "public" | "private"
) {
  const res = await apiClient.patch<{ data: { updated: number } }>(
    "/files/batch/visibility",
    { ids, visibility }
  );
  return res.data.data;
}

export async function deleteFilesBatch(ids: number[]) {
  const res = await apiClient.post<{ data: { deleted: number } }>(
    "/files/batch/delete",
    { ids }
  );
  return res.data.data;
}

export async function uploadFile(payload: {
  file: File;
  visibility: "public" | "private";
  strategyId?: number;
}) {
  const formData = new FormData();
  formData.append("file", payload.file, payload.file.name);
  formData.append("visibility", payload.visibility);
  if (payload.strategyId) {
    formData.append("strategyId", String(payload.strategyId));
  }
  const res = await apiClient.post<{ data: FileRecord }>(
    "/files",
    formData,
    {
      headers: { "Content-Type": "multipart/form-data" }
    }
  );
  return res.data.data;
}

export type FileAuditRecord = {
  status: "none" | "approved" | "pending" | "rejected" | "error";
  decision?: "pass" | "review" | "block" | "error";
  provider?: string;
  riskLevel?: string;
  label?: string;
  nsfwScore?: number;
  confidence?: number;
  message?: string;
  checkedAt?: string;
  reviewedAt?: string;
};

export type FileRecord = {
  id: number;
  key: string;
  originalName: string;
  size: number;
  mimeType?: string;
  extension?: string;
  checksumMd5?: string;
  checksumSha1?: string;
  viewUrl: string;
  directUrl: string;
  thumbnailUrl?: string;
  visibility: string;
  markdown: string;
  html: string;
  createdAt: string;
  ownerId?: string | number;
  ownerName?: string;
  ownerEmail?: string;
  ownerPublicProfile?: boolean;
  strategyId?: number;
  strategyName?: string;
  relativePath?: string;
  width?: number;
  height?: number;
  audit?: FileAuditRecord;
};

export async function fetchAccountProfile() {
  const res = await apiClient.get<{ data: any; globalLoginNotificationEnabled?: boolean }>("/account/profile");
  return {
    user: res.data.data,
    globalLoginNotificationEnabled: res.data.globalLoginNotificationEnabled ?? false,
  };
}

export async function updateAccountProfile(input: {
  name: string;
  url: string;
  password?: string;
  defaultVisibility?: "public" | "private";
  theme?: "light" | "dark" | "system";
  loginNotification?: boolean;
  publicProfile?: boolean;
  ticketStaffName?: string;
}) {
  const res = await apiClient.put<{ data: any }>("/account/profile", input);
  return res.data.data;
}

export async function deleteAccount() {
  await apiClient.delete("/account/profile");
}

export type UserNotificationMetadata = {
  fileId?: number;
  fileKey?: string;
  fileOriginalName?: string;
  reasonType?: "audit_block_delete" | "audit_error_delete" | "admin_delete";
  auditMessage?: string;
  adminReason?: string;
};

export type UserNotificationRecord = {
  id: number;
  type: string;
  title: string;
  message: string;
  metadata: UserNotificationMetadata;
  readAt?: string;
  createdAt: string;
  updatedAt: string;
};

export async function fetchAccountNotifications(params?: {
  status?: "all" | "unread" | "read";
  limit?: number;
  offset?: number;
}) {
  const res = await apiClient.get<{ data: UserNotificationRecord[] }>(
    "/account/notifications",
    { params }
  );
  return res.data.data;
}

export async function updateAccountNotificationRead(id: number, read = true) {
  const res = await apiClient.patch<{ data: UserNotificationRecord }>(
    `/account/notifications/${id}/read`,
    { read }
  );
  return res.data.data;
}

export async function markAllAccountNotificationsRead() {
  const res = await apiClient.post<{ data: { updated: number } }>(
    "/account/notifications/read-all"
  );
  return res.data.data;
}

export async function clearAccountNotifications() {
  const res = await apiClient.delete<{ data: { deleted: number } }>(
    "/account/notifications"
  );
  return res.data.data;
}

export type GroupRecord = {
  id: number;
  name: string;
  isDefault: boolean;
  isGuest?: boolean;
  configs: Record<string, any>;
};

export async function fetchGroups() {
  const res = await apiClient.get<{ data: GroupRecord[] }>("/admin/groups");
  return res.data.data;
}

export async function saveGroup(input: Partial<GroupRecord> & { name: string }) {
  if (input.id) {
    const res = await apiClient.put<{ data: GroupRecord }>(
      `/admin/groups/${input.id}`,
      input
    );
    return res.data.data;
  }
  const res = await apiClient.post<{ data: GroupRecord }>("/admin/groups", input);
  return res.data.data;
}

export async function deleteGroup(id: number) {
  await apiClient.delete(`/admin/groups/${id}`);
}

export type StrategyRecord = {
  id: number;
  key: number;
  name: string;
  intro: string;
  configs: Record<string, any>;
  groups?: GroupRecord[];
  groupIds?: number[];
};

export type UserStrategyOption = {
  id: number;
  name: string;
  intro: string;
};

export async function fetchUploadStrategies() {
  const res = await apiClient.get<{
    data: { strategies: UserStrategyOption[]; defaultStrategyId?: number };
  }>("/files/strategies");
  return res.data.data;
}

export async function fetchStrategies() {
  const res = await apiClient.get<{ data: StrategyRecord[] }>(
    "/admin/strategies"
  );
  return res.data.data;
}

export async function saveStrategy(
  input: Partial<StrategyRecord> & { name: string }
) {
  if (input.id) {
    const res = await apiClient.put<{ data: StrategyRecord }>(
      `/admin/strategies/${input.id}`,
      input
    );
    return res.data.data;
  }
  const res = await apiClient.post<{ data: StrategyRecord }>(
    "/admin/strategies",
    input
  );
  return res.data.data;
}

export async function deleteStrategy(id: number) {
  await apiClient.delete(`/admin/strategies/${id}`);
}

export type AuditProfileRecord = {
  id: number;
  name: string;
  provider: string;
  configs: Record<string, any>;
};

export async function fetchAuditProfiles() {
  const res = await apiClient.get<{ data: AuditProfileRecord[] }>("/admin/audits");
  return res.data.data;
}

export async function saveAuditProfile(
  input: Partial<AuditProfileRecord> & { name: string }
) {
  if (input.id) {
    const res = await apiClient.put<{ data: AuditProfileRecord }>(
      `/admin/audits/${input.id}`,
      input
    );
    return res.data.data;
  }
  const res = await apiClient.post<{ data: AuditProfileRecord }>("/admin/audits", input);
  return res.data.data;
}

export async function deleteAuditProfile(id: number) {
  await apiClient.delete(`/admin/audits/${id}`);
}

export async function fetchUserDetail(userId: string | number) {
  const res = await apiClient.get<{ data: any }>(`/admin/users/${userId}`);
  return res.data.data;
}

export async function assignUserGroup(userId: string | number, groupId: number | null) {
  const res = await apiClient.patch<{ data: any }>(
    `/admin/users/${userId}/group`,
    { groupId }
  );
  return res.data.data;
}

export async function adjustUserCapacityBonus(
  userId: string | number,
  input: { deltaBytes?: number; bonusBytes?: number }
) {
  const res = await apiClient.patch<{ data: any }>(
    `/admin/users/${userId}/capacity-bonus`,
    input
  );
  return res.data.data;
}

export async function fetchAdminImages(params?: {
  limit?: number;
  offset?: number;
  auditStatus?: string;
}) {
  const res = await apiClient.get<{ data: FileRecord[] }>("/admin/images", {
    params
  });
  return res.data.data;
}

export async function deleteAdminImage(id: number, reason?: string) {
  await apiClient.delete(`/admin/images/${id}`, {
    data: reason ? { reason } : undefined
  });
}

export async function updateAdminImageVisibility(
  id: number,
  visibility: "public" | "private"
) {
  const res = await apiClient.patch<{ data: FileRecord }>(
    `/admin/images/${id}/visibility`,
    { visibility }
  );
  return res.data.data;
}

export async function updateAdminImageAuditStatus(
  id: number,
  status: "approved"
) {
  const res = await apiClient.patch<{ data: FileRecord }>(
    `/admin/images/${id}/audit-status`,
    { status }
  );
  return res.data.data;
}

export async function updateAdminImagesVisibilityBatch(
  ids: number[],
  visibility: "public" | "private"
) {
  const res = await apiClient.patch<{ data: { updated: number } }>(
    "/admin/images/batch/visibility",
    { ids, visibility }
  );
  return res.data.data;
}

export async function deleteAdminImagesBatch(ids: number[], reason?: string) {
  const res = await apiClient.post<{ data: { deleted: number } }>(
    "/admin/images/batch/delete",
    { ids, reason }
  );
  return res.data.data;
}

// ── Site Settings ──

export type RegistrationMode = "open" | "oauth_only" | "closed";

export type SiteSettings = {
  siteTitle: string;
  consoleUrl: string;
  siteDescription: string;
  siteSlogan: string;
  siteLogo: string;
  about: string;
  aboutTitle: string;
  notFoundMode: string;
  notFoundHeading: string;
  notFoundText: string;
  notFoundHtml: string;
  homePageMode: "default" | "custom_html";
  homeCustomHtml: string;
  accountDisabledNotice: string;
};

// 站点设置的配置文件仅用于部分更新，提交时只携带修改过的字段。
export type SiteSettingsUpdate = Partial<Omit<SiteSettings, "termsOfService" | "privacyPolicy">>;

export type OAuthProviderSettings = {
  enabled: boolean;
  clientId: string;
  clientSecret: string;
  name?: string;
  authUrl?: string;
  tokenUrl?: string;
  userInfoUrl?: string;
  scopes?: string;
};

export type OAuthSettings = {
  enabled: boolean;
  autoLinkByEmail: boolean;
  github: OAuthProviderSettings;
  google: OAuthProviderSettings;
  discord: OAuthProviderSettings;
  custom: OAuthProviderSettings;
};

export async function fetchOAuthSettings() {
  const res = await apiClient.get<{ data: OAuthSettings }>("/admin/system/oauth");
  return res.data.data;
}

export type OAuthProviderSettingsUpdate = {
  enabled?: boolean;
  clientId?: string;
  clientSecret?: string;
  name?: string;
  authUrl?: string;
  tokenUrl?: string;
  userInfoUrl?: string;
  scopes?: string;
};

export type OAuthSettingsUpdate = {
  enabled?: boolean;
  autoLinkByEmail?: boolean;
  github?: OAuthProviderSettingsUpdate;
  google?: OAuthProviderSettingsUpdate;
  discord?: OAuthProviderSettingsUpdate;
  custom?: OAuthProviderSettingsUpdate;
};

export async function updateOAuthSettings(input: OAuthSettingsUpdate) {
  await apiClient.patch("/admin/system/oauth", input);
}

// ── Database config & migration ──

export type DatabaseConfigView = {
  type: string;
  path?: string;
  host?: string;
  port?: string;
  name?: string;
  user?: string;
  hasPassword: boolean;
};

export type DatabaseTargetInput = {
  type: string;
  path?: string;
  host?: string;
  port?: string;
  name?: string;
  user?: string;
  password?: string;
};

export type DatabaseMigrateResult = {
  sourceType: string;
  targetType: string;
  tables: {
    table: string;
    rows: number;
    sourceRows?: number;
    copiedRows?: number;
    targetRows?: number;
  }[];
  switched: boolean;
};

export async function fetchDatabaseConfig() {
  const res = await apiClient.get<{ data: DatabaseConfigView }>("/admin/system/database");
  return res.data.data;
}

export async function testDatabaseConnection(input: DatabaseTargetInput) {
  const res = await apiClient.post<{ data: { ok: boolean } }>("/admin/system/database/test", input);
  return res.data.data;
}

export async function migrateDatabase(input: {
  target: DatabaseTargetInput;
  truncateTarget?: boolean;
  switchRuntime?: boolean;
  batchSize?: number;
}) {
  const res = await apiClient.post<{ data: DatabaseMigrateResult }>(
    "/admin/system/database/migrate",
    input
  );
  return res.data.data;
}

export async function fetchSiteSettings() {
  const res = await apiClient.get<{ data: SiteSettings }>(
    "/admin/system/site"
  );
  return res.data.data;
}

export async function updateSiteSettings(input: SiteSettingsUpdate) {
  await apiClient.patch("/admin/system/site", input);
}

export async function fetchAdminLegalContent(type: LegalType) {
  const res = await apiClient.get<{ data: { content: string } }>(
    `/admin/system/site/legal/${type}`
  );
  return res.data.data.content;
}

export async function updateAdminLegalContent(type: LegalType, content: string) {
  await apiClient.put(`/admin/system/site/legal/${type}`, { content });
}

// ── General Settings ──

export type GeneralSettings = {
  imageLoadRows: number;
  userNotificationLimit: number;
  adminImageDeleteDefaultReason: string;
  systemAutoDeleteDefaultReason: string;
  enableCDN: boolean;
  enableGallery: boolean;
  enableHome: boolean;
  enableApi: boolean;
  enablePasskey: boolean;
  allowRegistration: boolean;
  registrationMode: RegistrationMode;
  hiddenSidebarItems: string[];
};

// 通用设置的更新仅携带修改过的字段。
export type GeneralSettingsUpdate = Partial<GeneralSettings>;

export type TicketSettings = {
  attachmentStrategyId: number;
  emailNotifyEnabled: boolean;
  emailNotifyMode: "all_admins" | "selected";
  emailNotifyAdminIds: string[];
};

export async function fetchTicketSettings() {
  const res = await apiClient.get<{ data: TicketSettings }>("/admin/system/tickets");
  return res.data.data;
}

// 工单设置的更新仅携带修改过的字段。
export type TicketSettingsUpdate = Partial<TicketSettings>;

export async function updateTicketSettings(input: TicketSettingsUpdate) {
  await apiClient.patch("/admin/system/tickets", input);
}

export async function fetchGeneralSettings() {
  const res = await apiClient.get<{ data: GeneralSettings }>(
    "/admin/system/general"
  );
  return res.data.data;
}

export async function updateGeneralSettings(input: GeneralSettingsUpdate) {
  await apiClient.patch("/admin/system/general", input);
}

// ── Email Settings ──

export type EmailSettings = {
  smtpHost: string;
  smtpPort: string;
  smtpUsername: string;
  smtpPassword: string;
  smtpFrom: string;
  smtpSecure: boolean;
  mailTestSubject: string;
  mailTestBody: string;
  mailRegisterVerifySubject: string;
  mailRegisterVerifyBody: string;
  mailRegisterSuccessSubject: string;
  mailRegisterSuccessBody: string;
  mailLoginNotificationSubject: string;
  mailLoginNotificationBody: string;
  mailForgotPasswordSubject: string;
  mailForgotPasswordBody: string;
  mailTicketCreatedSubject: string;
  mailTicketCreatedBody: string;
  mailTicketReplyUserSubject: string;
  mailTicketReplyUserBody: string;
  mailTicketReplyAdminSubject: string;
  mailTicketReplyAdminBody: string;
  mailTicketStatusSubject: string;
  mailTicketStatusBody: string;
  enableRegisterVerify: boolean;
  enableLoginNotification: boolean;
  enableForgotPassword: boolean;
  enableForgotPasswordTurnstile: boolean;
  enableForgotPasswordTurnstileRequest: boolean;
  enableForgotPasswordTurnstileReset: boolean;
};

export async function fetchEmailSettings() {
  const res = await apiClient.get<{ data: EmailSettings }>(
    "/admin/system/email"
  );
  return res.data.data;
}

// 邮件设置的更新仅携带修改过的字段。
export type EmailSettingsUpdate = Partial<EmailSettings>;

export async function updateEmailSettings(input: EmailSettingsUpdate) {
  await apiClient.patch("/admin/system/email", input);
}

// ── Captcha Settings ──

export type CaptchaSettings = {
  enableCaptcha: boolean;
  captchaProvider: "cloudflare" | "geetest" | "cap" | "";
  cloudflareSiteKey: string;
  cloudflareSecretKey: string;
  geetestCaptchaId: string;
  geetestCaptchaKey: string;
  capInstanceUrl: string;
  capSiteKey: string;
  capSecretKey: string;
  enableLoginCaptcha: boolean;
  enableRegisterCaptcha: boolean;
  enableRegisterVerifyCaptcha: boolean;
  enableForgotPasswordRequestCaptcha: boolean;
  enableForgotPasswordResetCaptcha: boolean;
  enableRedeemCaptcha: boolean;
  enableTicketCaptcha: boolean;
  cloudflareVerified: boolean;
  cloudflareLastVerifiedAt?: string;
  geetestVerified: boolean;
  geetestLastVerifiedAt?: string;
  capVerified: boolean;
  capLastVerifiedAt?: string;
};

export async function fetchCaptchaSettings() {
  const res = await apiClient.get<{ data: CaptchaSettings }>(
    "/admin/system/captcha"
  );
  return res.data.data;
}

// 验证码设置的更新仅携带修改过的字段。
export type CaptchaSettingsUpdate = Partial<Omit<CaptchaSettings, "cloudflareVerified" | "cloudflareLastVerifiedAt" | "geetestVerified" | "geetestLastVerifiedAt" | "capVerified" | "capLastVerifiedAt">>;

export async function updateCaptchaSettings(input: CaptchaSettingsUpdate) {
  await apiClient.patch("/admin/system/captcha", input);
}

// ── Test APIs ──

export type TestSmtpPayload = {
  testEmail: string;
  siteTitle: string;
  smtpHost: string;
  smtpPort: string;
  smtpUsername: string;
  smtpPassword: string;
  smtpFrom: string;
  smtpSecure: boolean;
  mailTestSubject: string;
  mailTestBody: string;
};

export type TestSmtpResponse = {
  success: boolean;
  message: string;
};

export async function testSmtpEmail(payload: TestSmtpPayload) {
  const res = await apiClient.post<{ data: TestSmtpResponse }>(
    "/admin/system/email/test",
    payload
  );
  return res.data.data;
}

export type TestTurnstilePayload = {
  siteKey: string;
  secretKey: string;
  token: string;
};

export type TestCaptchaPayload = {
  provider: "cloudflare" | "geetest" | "cap";
  siteKey?: string;
  secretKey?: string;
  captchaId?: string;
  captchaKey?: string;
  instanceUrl?: string;
  token: string;
  extraData?: Record<string, string>;
};

export type TestTurnstileResponse = {
  success: boolean;
  verifiedAt?: string;
  message?: string;
};

export type TestCaptchaResponse = {
  success: boolean;
  verifiedAt?: string;
  message?: string;
};

export async function testTurnstileConfig(payload: TestTurnstilePayload) {
  const res = await apiClient.post<{ data: TestTurnstileResponse }>(
    "/admin/system/captcha/test-turnstile",
    payload
  );
  return res.data.data;
}

export async function testCaptchaConfig(payload: TestCaptchaPayload) {
  const res = await apiClient.post<{ data: TestCaptchaResponse }>(
    "/admin/system/captcha/test",
    payload
  );
  return res.data.data;
}

export type LegalDefaults = {
  termsOfService: string;
  privacyPolicy: string;
};

export async function fetchLegalDefaults() {
  const res = await apiClient.get<{ data: LegalDefaults }>("/installer/defaults");
  return res.data.data;
}

export type ApiTokenResponse = {
  token: string;
};

export type ApiTokenRecord = {
  id: number;
  token: string;
  tokenMasked?: string;
  createdAt: string;
  expiresAt: string;
  lastUsedAt?: string;
};

function readTokenField(value: Record<string, unknown>, keys: string[]): string {
  for (const key of keys) {
    const v = value[key];
    if (typeof v === "string" && v.trim() !== "") {
      return v;
    }
  }
  return "";
}

function readNumberField(value: Record<string, unknown>, keys: string[]): number {
  for (const key of keys) {
    const v = value[key];
    if (typeof v === "number" && Number.isFinite(v)) {
      return v;
    }
    if (typeof v === "string" && v.trim() !== "") {
      const parsed = Number(v);
      if (Number.isFinite(parsed)) {
        return parsed;
      }
    }
  }
  return 0;
}

function normalizeApiToken(value: unknown): ApiTokenRecord | null {
  if (!value || typeof value !== "object") {
    return null;
  }
  const source = value as Record<string, unknown>;
  const masked = readTokenField(source, ["tokenMasked", "token_masked", "token", "Token"]);
  if (!masked) {
    return null;
  }
  return {
    id: readNumberField(source, ["id", "ID"]),
    token: masked,
    tokenMasked: masked,
    createdAt: readTokenField(source, ["createdAt", "created_at", "CreatedAt"]),
    expiresAt: readTokenField(source, ["expiresAt", "expires_at", "ExpiresAt"]),
    lastUsedAt: readTokenField(source, ["lastUsedAt", "last_used_at", "LastUsedAt"]) || undefined
  };
}

export async function generateApiToken(input?: { expiresAt?: string }) {
  const res = await apiClient.post<{ data: ApiTokenResponse }>(
    "/account/api-token",
    input
  );
  return res.data.data;
}

export async function fetchApiTokens() {
  const res = await apiClient.get<{ data: unknown[] }>("/account/api-tokens");
  const source = Array.isArray(res.data.data) ? res.data.data : [];
  return source.map(normalizeApiToken).filter((item): item is ApiTokenRecord => item !== null);
}

export async function deleteApiToken(id: number) {
  await apiClient.delete(`/account/api-token/${id}`);
}

export async function updateApiToken(id: number, input: { expiresAt: string }) {
  await apiClient.patch(`/account/api-token/${id}`, input);
}

export async function deleteApiTokens() {
  await apiClient.delete("/account/api-token");
}

// Captcha configuration API
export type CaptchaConfig = {
  enabled: boolean;
  provider: "cloudflare" | "geetest" | "cap" | "";
  siteKey?: string;
  apiEndpoint?: string;
};

export async function fetchCaptchaConfig(context: string): Promise<CaptchaConfig> {
  const res = await apiClient.get<{ data: CaptchaConfig }>("/auth/captcha-config", {
    params: { context }
  });
  return res.data.data;
}

export type RedeemCodeRecord = {
  id: number;
  code: string;
  rewardType: "group" | "capacity" | string;
  groupId?: number | null;
  capacityDelta?: number;
  maxUses: number;
  usedCount: number;
  allowMultiRedeem: boolean;
  enabled: boolean;
  note?: string;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
  group?: GroupRecord | null;
};

export type RedeemCodeUsageRecord = {
  id: number;
  redeemCodeId: number;
  userId: number;
  createdAt: string;
  user?: {
    id: number;
    name: string;
    email: string;
  };
};

export async function fetchRedeemCodes() {
  const res = await apiClient.get<{ data: RedeemCodeRecord[] }>("/admin/redeem-codes");
  return res.data.data;
}

export async function createRedeemCode(input: {
  code?: string;
  rewardType: "group" | "capacity";
  groupId?: number | null;
  capacityDelta?: number;
  maxUses: number;
  allowMultiRedeem: boolean;
  enabled?: boolean;
  note?: string;
  autoGenerate?: boolean;
}) {
  const res = await apiClient.post<{ data: RedeemCodeRecord }>("/admin/redeem-codes", input);
  return res.data.data;
}

export async function updateRedeemCode(
  id: number,
  input: {
    groupId?: number;
    maxUses?: number;
    allowMultiRedeem?: boolean;
    enabled?: boolean;
    note?: string;
  }
) {
  const res = await apiClient.put<{ data: RedeemCodeRecord }>(`/admin/redeem-codes/${id}`, input);
  return res.data.data;
}

export async function deleteRedeemCode(id: number) {
  await apiClient.delete(`/admin/redeem-codes/${id}`);
}

export async function fetchRedeemCodeUsages(id: number) {
  const res = await apiClient.get<{ data: RedeemCodeUsageRecord[] }>(
    `/admin/redeem-codes/${id}/usages`
  );
  return res.data.data;
}

export async function redeemCode(input: {
  code: string;
  captchaToken?: string;
  captchaData?: Record<string, string>;
  captchaProvider?: string;
}) {
  const res = await apiClient.post<{
    data: {
      user: any;
      group?: GroupRecord | null;
      code: RedeemCodeRecord;
    };
    message?: string;
  }>("/account/redeem", input);
  return { ...res.data.data, message: res.data.message };
}

// --- Shop / Payment ---

export type ShopProduct = {
  id: number;
  name: string;
  description?: string;
  priceCents: number;
  currency: string;
  durationDays: number;
  groupId: number;
  enabled: boolean;
  sort: number;
  group?: GroupRecord;
  createdAt?: string;
  updatedAt?: string;
};

export type ShopOrder = {
  id: number;
  orderNo: string;
  userId: number;
  productId: number;
  productName: string;
  priceCents: number;
  currency: string;
  durationDays: number;
  groupId: number;
  status: string;
  provider: string;
  providerTradeNo?: string;
  paidAt?: string | null;
  fulfilledAt?: string | null;
  membershipExpiresAt?: string | null;
  createdAt: string;
  group?: GroupRecord;
  user?: { id: number; name: string; email: string };
};

export type MembershipInfo = {
  active: boolean;
  expiresAt?: string | null;
  groupId?: number | null;
  groupName?: string;
  previousGroupId?: number | null;
};

export type PaymentSettings = {
  enabled: boolean;
  epay: {
    enabled: boolean;
    apiUrl: string;
    pid: string;
    key: string;
    defaultType: string;
  };
  alipay: {
    enabled: boolean;
    appId: string;
    privateKey: string;
    alipayPublicKey: string;
    gateway: string;
  };
  wechat: {
    enabled: boolean;
    appId: string;
    mchId: string;
    apiKey: string;
  };
  stripe: {
    enabled: boolean;
    secretKey: string;
    webhookSecret: string;
  };
};

export async function fetchShopProducts() {
  const res = await apiClient.get<{ data: ShopProduct[] }>("/shop/products");
  return res.data.data;
}

export async function fetchShopProviders() {
  const res = await apiClient.get<{ data: string[] }>("/shop/providers");
  return res.data.data;
}

export async function createShopOrder(input: {
  productId: number;
  provider: string;
  epayType?: string;
  returnUrl?: string;
}) {
  const res = await apiClient.post<{
    data: {
      order: ShopOrder;
      payUrl?: string;
      qrContent?: string;
      extra?: Record<string, string>;
    };
  }>("/shop/orders", input);
  return res.data.data;
}

export async function fetchMyShopOrders(params?: { limit?: number; offset?: number }) {
  const res = await apiClient.get<{ data: ShopOrder[] }>("/shop/orders", { params });
  return res.data.data;
}

export async function fetchMyShopOrder(orderNo: string) {
  const res = await apiClient.get<{ data: ShopOrder }>(`/shop/orders/${orderNo}`);
  return res.data.data;
}

export async function fetchMembership() {
  const res = await apiClient.get<{ data: MembershipInfo }>("/shop/membership");
  return res.data.data;
}

export async function fetchAdminShopProducts() {
  const res = await apiClient.get<{ data: ShopProduct[] }>("/admin/shop/products");
  return res.data.data;
}

export async function createAdminShopProduct(input: {
  name: string;
  description?: string;
  priceCents: number;
  currency?: string;
  durationDays: number;
  groupId: number;
  enabled?: boolean;
  sort?: number;
}) {
  const res = await apiClient.post<{ data: ShopProduct }>("/admin/shop/products", input);
  return res.data.data;
}

export async function updateAdminShopProduct(
  id: number,
  input: {
    name: string;
    description?: string;
    priceCents: number;
    currency?: string;
    durationDays: number;
    groupId: number;
    enabled?: boolean;
    sort?: number;
  }
) {
  const res = await apiClient.put<{ data: ShopProduct }>(`/admin/shop/products/${id}`, input);
  return res.data.data;
}

export async function deleteAdminShopProduct(id: number) {
  await apiClient.delete(`/admin/shop/products/${id}`);
}

export async function fetchAdminShopOrders(params?: { status?: string; limit?: number; offset?: number }) {
  const res = await apiClient.get<{ data: ShopOrder[] }>("/admin/shop/orders", { params });
  return res.data.data;
}

export async function fetchPaymentSettings() {
  const res = await apiClient.get<{ data: PaymentSettings }>("/admin/system/payment");
  return res.data.data;
}

export async function updatePaymentSettings(input: PaymentSettings) {
  const res = await apiClient.put<{ data: boolean }>("/admin/system/payment", input);
  return res.data.data;
}

export function formatPriceCents(cents: number, currency = "CNY") {
  const amount = (cents / 100).toFixed(2);
  if (currency === "CNY" || currency === "cny") return `¥${amount}`;
  if (currency === "USD" || currency === "usd") return `$${amount}`;
  return `${amount} ${currency}`;
}

// ── Tickets ──

export type TicketStatus = "open" | "pending" | "resolved" | "closed";
export type TicketPriority = "low" | "normal" | "high" | "urgent";

export type TicketUser = {
  id: string;
  name: string;
  email: string;
};

export type TicketRecord = {
  id: number;
  ticketNo: string;
  userId: string;
  subject: string;
  status: TicketStatus;
  priority: TicketPriority;
  lastReplyAt?: string;
  closedAt?: string;
  createdAt: string;
  updatedAt: string;
  user?: TicketUser;
};

export type TicketAttachment = {
  id: number;
  ticketId: number;
  messageId?: number;
  name: string;
  size: number;
  mimeType: string;
  url: string;
  createdAt: string;
};

export type TicketMessage = {
  id: number;
  ticketId: number;
  userId: string;
  body: string;
  isStaff: boolean;
  createdAt: string;
  user?: TicketUser;
  attachments?: TicketAttachment[];
};

export type TicketDetail = {
  ticket: TicketRecord;
  messages: TicketMessage[];
  attachments: TicketAttachment[];
};

export async function fetchMyTickets(params?: {
  status?: string;
  priority?: string;
  limit?: number;
  offset?: number;
}) {
  const res = await apiClient.get<{ data: TicketRecord[] }>("/account/tickets", { params });
  return res.data.data;
}

export async function fetchMyTicket(id: number) {
  const res = await apiClient.get<{ data: TicketDetail }>(`/account/tickets/${id}`);
  return res.data.data;
}

export async function createTicket(input: {
  subject: string;
  body: string;
  priority?: TicketPriority;
  captchaToken?: string;
  captchaData?: Record<string, string>;
  captchaProvider?: string;
}) {
  const res = await apiClient.post<{ data: TicketDetail }>("/account/tickets", input);
  return res.data.data;
}

export async function replyMyTicket(
  id: number,
  body: string,
  captcha?: {
    captchaToken?: string;
    captchaData?: Record<string, string>;
    captchaProvider?: string;
  }
) {
  const res = await apiClient.post<{ data: TicketMessage }>(`/account/tickets/${id}/replies`, {
    body,
    ...captcha
  });
  return res.data.data;
}

export async function closeMyTicket(id: number) {
  const res = await apiClient.post<{ data: TicketRecord }>(`/account/tickets/${id}/close`);
  return res.data.data;
}

export async function uploadMyTicketAttachment(
  id: number,
  file: File,
  messageId?: number,
  captcha?: {
    captchaToken?: string;
    captchaProvider?: string;
  }
) {
  const form = new FormData();
  form.append("file", file);
  if (messageId) {
    form.append("messageId", String(messageId));
  }
  if (captcha?.captchaToken) {
    form.append("captchaToken", captcha.captchaToken);
  }
  if (captcha?.captchaProvider) {
    form.append("captchaProvider", captcha.captchaProvider);
  }
  const res = await apiClient.post<{ data: TicketAttachment }>(
    `/account/tickets/${id}/attachments`,
    form
  );
  return res.data.data;
}

export async function fetchTicketAttachmentStrategy() {
  const res = await apiClient.get<{ data: { strategyId: number; enabled: boolean } }>(
    "/account/tickets/attachment-strategy"
  );
  return res.data.data;
}

export async function fetchAdminTickets(params?: {
  status?: string;
  priority?: string;
  limit?: number;
  offset?: number;
}) {
  const res = await apiClient.get<{ data: TicketRecord[] }>("/admin/tickets", { params });
  return res.data.data;
}

export async function fetchAdminTicket(id: number) {
  const res = await apiClient.get<{ data: TicketDetail }>(`/admin/tickets/${id}`);
  return res.data.data;
}

export async function updateAdminTicket(
  id: number,
  input: { status?: TicketStatus; priority?: TicketPriority }
) {
  const res = await apiClient.patch<{ data: TicketRecord }>(`/admin/tickets/${id}`, input);
  return res.data.data;
}

export async function replyAdminTicket(id: number, body: string) {
  const res = await apiClient.post<{ data: TicketMessage }>(`/admin/tickets/${id}/replies`, {
    body
  });
  return res.data.data;
}

export async function uploadAdminTicketAttachment(
  id: number,
  file: File,
  messageId?: number
) {
  const form = new FormData();
  form.append("file", file);
  if (messageId) {
    form.append("messageId", String(messageId));
  }
  const res = await apiClient.post<{ data: TicketAttachment }>(
    `/admin/tickets/${id}/attachments`,
    form
  );
  return res.data.data;
}
