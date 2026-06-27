"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import {
  Container, Typography, Box, Card, CardContent, Button, Dialog,
  DialogTitle, DialogContent, DialogActions, TextField, FormControl,
  InputLabel, Select, MenuItem, Alert,
} from "@mui/material";
import RichTextEditor from "@/components/RichTextEditor";

interface BlogPost {
  id: string;
  title: string;
  slug: string;
  excerpt: string;
  content: string;
  authorId: string;
  status: string;
  publishDate: string;
  featuredImage: string;
  createdAt: string;
}

function generateSlug(name: string) {
  return name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "");
}

export default function BlogListPage() {
  const [posts, setPosts] = useState<BlogPost[]>([]);
  const [blogEnabled, setBlogEnabled] = useState(true);
  const [loaded, setLoaded] = useState(false);
  const [currentUser, setCurrentUser] = useState<{ id: string; role: string } | null>(null);
  const [myPostsOnly, setMyPostsOnly] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [formData, setFormData] = useState({ title: "", content: "", status: "published" as "draft" | "published" });
  const [alert, setAlert] = useState<{ type: "success" | "error"; message: string } | null>(null);

  function loadData() {
    Promise.all([
      fetch("/api/settings").then((r) => r.json()),
      fetch("/api/blog/posts").then((r) => r.json()),
      fetch("/api/settings/profile").then((r) => r.ok ? r.json() : null),
    ])
      .then(([settings, data, profile]) => {
        if (settings.blogEnabled !== undefined) setBlogEnabled(settings.blogEnabled);
        setPosts((data as BlogPost[]).filter((p) => p.status === "published"));
        if (profile) setCurrentUser(profile);
      })
      .catch(() => {})
      .finally(() => setLoaded(true));
  }

  useEffect(() => { loadData(); }, []);

  const displayedPosts = myPostsOnly && currentUser
    ? currentUser.role === "admin" ? posts : posts.filter((p) => p.authorId === currentUser.id)
    : posts;

  async function handleCreate() {
    const slug = generateSlug(formData.title) || "post";
    const res = await fetch("/api/blog/posts", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ...formData, slug, excerpt: "" }),
    });
    if (res.ok) {
      setFormOpen(false);
      setFormData({ title: "", content: "", status: "published" });
      setAlert({ type: "success", message: "Post created" });
      loadData();
    } else {
      setAlert({ type: "error", message: "Failed to create post" });
    }
  }

  if (!loaded) return null;

  if (!blogEnabled) {
    return (
      <Container maxWidth="md" sx={{ mt: 4, textAlign: "center" }}>
        <Typography variant="h4" gutterBottom>Blog</Typography>
        <Typography color="text.secondary">Blog is disabled. Check back later!</Typography>
      </Container>
    );
  }

  return (
    <Container maxWidth="md" sx={{ mt: 4 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3, flexWrap: "wrap", gap: 1 }}>
        <Typography variant="h4">Blog</Typography>
        <Box sx={{ display: "flex", gap: 1 }}>
          {currentUser && (
            <Button
              variant={myPostsOnly ? "contained" : "outlined"}
              onClick={() => setMyPostsOnly(!myPostsOnly)}
            >
              My Posts
            </Button>
          )}
          {currentUser && (
            <Button variant="contained" onClick={() => setFormOpen(true)}>
              New Post
            </Button>
          )}
        </Box>
      </Box>

      {alert && (
        <Alert severity={alert.type} sx={{ mb: 2 }} onClose={() => setAlert(null)}>
          {alert.message}
        </Alert>
      )}

      {displayedPosts.length === 0 ? (
        <Typography color="text.secondary" sx={{ textAlign: "center" }}>
          {myPostsOnly ? "You haven't created any posts yet." : "No posts yet. Check back soon!"}
        </Typography>
      ) : (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {displayedPosts.map((post) => (
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
      )}

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>New Blog Post</DialogTitle>
        <DialogContent>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
            <TextField
              label="Title"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              required
            />
            <Typography variant="body2" color="text.secondary">Content</Typography>
            <RichTextEditor
              value={formData.content}
              onChange={(html: string) => setFormData({ ...formData, content: html })}
            />
            <FormControl fullWidth>
              <InputLabel>Status</InputLabel>
              <Select
                label="Status"
                value={formData.status}
                onChange={(e) => setFormData({ ...formData, status: e.target.value as "draft" | "published" })}
              >
                <MenuItem value="draft">Draft</MenuItem>
                <MenuItem value="published">Published</MenuItem>
              </Select>
            </FormControl>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setFormOpen(false)}>Cancel</Button>
          {formData.status === "draft" && (
            <Button variant="outlined" onClick={handleCreate}>Save Draft</Button>
          )}
          <Button variant="contained" onClick={handleCreate}>
            {formData.status === "published" ? "Publish" : "Save Draft"}
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
