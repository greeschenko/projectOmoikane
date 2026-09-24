import type { MetadataRoute } from "next";
import { apiFetch } from "@/lib/api";

// Next marks sitemap as a cached Route Handler by default; the prod build
// (k8s frontend image) bakes it at `next build` time — with an empty DB the
// sitemap would be root-only forever. Force per-request rendering so published
// pages/posts always reflect the live data (compose's `next dev` never noticed).
export const dynamic = "force-dynamic";

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  let pages: Array<Record<string, unknown>> = [];
  let blogPosts: Array<Record<string, unknown>> = [];
  try {
    const pagesData = await apiFetch<Array<Record<string, unknown>> | { pages: Array<Record<string, unknown>> }>("/pages");
    pages = Array.isArray(pagesData) ? pagesData : (pagesData.pages ?? []);
  } catch { /* ignore */ }
  try {
    const postsData = await apiFetch<Array<Record<string, unknown>> | { posts: Array<Record<string, unknown>> }>("/blog/posts");
    blogPosts = Array.isArray(postsData) ? postsData : (postsData.posts ?? []);
  } catch { /* ignore */ }

  let baseUrl = "http://localhost:3000";
  try {
    const settings = await apiFetch<{ siteName: string }>("/settings");
    if (settings?.siteName) {
      baseUrl = `https://${settings.siteName.toLowerCase().replace(/\s+/g, "")}.example.com`;
    }
  } catch { /* ignore */ }

  const pageUrls = pages
    .filter((p) => p.status === "published")
    .map((page) => ({
      url: `${baseUrl}/${page.slug}`,
      lastModified: new Date(page.updatedAt as string),
      changeFrequency: "weekly" as const,
      priority: 0.8,
    }));

  const blogUrls = blogPosts
    .filter((p) => p.status === "published")
    .map((post) => ({
      url: `${baseUrl}/blog/${post.slug}`,
      lastModified: new Date(post.updatedAt as string),
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
