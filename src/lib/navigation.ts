import {
  Activity,
  Bell,
  Brush,
  CloudUpload,
  GaugeCircle,
  Image as ImageIcon,
  Info,
  Key,
  Layers3,
  LinkIcon,
  Package,
  Receipt,
  ServerCog,
  Settings2,
  ShieldAlert,
  ShoppingBag,
  LifeBuoy,
  Ticket,
  Users,
  Users2,
  type LucideIcon
} from "lucide-react";
import type { SiteConfig } from "@/lib/api";

type Translate = (key: string) => string;

export type NavNode = {
  title: string;
  url?: string;
  icon?: LucideIcon;
  items?: NavNode[];
};

export type NavSection = {
  title?: string;
  items: NavNode[];
};

// normalizeNavUrl 规整 URL：去掉首尾空格与结尾斜杠，便于隐藏项的精确匹配。
export function normalizeNavUrl(url: string): string {
  const trimmed = url.trim();
  return trimmed.length > 1 ? trimmed.replace(/\/+$/, "") : trimmed;
}

export type SidebarItemConfig = {
  url: string;
  /** 菜单显示标题的 i18n key，例如 "nav.upload" */
  nameKey: string;
  /** 分组块标题的 i18n key，例如 "nav.mine" */
  groupKey: string;
  /** 关键页面不可隐藏时置为 true */
  critical?: boolean;
};

// 可隐藏的侧边栏项统一注册表：同时供侧边栏过滤、路由 404 守卫、系统设置页开关使用。
export const HIDEABLE_SIDEBAR_ITEMS: SidebarItemConfig[] = [
  // 首页
  { url: "/dashboard", nameKey: "nav.dashboard", groupKey: "nav.dashboard", critical: true },
  // 我的
  { url: "/dashboard/upload", nameKey: "nav.upload", groupKey: "nav.mine", critical: true },
  { url: "/dashboard/images", nameKey: "nav.images", groupKey: "nav.mine", critical: true },
  { url: "/dashboard/shop", nameKey: "nav.shop", groupKey: "nav.mine" },
  { url: "/dashboard/orders", nameKey: "nav.orders", groupKey: "nav.mine" },
  { url: "/dashboard/tickets", nameKey: "nav.tickets", groupKey: "nav.mine" },
  { url: "/dashboard/settings", nameKey: "nav.settings", groupKey: "nav.mine", critical: true },
  { url: "/dashboard/notifications", nameKey: "nav.notifications", groupKey: "nav.mine" },
  // 公开
  { url: "/shop", nameKey: "nav.shop", groupKey: "nav.public" },
  { url: "/dashboard/gallery", nameKey: "nav.gallery", groupKey: "nav.public" },
  { url: "/dashboard/api", nameKey: "nav.apiDocs", groupKey: "nav.public", critical: true },
  { url: "/dashboard/api-tokens", nameKey: "nav.apiTokens", groupKey: "nav.public", critical: true },
  { url: "/dashboard/about", nameKey: "nav.about", groupKey: "nav.public", critical: true },
  // 系统（管理员）
  { url: "/dashboard/admin/console", nameKey: "nav.console", groupKey: "nav.system", critical: true },
  { url: "/dashboard/admin/images", nameKey: "nav.adminImages", groupKey: "nav.system", critical: true },
  { url: "/dashboard/admin/audits", nameKey: "nav.audits", groupKey: "nav.system" },
  { url: "/dashboard/admin/groups", nameKey: "nav.groups", groupKey: "nav.system", critical: true },
  { url: "/dashboard/admin/redeem-codes", nameKey: "nav.redeemCodes", groupKey: "nav.system" },
  { url: "/dashboard/admin/shop/products", nameKey: "nav.shopProducts", groupKey: "nav.system" },
  { url: "/dashboard/admin/shop/orders", nameKey: "nav.shopOrders", groupKey: "nav.system" },
  { url: "/dashboard/admin/tickets", nameKey: "nav.adminTickets", groupKey: "nav.system" },
  { url: "/dashboard/admin/users", nameKey: "nav.users", groupKey: "nav.system", critical: true },
  { url: "/dashboard/admin/strategies", nameKey: "nav.strategies", groupKey: "nav.system", critical: true },
  // 系统设置子项（关键，避免误锁设置入口）
  { url: "/dashboard/admin/settings/site", nameKey: "nav.siteSettings", groupKey: "nav.systemSettings", critical: true },
  { url: "/dashboard/admin/settings/email", nameKey: "nav.emailSettings", groupKey: "nav.systemSettings", critical: true },
  { url: "/dashboard/admin/settings/system", nameKey: "nav.systemSettings", groupKey: "nav.systemSettings", critical: true },
  { url: "/dashboard/admin/settings/tickets", nameKey: "nav.ticketSettings", groupKey: "nav.systemSettings", critical: true },
  { url: "/dashboard/admin/settings/captcha", nameKey: "nav.captchaSettings", groupKey: "nav.systemSettings", critical: true },
  { url: "/dashboard/admin/settings/oauth", nameKey: "nav.oauthSettings", groupKey: "nav.systemSettings", critical: true },
  { url: "/dashboard/admin/settings/payment", nameKey: "nav.paymentSettings", groupKey: "nav.systemSettings", critical: true },
  { url: "/dashboard/admin/settings/database", nameKey: "nav.databaseSettings", groupKey: "nav.systemSettings", critical: true },
];

