"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Button, TextField, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Box, Select, MenuItem, InputLabel, FormControl, Alert,
  IconButton, Chip,
} from "@mui/material";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import AddIcon from "@mui/icons-material/Add";
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

export default function AdminBlog() {
  const [posts, setPosts] = useState<BlogPost[]>([]);
  const [formOpen, setFormOpen] = useState(false);
  const [editingPost, setEditingPost] = useState<BlogPost | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<BlogPost | null>(null);
  const [formData, setFormData] = useState({
    title: "", slug: "", content: "", status: "draft" as "draft" | "published",
  });
  const [alert, setAlert] = useState<{ type: "success" | "error"; message: string } | null>(null);

  const fetchPosts = useCallback(async () => {
    const res = await fetch("/api/blog/posts");
    if (res.ok) setPosts(await res.json());
  }, []);

  useEffect(() => { fetchPosts(); }, [fetchPosts]);

  function openCreate() {
    setEditingPost(null);
    setFormData({ title: "", slug: "", content: "", status: "published" });
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
    setFormOpen(true);
  }

  async function handleSave() {
    const url = editingPost
      ? `/api/blog/posts/${editingPost.id}`
      : "/api/blog/posts";
    const method = editingPost ? "PUT" : "POST";
    const res = await fetch(url, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(formData),
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
    const res = await fetch(`/api/blog/posts/${deleteTarget.id}`, { method: "DELETE" });
    if (res.ok) {
      setDeleteTarget(null);
      setAlert({ type: "success", message: "Post deleted" });
      fetchPosts();
    } else {
      setAlert({ type: "error", message: "Failed to delete post" });
    }
  }

  return (
    <Container maxWidth="lg" sx={{ mt: 4 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
        <Typography variant="h4">Blog Posts</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={openCreate}>
          New Blog Post
        </Button>
      </Box>

      {alert && (
        <Alert severity={alert.type} sx={{ mb: 2 }} onClose={() => setAlert(null)}>
          {alert.message}
        </Alert>
      )}

      {posts.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: "center" }}>
          <Typography color="text.secondary">
            No blog posts yet. Create your first post!
          </Typography>
        </Paper>
      ) : (
        <Paper>
          <Box sx={{ display: "flex", flexDirection: "column" }}>
            {posts.map((post) => (
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
                <IconButton onClick={() => openEdit(post)} size="small">
                  <EditIcon />
                </IconButton>
                <IconButton onClick={() => setDeleteTarget(post)} size="small" color="error">
                  <DeleteIcon />
                </IconButton>
              </Box>
            ))}
          </Box>
        </Paper>
      )}

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editingPost ? "Edit Post" : "New Blog Post"}</DialogTitle>
        <DialogContent>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
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
              required
            />
            <Typography variant="body2" color="text.secondary">Content</Typography>
            <RichTextEditor
              content={formData.content}
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
          {!editingPost && formData.status === "draft" && (
            <Button variant="outlined" onClick={() => handleSave()}>
              Save Draft
            </Button>
          )}
          <Button variant="contained" onClick={handleSave}>
            {editingPost ? "Update" : "Publish"}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Post</DialogTitle>
        <DialogContent>
          Are you sure you want to delete "{deleteTarget?.title}"?
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={handleDelete}>Delete</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
