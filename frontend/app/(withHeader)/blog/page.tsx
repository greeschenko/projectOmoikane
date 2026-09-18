"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import {
  Container, Typography, Box, Card, CardContent, Button, Dialog,
  DialogTitle, DialogContent, DialogActions, TextField, FormControl,
  InputLabel, Select, MenuItem, Alert, Chip, Autocomplete, Grid,
} from "@mui/material";
import RichTextEditor from "@/components/RichTextEditor";

interface BlogPost {
  id: string;
  title: string;
  slug: string;
  excerpt: string;
  content: string;
  authorId: string;
  authorName?: string;
  status: string;
  publishDate: string;
  featuredImage: string;
  tags: string[];
  categoryId: string | null;
  likeCount: number;
  createdAt: string;
}

interface Category {
  id: string;
  name: string;
  slug: string;
}

interface Tag {
  id: string;
  name: string;
  slug: string;
}

function generateSlug(name: string) {
  return name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "");
}

export default function BlogListPage() {
  const [posts, setPosts] = useState<BlogPost[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [blogEnabled, setBlogEnabled] = useState(true);
  const [loaded, setLoaded] = useState(false);
  const [currentUser, setCurrentUser] = useState<{ id: string; role: string } | null>(null);
  const [myPostsOnly, setMyPostsOnly] = useState(false);
  const [categoryFilter, setCategoryFilter] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [formData, setFormData] = useState({
    title: "", slug: "", content: "", status: "published" as "draft" | "published",
    tags: [] as string[], categoryId: "" as string,
  });
  const [alert, setAlert] = useState<{ type: "success" | "error"; message: string } | null>(null);

  function loadData() {
    Promise.all([
      fetch("/api/settings").then((r) => r.json()),
      fetch("/api/blog/posts").then((r) => r.json()),
      fetch("/api/blog/categories").then((r) => r.ok ? r.json() : []),
      fetch("/api/blog/tags").then((r) => r.ok ? r.json() : []),
      fetch("/api/settings/profile").then((r) => r.ok ? r.json() : null),
    ])
      .then(([settings, data, categoryData, tagData, profile]) => {
        if (settings.blogEnabled !== undefined) setBlogEnabled(settings.blogEnabled);
        setPosts((data as BlogPost[]).filter((p) => p.status === "published"));
        setCategories(categoryData as Category[]);
        setTags(tagData as Tag[]);
        if (profile) setCurrentUser(profile);
      })
      .catch(() => {})
      .finally(() => setLoaded(true));
  }

  useEffect(() => { loadData(); }, []);

  const displayedPosts = (myPostsOnly && currentUser
    ? currentUser.role === "admin" ? posts : posts.filter((p) => p.authorId === currentUser.id)
    : posts
  ).filter((p) => !categoryFilter || p.categoryId === categoryFilter);

  async function handleCreate() {
    const slug = formData.slug || generateSlug(formData.title) || "post";
    const res = await fetch("/api/blog/posts", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ...formData,
        slug,
        excerpt: "",
        categoryId: formData.categoryId || null,
      }),
    });
    if (res.ok) {
      setFormOpen(false);
      setFormData({ title: "", slug: "", content: "", status: "published", tags: [], categoryId: "" });
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
        <Box sx={{ display: "flex", gap: 1, flexWrap: "wrap", alignItems: "center" }}>
          {categories.length > 0 && (
            <FormControl size="small" sx={{ minWidth: 150 }}>
              <InputLabel id="category-filter-label">Category</InputLabel>
              <Select
                labelId="category-filter-label"
                label="Category"
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
              >
                <MenuItem value="">All</MenuItem>
                {categories.map((c) => (
                  <MenuItem key={c.id} value={c.id}>{c.name}</MenuItem>
                ))}
              </Select>
            </FormControl>
          )}
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
          {displayedPosts.map((post) => {
            const categoryName = post.categoryId
              ? categories.find((c) => c.id === post.categoryId)?.name
              : undefined;
            return (
              <Card key={post.id} component={Link} href={`/blog/${post.slug}`} sx={{ textDecoration: "none" }}>
                <CardContent>
                  <Typography variant="h5" component="h2">{post.title}</Typography>
                  {(categoryName || (post.tags && post.tags.length > 0)) && (
                    <Box sx={{ mt: 1, display: "flex", gap: 0.5, flexWrap: "wrap", alignItems: "center" }}>
                      {categoryName && (
                        <Chip label={categoryName} size="small" color="primary" />
                      )}
                      {post.tags?.slice(0, 5).map((tag) => (
                        <Chip key={tag} label={tag} size="small" variant="outlined" />
                      ))}
                    </Box>
                  )}
                  {post.excerpt && (
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                      {post.excerpt}
                    </Typography>
                  )}
                  <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: "block" }}>
                    {post.authorName ? `by ${post.authorName} · ` : ""}
                    {new Date(post.publishDate || post.createdAt).toLocaleDateString()}
                  </Typography>
                </CardContent>
              </Card>
            );
          })}
        </Box>
      )}

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="lg" fullWidth>
        <DialogTitle>New Blog Post</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0 }}>
            <Grid size={{ xs: 12, md: 8.4 }}>
              <Box sx={{ mb: 1 }}>
                <Typography variant="body2" color="text.secondary">Content</Typography>
              </Box>
              <RichTextEditor
                value={formData.content}
                onChange={(html: string) => setFormData({ ...formData, content: html })}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3.6 }}>
              <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <TextField
                  label="Title"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  required
                />
                <TextField
                  label="Slug"
                  value={formData.slug}
                  onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                  helperText="Leave blank to auto-generate"
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
                <Autocomplete
                  multiple
                  freeSolo
                  options={tags.map((t) => t.name)}
                  value={formData.tags}
                  onChange={(_, newValue) => setFormData({ ...formData, tags: newValue })}
                  renderInput={(params) => <TextField {...params} label="Tags" placeholder="Add tag" />}
                  renderTags={(value, getTagProps) =>
                    value.map((option, index) => (
                      <Chip variant="outlined" label={option} size="small" {...getTagProps({ index })} key={option} />
                    ))
                  }
                />
                <FormControl fullWidth>
                  <InputLabel>Category</InputLabel>
                  <Select
                    label="Category"
                    value={formData.categoryId}
                    onChange={(e) => setFormData({ ...formData, categoryId: e.target.value })}
                  >
                    <MenuItem value="">None</MenuItem>
                    {categories.map((cat) => (
                      <MenuItem key={cat.id} value={cat.id}>{cat.name}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Box>
            </Grid>
          </Grid>
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