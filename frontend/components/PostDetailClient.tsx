"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Container, Typography, Box, Chip, Button, Dialog, DialogTitle,
  DialogContent, DialogActions, TextField, IconButton,
} from "@mui/material";
import FavoriteBorderIcon from "@mui/icons-material/FavoriteBorder";
import FavoriteIcon from "@mui/icons-material/Favorite";

interface Post {
  id: string;
  title: string;
  slug: string;
  content: string;
  authorId: string;
  status: string;
  publishDate: string;
  likeCount: number;
  createdAt: string;
  authorName?: string;
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
  const [editContent, setEditContent] = useState(post.content);
  const [editStatus, setEditStatus] = useState(post.status);
  const [liked, setLiked] = useState(false);
  const [likeCount, setLikeCount] = useState(post.likeCount);

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
      body: JSON.stringify({ title: editTitle, content: editContent, status: editStatus }),
    });
    if (res.ok) {
      setEditOpen(false);
      router.refresh();
    }
  }

  return (
    <>
      <Box sx={{ display: "flex", gap: 1, alignItems: "center", mb: 2 }}>
        {post.authorName && (
          <Typography variant="body2" color="text.secondary">
            By {post.authorName}
          </Typography>
        )}
        <Chip label={post.status} size="small" color="success" />
        <Typography variant="body2" color="text.secondary">
          {new Date(post.publishDate || post.createdAt).toLocaleDateString()}
        </Typography>
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

      <Dialog open={editOpen} onClose={() => setEditOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Edit Post</DialogTitle>
        <DialogContent>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
            <TextField label="Title" value={editTitle} onChange={(e) => setEditTitle(e.target.value)} required />
            <Typography variant="body2" color="text.secondary">Content</Typography>
            <Box sx={{ border: 1, borderColor: "divider", borderRadius: 1, p: 1 }}>
              <div
                contentEditable
                suppressContentEditableWarning
                style={{ minHeight: 120, outline: "none" }}
                dangerouslySetInnerHTML={{ __html: editContent }}
                onInput={(e) => setEditContent((e.target as HTMLElement).innerHTML)}
              />
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave}>Save</Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
