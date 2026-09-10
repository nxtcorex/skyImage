import {
  Bell,
  Code2,
  Database,
  GaugeCircle,
  Globe,
  HardDrive,
  Images,
  KeyRound,
  Layers,
  Link2,
  MonitorSmartphone,
  Server,
  ShieldCheck,
  ShoppingCart,
  Ticket,
  UploadCloud,
  Users,
  type LucideIcon
} from "lucide-react";

// 文案全部集中在 src/i18n.tsx。本文件只放「与语言无关」的数据：
// 链接、图标、专有名词列表、演示账号。
// 注意：图标数组的顺序必须与 i18n.tsx 中对应文案数组一一对应。

export const SITE = {
  name: "SkyImage",
  github: "https://github.com/fishcpy/skyImage",
  demo: "https://skyimage.demo.123123223.xyz",
  license: "AGPL-3.0"
};

/// 演示站账号，与 README「演示站」一节一致
export const DEMO_ACCOUNTS = [
  { role: "admin", email: "demo@example.com", password: "adminpassword" },
  { role: "user", email: "user@example.com", password: "userpassword" }
] as const;

/// 网站备案信息（工信部 ICP 备案 + 公安部联网备案），与语言无关，固定显示中文
export const FILINGS = [
  {
    label: "京ICP备2025138063号-2",
    href: "https://beian.miit.gov.cn/"
  },
  {
    label: "京公网安备11011402056544号",
    href: "https://beian.mps.gov.cn/#/query/webSearch?code=11011402056544"
  }
] as const;

/// 顺序对应 copy.features.items
export const FEATURE_ICONS: LucideIcon[] = [
  HardDrive,
  Users,
  Database,
  Images,
  Code2,
  KeyRound,
  ShieldCheck,
  Ticket
];

/// 顺序对应 copy.steps.items
export const STEP_ICONS: LucideIcon[] = [UploadCloud, Link2, Layers];

/// 顺序对应 copy.scenes.items
export const SCENE_ICONS: LucideIcon[] = [Users, Code2, Globe];

/// 顺序对应 copy.deploy.points
export const DEPLOY_POINT_ICONS: LucideIcon[] = [GaugeCircle, Bell, ShoppingCart];

/// 顺序对应 copy.stack.groups
export const STACK_GROUP_ICONS: LucideIcon[] = [Server, MonitorSmartphone, Database];

/// 以下为专有名词，无需翻译
export const STACK_BACKEND = ["Go 1.24+", "Gin", "GORM", "Viper", "Cookie + Session"];
export const STACK_FRONTEND = ["React 18", "TypeScript", "Vite", "Tailwind CSS", "Radix UI", "Zustand"];
export const STACK_DATABASE = ["SQLite", "MySQL", "PostgreSQL"];

/// 界面预览里的图库卡片数据
export type PreviewImage = {
  src: string;
  /// 宽高比：按真实图库的算法等比撑满一行
  ratio: number;
  name: string;
  /// 与后台一致：ImageGrid 卡片渲染的是 `(size / 1024).toFixed(1) + " KB"`
  size: string;
  visibility: "public" | "private";
};

// 缩略图用内联 SVG 占位，纯灰阶——不引入任何彩色，发布时无需额外请求。
const svgUri = (markup: string) =>
  `data:image/svg+xml,${encodeURIComponent(markup)}`;

// 行内条目按后台算法（ROW_HEIGHT=240、GAP=16、容器宽 ≈832px）贪心换行的结果排列：
// 第一行 [16:9, 3:2]，第二行 [3:4, 1:1, 4:3]。改宽高比前请先复核换行是否仍成立。
export const PREVIEW_ROWS: PreviewImage[][] = [
  [
    {
      src: svgUri(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 480 270"><defs><linearGradient id="a" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#525252"/><stop offset="0.6" stop-color="#a3a3a3"/><stop offset="1" stop-color="#404040"/></linearGradient></defs><rect width="480" height="270" fill="url(#a)"/><circle cx="374" cy="64" r="21" fill="#fafafa" opacity="0.7"/><path d="M0 194 96 130l72 54 100-88 92 82 120-58v150H0z" fill="#262626"/><path d="M0 232 120 186l112 50 112-56 136 46v44H0z" fill="#171717"/></svg>'
      ),
      ratio: 16 / 9,
      name: "cover-2026.webp",
      size: "1240.6 KB",
      visibility: "public"
    },
    {
      src: svgUri(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 450 300"><rect width="450" height="300" fill="#8a8a8a"/><rect x="56" y="38" width="148" height="224" rx="6" fill="#d4d4d4"/><rect x="236" y="38" width="148" height="224" rx="6" fill="#5c5c5c"/><rect y="248" width="450" height="52" fill="#6e6e6e"/></svg>'
      ),
      ratio: 3 / 2,
      name: "studio-light.jpg",
      size: "864.3 KB",
      visibility: "public"
    }
  ],
  [
    {
      src: svgUri(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 400"><defs><radialGradient id="c" cx="0.5" cy="0.28" r="0.78"><stop offset="0" stop-color="#e5e5e5"/><stop offset="1" stop-color="#171717"/></radialGradient></defs><rect width="300" height="400" fill="url(#c)"/><circle cx="150" cy="156" r="56" fill="#404040"/><path d="M54 400c0-86 43-126 96-126s96 40 96 126z" fill="#404040"/></svg>'
      ),
      ratio: 3 / 4,
      name: "portrait-final.png",
      size: "2058.2 KB",
      visibility: "private"
    },
    {
      src: svgUri(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 300"><rect width="300" height="300" fill="#a3a3a3"/><path d="M0 208c70-40 110 40 180 0 60-34 90 4 120-12v104H0z" fill="#c9c9c9"/><path d="M0 244c80-34 120 32 200-4 52-24 76 6 100-4v64H0z" fill="#7a7a7a"/><path d="M0 276c90-24 140 22 220-2 44-14 64 4 80-2v28H0z" fill="#5c5c5c"/></svg>'
      ),
      ratio: 1,
      name: "dune-study.webp",
      size: "742.9 KB",
      visibility: "public"
    },
    {
      src: svgUri(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 300"><defs><radialGradient id="d" cx="0.34" cy="0.28" r="0.82"><stop offset="0" stop-color="#fafafa"/><stop offset="0.45" stop-color="#737373"/><stop offset="1" stop-color="#171717"/></radialGradient></defs><rect width="400" height="300" fill="#111111"/><ellipse cx="200" cy="262" rx="126" ry="14" fill="#000" opacity="0.55"/><circle cx="200" cy="146" r="98" fill="url(#d)"/></svg>'
      ),
      ratio: 4 / 3,
      name: "render-orb.png",
      size: "1904.4 KB",
      visibility: "public"
    }
  ]
];
