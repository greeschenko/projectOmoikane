"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Container, Typography, Box, Card, CardContent } from "@mui/material";

interface BlogPost {
  id: string;
  title: string;
  slug: string;
  excerpt: string;
  content: string;
  status: string;
  publishDate: string;
  featuredImage: string;
  createdAt: string;
}

export default function BlogListPage() {
  const [posts, setPosts] = useState<BlogPost[]>([]);

  useEffect(() => {
    fetch("/api/blog/posts")
      .then((r) => r.json())
      .then((data: BlogPost[]) => setPosts(data.filter((p) => p.status === "published")))
      .catch(() => {});
  }, []);

  if (posts.length === 0) {
    return (
      <Container maxWidth="md" sx={{ mt: 4, textAlign: "center" }}>
        <Typography variant="h4" gutterBottom>Blog</Typography>
        <Typography color="text.secondary">No posts yet. Check back soon!</Typography>
      </Container>
    );
  }

  return (
    <Container maxWidth="md" sx={{ mt: 4 }}>
      <Typography variant="h4" gutterBottom>Blog</Typography>
      <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
        {posts.map((post) => (
          <Card key={post.id} component={Link} href={`/blog/${post.slug}`} sx={{ textDecoration: "none" }}>
            <CardContent>
              <Typography variant="h5">{post.title}</Typography>
              {post.excerpt && (
                <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                  {post.excerpt}
                </Typography>
              )}
              <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: "block" }}>
                {new Date(post.publishDate || post.createdAt).toLocaleDateString()}
              </Typography>
            </CardContent>
          </Card>
        ))}
      </Box>
    </Container>
  );
}
