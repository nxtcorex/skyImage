import { Palette } from "lucide-react";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import { useTheme } from "@/components/ThemeProvider";
import { themePalettes } from "@/lib/theme-palettes";

type PaletteToggleProps = {
  iconOnly?: boolean;
};

// 与主项目 src/components/PaletteToggle.tsx 视觉一致：切换 data-palette 令牌组
export function PaletteToggle({ iconOnly = false }: PaletteToggleProps) {
  const { palette, setPalette } = useTheme();

  return (
    <Select value={palette} onValueChange={setPalette}>
      <SelectTrigger
        className={
          iconOnly
            ? "h-9 w-9 justify-center px-0 border-0 shadow-none"
            : "h-9 w-[148px] gap-2 px-3"
        }
        aria-label="配色方案"
        title="配色方案"
      >
        <Palette className="h-4 w-4" />
        {!iconOnly ? <SelectValue placeholder="配色方案" /> : null}
      </SelectTrigger>
      <SelectContent>
        {themePalettes.map((item) => (
          <SelectItem key={item.value} value={item.value}>
            {item.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
