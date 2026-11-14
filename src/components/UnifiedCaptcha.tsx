import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import { Turnstile, type TurnstileRef } from "./Turnstile";
import { CapWidget, type CapWidgetRef } from "./CapWidget";
import { loadTurnstileScript } from "@/lib/turnstile";
import { loadGeetestScript } from "@/lib/geetest";
import { extractCapInstanceFromEndpoint, loadCapWidget, type CapErrorDetail } from "@/lib/cap";

export type CaptchaProvider = "cloudflare" | "geetest" | "cap";

export interface UnifiedCaptchaProps {
  provider: CaptchaProvider;
  siteKey: string;
  apiEndpoint?: string;
  onVerify: (token: string, extraData?: Record<string, string>) => void;
  onError?: (detail?: CapErrorDetail) => void;
  onExpire?: () => void;
}

export interface UnifiedCaptchaRef {
  reset: () => void;
}

export const UnifiedCaptcha = forwardRef<UnifiedCaptchaRef, UnifiedCaptchaProps>(
  ({ provider, siteKey, apiEndpoint, onVerify, onError, onExpire }, ref) => {
    const [ready, setReady] = useState(false);
    const turnstileRef = useRef<TurnstileRef>(null);
    const capRef = useRef<CapWidgetRef>(null);
    const geetestRef = useRef<any>(null);
    const geetestInitialized = useRef(false);
    const onVerifyRef = useRef(onVerify);
    const onErrorRef = useRef(onError);
    const onExpireRef = useRef(onExpire);

    useEffect(() => {
      onVerifyRef.current = onVerify;
    }, [onVerify]);
    useEffect(() => {
      onErrorRef.current = onError;
    }, [onError]);
    useEffect(() => {
      onExpireRef.current = onExpire;
    }, [onExpire]);

    const handleVerify = (token: string, extraData?: Record<string, string>) => {
      onVerifyRef.current?.(token, extraData);
    };
    const handleError = (detail?: CapErrorDetail) => {
      onErrorRef.current?.(detail);
    };
    const handleExpire = () => {
      onExpireRef.current?.();
    };

    useImperativeHandle(ref, () => ({
      reset: () => {
        if (provider === "cloudflare" && turnstileRef.current) {
          turnstileRef.current.reset();
        } else if (provider === "geetest" && geetestRef.current) {
          geetestRef.current.reset();
        } else if (provider === "cap" && capRef.current) {
          capRef.current.reset();
        }
      }
    }));

    // 只在 provider 变化时加载脚本，避免父组件每次 render 传入新 onError 导致反复卸载
    useEffect(() => {
      let cancelled = false;
      setReady(false);
      geetestInitialized.current = false;

      if (provider === "cloudflare") {
        loadTurnstileScript()
          .then(() => {
            if (!cancelled) setReady(true);
          })
          .catch((err) => {
            console.error("Failed to load Turnstile:", err);
            if (!cancelled) handleError();
          });
      } else if (provider === "geetest") {
        loadGeetestScript()
          .then(() => {
            if (!cancelled) setReady(true);
          })
          .catch((err) => {
            console.error("Failed to load Geetest:", err);
            if (!cancelled) handleError();
          });
      } else if (provider === "cap") {
        const capInstance = extractCapInstanceFromEndpoint(apiEndpoint || "", siteKey);
        loadCapWidget(capInstance || undefined)
          .then(() => {
            if (!cancelled) setReady(true);
          })
          .catch((err) => {
            console.error("Failed to load Cap widget:", err);
            if (!cancelled) handleError();
          });
      }

      return () => {
        cancelled = true;
      };
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [provider, apiEndpoint]);

    if (provider === "cloudflare") {
      return ready ? (
        <Turnstile
          ref={turnstileRef}
          siteKey={siteKey}
          onVerify={handleVerify}
          onError={handleError}
          onExpire={handleExpire}
        />
      ) : (
        <div className="flex items-center justify-center p-4">
          <div className="text-sm text-muted-foreground">加载验证组件中...</div>
        </div>
      );
    }

    if (provider === "cap") {
      const endpoint = apiEndpoint || "";
      if (!endpoint) {
        return (
          <div className="flex items-center justify-center p-4">
            <div className="text-sm text-muted-foreground">Cap 配置不完整</div>
          </div>
        );
      }
      return ready ? (
        <CapWidget
          ref={capRef}
          apiEndpoint={endpoint}
          onVerify={handleVerify}
          onError={handleError}
          onExpire={handleExpire}
        />
      ) : (
        <div className="flex items-center justify-center p-4">
          <div className="text-sm text-muted-foreground">加载验证组件中...</div>
        </div>
      );
    }

    if (provider === "geetest") {
      return ready ? (
        <div
          id="geetest-captcha"
          className="flex items-center justify-center"
          ref={(el) => {
            if (el && !geetestInitialized.current) {
              if (!(window as any).initGeetest4) {
                const retry = () => {
                  if (geetestInitialized.current) return;
                  if ((window as any).initGeetest4) {
                    geetestInitialized.current = true;
                    initGeetest4(el, siteKey, (captchaObj) => {
                      geetestRef.current = captchaObj;
                    }, handleVerify, handleError);
                  } else {
                    setTimeout(retry, 100);
                  }
                };
                setTimeout(retry, 100);
                return;
              }
              geetestInitialized.current = true;
              initGeetest4(el, siteKey, (captchaObj) => {
                geetestRef.current = captchaObj;
              }, handleVerify, handleError);
            }
          }}
        />
      ) : (
        <div className="flex items-center justify-center p-4">
          <div className="text-sm text-muted-foreground">加载极验组件中...</div>
        </div>
      );
    }

    return null;
  }
);

UnifiedCaptcha.displayName = "UnifiedCaptcha";

function initGeetest4(
  element: HTMLElement,
  captchaId: string,
  onCaptchaReady: (captchaObj: any) => void,
  onVerify: (token: string, extraData?: Record<string, string>) => void,
  onError?: () => void
): void {
  if (typeof window === "undefined" || !(window as any).initGeetest4) {
    return;
  }

  (window as any).initGeetest4(
    {
      captchaId: captchaId,
      product: "popup",
    },
    (captchaObj: any) => {
      onCaptchaReady(captchaObj);

      captchaObj.onSuccess(() => {
        const result = captchaObj.getValidate();
        if (result) {
          onVerify(result.lot_number, {
            challenge: result.lot_number,
            validate: result.pass_token,
            seccode: result.gen_time,
            captcha_output: result.captcha_output,
          });
        }
      });

      captchaObj.onError(() => {
        onError?.();
      });

      captchaObj.appendTo(element);
    }
  );
}
