"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Typography, Box, Chip, Button, Dialog, DialogTitle,
  DialogContent, DialogActions, TextField, IconButton, FormControl,
  InputLabel, Select, MenuItem, Autocomplete, Grid,
} from "@mui/material";
import FavoriteBorderIcon from "@mui/icons-material/FavoriteBorder";
import FavoriteIcon from "@mui/icons-material/Favorite";
import RichTextEditor from "@/components/RichTextEditor";

interface Post {
  id: string;
  title: string;
  slug: string;
  content: string;
  authorId: string;
  authorName?: string;
  status: string;
  publishDate: string;
  likeCount: number;
  tags?: string[];
  categoryId?: string | null;
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

export default function PostDetailClient({
  post,
  canEdit,
}: {
  post: Post;
  canEdit: boolean;
}) {
  const router = useRouter();
  const [editOpen, setEditOpen] = useState(false);
  const [editTitle, setEditTitle] = useState(post.title);
  const [editSlug, setEditSlug] = useState(post.slug);
  const [editContent, setEditContent] = useState(post.content);
  const [editStatus, setEditStatus] = useState(post.status || "draft");
  const [editTags, setEditTags] = useState<string[]>(post.tags || []);
  const [editCategoryId, setEditCategoryId] = useState(post.categoryId || "");
  const [categories, setCategories] = useState<Category[]>([]);
  const [allTags, setAllTags] = useState<Tag[]>([]);
  const [liked, setLiked] = useState(false);
  const [likeCount, setLikeCount] = useState(post.likeCount);

  useEffect(() => {
    fetch("/api/blog/categories")
      .then((r) => (r.ok ? r.json() : []))
      .then(setCategories)
      .catch(() => {});
    fetch("/api/blog/tags")
      .then((r) => (r.ok ? r.json() : []))
      .then(setAllTags)
      .catch(() => {});
  }, []);

  async function handleLike() {
    try {
      const res = await fetch(`/api/blog/posts/${post.id}/like`, { method: "POST" });
      if (res.ok) {
        const data = await res.json();
        setLiked(data.liked);
        setLikeCount(data.count);
      }
    } catch { /* ignore */ }
  }

  async function handleSave() {
    const res = await fetch(`/api/blog/posts/${post.id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        title: editTitle,
        slug: editSlug,
        content: editContent,
        status: editStatus,
        tags: editTags,
        categoryId: editCategoryId || null,
      }),
    });
    if (res.ok) {
      setEditOpen(false);
      router.refresh();
    }
  }

  const categoryName = post.categoryId
    ? categories.find((c) => c.id === post.categoryId)?.name
    : undefined;

  return (
    <>
      <Box sx={{ display: "flex", gap: 1, alignItems: "center", flexWrap: "wrap", mb: 2 }}>
        {post.authorName && (
          <Typography variant="body2" color="text.secondary">
            By {post.authorName}
          </Typography>
        )}
        <Chip label={post.status} size="small" color="success" />
        <Typography variant="body2" color="text.secondary">
          {new Date(post.publishDate || post.createdAt).toLocaleDateString()}
        </Typography>
        {categoryName && <Chip label={categoryName} size="small" color="primary" />}
        {post.tags?.slice(0, 5).map((tag) => (
          <Chip key={tag} label={tag} size="small" variant="outlined" />
        ))}
        <IconButton onClick={handleLike} size="small" color={liked ? "error" : "default"} aria-label={liked ? "Unlike" : "Like"}>
          {liked ? <FavoriteIcon fontSize="small" /> : <FavoriteBorderIcon fontSize="small" />}
        </IconButton>
        <Typography variant="body2" color="text.secondary">
          {likeCount} {likeCount === 1 ? "like" : "likes"}
        </Typography>
        {canEdit && (
          <Button variant="outlined" size="small" onClick={() => setEditOpen(true)}>
            Edit
          </Button>
        )}
      </Box>
      <Box sx={{ mt: 2 }} dangerouslySetInnerHTML={{ __html: post.content }} />

      <Dialog open={editOpen} onClose={() => setEditOpen(false)} maxWidth="lg" fullWidth>
        <DialogTitle>Edit Post</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0 }}>
            <Grid size={{ xs: 12, md: 8.4 }}>
              <Box sx={{ mb: 1 }}>
                <Typography variant="body2" color="text.secondary">Content</Typography>
              </Box>
              <RichTextEditor
                value={editContent}
                onChange={(html: string) => setEditContent(html)}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3.6 }}>
              <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <TextField label="Title" value={editTitle} onChange={(e) => setEditTitle(e.target.value)} required />
                <TextField label="Slug" value={editSlug} onChange={(e) => setEditSlug(e.target.value)} required />
                <FormControl fullWidth>
                  <InputLabel>Status</InputLabel>
                  <Select
                    label="Status"
                    value={editStatus}
                    onChange={(e) => setEditStatus(e.target.value)}
                  >
                    <MenuItem value="draft">Draft</MenuItem>
                    <MenuItem value="published">Published</MenuItem>
                  </Select>
                </FormControl>
                <Autocomplete
                  multiple
                  freeSolo
                  options={allTags.map((t) => t.name)}
                  value={editTags}
                  onChange={(_, newValue) => setEditTags(newValue)}
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
                    value={editCategoryId}
                    onChange={(e) => setEditCategoryId(e.target.value)}
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
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave}>Save</Button>
        </DialogActions>
      </Dialog>
    </>
  );
}