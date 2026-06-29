export const dynamic = "force-dynamic";

const apiBase = process.env.API_URL || "http://backend:8080";

export async function GET() {
  const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || "http://localhost:3000";

  const settingsRes = await fetch(`${apiBase}/settings`);
  const settings = settingsRes.ok ? await settingsRes.json() : null;
  const siteName = settings?.siteName || "Omoikane";

  const postsRes = await fetch(`${apiBase}/blog/posts`);
  const allPosts = postsRes.ok ? await postsRes.json() : [];
  const posts = (Array.isArray(allPosts) ? allPosts : (allPosts.posts ?? []))
    .filter((p: any) => p.status === "published")
    .sort((a: any, b: any) => new Date(b.publishDate || b.createdAt).getTime() - new Date(a.publishDate || a.createdAt).getTime());

  const items = posts.map((post: any) => `
    <item>
      <title><![CDATA[${post.title}]]></title>
      <link>${siteUrl}/blog/${post.slug}</link>
      <description><![CDATA[${post.excerpt || (post.content || "").substring(0, 200)}]]></description>
      <pubDate>${new Date(post.publishDate || post.createdAt).toUTCString()}</pubDate>
      <guid>${siteUrl}/blog/${post.slug}</guid>
    </item>
  `).join("\n");

  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>${siteName}</title>
    <link>${siteUrl}</link>
    <description>Latest blog posts from ${siteName}</description>
    <language>en</language>
    <atom:link href="${siteUrl}/rss" rel="self" type="application/rss+xml"/>
    ${items}
  </channel>
</rss>`;

  return new Response(xml, {
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
      "Cache-Control": "no-cache",
    },
  });
}
