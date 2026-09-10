import { Moon, Monitor, Sun } from "lucide-react";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import { useTheme } from "@/components/ThemeProvider";

type ThemeToggleProps = {
  iconOnly?: boolean;
};

// 与主项目 src/components/ThemeToggle.tsx 视觉一致：radix Select 下拉，图标随状态切换
export function ThemeToggle({ iconOnly = false }: ThemeToggleProps) {
  const { theme, resolvedTheme, setTheme } = useTheme();
  const currentValue = theme ?? "system";

  const Icon =
    currentValue === "system" ? Monitor : resolvedTheme === "dark" ? Moon : Sun;

  return (
    <Select value={currentValue} onValueChange={setTheme}>
      <SelectTrigger
        className={
          iconOnly
            ? "h-9 w-9 justify-center px-0 border-0 shadow-none"
            : "h-9 w-[140px] gap-2 px-3"
        }
        aria-label="主题"
        title="主题"
      >
        <Icon className="h-4 w-4" />
        {!iconOnly ? <SelectValue placeholder="主题" /> : null}
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="system">跟随系统</SelectItem>
        <SelectItem value="light">浅色</SelectItem>
        <SelectItem value="dark">深色</SelectItem>
      </SelectContent>
    </Select>
  );
}
