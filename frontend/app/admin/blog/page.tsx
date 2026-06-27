"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Button, TextField, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Box, Select, MenuItem, InputLabel, FormControl, Alert,
  IconButton, Chip, Tabs, Tab, CircularProgress, FormHelperText,
  FormControlLabel, Switch,
} from "@mui/material";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import AddIcon from "@mui/icons-material/Add";
import VisibilityIcon from "@mui/icons-material/Visibility";
import Link from "next/link";
import RichTextEditor from "@/components/RichTextEditor";

interface BlogPost {
  id: string;
  title: string;
  slug: string;
  content: string;
  excerpt: string;
  authorId: string;
  status: "draft" | "published";
  publishDate: string;
  featuredImage: string;
  tags: string[];
  categoryId: string | null;
  likeCount: number;
  createdAt: string;
  updatedAt: string;
}

interface Tag {
  id: string;
  name: string;
  slug: string;
}

interface Category {
  id: string;
  name: string;
  slug: string;
  description: string;
}

function generateSlug(name: string) {
  return name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "");
}

export default function AdminBlog() {
  const [tabIndex, setTabIndex] = useState(0);
  const [posts, setPosts] = useState<BlogPost[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [editingPost, setEditingPost] = useState<BlogPost | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<BlogPost | Tag | Category | null>(null);
  const [deleteType, setDeleteType] = useState<"post" | "tag" | "category">("post");
  const [formData, setFormData] = useState({
    title: "", slug: "", content: "", status: "draft" as "draft" | "published",
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [tagFormOpen, setTagFormOpen] = useState(false);
  const [tagFormData, setTagFormData] = useState({ name: "", slug: "" });
  const [catFormOpen, setCatFormOpen] = useState(false);
  const [catFormData, setCatFormData] = useState({ name: "", slug: "", description: "" });
  const [blogEnabled, setBlogEnabled] = useState(true);
  const [alert, setAlert] = useState<{ type: "success" | "error"; message: string } | null>(null);

  useEffect(() => {
    fetch("/api/settings")
      .then((r) => r.json())
      .then((data) => {
        if (data.blogEnabled !== undefined) setBlogEnabled(data.blogEnabled);
      })
      .catch(() => {});
  }, []);

  const fetchPosts = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/blog/posts");
      if (res.ok) setPosts(await res.json());
    } catch { /* ignore */ }
    setLoading(false);
  }, []);

  const fetchTags = useCallback(async () => {
    try {
      const res = await fetch("/api/blog/tags");
      if (res.ok) setTags(await res.json());
    } catch { /* ignore */ }
  }, []);

  const fetchCategories = useCallback(async () => {
    try {
      const res = await fetch("/api/blog/categories");
      if (res.ok) setCategories(await res.json());
    } catch { /* ignore */ }
  }, []);

  useEffect(() => { fetchPosts(); fetchTags(); fetchCategories(); }, [fetchPosts, fetchTags, fetchCategories]);

  const filteredPosts = posts.filter((p) =>
    !search || p.title.toLowerCase().includes(search.toLowerCase())
  );

  function openCreate() {
    setEditingPost(null);
    setFormData({ title: "", slug: "", content: "", status: "published" });
    setFormErrors({});
    setFormOpen(true);
  }

  function openEdit(post: BlogPost) {
    setEditingPost(post);
    setFormData({
      title: post.title,
      slug: post.slug,
      content: post.content,
      status: post.status,
    });
    setFormErrors({});
    setFormOpen(true);
  }

  function validatePostForm(slug: string) {
    const errors: Record<string, string> = {};
    if (!formData.title.trim()) errors.title = "Title is required";
    if (!slug.trim()) errors.slug = "Slug is required";
    if (!formData.content.trim()) errors.content = "Content is required";
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function handleSave() {
    const slug = formData.slug || generateSlug(formData.title);
    if (!slug) {
      setFormErrors({ slug: "Slug is required" });
      return;
    }
    if (!validatePostForm(slug)) return;
    const payload = { ...formData, slug };
    const url = editingPost
      ? `/api/blog/posts/${editingPost.id}`
      : "/api/blog/posts";
    const method = editingPost ? "PUT" : "POST";
    const res = await fetch(url, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (res.ok) {
      setFormOpen(false);
      setAlert({ type: "success", message: editingPost ? "Post updated" : "Post created" });
      fetchPosts();
    } else {
      setAlert({ type: "error", message: "Failed to save post" });
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    if (deleteType === "post") {
      const res = await fetch(`/api/blog/posts/${deleteTarget.id}`, { method: "DELETE" });
      if (res.ok) {
        setDeleteTarget(null);
        setAlert({ type: "success", message: "Post deleted" });
        fetchPosts();
      } else {
        setAlert({ type: "error", message: "Failed to delete post" });
      }
    } else if (deleteType === "tag") {
      const res = await fetch(`/api/blog/tags/${deleteTarget.id}`, { method: "DELETE" });
      if (res.ok) {
        setDeleteTarget(null);
        setAlert({ type: "success", message: "Tag deleted" });
        fetchTags();
      } else {
        setAlert({ type: "error", message: "Failed to delete tag" });
      }
    } else if (deleteType === "category") {
      const res = await fetch(`/api/blog/categories/${deleteTarget.id}`, { method: "DELETE" });
      if (res.ok) {
        setDeleteTarget(null);
        setAlert({ type: "success", message: "Category deleted" });
        fetchCategories();
      } else {
        setAlert({ type: "error", message: "Failed to delete category" });
      }
    }
  }

  async function handleBlogToggle(checked: boolean) {
    const res = await fetch("/api/settings", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ blogEnabled: checked }),
    });
    if (res.ok) {
      setBlogEnabled(checked);
      setAlert({ type: "success", message: checked ? "Blog enabled" : "Blog disabled" });
    } else {
      setAlert({ type: "error", message: "Failed to update setting" });
    }
  }

  async function handleTagSave() {
    const res = await fetch("/api/blog/tags", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: tagFormData.name, slug: tagFormData.slug || generateSlug(tagFormData.name) }),
    });
    if (res.ok) {
      setTagFormOpen(false);
      setTagFormData({ name: "", slug: "" });
      setAlert({ type: "success", message: "Tag created" });
      fetchTags();
    } else {
      setAlert({ type: "error", message: "Failed to create tag" });
    }
  }

  async function handleCategorySave() {
    const res = await fetch("/api/blog/categories", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: catFormData.name,
        slug: catFormData.slug || generateSlug(catFormData.name),
        description: catFormData.description,
      }),
    });
    if (res.ok) {
      setCatFormOpen(false);
      setCatFormData({ name: "", slug: "", description: "" });
      setAlert({ type: "success", message: "Category created" });
      fetchCategories();
    } else {
      setAlert({ type: "error", message: "Failed to create category" });
    }
  }

  return (
    <Container maxWidth="lg" sx={{ mt: 4 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3, flexWrap: "wrap", gap: 1 }}>
        <Typography variant="h4">Blog Posts</Typography>
        <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
          <FormControlLabel
            control={<Switch checked={blogEnabled} onChange={(e) => handleBlogToggle(e.target.checked)} />}
            label="Enable Blog"
          />
          <Button variant="contained" startIcon={<AddIcon />} onClick={openCreate}>
            New Blog Post
          </Button>
        </Box>
      </Box>

      {alert && (
        <Alert severity={alert.type} sx={{ mb: 2 }} onClose={() => setAlert(null)}>
          {alert.message}
        </Alert>
      )}

      <Tabs value={tabIndex} onChange={(_, i) => setTabIndex(i)} sx={{ mb: 2 }}>
        <Tab label="Posts" />
        <Tab label="Tags" />
        <Tab label="Categories" />
      </Tabs>

      {tabIndex === 0 && (
        <>
          <TextField
            placeholder="Search posts..."
            size="small"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            sx={{ mb: 2, maxWidth: 320 }}
          />

          {loading ? (
            <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
              <CircularProgress />
            </Box>
          ) : filteredPosts.length === 0 ? (
            <Paper sx={{ p: 4, textAlign: "center" }}>
              <Typography color="text.secondary">
                {search ? "No posts match your search." : "No blog posts yet. Create your first post!"}
              </Typography>
            </Paper>
          ) : (
            <Paper>
              <Box sx={{ display: "flex", flexDirection: "column" }}>
                {filteredPosts.map((post) => (
                  <Box
                    key={post.id}
                    sx={{
                      display: "flex", alignItems: "center", gap: 2,
                      p: 2, borderBottom: "1px solid", borderColor: "divider",
                    }}
                  >
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="subtitle1">{post.title}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        /blog/{post.slug}
                      </Typography>
                    </Box>
                    <Chip
                      label={post.status}
                      color={post.status === "published" ? "success" : "default"}
                      size="small"
                    />
                    <IconButton component={Link} href={`/blog/${post.slug}`} target="_blank" size="small" aria-label="view">
                      <VisibilityIcon />
                    </IconButton>
                    <IconButton onClick={() => openEdit(post)} size="small">
                      <EditIcon />
                    </IconButton>
                    <IconButton onClick={() => { setDeleteTarget(post); setDeleteType("post"); }} size="small" color="error">
                      <DeleteIcon />
                    </IconButton>
                  </Box>
                ))}
              </Box>
            </Paper>
          )}
        </>
      )}

      {tabIndex === 1 && (
        <>
          <Box sx={{ display: "flex", justifyContent: "flex-end", mb: 2 }}>
            <Button variant="contained" startIcon={<AddIcon />} onClick={() => setTagFormOpen(true)}>
              New Tag
            </Button>
          </Box>
          {tags.length === 0 ? (
            <Paper sx={{ p: 4, textAlign: "center" }}>
              <Typography color="text.secondary">No tags yet. Create your first tag!</Typography>
            </Paper>
          ) : (
            <Paper>
              <Box sx={{ display: "flex", flexDirection: "column" }}>
                {tags.map((tag) => (
                  <Box key={tag.id} sx={{ display: "flex", alignItems: "center", gap: 2, p: 2, borderBottom: "1px solid", borderColor: "divider" }}>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="subtitle1">{tag.name}</Typography>
                      <Typography variant="body2" color="text.secondary">/{tag.slug}</Typography>
                    </Box>
                    <IconButton onClick={() => { setDeleteTarget(tag); setDeleteType("tag"); }} size="small" color="error">
                      <DeleteIcon />
                    </IconButton>
                  </Box>
                ))}
              </Box>
            </Paper>
          )}
        </>
      )}

      {tabIndex === 2 && (
        <>
          <Box sx={{ display: "flex", justifyContent: "flex-end", mb: 2 }}>
            <Button variant="contained" startIcon={<AddIcon />} onClick={() => setCatFormOpen(true)}>
              New Category
            </Button>
          </Box>
          {categories.length === 0 ? (
            <Paper sx={{ p: 4, textAlign: "center" }}>
              <Typography color="text.secondary">No categories yet. Create your first category!</Typography>
            </Paper>
          ) : (
            <Paper>
              <Box sx={{ display: "flex", flexDirection: "column" }}>
                {categories.map((cat) => (
                  <Box key={cat.id} sx={{ display: "flex", alignItems: "center", gap: 2, p: 2, borderBottom: "1px solid", borderColor: "divider" }}>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="subtitle1">{cat.name}</Typography>
                      <Typography variant="body2" color="text.secondary">/{cat.slug}{cat.description ? ` — ${cat.description}` : ""}</Typography>
                    </Box>
                    <IconButton onClick={() => { setDeleteTarget(cat); setDeleteType("category"); }} size="small" color="error">
                      <DeleteIcon />
                    </IconButton>
                  </Box>
                ))}
              </Box>
            </Paper>
          )}
        </>
      )}

      {/* Post Form Dialog */}
      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editingPost ? "Edit Post" : "New Blog Post"}</DialogTitle>
        <DialogContent>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
            <TextField
              label="Title"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              error={!!formErrors.title}
              helperText={formErrors.title}
              required
            />
            <TextField
              label="Slug"
              value={formData.slug}
              onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
              error={!!formErrors.slug}
              helperText={formErrors.slug}
              required
            />
            <Typography variant="body2" color="text.secondary">Content</Typography>
            <RichTextEditor
              value={formData.content}
              onChange={(html: string) => setFormData({ ...formData, content: html })}
            />
            {formErrors.content && <FormHelperText error>{formErrors.content}</FormHelperText>}
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
          {!editingPost && formData.status === "draft" && (
            <Button variant="outlined" onClick={handleSave}>
              Save Draft
            </Button>
          )}
          <Button variant="contained" onClick={handleSave}>
            {editingPost ? "Update" : "Publish"}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Tag Form Dialog */}
      <Dialog open={tagFormOpen} onClose={() => setTagFormOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>New Tag</DialogTitle>
        <DialogContent>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
            <TextField label="Name" value={tagFormData.name} onChange={(e) => setTagFormData({ ...tagFormData, name: e.target.value })} required />
            <TextField label="Slug" value={tagFormData.slug} onChange={(e) => setTagFormData({ ...tagFormData, slug: e.target.value })} helperText="Leave blank to auto-generate" />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setTagFormOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleTagSave}>Create</Button>
        </DialogActions>
      </Dialog>

      {/* Category Form Dialog */}
      <Dialog open={catFormOpen} onClose={() => setCatFormOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>New Category</DialogTitle>
        <DialogContent>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
            <TextField label="Name" value={catFormData.name} onChange={(e) => setCatFormData({ ...catFormData, name: e.target.value })} required />
            <TextField label="Slug" value={catFormData.slug} onChange={(e) => setCatFormData({ ...catFormData, slug: e.target.value })} helperText="Leave blank to auto-generate" />
            <TextField label="Description" value={catFormData.description} onChange={(e) => setCatFormData({ ...catFormData, description: e.target.value })} multiline rows={2} />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCatFormOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCategorySave}>Create</Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation */}
      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete {deleteType === "post" ? "Post" : deleteType === "tag" ? "Tag" : "Category"}</DialogTitle>
        <DialogContent>
          Are you sure you want to delete &quot;{deleteTarget?.name || (deleteTarget as BlogPost)?.title}&quot;?
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={handleDelete}>Delete</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
