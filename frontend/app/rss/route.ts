import store from "@/lib/store";

export const dynamic = "force-dynamic";

export async function GET() {
  const settings = store.getSettings();
  const siteName = settings?.siteName || "Omoikane";
  const siteUrl = "http://localhost:3000";

  const posts = store.getBlogPosts()
    .filter((p) => p.status === "published")
    .sort((a, b) => new Date(b.publishDate || b.createdAt).getTime() - new Date(a.publishDate || a.createdAt).getTime());

  const items = posts.map((post) => `
    <item>
      <title><![CDATA[${post.title}]]></title>
      <link>${siteUrl}/blog/${post.slug}</link>
      <description><![CDATA[${post.excerpt || post.content.substring(0, 200)}]]></description>
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
