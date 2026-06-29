import { getSession } from "@/lib/auth";
import { apiFetch } from "@/lib/api";
import { notFound } from "next/navigation";
import { Container, Typography } from "@mui/material";
import PostDetailClient from "@/components/PostDetailClient";

export const dynamic = "force-dynamic";

export default async function BlogPostPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const settings = await apiFetch<{ siteName: string; blogEnabled: boolean }>("/settings");
  if (!settings.blogEnabled) notFound();

  const { slug } = await params;

  let post: Record<string, unknown>;
  try {
    post = await apiFetch<Record<string, unknown>>(`/blog/posts/slug/${slug}`);
  } catch {
    notFound();
  }

  const session = await getSession();
  const canEdit =
    !!session &&
    (session.role === "admin" || session.userId === String(post.authorId));

  return (
    <Container maxWidth="md" sx={{ mt: 4 }}>
      <Typography variant="h3" gutterBottom>{post.title as string}</Typography>
      <PostDetailClient
        post={{
          id: post.id as string,
          title: post.title as string,
          slug: post.slug as string,
          content: post.content as string,
          authorId: post.authorId as string,
          status: post.status as string,
          publishDate: post.publishDate as string | undefined,
          likeCount: post.likeCount as number,
          createdAt: post.createdAt as string,
          authorName: post.authorName as string | undefined,
        }}
        canEdit={canEdit}
      />
    </Container>
  );
}
