import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

export type Locale = "zh-CN" | "en";

// 与主项目 src/i18n.tsx 共用同一个 localStorage key，保持语言偏好一致
const LANGUAGE_STORAGE_KEY = "skyimage-language";

export type DemoRole = "admin" | "user";

export type SiteCopy = {
  lang: {
    switcher: string;
    zh: string;
    en: string;
  };
  meta: {
    title: string;
    description: string;
  };
  tagline: string;
  nav: {
    items: { label: string; href: string }[];
    github: string;
    demo: string;
  };
  hero: {
    slogan: string;
    intro: string;
    demo: string;
    source: string;
    stats: { value: string; label: string }[];
  };
  demo: {
    title: string;
    desc: string;
    emailLabel: string;
    passwordLabel: string;
    copy: string;
    copied: string;
    open: string;
    hint: string;
    roles: Record<DemoRole, string>;
  };
  preview: {
    /// 侧边栏里的站点描述（演示站 site.description）
    siteDesc: string;
    /// 分组标题，对应主项目 nav.mine / nav.public / nav.system
    groups: { mine: string; public: string; system: string };
    /// 侧边栏条目，逐条对应主项目 src/lib/navigation.ts 的 buildNavSections
    nav: {
      dashboard: string;
      upload: string;
      images: string;
      shop: string;
      orders: string;
      tickets: string;
      settings: string;
      notifications: string;
      gallery: string;
      apiDocs: string;
      apiTokens: string;
      about: string;
      console: string;
      adminImages: string;
      audits: string;
      groups: string;
      redeemCodes: string;
      shopProducts: string;
      shopOrders: string;
      adminTickets: string;
      users: string;
      strategies: string;
      systemSettings: string;
    };
    /// 顶栏
    search: string;
    home: string;
    /// 主区
    pageTitle: string;
    pageDesc: string;
    visibility: { public: string; private: string };
    /// 侧边栏底部
    capacityLabel: string;
    capacityUsed: string;
    capacityTotal: string;
    account: { name: string; initials: string; email: string };
  };
  features: {
    title: string;
    desc: string;
    items: { title: string; desc: string }[];
  };
  steps: {
    title: string;
    desc: string;
    items: { title: string; desc: string }[];
  };
  scenes: {
    title: string;
    desc: string;
    items: { text: string }[];
  };
  storage: {
    title: string;
    desc: string;
    supported: string;
    untested: string;
    untestedNote: string;
    supportedItems: string[];
    untestedItems: string[];
  };
  stack: {
    title: string;
    desc: string;
    groups: string[];
  };
  deploy: {
    title: string;
    desc: string;
    quickStart: string;
    docker: string;
    binary: string;
    note: string;
    points: { title: string; desc: string }[];
    commands: { docker: { text: string; type: "cmd" | "cmt" }[]; binary: { text: string; type: "cmd" | "cmt" }[] };
  };
  migration: {
    badge: string;
    title: string;
    desc: string;
    cta: string;
    points: string[];
  };
  cta: {
    title: string;
    desc: string;
    demo: string;
    github: string;
  };
  footer: {
    copyright: string;
    github: string;
    license: string;
    demo: string;
  };
};

