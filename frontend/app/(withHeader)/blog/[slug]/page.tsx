import store from "@/lib/store";
import { notFound } from "next/navigation";
import { Container, Typography, Box, Chip } from "@mui/material";

export const dynamic = "force-dynamic";

export default async function BlogPostPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const posts = store.getBlogPosts().filter(
    (p) => p.status === "published" && p.slug === slug
  );
  const post = posts[0];
  if (!post) notFound();

  const author = store.getUser(post.authorId);

  return (
    <Container maxWidth="md" sx={{ mt: 4 }}>
      <Typography variant="h3" gutterBottom>{post.title}</Typography>
      <Box sx={{ display: "flex", gap: 1, alignItems: "center", mb: 2 }}>
        {author && (
          <Typography variant="body2" color="text.secondary">
            By {author.name || author.email}
          </Typography>
        )}
        <Chip label={post.status} size="small" color="success" />
        <Typography variant="body2" color="text.secondary">
          {new Date(post.publishDate || post.createdAt).toLocaleDateString()}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {post.likeCount} {post.likeCount === 1 ? "like" : "likes"}
        </Typography>
      </Box>
      <Box sx={{ mt: 2 }} dangerouslySetInnerHTML={{ __html: post.content }} />
    </Container>
  );
}