// 关键页面的 URL 集合：这些页面即使出现在持久化配置中也会被忽略，确保永不隐藏。
const CRITICAL_SIDEBAR_URLS = new Set(
  HIDEABLE_SIDEBAR_ITEMS.filter((item) => item.critical).map((item) =>
    normalizeNavUrl(item.url)
  )
);

// getHiddenSidebarUrls 返回真正生效的被隐藏 URL（剔除关键页面），供侧边栏过滤与路由守卫共用。
export function getHiddenSidebarUrls(siteConfig?: SiteConfig): Set<string> {
  const hidden = new Set<string>();
  for (const url of siteConfig?.hiddenSidebarItems ?? []) {
    const norm = normalizeNavUrl(url);
    if (!CRITICAL_SIDEBAR_URLS.has(norm)) {
      hidden.add(norm);
    }
  }
  return hidden;
}

// isNavVisible 依据站点配置判断侧边栏项是否可见。
function isNavVisible(hidden: Set<string>, url?: string): boolean {
  if (!url) {
    return true;
  }
  return !hidden.has(normalizeNavUrl(url));
}

export const buildNavSections = ({
  t,
  isAdmin,
  siteConfig
}: {
  t: Translate;
  isAdmin: boolean;
  siteConfig?: SiteConfig;
}): NavSection[] => {
  const enableGallery = siteConfig?.enableGallery ?? true;
  const enableApi = siteConfig?.enableApi ?? true;
  const hidden = getHiddenSidebarUrls(siteConfig);

  const sections: NavSection[] = [
    {
      items: [{ url: "/dashboard", title: t("nav.dashboard"), icon: GaugeCircle }].filter(
        (item) => isNavVisible(hidden, item.url)
      )
    },
    {
      title: t("nav.mine"),
      items: [
        { url: "/dashboard/upload", title: t("nav.upload"), icon: CloudUpload },
        { url: "/dashboard/images", title: t("nav.images"), icon: ImageIcon },
        { url: "/dashboard/shop", title: t("nav.shop"), icon: ShoppingBag },
        { url: "/dashboard/orders", title: t("nav.orders"), icon: Receipt },
        { url: "/dashboard/tickets", title: t("nav.tickets"), icon: LifeBuoy },
        { url: "/dashboard/settings", title: t("nav.settings"), icon: Settings2 },
        { url: "/dashboard/notifications", title: t("nav.notifications"), icon: Bell }
      ].filter((item) => isNavVisible(hidden, item.url))
    },
    {
      title: t("nav.public"),
      items: [
        { url: "/shop", title: t("nav.shop"), icon: ShoppingBag },
        ...(enableGallery
          ? [{ url: "/dashboard/gallery", title: t("nav.gallery"), icon: Brush }]
          : []),
        ...(enableApi
          ? [
              { url: "/dashboard/api", title: t("nav.apiDocs"), icon: LinkIcon },
              { url: "/dashboard/api-tokens", title: t("nav.apiTokens"), icon: Key }
            ]
          : []),
        { url: "/dashboard/about", title: t("nav.about"), icon: Info }
      ].filter((item) => isNavVisible(hidden, item.url))
    }
  ];

  if (isAdmin) {
    sections.push({
      title: t("nav.system"),
      items: [
        { url: "/dashboard/admin/console", title: t("nav.console"), icon: Activity },
        { url: "/dashboard/admin/images", title: t("nav.adminImages"), icon: ImageIcon },
        { url: "/dashboard/admin/audits", title: t("nav.audits"), icon: ShieldAlert },
        { url: "/dashboard/admin/groups", title: t("nav.groups"), icon: Users },
        { url: "/dashboard/admin/redeem-codes", title: t("nav.redeemCodes"), icon: Ticket },
        { url: "/dashboard/admin/shop/products", title: t("nav.shopProducts"), icon: Package },
        { url: "/dashboard/admin/shop/orders", title: t("nav.shopOrders"), icon: Receipt },
        { url: "/dashboard/admin/tickets", title: t("nav.adminTickets"), icon: LifeBuoy },
        { url: "/dashboard/admin/users", title: t("nav.users"), icon: Users2 },
        { url: "/dashboard/admin/strategies", title: t("nav.strategies"), icon: Layers3 },
        {
          title: t("nav.systemSettings"),
          icon: ServerCog,
          items: [
            { url: "/dashboard/admin/settings/site", title: t("nav.siteSettings") },
            { url: "/dashboard/admin/settings/email", title: t("nav.emailSettings") },
            { url: "/dashboard/admin/settings/system", title: t("nav.systemSettings") },
            { url: "/dashboard/admin/settings/tickets", title: t("nav.ticketSettings") },
            { url: "/dashboard/admin/settings/captcha", title: t("nav.captchaSettings") },
            { url: "/dashboard/admin/settings/oauth", title: t("nav.oauthSettings") },
            { url: "/dashboard/admin/settings/payment", title: t("nav.paymentSettings") },
            { url: "/dashboard/admin/settings/database", title: t("nav.databaseSettings") }
          ].filter((item) => isNavVisible(hidden, item.url))
        }
      ].filter(
        (item) => (item.url ? isNavVisible(hidden, item.url) : true)
      )
    });
  }

  return sections;
};