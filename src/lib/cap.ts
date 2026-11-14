export interface CapErrorDetail {
  code?: string;
  message?: string;
}

export type CapProblem = "mixedContent" | "network" | "parse" | "generic";

const CAP_NETWORK_CODES = new Set([
  "network_error",
  "solve_failed",
  "missing_endpoint",
  "wasm_load_failed",
  "worker_spawn_failed",
]);

const CAP_PARSE_CODES = new Set(["challenge_parse_error", "redeem_failed"]);

function isLoopbackHost(host: string): boolean {
  const h = host.toLowerCase();
  return (
    h === "localhost" ||
    h.endsWith(".localhost") ||
    /^127\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(h) ||
    h === "[::1]" ||
    h === "::1"
  );
}

function instanceHost(instanceUrl: string): string {
  try {
    return new URL(instanceUrl.trim()).hostname;
  } catch {
    return "";
  }
}

export function diagnoseCapProblem(
  detail: CapErrorDetail | undefined,
  opts: { pageIsHttps: boolean; instanceUrl: string }
): CapProblem {
  const code = detail?.code || "";
  const message = detail?.message || "";
  const host = instanceHost(opts.instanceUrl);
  if (opts.pageIsHttps && /^http:\/\//i.test(opts.instanceUrl.trim()) && !isLoopbackHost(host)) {
    return "mixedContent";
  }
  if (CAP_PARSE_CODES.has(code)) return "parse";
  if (CAP_NETWORK_CODES.has(code)) return "network";
  if (/likely down|network|failed to fetch/i.test(message)) return "network";
  return "generic";
}

let loadPromise: Promise<void> | null = null;

export function extractCapInstanceFromEndpoint(apiEndpoint: string, siteKey: string): string {
  const base = apiEndpoint.trim().replace(/\/+$/, "");
  const key = siteKey.trim();
  if (!base || !key) {
    return "";
  }
  if (base.toLowerCase().endsWith(`/${key.toLowerCase()}`)) {
    return base.slice(0, base.length - key.length - 1);
  }
  return "";
}

export function buildCapAssetsUrl(instanceUrl: string): string {
  const base = instanceUrl.trim().replace(/\/+$/, "");
  if (!base) {
    return "";
  }
  return `${base}/assets/widget.js`;
}

function loadWidgetScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = src;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => {
      script.remove();
      reject(new Error(`Failed to load Cap widget script: ${src}`));
    };
    document.head.appendChild(script);
  });
}

function ensureElementDefined(): Promise<void> {
  if (customElements.get("cap-widget")) {
    return Promise.resolve();
  }
  return customElements.whenDefined("cap-widget").then(() => undefined);
}

function loadBundledWidget(): Promise<void> {
  return import("cap-widget").then(() => ensureElementDefined());
}

export function loadCapWidget(instanceUrl?: string): Promise<void> {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("Cap widget requires a browser environment"));
  }

  if (customElements.get("cap-widget")) {
    return Promise.resolve();
  }

  const assetsUrl = instanceUrl ? buildCapAssetsUrl(instanceUrl) : "";

  if (loadPromise) {
    return loadPromise;
  }

  loadPromise = (assetsUrl
    ? loadWidgetScript(assetsUrl)
        .then(() => {
          if (customElements.get("cap-widget")) {
            return;
          }
          return loadBundledWidget();
        })
        .catch(() => loadBundledWidget())
    : loadBundledWidget()
  ).catch((err) => {
    loadPromise = null;
    throw err instanceof Error ? err : new Error("Failed to load Cap widget");
  });

  return loadPromise;
}

export function buildCapApiEndpoint(instanceUrl: string, siteKey: string): string {
  const base = instanceUrl.trim().replace(/\/+$/, "");
  const key = siteKey.trim();
  if (!base || !key) {
    return "";
  }
  return `${base}/${key}/`;
}
