import {
  Activity,
  Bell,
  Brush,
  ChevronRight,
  ChevronsUpDown,
  CloudUpload,
  GaugeCircle,
  Home,
  Image as ImageIcon,
  Info,
  Key,
  Languages,
  Layers3,
  LifeBuoy,
  Link as LinkIcon,
  Monitor,
  Package,
  PanelLeft,
  Receipt,
  RefreshCw,
  Search,
  ServerCog,
  Settings,
  Settings2,
  ShieldAlert,
  ShoppingBag,
  Ticket,
  Users,
  Users2,
  type LucideIcon
} from "lucide-react";

import { PREVIEW_ROWS, SITE } from "@/data/site";
import { useI18n } from "@/i18n";

// 本组件是对后台控制台的静态还原，所有尺寸与类名都照抄真实源码：
//   - 骨架与顶栏：src/layouts/AppShell.tsx（默认 inset 变体）
//   - 侧边栏条目：src/lib/navigation.ts 的 buildNavSections + src/components/layout/NavGroup.tsx
//   - 尺寸令牌：src/components/ui/sidebar.tsx（16rem / h-8 / p-2 / gap-1）
//   - 容量条与账号行：src/components/CapacityMeter.tsx、src/components/layout/NavUser.tsx
//   - 图库卡片：src/features/files/MyImagesPage.tsx、src/features/files/components/ImageGrid.tsx

type PreviewNavItem = {
  icon: LucideIcon;
  label: string;
  active?: boolean;
  /// 对应 NavGroup 里的可折叠项（系统设置），右侧带 ChevronRight
  submenu?: boolean;
};

type PreviewNavGroup = {
  /// null 表示无标题分组（后台第一个分组）
  label: string | null;
  items: PreviewNavItem[];
};

