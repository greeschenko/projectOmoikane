"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Button, TextField, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Box, Alert, IconButton,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";

interface Category {
  id: string;
  name: string;
  slug: string;
  description: string;
}

export default function AdminCategories() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [formOpen, setFormOpen] = useState(false);
  const [formData, setFormData] = useState({ name: "", slug: "", description: "" });
  const [deleteTarget, setDeleteTarget] = useState<Category | null>(null);
  const [alert, setAlert] = useState<{ type: "success" | "error"; message: string } | null>(null);

  const fetchCategories = useCallback(async () => {
    const res = await fetch("/api/blog/categories");
    if (res.ok) setCategories(await res.json());
  }, []);

  useEffect(() => { fetchCategories(); }, [fetchCategories]);

  function generateSlug(name: string) {
    return name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "");
  }

  async function handleSave() {
    const res = await fetch("/api/blog/categories", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: formData.name,
        slug: formData.slug || generateSlug(formData.name),
        description: formData.description,
      }),
    });
    if (res.ok) {
      setFormOpen(false);
      setFormData({ name: "", slug: "", description: "" });
      setAlert({ type: "success", message: "Category created" });
      fetchCategories();
    } else {
      setAlert({ type: "error", message: "Failed to create category" });
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    const res = await fetch(`/api/blog/categories/${deleteTarget.id}`, { method: "DELETE" });
    if (res.ok) {
      setDeleteTarget(null);
      setAlert({ type: "success", message: "Category deleted" });
      fetchCategories();
    } else {
      setAlert({ type: "error", message: "Failed to delete category" });
    }
  }

  return (
    <Container maxWidth="lg" sx={{ mt: 4 }}>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
        <Typography variant="h4">Categories</Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => setFormOpen(true)}>
          New Category
        </Button>
      </Box>

      {alert && (
        <Alert severity={alert.type} sx={{ mb: 2 }} onClose={() => setAlert(null)}>
          {alert.message}
        </Alert>
      )}

      {categories.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: "center" }}>
          <Typography color="text.secondary">No categories yet. Create your first category!</Typography>
        </Paper>
      ) : (
        <Paper>
          <Box sx={{ display: "flex", flexDirection: "column" }}>
            {categories.map((cat) => (
              <Box
                key={cat.id}
                sx={{
                  display: "flex", alignItems: "center", gap: 2,
                  p: 2, borderBottom: "1px solid", borderColor: "divider",
                }}
              >
                <Box sx={{ flex: 1 }}>
                  <Typography variant="subtitle1">{cat.name}</Typography>
                  <Typography variant="body2" color="text.secondary">
                    /{cat.slug}{cat.description ? ` — ${cat.description}` : ""}
                  </Typography>
                </Box>
                <IconButton
                  onClick={() => setDeleteTarget(cat)}
                  size="small"
                  color="error"
                  data-testid={`delete-cat-${cat.id}`}
                >
                  <DeleteIcon />
                </IconButton>
              </Box>
            ))}
          </Box>
        </Paper>
      )}

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>New Category</DialogTitle>
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
            <TextField
              label="Description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              multiline
              rows={2}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setFormOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave}>Create</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Category</DialogTitle>
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