const zh: SiteCopy = {
  lang: {
    switcher: "语言",
    zh: "简体中文",
    en: "English"
  },
  meta: {
    title: "SkyImage · 现代化开源图床系统",
    description:
      "SkyImage —— 现代化开源图床系统。支持多存储策略、用户组权限、数据库跨库迁移与 Lsky Pro 一键导入。"
  },
  tagline: "现代化开源图床系统",
  nav: {
    items: [
      { label: "功能特性", href: "#features" },
      { label: "存储支持", href: "#storage" },
      { label: "技术栈", href: "#stack" },
      { label: "部署", href: "#deploy" },
      { label: "数据迁移", href: "#migration" }
    ],
    github: "GitHub 仓库",
    demo: "在线演示"
  },
  hero: {
    slogan: "把图片交给自己托管",
    intro:
      "SkyImage 是一个前后端分离的现代化图床系统。多存储策略、细粒度权限、数据库跨库迁移与 Lsky Pro 一键导入，全部开箱即用。",
    demo: "在线演示",
    source: "查看源码",
    stats: [
      { value: "8+", label: "存储驱动" },
      { value: "3", label: "数据库支持" },
      { value: "2", label: "部署方式" },
      { value: "AGPL", label: "开源协议" }
    ]
  },
  demo: {
    title: "演示账号",
    desc: "用下面任意一个账号即可登录演示站，两种角色的权限不同。",
    emailLabel: "邮箱",
    passwordLabel: "密码",
    copy: "复制",
    copied: "已复制",
    open: "打开演示站",
    hint: "演示站数据会定期清理，请勿上传重要文件。",
    roles: {
      admin: "管理员",
      user: "普通用户"
    }
  },
  preview: {
    siteDesc: "云端图床",
    groups: { mine: "我的", public: "公共", system: "系统" },
    nav: {
      dashboard: "仪表盘",
      upload: "上传图片",
      images: "我的图片",
      shop: "商店",
      orders: "我的订单",
      tickets: "工单",
      settings: "设置",
      notifications: "通知",
      gallery: "图片广场",
      apiDocs: "接口文档",
      apiTokens: "API Token",
      about: "关于",
      console: "控制台",
      adminImages: "图片管理",
      audits: "审核",
      groups: "角色组",
      redeemCodes: "兑换码",
      shopProducts: "商品管理",
      shopOrders: "商店订单",
      adminTickets: "工单管理",
      users: "用户管理",
      strategies: "储存策略",
      systemSettings: "系统设置"
    },
    search: "搜索菜单...",
    home: "首页",
    pageTitle: "我的图片",
    pageDesc: "查看、管理、删除你已经上传的所有内容。",
    visibility: { public: "公开", private: "私有" },
    capacityLabel: "容量使用",
    capacityUsed: "12.40 GB",
    capacityTotal: "50.00 GB",
    account: { name: "demo", initials: "D", email: "demo@example.com" }
  },
  features: {
    title: "功能特性",
    desc: "覆盖上传、存储、权限与运营的完整能力，装好即能用。",
    items: [
      {
        title: "多存储策略",
        desc: "本地、S3、OSS、COS、七牛、又拍、WebDAV、MinIO，按需选择并随时切换。"
      },
      {
        title: "用户组与权限",
        desc: "角色组、容量配额、上传权限精细化管控，适配个人到团队的不同场景。"
      },
      {
        title: "数据库跨库迁移",
        desc: "SQLite / MySQL / PostgreSQL 之间一键互迁，命令行与后台均可操作。"
      },
      {
        title: "缩略图与加载优化",
        desc: "上传即生成缩略图，图库加载更快，流量更省。"
      },
      {
        title: "开放 API 与令牌",
        desc: "完整的 REST API 与访问令牌，轻松对接第三方应用与自动化脚本。"
      },
      {
        title: "Passkey 与 OAuth",
        desc: "支持 Passkey 通行密钥免密登录，以及自定义 OAuth 账号绑定。"
      },
      {
        title: "内容安全",
        desc: "内置腾讯云图片审核与多种人机验证，自动拦截违规内容。"
      },
      {
        title: "工单与商店",
        desc: "内置工单系统与用户组商店，支持容量、角色组兑换码。"
      }
    ]
  },
  steps: {
    title: "三步开始使用",
    desc: "从上传到分享，操作路径足够短。",
    items: [
      { title: "上传图片", desc: "拖拽或粘贴即可上传，自动应用所选存储策略。" },
      { title: "获取链接", desc: "多种链接格式一键复制，支持直接对外访问。" },
      { title: "集中管理", desc: "批量选择、公开与私有切换、容量统计，已上传的内容随时检索。" }
    ]
  },
  scenes: {
    title: "适用场景",
    desc: "个人、团队与内容站点，都能找到合适的位置。",
    items: [
      { text: "个人图床：把博客与笔记里的图片收归自己，告别外部图床失效。" },
      { text: "团队素材库：为设计、运营团队提供统一的素材托管与权限管理。" },
      { text: "内容分发：配合对象存储与 CDN，为站点提供稳定的图片加速。" }
    ]
  },
  storage: {
    title: "存储支持",
    desc: "对象存储统一通过 S3 兼容驱动接入，换云商不用改代码。",
    supported: "已支持",
    untested: "未测试",
    untestedNote: "驱动已实现但尚未经过完整测试，欢迎在社区反馈使用情况。",
    supportedItems: ["本地存储", "AWS S3", "阿里云 OSS", "腾讯云 COS", "七牛云", "又拍云", "WebDAV", "MinIO"],
    untestedItems: ["FTP", "SFTP"]
  },
  stack: {
    title: "技术栈",
    desc: "前后端分离架构，现代化工具链，易于二次开发与部署。",
    groups: ["后端", "前端", "数据库"]
  },
  deploy: {
    title: "部署",
    desc: "推荐使用 Docker 部署；也提供各平台预编译二进制包。",
    quickStart: "快速开始",
    docker: "Docker",
    binary: "二进制",
    note: "支持 Linux / Windows / macOS，Docker 与二进制双通道",
    points: [
      {
        title: "一键部署",
        desc: "Docker Compose 三条命令启动，数据目录与配置自动持久化。"
      },
      {
        title: "多架构镜像",
        desc: "Docker Hub、GHCR、CNB 三仓库可选，覆盖国内外网络环境。"
      },
      {
        title: "开箱即用",
        desc: "内置安装向导，引导完成数据库与管理员配置，无需手工建表。"
      }
    ],
    commands: {
      docker: [
        { text: "mkdir skyimage && cd skyimage", type: "cmd" },
        {
          text: "curl -O https://raw.githubusercontent.com/nxtcorex/skyImage/refs/heads/main/docker-compose.yml",
          type: "cmd"
        },
        {
          text: "curl -o .env https://raw.githubusercontent.com/nxtcorex/skyImage/refs/heads/main/.env.example",
          type: "cmd"
        },
        { text: "", type: "cmd" },
        { text: "# 启动服务，访问 http://localhost:8080", type: "cmt" },
        { text: "docker-compose up -d", type: "cmd" }
      ],
      binary: [
        { text: "# 前往 GitHub Releases 下载对应平台预编译包", type: "cmt" },
        { text: "tar -xzf skyimage-*-linux-amd64.tar.gz", type: "cmd" },
        { text: "cp .env.example .env", type: "cmd" },
        { text: "", type: "cmd" },
        { text: "# 运行并完成安装向导", type: "cmt" },
        { text: "./skyimage", type: "cmd" }
      ]
    }
  },
  migration: {
    badge: "从 Lsky Pro 迁移",
    title: "旧站数据，一次导入",
    desc: "安装向导内置迁移工具，把 Lsky Pro 开源版（V2.x）的数据平滑搬到 SkyImage，图片链接路径保持不变。",
    cta: "查看迁移文档",
    points: [
      "用户、角色组、存储策略、相册与图片记录一并迁移",
      "密码哈希原样保留，老用户可用原密码直接登录",
      "导入过程只读源库，SQLite 以只读模式打开，绝不改动源数据",
      "填写旧站图片目录后按原路径拷贝文件，图片链接保持不变"
    ]
  },
  cta: {
    title: "现在就把图床搭起来",
    desc: "一条 Docker 命令即可启动，几分钟内完成安装向导。开源免费，数据握在自己手里。",
    demo: "体验演示站",
    github: "GitHub"
  },
  footer: {
    copyright: "© {year} SkyImage · 基于 {license} 协议开源",
    github: "GitHub",
    license: "许可证",
    demo: "演示站"
  }
};

