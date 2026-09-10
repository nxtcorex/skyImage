import { useState } from "react";
import { ArrowRight, Check, Copy, KeyRound } from "lucide-react";

import { DEMO_ACCOUNTS, SITE } from "@/data/site";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

type DemoAccessDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/** 演示站账号弹窗：账号密码默认隐藏，点「在线演示」才弹出。 */
export function DemoAccessDialog({ open, onOpenChange }: DemoAccessDialogProps) {
  const { copy } = useI18n();
  const [copied, setCopied] = useState<string | null>(null);

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(text);
      window.setTimeout(() => {
        setCopied((current) => (current === text ? null : current));
      }, 2000);
    } catch {
      // 剪贴板不可用时忽略：账号本身已可选中复制
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <KeyRound className="h-4 w-4" />
            {copy.demo.title}
          </DialogTitle>
          <DialogDescription>{copy.demo.desc}</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          {DEMO_ACCOUNTS.map((account) => {
            const credential = `${account.email} / ${account.password}`;
            const isCopied = copied === credential;
            return (
              <div key={account.email} className="rounded-lg border bg-card/60 p-3">
                <span className="inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-semibold">
                  {copy.demo.roles[account.role]}
                </span>

                <div className="mt-3 grid grid-cols-[3.5rem_1fr] items-center gap-y-2 text-sm">
                  <span className="text-muted-foreground">{copy.demo.emailLabel}</span>
                  <span className="select-all break-all font-mono text-[13px]">{account.email}</span>

                  <span className="text-muted-foreground">{copy.demo.passwordLabel}</span>
                  <span className="flex items-center justify-between gap-2">
                    <span className="select-all break-all font-mono text-[13px]">
                      {account.password}
                    </span>
                    <button
                      type="button"
                      onClick={() => void handleCopy(credential)}
                      className="press inline-flex h-6 shrink-0 items-center gap-1 rounded-md border px-2 text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
                    >
                      {isCopied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
                      {isCopied ? copy.demo.copied : copy.demo.copy}
                    </button>
                  </span>
                </div>
              </div>
            );
          })}
        </div>

        <p className="text-xs leading-relaxed text-muted-foreground">{copy.demo.hint}</p>

        <DialogFooter>
          <Button asChild className="gap-2">
            <a href={SITE.demo} target="_blank" rel="noreferrer">
              {copy.demo.open}
              <ArrowRight className="h-4 w-4" />
            </a>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type DemoDialogButtonProps = {
  label: string;
  className?: string;
};

/** 「在线演示」入口按钮：点击弹出账号弹窗，而不是直接跳站。 */
export function DemoDialogButton({ label, className }: DemoDialogButtonProps) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button size="lg" className={cn("gap-2", className)} onClick={() => setOpen(true)}>
        {label}
        <ArrowRight className="h-4 w-4" />
      </Button>
      <DemoAccessDialog open={open} onOpenChange={setOpen} />
    </>
  );
}