export function ProductPreview() {
  const { copy } = useI18n();
  const nav = copy.preview.nav;

  const navGroups: PreviewNavGroup[] = [
    {
      label: null,
      items: [{ icon: GaugeCircle, label: nav.dashboard }]
    },
    {
      label: copy.preview.groups.mine,
      items: [
        { icon: CloudUpload, label: nav.upload },
        { icon: ImageIcon, label: nav.images, active: true },
        { icon: ShoppingBag, label: nav.shop },
        { icon: Receipt, label: nav.orders },
        { icon: LifeBuoy, label: nav.tickets },
        { icon: Settings2, label: nav.settings },
        { icon: Bell, label: nav.notifications }
      ]
    },
    {
      label: copy.preview.groups.public,
      items: [
        { icon: ShoppingBag, label: nav.shop },
        { icon: Brush, label: nav.gallery },
        { icon: LinkIcon, label: nav.apiDocs },
        { icon: Key, label: nav.apiTokens },
        { icon: Info, label: nav.about }
      ]
    },
    {
      label: copy.preview.groups.system,
      items: [
        { icon: Activity, label: nav.console },
        { icon: ImageIcon, label: nav.adminImages },
        { icon: ShieldAlert, label: nav.audits },
        { icon: Users, label: nav.groups },
        { icon: Ticket, label: nav.redeemCodes },
        { icon: Package, label: nav.shopProducts },
        { icon: Receipt, label: nav.shopOrders },
        { icon: LifeBuoy, label: nav.adminTickets },
        { icon: Users2, label: nav.users },
        { icon: Layers3, label: nav.strategies },
        { icon: ServerCog, label: nav.systemSettings, submenu: true }
      ]
    }
  ];

  return (
    <section className="container mx-auto max-w-6xl px-4 pt-4 sm:px-8">
      <div className="animate-enter animate-enter-2 overflow-hidden rounded-2xl border bg-card shadow-[0_24px_60px_-32px_hsl(var(--foreground)/0.28)]">
        {/* 浏览器窗框 */}
        <div className="flex items-center gap-2 border-b bg-muted px-3.5 py-2.5">
          <div className="flex gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-muted-foreground/35" />
            <span className="h-2.5 w-2.5 rounded-full bg-muted-foreground/35" />
            <span className="h-2.5 w-2.5 rounded-full bg-muted-foreground/35" />
          </div>
          <span className="ml-2 truncate font-mono text-xs text-muted-foreground">
            {SITE.demo.replace(/^https?:\/\//, "")}/dashboard/images
          </span>
        </div>

        {/* 默认 inset 变体：整体底色 bg-sidebar，侧边栏与主区各是一张浮起的卡片。
            高度写在两侧列上（而不是 grid 上），这样 `h-full` 才有确定高度可解析——
            否则侧边栏底部会被内容顶出可视区。 */}
        <div className="grid bg-sidebar md:grid-cols-[16rem_1fr]">
          {/* ---------- 侧边栏 ---------- */}
          <aside className="hidden p-2 md:flex md:h-[44.5rem]">
            <div className="flex h-full min-h-0 w-full flex-col overflow-hidden rounded-lg border border-sidebar-border bg-sidebar shadow-sm">
              {/* SidebarHeader：站名 + 站点描述，无 logo */}
              <div className="flex shrink-0 flex-col gap-2 p-2">
                <div className="flex flex-col gap-0.5 px-2 py-1">
                  <p className="text-lg font-semibold">{SITE.name}</p>
                  <p className="text-sm text-muted-foreground">{copy.preview.siteDesc}</p>
                </div>
              </div>

              {/* SidebarContent：真实侧边栏此处可滚动，预览里裁切并做淡出 */}
              <div
                className="flex min-h-0 flex-1 flex-col gap-2 overflow-hidden"
                style={{
                  maskImage: "linear-gradient(to bottom, black 88%, transparent 100%)",
                  WebkitMaskImage: "linear-gradient(to bottom, black 88%, transparent 100%)"
                }}
              >
                {navGroups.map((group) => (
                  <div
                    key={group.label ?? "group-0"}
                    className="relative flex w-full min-w-0 flex-col p-2"
                  >
                    {group.label ? (
                      <div className="flex h-8 shrink-0 items-center rounded-md px-2 text-xs font-medium text-sidebar-foreground/70">
                        {group.label}
                      </div>
                    ) : null}
                    <ul className="flex w-full min-w-0 flex-col gap-1">
                      {group.items.map((item) => (
                        <li key={`${group.label}-${item.label}`} className="group/menu-item relative">
                          {/* 预览里的条目只做视觉反馈：悬停/按下都要有响应，点击不跳转 */}
                          <div
                            className={[
                              "press flex h-8 w-full cursor-pointer select-none items-center gap-2 overflow-hidden rounded-md p-2 text-start text-sm",
                              item.active
                                ? "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
                                : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                            ].join(" ")}
                          >
                            <item.icon className="size-4 shrink-0" />
                            <span className="truncate">{item.label}</span>
                            {item.submenu ? (
                              <ChevronRight className="ms-auto size-4 shrink-0" />
                            ) : null}
                          </div>
                        </li>
                      ))}
                    </ul>
                  </div>
                ))}
              </div>

              {/* SidebarFooter：容量条 + 账号行 */}
              <div className="flex shrink-0 flex-col gap-2 p-2">
                <div className="space-y-2 rounded-lg border p-3">
                  <div className="flex items-center justify-between">
                    <p className="text-sm text-muted-foreground">{copy.preview.capacityLabel}</p>
                    <span className="press-icon -m-1 inline-flex cursor-pointer items-center justify-center rounded p-1 text-muted-foreground">
                      <RefreshCw className="h-3.5 w-3.5" />
                    </span>
                  </div>
                  <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-primary"
                      style={{ width: "24.8%" }}
                    />
                  </div>
                  <p className="text-sm text-muted-foreground">
                    <span className="text-foreground">{copy.preview.capacityUsed}</span> /{" "}
                    {copy.preview.capacityTotal}
                  </p>
                </div>

                <div className="press flex h-12 w-full cursor-pointer select-none items-center gap-2 overflow-hidden rounded-md p-2 text-start text-sm hover:bg-sidebar-accent">
                  <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-xs font-medium">
                    {copy.preview.account.initials}
                  </span>
                  <span className="grid flex-1 text-start text-sm leading-tight">
                    <span className="truncate font-semibold">{copy.preview.account.name}</span>
                    <span className="truncate text-xs">{copy.preview.account.email}</span>
                  </span>
                  <ChevronsUpDown className="ms-auto size-4 shrink-0" />
                </div>
              </div>
            </div>
          </aside>

          {/* ---------- 主区（inset：m-2 + rounded-xl） ---------- */}
          <div className="flex min-h-0 min-w-0 flex-col md:h-[44.5rem] md:py-2 md:pr-2">
            <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden md:rounded-xl md:bg-background md:shadow-sm">
              {/* 顶栏：折叠按钮 / 居中搜索 / 首页·仪表盘 + 语言·主题·设置 */}
              <header className="flex h-14 shrink-0 items-center justify-between border-b border-border/60 px-3 sm:px-4">
                <div className="flex items-center gap-1">
                  <span className="press-icon inline-flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-md bg-sidebar text-sidebar-foreground">
                    <PanelLeft className="h-4 w-4" />
                  </span>
                </div>

                <div className="mx-4 hidden w-full max-w-sm md:block">
                  <div className="press relative flex h-8 w-full cursor-pointer items-center rounded-md border bg-muted/25 pl-8 pr-12 text-sm text-muted-foreground">
                    <Search
                      aria-hidden="true"
                      className="absolute left-1.5 top-1/2 size-4 -translate-y-1/2"
                    />
                    <span className="truncate">{copy.preview.search}</span>
                    <kbd className="absolute right-1.5 top-1.5 hidden h-5 items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium sm:flex">
                      <span className="text-xs">⌘</span>K
                    </kbd>
                  </div>
                </div>

                <div className="ml-auto flex items-center gap-2">
                  <nav className="flex items-center gap-1">
                    <TopbarButton icon={Home} label={copy.preview.home} />
                    <TopbarButton icon={GaugeCircle} label={nav.dashboard} />
                  </nav>
                  <IconButton icon={Languages} />
                  <IconButton icon={Monitor} />
                  <IconButton icon={Settings} />
                </div>
              </header>

              {/* 主区内容：我的图片 */}
              <div className="min-h-0 flex-1 overflow-hidden p-8">
                <div className="space-y-6">
                  <div>
                    <h1 className="text-2xl font-semibold">{copy.preview.pageTitle}</h1>
                    <p className="text-muted-foreground">{copy.preview.pageDesc}</p>
                  </div>

                  {/* ImageGrid：ROW_HEIGHT 240 + GAP 16，行内按宽高比等比撑满 */}
                  <div className="flex w-full flex-col" style={{ gap: 16 }}>
                    {PREVIEW_ROWS.map((row, rowIndex) => (
                      <div
                        key={rowIndex}
                        className="flex w-full"
                        style={{ gap: 16, height: 240 }}
                      >
                        {row.map((image) => (
                          <div
                            key={image.name}
                            className="press relative min-w-0 cursor-pointer select-none overflow-hidden rounded-xl border bg-muted/30 text-left shadow-sm"
                            style={{ flexGrow: image.ratio, flexBasis: 0 }}
                          >
                            <img
                              src={image.src}
                              alt=""
                              className="h-full w-full object-cover"
                            />
                            <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/60 via-black/10 to-transparent opacity-80" />

                            <span className="press-icon absolute left-3 top-3 inline-flex h-5 w-5 cursor-pointer items-center justify-center rounded border border-white/50 bg-black/30 text-[11px] text-white/80" />
                            <span className="press absolute right-3 top-3 inline-flex cursor-pointer items-center rounded-md border border-white/40 bg-black/35 px-2 py-0.5 text-xs text-white/80 backdrop-blur">
                              •••
                            </span>

                            <div className="pointer-events-none absolute inset-x-0 bottom-0 p-3">
                              <div className="flex items-center gap-2 text-white">
                                <p className="min-w-0 flex-1 truncate text-sm font-semibold">
                                  {image.name}
                                </p>
                                <span className="shrink-0 rounded-full bg-black/50 px-2 py-0.5 text-[11px]">
                                  {image.visibility === "public"
                                    ? copy.preview.visibility.public
                                    : copy.preview.visibility.private}
                                </span>
                              </div>
                              <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-white/80">
                                <span>{image.size}</span>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

/// 顶栏文字按钮：对应 AppShell 里的 Button variant="ghost" size="sm" h-8 gap-1.5 px-2
function TopbarButton({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
  return (
    <span className="press inline-flex h-8 shrink-0 cursor-pointer select-none items-center gap-1.5 rounded-md px-2 text-xs text-muted-foreground hover:bg-accent hover:text-accent-foreground">
      <Icon className="size-4 shrink-0" />
      <span className="hidden lg:inline">{label}</span>
    </span>
  );
}

/// 顶栏图标按钮：对应独立的 LanguageToggle / ThemeToggle / ConfigDrawer 触发器
function IconButton({ icon: Icon }: { icon: LucideIcon }) {
  return (
    <span className="press-icon inline-flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground">
      <Icon className="size-4" />
    </span>
  );
}
