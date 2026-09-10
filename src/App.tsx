import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import { InstallerPage } from "@/features/installer/InstallerPage";
import { UploadPage } from "@/features/files/UploadPage";
import { UserManagementPage } from "@/features/users/UserManagementPage";
import { LoginPage } from "@/features/auth/LoginPage";
import { RegisterPage } from "@/features/auth/RegisterPage";
import { ForgotPasswordPage } from "@/features/auth/ForgotPasswordPage";
import { ResetPasswordPage } from "@/features/auth/ResetPasswordPage";
import { fetchInstallerStatus, fetchSiteConfig } from "@/lib/api";
import { SplashScreen } from "@/components/SplashScreen";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import { AppShell } from "@/layouts/AppShell";
import { DashboardPage } from "@/features/dashboard/DashboardPage";
import { MyImagesPage } from "@/features/files/MyImagesPage";
import { ProfileSettingsPage } from "@/features/settings/ProfileSettingsPage";
import { NotificationsPage } from "@/features/notifications/NotificationsPage";
import { GalleryPage } from "@/features/gallery/GalleryPage";
import { PublicUserPage } from "@/features/profile/PublicUserPage";
import { ApiDocsPage } from "@/features/api/ApiDocsPage";
import { ApiTokensPage } from "@/features/api/ApiTokensPage";
import { ApiTokenEditorPage } from "@/features/api/ApiTokenEditorPage";
import { AdminConsolePage } from "@/features/admin/AdminDashboard";
import { AdminGroupsPage } from "@/features/admin/AdminGroupsPage";
import { AdminImagesPage } from "@/features/admin/AdminImagesPage";
import { AdminAuditsPage } from "@/features/admin/AdminAuditsPage";
import { AdminStrategiesPage } from "@/features/admin/AdminStrategiesPage";
import { AdminSystemSettingsPage } from "@/features/admin/AdminSystemSettingsPage";
import { AdminSiteSettingsPage } from "@/features/admin/AdminSiteSettingsPage";
import { AdminEmailSettingsPage } from "@/features/admin/AdminEmailSettingsPage";
import { AdminCaptchaSettingsPage } from "@/features/admin/AdminCaptchaSettingsPage";
import { AdminOAuthSettingsPage } from "@/features/admin/AdminOAuthSettingsPage";
import { AdminDatabaseSettingsPage } from "@/features/admin/AdminDatabaseSettingsPage";
import { AdminGroupEditorPage } from "@/features/admin/AdminGroupEditorPage";
import { AdminRedeemCodesPage } from "@/features/admin/AdminRedeemCodesPage";
import { AdminRedeemCodeEditorPage } from "@/features/admin/AdminRedeemCodeEditorPage";
import { AdminStrategyEditorPage } from "@/features/admin/AdminStrategyEditorPage";
import { AdminAuditEditorPage } from "@/features/admin/AdminAuditEditorPage";
import { AdminShopProductsPage } from "@/features/admin/AdminShopProductsPage";
import { AdminShopProductEditorPage } from "@/features/admin/AdminShopProductEditorPage";
import { AdminShopOrdersPage } from "@/features/admin/AdminShopOrdersPage";
import { AdminPaymentSettingsPage } from "@/features/admin/AdminPaymentSettingsPage";
import { AdminUserCreatePage } from "@/features/users/AdminUserCreatePage";
import { AdminUserDetailPage } from "@/features/users/AdminUserDetailPage";
import { AboutPage } from "@/features/about/AboutPage";
import { TermsPage } from "@/features/legal/TermsPage";
import { PrivacyPage } from "@/features/legal/PrivacyPage";
import { PublicShopPage } from "@/features/shop/PublicShopPage";
import { ShopPage } from "@/features/shop/ShopPage";
import { OrdersPage } from "@/features/shop/OrdersPage";
import { TicketsPage } from "@/features/tickets/TicketsPage";
import { TicketCreatePage } from "@/features/tickets/TicketCreatePage";
import { TicketDetailPage } from "@/features/tickets/TicketDetailPage";
import { AdminTicketsPage } from "@/features/admin/AdminTicketsPage";
import { AdminTicketDetailPage } from "@/features/admin/AdminTicketDetailPage";
import { AdminTicketSettingsPage } from "@/features/admin/AdminTicketSettingsPage";
import { AdminRoute } from "@/components/AdminRoute";
import { SiteMetaWatcher } from "@/components/SiteMetaWatcher";
import { NavigationProgress } from "@/components/NavigationProgress";
import { Button } from "@/components/ui/button";
import { NotFoundPage } from "@/features/misc/NotFoundPage";
import { HiddenRoute } from "@/components/layout/HiddenRoute";
import { HomePage } from "@/features/home/HomePage";
import { SearchProvider } from "@/context/search-provider";
import { useI18n } from "@/i18n";

