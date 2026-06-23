"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Button, TextField, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Box, Alert, IconButton, CircularProgress,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";

interface Tag {
  id: string;
  name: string;
  slug: string;
}

export default function AdminTags() {
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [formData, setFormData] = useState({ name: "", slug: "" });
  const [deleteTarget, setDeleteTarget] = useState<Tag | null>(null);
  const [alert, setAlert] = useState<{ type: "success" | "error"; message: string } | null>(null);

  const fetchTags = useCallback(async () => {
    setLoading(true);
    const res = await fetch("/api/blog/tags");
    if (res.ok) setTags(await res.json());
    setLoading(false);
  }, []);

  useEffect(() => { fetchTags(); }, [fetchTags]);

  function generateSlug(name: string) {
    return name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "");
  }

  async function handleSave() {
    const res = await fetch("/api/blog/tags", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: formData.name, slug: formData.slug || generateSlug(formData.name) }),
    });
    if (res.ok) {
      setFormOpen(false);
      setFormData({ name: "", slug: "" });
      setAlert({ type: "success", message: "Tag created" });
      fetchTags();
    } else {
      setAlert({ type: "error", message: "Failed to create tag" });
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    const res = await fetch(`/api/blog/tags/${deleteTarget.id}`, { method: "DELETE" });
    if (res.ok) {
      setDeleteTarget(null);
      setAlert({ type: "success", message: "Tag deleted" });
      fetchTags();
    } else {
      setAlert({ type: "error", message: "Failed to delete tag" });
    }
  }

  return (
    <Container maxWidth="lg" sx={{ mt: 4 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
        <Typography variant="h4">Tags</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => setFormOpen(true)}>
          New Tag
        </Button>
      </Box>

      {alert && (
        <Alert severity={alert.type} sx={{ mb: 2 }} onClose={() => setAlert(null)}>
          {alert.message}
        </Alert>
      )}

      {loading ? (
        <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
          <CircularProgress />
        </Box>
      ) : tags.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: "center" }}>
          <Typography color="text.secondary">No tags yet. Create your first tag!</Typography>
        </Paper>
      ) : (
        <Paper>
          <Box sx={{ display: "flex", flexDirection: "column" }}>
            {tags.map((tag) => (
              <Box
                key={tag.id}
                sx={{
                  display: "flex", alignItems: "center", gap: 2,
                  p: 2, borderBottom: "1px solid", borderColor: "divider",
                }}
                data-testid={`tag-row-${tag.id}`}
              >
                <Box sx={{ flex: 1 }}>
                  <Typography variant="subtitle1">{tag.name}</Typography>
                  <Typography variant="body2" color="text.secondary">
                    /{tag.slug}
                  </Typography>
                </Box>
                <IconButton
                  onClick={() => setDeleteTarget(tag)}
                  size="small"
                  color="error"
                  data-testid={`delete-tag-${tag.id}`}
                >
                  <DeleteIcon />
                </IconButton>
              </Box>
            ))}
          </Box>
        </Paper>
      )}

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>New Tag</DialogTitle>
        <DialogContent>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
            <TextField
              label="Name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
            <TextField
              label="Slug"
              value={formData.slug}
              onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
              helperText="Leave blank to auto-generate"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setFormOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave}>Create</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Tag</DialogTitle>
        <DialogContent>
          Are you sure you want to delete "{deleteTarget?.name}"?
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={handleDelete}>Delete</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
