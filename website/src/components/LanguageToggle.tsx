import { Languages } from "lucide-react";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import { useI18n, type Locale } from "@/i18n";

type LanguageToggleProps = {
  iconOnly?: boolean;
};

// 与主项目 src/components/LanguageToggle.tsx 视觉一致
export function LanguageToggle({ iconOnly = false }: LanguageToggleProps) {
  const { locale, setLocale, copy } = useI18n();

  return (
    <Select value={locale} onValueChange={(value) => setLocale(value as Locale)}>
      <SelectTrigger
        className={
          iconOnly
            ? "h-9 w-9 justify-center px-0 border-0 shadow-none"
            : "h-9 w-[140px] gap-2 px-3"
        }
        aria-label={copy.lang.switcher}
        title={copy.lang.switcher}
      >
        <Languages className="h-4 w-4" />
        {!iconOnly ? <SelectValue placeholder={copy.lang.switcher} /> : null}
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="zh-CN">{copy.lang.zh}</SelectItem>
        <SelectItem value="en">{copy.lang.en}</SelectItem>
      </SelectContent>
    </Select>
  );
}