const noIndexPaths = [
  "/installer",
  "/login",
  "/register",
  "/forgot-password",
  "/reset-password",
  "/dashboard"
];

function NoIndexMetaWatcher() {
  const location = useLocation();

  useEffect(() => {
    const path = location.pathname;
    const shouldNoIndex = noIndexPaths.some(p => path === p || path.startsWith(p + "/"));

    let meta = document.querySelector('meta[name="robots"]');

    if (shouldNoIndex) {
      if (!meta) {
        meta = document.createElement("meta");
        meta.setAttribute("name", "robots");
        document.head.appendChild(meta);
      }
      meta.setAttribute("content", "noindex, nofollow");
    } else {
      meta?.remove();
    }
  }, [location.pathname]);

  return null;
}

function HomeEntry() {
  const { data: siteConfig, isLoading } = useQuery({
    queryKey: ["site-config"],
    queryFn: fetchSiteConfig,
    staleTime: 5 * 60 * 1000
  });

  useEffect(() => {
    if (typeof window === "undefined" || siteConfig?.enableHome === undefined) {
      return;
    }
    window.localStorage.setItem("site-config:enable-home", String(siteConfig.enableHome));
  }, [siteConfig?.enableHome]);

  if (siteConfig && siteConfig.enableHome === false) {
    return <Navigate to="/login" replace />;
  }

  if (isLoading && !siteConfig) {
    return <SplashScreen />;
  }

  return <HomePage siteConfig={siteConfig} />;
}

