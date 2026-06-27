import type { MetadataRoute } from "next";
import { apiFetch } from "@/lib/api";

export default async function robots(): Promise<MetadataRoute.Robots> {
  let baseUrl = "http://localhost:3000";
  try {
    const settings = await apiFetch<{ siteName: string }>("/settings");
    if (settings?.siteName) {
      baseUrl = `https://${settings.siteName.toLowerCase().replace(/\s+/g, "")}.example.com`;
    }
  } catch { /* ignore */ }

  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/admin", "/api"],
    },
    sitemap: `${baseUrl}/sitemap.xml`,
  };
}