const en: SiteCopy = {
  lang: {
    switcher: "Language",
    zh: "简体中文",
    en: "English"
  },
  meta: {
    title: "SkyImage · A modern open-source image hosting system",
    description:
      "SkyImage is a modern open-source image hosting system with multiple storage strategies, user group permissions, cross-database migration and one-click Lsky Pro import."
  },
  tagline: "Modern open-source image hosting",
  nav: {
    items: [
      { label: "Features", href: "#features" },
      { label: "Storage", href: "#storage" },
      { label: "Stack", href: "#stack" },
      { label: "Deploy", href: "#deploy" },
      { label: "Migration", href: "#migration" }
    ],
    github: "GitHub repository",
    demo: "Live Demo"
  },
  hero: {
    slogan: "Keep your image hosting in your own hands",
    intro:
      "SkyImage is a modern image hosting system with a decoupled frontend and backend. Multiple storage strategies, fine-grained permissions, cross-database migration and one-click Lsky Pro import — all ready out of the box.",
    demo: "Live Demo",
    source: "View source",
    stats: [
      { value: "8+", label: "Storage drivers" },
      { value: "3", label: "Databases" },
      { value: "2", label: "Deploy options" },
      { value: "AGPL", label: "License" }
    ]
  },
  demo: {
    title: "Demo accounts",
    desc: "Sign in to the live demo with either account below — the two roles have different permissions.",
    emailLabel: "Email",
    passwordLabel: "Password",
    copy: "Copy",
    copied: "Copied",
    open: "Open the demo",
    hint: "Demo data is reset periodically — please don't upload anything important.",
    roles: {
      admin: "Administrator",
      user: "Regular user"
    }
  },
  preview: {
    siteDesc: "Cloud image hosting",
    groups: { mine: "My Space", public: "Public", system: "System" },
    nav: {
      dashboard: "Dashboard",
      upload: "Upload",
      images: "My Images",
      shop: "Shop",
      orders: "Orders",
      tickets: "Tickets",
      settings: "Settings",
      notifications: "Notifications",
      gallery: "Image Square",
      apiDocs: "API Docs",
      apiTokens: "API Tokens",
      about: "About",
      console: "Console",
      adminImages: "Image Admin",
      audits: "Audits",
      groups: "Groups",
      redeemCodes: "Redeem codes",
      shopProducts: "Products",
      shopOrders: "Shop orders",
      adminTickets: "Ticket Admin",
      users: "Users",
      strategies: "Storage",
      systemSettings: "Settings"
    },
    search: "Search menu...",
    home: "Home",
    pageTitle: "My images",
    pageDesc: "View, manage, and delete everything you have uploaded.",
    visibility: { public: "Public", private: "Private" },
    capacityLabel: "Storage usage",
    capacityUsed: "12.40 GB",
    capacityTotal: "50.00 GB",
    account: { name: "demo", initials: "D", email: "demo@example.com" }
  },
  features: {
    title: "Features",
    desc: "Everything from upload and storage to permissions and operations — usable the moment it's installed.",
    items: [
      {
        title: "Multiple storage strategies",
        desc: "Local, S3, OSS, COS, Qiniu, Upyun, WebDAV and MinIO — pick what you need and switch at any time."
      },
      {
        title: "User groups & permissions",
        desc: "Role groups, storage quotas and fine-grained upload permissions, from solo use to full teams."
      },
      {
        title: "Cross-database migration",
        desc: "One-click migration between SQLite, MySQL and PostgreSQL, from the CLI or the admin panel."
      },
      {
        title: "Thumbnails & fast loading",
        desc: "Thumbnails are generated on upload, so galleries load faster and use less bandwidth."
      },
      {
        title: "Open API & tokens",
        desc: "A complete REST API and access tokens make it easy to integrate third-party apps and automation scripts."
      },
      {
        title: "Passkey & OAuth",
        desc: "Passwordless sign-in with passkeys, plus custom OAuth account binding."
      },
      {
        title: "Content safety",
        desc: "Built-in Tencent Cloud image moderation and multiple CAPTCHA options automatically block offending content."
      },
      {
        title: "Tickets & store",
        desc: "A built-in ticket system and group store, with redeem codes for storage and role groups."
      }
    ]
  },
  steps: {
    title: "Start in three steps",
    desc: "From upload to share — a path short enough.",
    items: [
      { title: "Upload images", desc: "Drag and drop or paste to upload; the selected storage strategy is applied automatically." },
      { title: "Get the link", desc: "Copy any of several link formats in one click, ready for public access." },
      { title: "Manage in one place", desc: "Batch selection, public/private toggles and storage statistics — find what you uploaded at any time." }
    ]
  },
  scenes: {
    title: "Use cases",
    desc: "Individuals, teams and content sites all find their fit.",
    items: [
      {
        text: "Personal hosting: keep the images in your blog and notes under your own control, and stop worrying about third-party hosts going down."
      },
      {
        text: "Team asset library: unified asset hosting and permission management for design and marketing teams."
      },
      {
        text: "Content delivery: pair object storage with a CDN for stable image acceleration."
      }
    ]
  },
  storage: {
    title: "Storage support",
    desc: "Object storage is integrated through S3-compatible drivers — switch providers without touching code.",
    supported: "Supported",
    untested: "Untested",
    untestedNote: "The driver is implemented but not fully tested yet — community feedback is welcome.",
    supportedItems: ["Local", "AWS S3", "Aliyun OSS", "Tencent COS", "Qiniu", "Upyun", "WebDAV", "MinIO"],
    untestedItems: ["FTP", "SFTP"]
  },
  stack: {
    title: "Tech stack",
    desc: "A decoupled architecture and a modern toolchain — easy to extend and deploy.",
    groups: ["Backend", "Frontend", "Database"]
  },
  deploy: {
    title: "Deploy",
    desc: "Docker is recommended; prebuilt binaries are also available for every platform.",
    quickStart: "Quick start",
    docker: "Docker",
    binary: "Binary",
    note: "Linux / Windows / macOS, via either Docker or a prebuilt binary",
    points: [
      {
        title: "One-command deploy",
        desc: "Start with three Docker Compose commands; data and configuration persist automatically."
      },
      {
        title: "Multi-registry images",
        desc: "Choose from Docker Hub, GHCR or CNB to suit networks in and outside China."
      },
      {
        title: "Ready out of the box",
        desc: "A built-in setup wizard walks you through database and admin configuration — no manual table creation."
      }
    ],
    commands: {
      docker: [
        { text: "mkdir skyimage && cd skyimage", type: "cmd" },
        {
          text: "curl -O https://raw.githubusercontent.com/nxtcorex/skyImage/refs/heads/main/docker-compose.yml",
          type: "cmd"
        },
        {
          text: "curl -o .env https://raw.githubusercontent.com/nxtcorex/skyImage/refs/heads/main/.env.example",
          type: "cmd"
        },
        { text: "", type: "cmd" },
        { text: "# Start the service, then open http://localhost:8080", type: "cmt" },
        { text: "docker-compose up -d", type: "cmd" }
      ],
      binary: [
        { text: "# Download the prebuilt package for your platform from GitHub Releases", type: "cmt" },
        { text: "tar -xzf skyimage-*-linux-amd64.tar.gz", type: "cmd" },
        { text: "cp .env.example .env", type: "cmd" },
        { text: "", type: "cmd" },
        { text: "# Run it and finish the setup wizard", type: "cmt" },
        { text: "./skyimage", type: "cmd" }
      ]
    }
  },
  migration: {
    badge: "Migrate from Lsky Pro",
    title: "Move your old site in one pass",
    desc: "The setup wizard includes a migration tool that moves data from Lsky Pro (open-source V2.x) to SkyImage while keeping image paths unchanged.",
    cta: "Read the migration docs",
    points: [
      "Users, role groups, storage strategies, albums and image records all migrate together",
      "Password hashes are preserved, so existing users can sign in with their old passwords",
      "The import only reads the source database — SQLite is opened read-only and source data is never modified",
      "Point it at the old image directory and files are copied along the same paths, so links keep working"
    ]
  },
  cta: {
    title: "Stand up your image host today",
    desc: "One Docker command gets you started, and the wizard takes only minutes. Open source and free — your data stays with you.",
    demo: "Try the demo",
    github: "GitHub"
  },
  footer: {
    copyright: "© {year} SkyImage · Open source under the {license} license",
    github: "GitHub",
    license: "License",
    demo: "Demo"
  }
};

