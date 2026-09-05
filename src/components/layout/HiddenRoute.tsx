import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";

import { fetchSiteConfig } from "@/lib/api";
import { getHiddenSidebarUrls, normalizeNavUrl } from "@/lib/navigation";
import { NotFoundPage } from "@/features/misc/NotFoundPage";

function getCachedConfig() {
  if (typeof window === "undefined") {
    return undefined;
  }
  try {
    const cached = window.localStorage.getItem("skyimage-site-config");
    return cached ? JSON.parse(cached) : undefined;
  } catch {
    return undefined;
  }
}

type HiddenRouteProps = {
  /** 该路由对应的侧边栏 URL，例如 "/dashboard/images" */
  url: string;
  children: ReactNode;
};

// HiddenRoute 守卫被隐藏的侧边栏页面：即使通过路径直接访问，也会渲染 404。
export function HiddenRoute({ url, children }: HiddenRouteProps) {
  const { data: siteConfig } = useQuery({
    queryKey: ["site-config"],
    queryFn: fetchSiteConfig,
    initialData: getCachedConfig,
    staleTime: 5 * 60 * 1000
  });

  // 站点配置尚未成功返回（如首次请求失败）时放行渲染，避免页面静默空白。
  // 404 判断只建立在已有配置成功返回之上。
  if (!siteConfig) {
    return <>{children}</>;
  }

  const hidden = getHiddenSidebarUrls(siteConfig);
  if (hidden.has(normalizeNavUrl(url))) {
    return <NotFoundPage />;
  }

  return <>{children}</>;
}