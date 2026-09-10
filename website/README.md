# SkyImage 官网

SkyImage 官方网站（产品官网 / 落地页），独立于主应用的静态展示站点。

## 技术栈

与主应用保持一致：

- **React 18** + **TypeScript**
- **Vite 6** 构建
- **Tailwind CSS 3** + **shadcn/ui**（new-york 风格，neutral 中性色）
- **lucide-react** 图标

设计令牌直接复用主项目 `src/index.css` 的 HSL 变量与 `--radius`，视觉语言与后台完全统一：纯灰阶、无彩色渐变、无飞入动画，仅保留克制的上滑淡入。

## 开发

```bash
cd website
pnpm install
pnpm dev      # http://localhost:5174
```

## 构建

```bash
pnpm build    # 产物输出到 website/dist
pnpm preview
```

## 目录结构

```
website/
├── index.html                 # 入口，含主题/配色/语言初始化脚本
├── src/
│   ├── main.tsx               # I18nProvider > ThemeProvider > App
│   ├── App.tsx                # 页面装配
│   ├── index.css              # 设计令牌（与主项目一致，含 5 套 data-palette）
│   ├── i18n.tsx               # 全站中英文案字典 + I18nProvider / useI18n
│   ├── data/site.ts           # 与语言无关的数据：链接、图标、演示账号、预览占位图、专有名词
│   ├── lib/
│   │   ├── utils.ts
│   │   └── theme-palettes.ts  # 5 套配色方案定义
│   └── components/
│       ├── ui/                # button / card / badge / select / dialog（与主项目同源）
│       ├── BrandMark.tsx
│       ├── ThemeToggle.tsx    # 深浅（跟随系统 / 浅色 / 深色）
│       ├── PaletteToggle.tsx  # 配色方案
│       ├── LanguageToggle.tsx # 语言（简体中文 / English）
│       ├── DemoAccessDialog.tsx # 「在线演示」按钮 + 演示账号弹窗
│       └── sections/          # 各区块
└── public/favicon.svg
```

## 内容维护

- **文案**：全部集中在 `src/i18n.tsx` 的 `zh` / `en` 两本字典里，改文案只动这一个文件。两本字典共用 `SiteCopy` 类型，**漏翻会直接编译报错**。
- **数据**：`src/data/site.ts` 只放与语言无关的内容——仓库/演示站链接、图标数组、技术栈专有名词、演示账号、界面预览的图库卡片数据。
- **图标与文案的对应**：`site.ts` 里的图标数组按顺序对应 `i18n.tsx` 中的文案数组（如 `FEATURE_ICONS[i]` 对应 `copy.features.items[i]`），增删条目时两边要同步调整顺序。
- **演示账号**：改 `site.ts` 的 `DEMO_ACCOUNTS`（邮箱/密码）与 `i18n.tsx` 的 `copy.demo.roles.admin` / `.user`（角色名）。弹窗由 `DemoAccessDialog.tsx` 提供，Hero 和 CtaBand 都通过 `DemoDialogButton` 触发。
- **界面预览**：`ProductPreview.tsx` 还原后台真实控制台，缩略图来自 `site.ts` 的 `PREVIEW_ROWS`（内联灰阶 SVG，按 `ratio` 等比撑满一行）。