export const dictionaries: Record<Locale, SiteCopy> = {
  "zh-CN": zh,
  en
};

export function format(template: string, variables?: Record<string, string | number>) {
  if (!variables) {
    return template;
  }

  return template.replace(/\{(\w+)\}/g, (_, key) => String(variables[key] ?? `{${key}}`));
}

function detectInitialLocale(): Locale {
  if (typeof window === "undefined") {
    return "zh-CN";
  }

  const saved = window.localStorage.getItem(LANGUAGE_STORAGE_KEY);
  if (saved === "zh-CN" || saved === "en") {
    return saved;
  }

  return window.navigator.language.toLowerCase().startsWith("zh") ? "zh-CN" : "en";
}

type I18nContextValue = {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  copy: SiteCopy;
};

const I18nContext = createContext<I18nContextValue>({
  locale: "zh-CN",
  setLocale: () => {},
  copy: zh
});

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocale] = useState<Locale>(detectInitialLocale);

  useEffect(() => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(LANGUAGE_STORAGE_KEY, locale);
    document.documentElement.lang = locale;
    document.title = dictionaries[locale].meta.title;
    const description = document.querySelector('meta[name="description"]');
    if (description) {
      description.setAttribute("content", dictionaries[locale].meta.description);
    }
  }, [locale]);

  const value = useMemo<I18nContextValue>(
    () => ({ locale, setLocale, copy: dictionaries[locale] }),
    [locale]
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  return useContext(I18nContext);
}