export default function App() {
  const { t } = useI18n();
  const {
    data,
    isLoading,
    error,
    refetch
  } = useQuery({
    queryKey: ["installer"],
    queryFn: fetchInstallerStatus
  });

  if (isLoading) {
    return <SplashScreen />;
  }

  if (error) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-muted/30 p-4 text-center">
        <p className="text-lg font-semibold">{t("app.statusError.title")}</p>
        <p className="text-sm text-muted-foreground">
          {error instanceof Error ? error.message : t("app.statusError.description")}
        </p>
        <Button onClick={() => refetch()}>{t("app.statusError.retry")}</Button>
      </div>
    );
  }

  const installed = data?.installed;

  return (
    <>
      <NavigationProgress />
      <SiteMetaWatcher active={Boolean(installed)} />
      <NoIndexMetaWatcher />
      <Routes>
        <Route
          path="/installer"
          element={
            // 安装流程结束后 /installer 不再跳转，直接展示 404 页面
            installed ? <NotFoundPage /> : <InstallerPage />
          }
        />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
        <Route path="/terms" element={<TermsPage />} />
        <Route path="/privacy" element={<PrivacyPage />} />
        {installed && <Route path="/u/:id" element={<PublicUserPage />} />}
        {installed && (
          <Route
            path="/shop"
            element={
              <HiddenRoute url="/shop">
                <PublicShopPage />
              </HiddenRoute>
            }
          />
        )}
        {installed && <Route path="/" element={<HomeEntry />} />}
        {installed && (
          <Route element={<ProtectedRoute />}>
            <Route path="/dashboard/*" element={<SearchProvider><AppShell /></SearchProvider>}>
              <Route index element={<DashboardPage />} />
              <Route path="upload" element={<UploadPage />} />
              <Route path="images" element={<MyImagesPage />} />
              <Route path="shop" element={<HiddenRoute url="/dashboard/shop"><ShopPage /></HiddenRoute>} />
              <Route path="orders" element={<HiddenRoute url="/dashboard/orders"><OrdersPage /></HiddenRoute>} />
              <Route path="tickets" element={<HiddenRoute url="/dashboard/tickets"><TicketsPage /></HiddenRoute>} />
              <Route path="tickets/new" element={<HiddenRoute url="/dashboard/tickets"><TicketCreatePage /></HiddenRoute>} />
              <Route path="tickets/:id" element={<HiddenRoute url="/dashboard/tickets"><TicketDetailPage /></HiddenRoute>} />
              <Route path="settings" element={<ProfileSettingsPage />} />
              <Route path="notifications" element={<HiddenRoute url="/dashboard/notifications"><NotificationsPage /></HiddenRoute>} />
              <Route path="gallery" element={<HiddenRoute url="/dashboard/gallery"><GalleryPage /></HiddenRoute>} />
              <Route path="api" element={<ApiDocsPage />} />
              <Route path="api-tokens" element={<ApiTokensPage />} />
              <Route path="api-tokens/new" element={<ApiTokenEditorPage />} />
              <Route path="api-tokens/:id" element={<ApiTokenEditorPage />} />
              <Route path="about" element={<AboutPage />} />

              <Route element={<AdminRoute />}>
                <Route path="admin" element={<Navigate to="admin/console" replace />} />
                <Route path="admin/console" element={<AdminConsolePage />} />
                <Route path="admin/groups" element={<AdminGroupsPage />} />
                <Route path="admin/groups/new" element={<AdminGroupEditorPage />} />
                <Route path="admin/groups/:id" element={<AdminGroupEditorPage />} />
                <Route path="admin/redeem-codes" element={<HiddenRoute url="/dashboard/admin/redeem-codes"><AdminRedeemCodesPage /></HiddenRoute>} />
                <Route path="admin/redeem-codes/new" element={<HiddenRoute url="/dashboard/admin/redeem-codes"><AdminRedeemCodeEditorPage /></HiddenRoute>} />
                <Route path="admin/shop/products" element={<HiddenRoute url="/dashboard/admin/shop/products"><AdminShopProductsPage /></HiddenRoute>} />
                <Route path="admin/shop/products/new" element={<HiddenRoute url="/dashboard/admin/shop/products"><AdminShopProductEditorPage /></HiddenRoute>} />
                <Route path="admin/shop/products/:id" element={<HiddenRoute url="/dashboard/admin/shop/products"><AdminShopProductEditorPage /></HiddenRoute>} />
                <Route path="admin/shop/orders" element={<HiddenRoute url="/dashboard/admin/shop/orders"><AdminShopOrdersPage /></HiddenRoute>} />
                <Route path="admin/tickets" element={<HiddenRoute url="/dashboard/admin/tickets"><AdminTicketsPage /></HiddenRoute>} />
                <Route path="admin/tickets/:id" element={<HiddenRoute url="/dashboard/admin/tickets"><AdminTicketDetailPage /></HiddenRoute>} />
                <Route path="admin/users" element={<UserManagementPage />} />
                <Route path="admin/users/new" element={<AdminUserCreatePage />} />
                <Route path="admin/users/:id" element={<AdminUserDetailPage />} />
                <Route path="admin/images" element={<AdminImagesPage />} />
                <Route path="admin/audits" element={<HiddenRoute url="/dashboard/admin/audits"><AdminAuditsPage /></HiddenRoute>} />
                <Route path="admin/audits/new" element={<HiddenRoute url="/dashboard/admin/audits"><AdminAuditEditorPage /></HiddenRoute>} />
                <Route path="admin/audits/:id" element={<HiddenRoute url="/dashboard/admin/audits"><AdminAuditEditorPage /></HiddenRoute>} />
                <Route path="admin/strategies" element={<AdminStrategiesPage />} />
                <Route path="admin/strategies/new" element={<AdminStrategyEditorPage />} />
                <Route path="admin/strategies/:id" element={<AdminStrategyEditorPage />} />
                <Route path="admin/settings" element={<Navigate to="admin/settings/site" replace />} />
                <Route path="admin/settings/site" element={<AdminSiteSettingsPage />} />
                <Route path="admin/settings/email" element={<AdminEmailSettingsPage />} />
                <Route path="admin/settings/system" element={<AdminSystemSettingsPage />} />
                <Route path="admin/settings/tickets" element={<AdminTicketSettingsPage />} />
                <Route path="admin/settings/captcha" element={<AdminCaptchaSettingsPage />} />
                <Route path="admin/settings/oauth" element={<AdminOAuthSettingsPage />} />
                <Route path="admin/settings/payment" element={<AdminPaymentSettingsPage />} />
                <Route path="admin/settings/database" element={<AdminDatabaseSettingsPage />} />
              </Route>
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Route>
        )}
        <Route
          path="*"
          element={
            installed ? (
              <NotFoundPage />
            ) : (
              <Navigate to="/installer" />
            )
          }
        />
      </Routes>
    </>
  );
}
