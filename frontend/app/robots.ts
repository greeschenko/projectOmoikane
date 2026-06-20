import type { MetadataRoute } from "next";
import store from "@/lib/store";

export default function robots(): MetadataRoute.Robots {
  const settings = store.getSettings();

  const baseUrl = settings?.siteName
    ? `https://${settings.siteName.toLowerCase().replace(/\s+/g, "")}.example.com`
    : "http://localhost:3000";

  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/admin", "/api"],
    },
    sitemap: `${baseUrl}/sitemap.xml`,
  };
}
