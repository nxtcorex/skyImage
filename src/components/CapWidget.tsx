import { forwardRef, useEffect, useImperativeHandle, useLayoutEffect, useRef } from "react";
import { type CapErrorDetail } from "@/lib/cap";

export interface CapWidgetProps {
  apiEndpoint: string;
  onVerify: (token: string) => void;
  onError?: (detail?: CapErrorDetail) => void;
  onExpire?: () => void;
}

export interface CapWidgetRef {
  reset: () => void;
}

type CapWidgetElement = HTMLElement & {
  reset?: () => void;
};

export const CapWidget = forwardRef<CapWidgetRef, CapWidgetProps>(
  ({ apiEndpoint, onVerify, onError, onExpire }, ref) => {
    const hostRef = useRef<HTMLDivElement>(null);
    const widgetRef = useRef<CapWidgetElement | null>(null);
    const verifyRef = useRef(onVerify);
    const errorRef = useRef(onError);
    const expireRef = useRef(onExpire);
    // 主动 reset() 时不触发 onExpire
    const suppressExpireRef = useRef(false);

    useEffect(() => {
      verifyRef.current = onVerify;
    }, [onVerify]);

    useEffect(() => {
      errorRef.current = onError;
    }, [onError]);

    useEffect(() => {
      expireRef.current = onExpire;
    }, [onExpire]);

    useImperativeHandle(ref, () => ({
      reset: () => {
        suppressExpireRef.current = true;
        try {
          widgetRef.current?.reset?.();
        } finally {
          // Cap reset 同步 dispatch，处理完后再放开
          suppressExpireRef.current = false;
        }
      },
    }));

    useLayoutEffect(() => {
      const host = hostRef.current;
      if (!host || !apiEndpoint) {
        return;
      }

      suppressExpireRef.current = false;
      host.innerHTML = "";
      const widget = document.createElement("cap-widget") as CapWidgetElement;
      widget.setAttribute("data-cap-api-endpoint", apiEndpoint);

      const handleSolve = (event: Event) => {
        const detail = (event as CustomEvent<{ token?: string }>).detail;
        if (detail?.token) {
          verifyRef.current?.(detail.token);
        }
      };
      const handleError = (event: Event) => {
        const detail = (event as CustomEvent<CapErrorDetail>).detail;
        errorRef.current?.(detail);
      };
      // Cap reset 来源：
      // 1) token TTL 到期（仍在文档中，isConnected=true）→ 视为过期
      // 2) disconnectedCallback 卸载（isConnected=false）→ 忽略，避免登录成功后误报
      // 3) 主动 reset() → suppressExpireRef
      const handleReset = () => {
        if (suppressExpireRef.current) return;
        if (!widget.isConnected) return;
        expireRef.current?.();
      };

      widget.addEventListener("solve", handleSolve as EventListener);
      widget.addEventListener("error", handleError);
      widget.addEventListener("reset", handleReset);
      host.appendChild(widget);
      widgetRef.current = widget;

      return () => {
        suppressExpireRef.current = true;
        widget.removeEventListener("solve", handleSolve as EventListener);
        widget.removeEventListener("error", handleError);
        widget.removeEventListener("reset", handleReset);
        widgetRef.current = null;
        if (host.contains(widget)) {
          host.removeChild(widget);
        } else {
          host.innerHTML = "";
        }
      };
    }, [apiEndpoint]);

    return <div ref={hostRef} className="flex justify-center" />;
  }
);

CapWidget.displayName = "CapWidget";
