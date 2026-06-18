"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Button, TextField, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Box, FormControlLabel, Switch, Select, MenuItem, InputLabel, FormControl, Alert,
  IconButton,
} from "@mui/material";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";

interface Page {
  id: string;
  title: string;
  slug: string;
  content: string;
  metaTitle?: string;
  metaDescription?: string;
  metaKeywords?: string;
  parentId: string | null;
  status: "draft" | "published";
  inMenu: boolean;
}

export default function AdminPages() {
  const [pages, setPages] = useState<Page[]>([]);
  const [formOpen, setFormOpen] = useState(false);
  const [editingPage, setEditingPage] = useState<Page | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Page | null>(null);
  const [formData, setFormData] = useState({
    title: "", slug: "", content: "",
    metaTitle: "", metaDescription: "", metaKeywords: "",
    parentId: "", status: "draft" as "draft" | "published", inMenu: false,
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});

  const fetchPages = useCallback(async () => {
    const res = await fetch("/api/pages");
    if (res.ok) setPages(await res.json());
  }, []);

  useEffect(() => { fetchPages(); }, [fetchPages]);

  const rootPages = pages.filter((p) => !p.parentId);

  function renderTree(parentId: string | null, depth = 0): Page[] {
    return pages
      .filter((p) => p.parentId === parentId)
      .sort((a, b) => a.slug.localeCompare(b.slug))
      .flatMap((p) => [p, ...renderTree(p.id, depth + 1)]);
  }

  const treeItems = renderTree(null);

  function openCreate(parentId?: string) {
    setEditingPage(null);
    setFormData({ title: "", slug: "", content: "", metaTitle: "", metaDescription: "", metaKeywords: "", parentId: parentId || "", status: "draft", inMenu: false });
    setFormErrors({});
    setFormOpen(true);
  }

  function openEdit(page: Page) {
    setEditingPage(page);
    setFormData({
      title: page.title,
      slug: page.slug,
      content: page.content,
      metaTitle: page.metaTitle || "",
      metaDescription: page.metaDescription || "",
      metaKeywords: page.metaKeywords || "",
      parentId: page.parentId || "",
      status: page.status || "draft",
      inMenu: page.inMenu || false,
    });
    setFormErrors({});
    setFormOpen(true);
  }

  function validateForm() {
    const errors: Record<string, string> = {};
    if (!formData.title.trim()) errors.title = "Title is required";
    if (!formData.slug.trim()) errors.slug = "Slug is required";
    if (!formData.content.trim()) errors.content = "Content is required";
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function handleSubmit() {
    if (!validateForm()) return;
    const url = editingPage ? `/api/pages/${editingPage.id}` : "/api/pages";
    const method = editingPage ? "PUT" : "POST";
    const body = { ...formData, parentId: formData.parentId || null };
    const res = await fetch(url, { method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
    if (res.ok) {
      setFormOpen(false);
      fetchPages();
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return;
    await fetch(`/api/pages/${deleteTarget.id}`, { method: "DELETE" });
    setDeleteTarget(null);
    fetchPages();
  }

  return (
    <Container>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
        <Typography variant="h4" component="h1">Pages</Typography>
        <Button variant="contained" onClick={() => openCreate()}>New Page</Button>
      </Box>
      <Paper sx={{ p: 2 }}>
        {treeItems.length === 0 ? (
          <Typography color="text.secondary">No pages yet.</Typography>
        ) : (
          <ul style={{ listStyle: "none", padding: 0 }}>
            {rootPages.map((page) => (
              <PageTreeItem
                key={page.id}
                page={page}
                pages={pages}
                depth={0}
                onEdit={openEdit}
                onDelete={setDeleteTarget}
              />
            ))}
          </ul>
        )}
      </Paper>

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editingPage ? "Edit Page" : "New Page"}</DialogTitle>
        <DialogContent>
          <TextField label="Title" fullWidth margin="dense" value={formData.title}
            onChange={(e) => setFormData({ ...formData, title: e.target.value })}
            error={!!formErrors.title} helperText={formErrors.title} required />
          <TextField label="Slug" fullWidth margin="dense" value={formData.slug}
            onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
            error={!!formErrors.slug} helperText={formErrors.slug} required />
          <TextField label="Content" fullWidth margin="dense" multiline rows={4} value={formData.content}
            onChange={(e) => setFormData({ ...formData, content: e.target.value })}
            error={!!formErrors.content} helperText={formErrors.content} required />
          <TextField label="Meta Title" fullWidth margin="dense" value={formData.metaTitle}
            onChange={(e) => setFormData({ ...formData, metaTitle: e.target.value })} />
          <TextField label="Meta Description" fullWidth margin="dense" value={formData.metaDescription}
            onChange={(e) => setFormData({ ...formData, metaDescription: e.target.value })} />
          <TextField label="Meta Keywords" fullWidth margin="dense" value={formData.metaKeywords}
            onChange={(e) => setFormData({ ...formData, metaKeywords: e.target.value })} />
          <FormControl fullWidth margin="dense">
            <InputLabel>Parent Page</InputLabel>
            <Select label="Parent Page" value={formData.parentId}
              onChange={(e) => setFormData({ ...formData, parentId: e.target.value })}>
              <MenuItem value="">None (root page)</MenuItem>
              {pages
                .filter((p) => p.id !== editingPage?.id)
                .map((p) => (
                  <MenuItem key={p.id} value={p.id}>{p.title}</MenuItem>
                ))}
            </Select>
          </FormControl>
          <FormControl fullWidth margin="dense">
            <InputLabel>Status</InputLabel>
            <Select label="Status" value={formData.status}
              onChange={(e) => setFormData({ ...formData, status: e.target.value as "draft" | "published" })}>
              <MenuItem value="draft">Draft</MenuItem>
              <MenuItem value="published">Published</MenuItem>
            </Select>
          </FormControl>
          <FormControlLabel
            control={<Switch checked={formData.inMenu}
              onChange={(e) => setFormData({ ...formData, inMenu: e.target.checked })} />}
            label="Show in menu"
            sx={{ mt: 1 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setFormOpen(false)}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained">{editingPage ? "Save" : "Create"}</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <Typography>Are you sure you want to delete {deleteTarget?.title}?</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button onClick={confirmDelete} color="error" variant="contained">Delete</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}

function PageTreeItem({
  page, pages, depth, onEdit, onDelete,
}: {
  page: Page;
  pages: Page[];
  depth: number;
  onEdit: (p: Page) => void;
  onDelete: (p: Page) => void;
}) {
  const children = pages.filter((p) => p.parentId === page.id);

  function buildViewUrl(p: Page): string {
    const segments: string[] = [p.slug];
    let current = p;
    while (current.parentId) {
      const parent = pages.find((pp) => pp.id === current.parentId);
      if (!parent) break;
      segments.unshift(parent.slug);
      current = parent;
    }
    return "/pages/" + segments.join("/");
  }

  return (
    <li style={{ marginLeft: depth * 24 }}>
      <Box sx={{ display: "flex", alignItems: "center", gap: 1, py: 0.5 }}>
        <Typography sx={{ flexGrow: 1 }}>{page.title}</Typography>
        <IconButton
          size="small"
          onClick={() => window.open(buildViewUrl(page), '_blank', 'noopener')}
          aria-label="view"
          data-href={buildViewUrl(page)}
        >
          <OpenInNewIcon fontSize="small" />
        </IconButton>
        <IconButton size="small" onClick={() => onEdit(page)} aria-label="edit">
          <EditIcon fontSize="small" />
        </IconButton>
        <IconButton size="small" onClick={() => onDelete(page)} aria-label="delete">
          <DeleteIcon fontSize="small" />
        </IconButton>
      </Box>
      {children.length > 0 && (
        <ul style={{ listStyle: "none", padding: 0 }}>
          {children.map((child) => (
            <PageTreeItem key={child.id} page={child} pages={pages} depth={depth + 1} onEdit={onEdit} onDelete={onDelete} />
          ))}
        </ul>
      )}
    </li>
  );
}
