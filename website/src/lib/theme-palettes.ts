// 与主项目 src/lib/theme-palettes.ts 保持一致（官网无需 i18n，标签直接内联）
export const themePalettes = [
  {
    value: "skyimage",
    label: "默认",
    description: "保留原有黑白高对比风格",
    swatches: ["0 0% 9%", "0 0% 96.1%", "0 0% 45.1%"]
  },
  {
    value: "shadcn-admin",
    label: "深蓝",
    description: "默认的深蓝后台质感",
    swatches: ["221 83% 53%", "214 95% 93%", "221 39% 46%"]
  },
  {
    value: "zinc",
    label: "Zinc",
    description: "中性灰后台配色",
    swatches: ["240 5.9% 10%", "240 4.8% 95.9%", "240 3.8% 46.1%"]
  },
  {
    value: "slate",
    label: "Slate",
    description: "偏蓝灰的管理界面配色",
    swatches: ["222.2 47.4% 11.2%", "210 40% 96.1%", "215.4 16.3% 46.9%"]
  },
  {
    value: "stone",
    label: "Stone",
    description: "偏暖灰的阅读型配色",
    swatches: ["24 9.8% 10%", "60 4.8% 95.9%", "25 5.3% 44.7%"]
  }
] as const;

export type ThemePalette = (typeof themePalettes)[number]["value"];

export const defaultThemePalette: ThemePalette = "skyimage";

export function isThemePalette(value: string | null | undefined): value is ThemePalette {
  return themePalettes.some((palette) => palette.value === value);
}
