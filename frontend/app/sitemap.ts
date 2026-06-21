import type { MetadataRoute } from "next";
import store from "@/lib/store";

export default function sitemap(): MetadataRoute.Sitemap {
  const pages = store.getPages();
  const blogPosts = store.getBlogPosts();
  const settings = store.getSettings();

  const baseUrl = settings?.siteName
    ? `https://${settings.siteName.toLowerCase().replace(/\s+/g, "")}.example.com`
    : "http://localhost:3000";

  const pageUrls = pages
    .filter((p) => p.status === "published")
    .map((page) => ({
      url: `${baseUrl}/${page.slug}`,
      lastModified: new Date(page.updatedAt),
      changeFrequency: "weekly" as const,
      priority: 0.8,
    }));

  const blogUrls = blogPosts
    .filter((p) => p.status === "published")
    .map((post) => ({
      url: `${baseUrl}/blog/${post.slug}`,
      lastModified: new Date(post.updatedAt),
      changeFrequency: "weekly" as const,
      priority: 0.6,
    }));

  return [
    {
      url: baseUrl,
      lastModified: new Date(),
      changeFrequency: "daily" as const,
      priority: 1,
    },
    ...pageUrls,
    ...blogUrls,
  ];
}
