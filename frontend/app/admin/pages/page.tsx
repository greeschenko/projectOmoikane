"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Button, TextField, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Box, FormControlLabel, Switch, Select, MenuItem, InputLabel, FormControl, Alert,
  IconButton, CircularProgress,
} from "@mui/material";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import PreviewIcon from "@mui/icons-material/Preview";
import DragIndicatorIcon from "@mui/icons-material/DragIndicator";
import RichTextEditor from "@/components/RichTextEditor";

interface Page {
  id: string;
  title: string;
  slug: string;
  content: string;
  metaTitle?: string;
  metaDescription?: string;
  metaKeywords?: string;
  parentId: string | null;
  sortOrder: number;
  status: "draft" | "published";
  inMenu: boolean;
  previewToken: string;
}

export default function AdminPages() {
  const [pages, setPages] = useState<Page[]>([]);
  const [pagesLoading, setPagesLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editingPage, setEditingPage] = useState<Page | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Page | null>(null);
  const [formData, setFormData] = useState({
    title: "", slug: "", content: "",
    metaTitle: "", metaDescription: "", metaKeywords: "",
    parentId: "", status: "draft" as "draft" | "published", inMenu: false,
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [dragOverId, setDragOverId] = useState<string | null>(null);

  const fetchPages = useCallback(async () => {
    setPagesLoading(true);
    const res = await fetch("/api/pages");
    if (res.ok) setPages(await res.json());
    setPagesLoading(false);
  }, []);

  useEffect(() => { fetchPages(); }, [fetchPages]);

  function getChildren(parentId: string | null): Page[] {
    return pages
      .filter((p) => p.parentId === parentId)
      .sort((a, b) => a.sortOrder - b.sortOrder);
  }

  const rootPages = getChildren(null);

  async function handleReorder(parentId: string | null, pageIds: string[]) {
    await fetch("/api/pages/reorder", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ parentId, pageIds }),
    });
    fetchPages();
  }

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
        {pagesLoading ? (
          <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
            <CircularProgress />
          </Box>
        ) : pages.length === 0 ? (
          <Typography color="text.secondary">No pages yet.</Typography>
        ) : (
          <PageTreeList
            parentId={null}
            pages={pages}
            getChildren={getChildren}
            onEdit={openEdit}
            onDelete={setDeleteTarget}
            onReorder={handleReorder}
            dragOverId={dragOverId}
            setDragOverId={setDragOverId}
            depth={0}
          />
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
          <RichTextEditor
            value={formData.content}
            onChange={(html) => setFormData({ ...formData, content: html })}
            error={!!formErrors.content}
            helperText={formErrors.content}
          />
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
          {editingPage && (
            <Button
              startIcon={<PreviewIcon />}
              onClick={() => window.open(`/preview/${editingPage.id}?token=${editingPage.previewToken}`, '_blank', 'noopener')}
            >
              Preview
            </Button>
          )}
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

function PageTreeList({
  parentId, pages, getChildren, onEdit, onDelete, onReorder, dragOverId, setDragOverId, depth,
}: {
  parentId: string | null;
  pages: Page[];
  getChildren: (pid: string | null) => Page[];
  onEdit: (p: Page) => void;
  onDelete: (p: Page) => void;
  onReorder: (pid: string | null, ids: string[]) => void;
  dragOverId: string | null;
  setDragOverId: (id: string | null) => void;
  depth: number;
}) {
  const children = getChildren(parentId);
  if (children.length === 0) return null;

  return (
    <ul style={{ listStyle: "none", padding: 0 }}>
      {children.map((page) => (
        <PageTreeItem
          key={page.id}
          page={page}
          pages={pages}
          getChildren={getChildren}
          depth={depth}
          onEdit={onEdit}
          onDelete={onDelete}
          onReorder={onReorder}
          dragOverId={dragOverId}
          setDragOverId={setDragOverId}
        />
      ))}
    </ul>
  );
}

function buildViewUrl(p: Page, pages: Page[]): string {
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

function PageTreeItem({
  page, pages, getChildren, depth, onEdit, onDelete, onReorder, dragOverId, setDragOverId,
}: {
  page: Page;
  pages: Page[];
  getChildren: (pid: string | null) => Page[];
  depth: number;
  onEdit: (p: Page) => void;
  onDelete: (p: Page) => void;
  onReorder: (pid: string | null, ids: string[]) => void;
  dragOverId: string | null;
  setDragOverId: (id: string | null) => void;
}) {
  const siblings = getChildren(page.parentId);

  function handleDragStart(e: React.DragEvent) {
    e.dataTransfer.setData("text/plain", page.id);
    e.dataTransfer.effectAllowed = "move";
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDragOverId(page.id);
  }

  function handleDragLeave() {
    setDragOverId(null);
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOverId(null);
    const draggedId = e.dataTransfer.getData("text/plain");
    if (draggedId === page.id) return;
    const draggedPage = pages.find((p) => p.id === draggedId);
    if (!draggedPage) return;

    const reordered = siblings.filter((p) => p.id !== draggedId);
    const dropIndex = reordered.findIndex((p) => p.id === page.id);
    reordered.splice(dropIndex, 0, draggedPage);

    onReorder(page.parentId, reordered.map((p) => p.id));
  }

  const isDragOver = dragOverId === page.id;

  return (
    <li>
      <Box
        draggable
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        sx={{
          display: "flex", alignItems: "center", gap: 1, py: 0.5, pl: depth * 24,
          bgcolor: isDragOver ? "action.hover" : "transparent",
          borderTop: isDragOver ? 2 : 0,
          borderColor: "primary.main",
          cursor: "grab",
          "&:active": { cursor: "grabbing" },
        }}
      >
        <DragIndicatorIcon fontSize="small" color="disabled" sx={{ cursor: "grab" }} />
        <Typography sx={{ flexGrow: 1 }}>{page.title}</Typography>
        <IconButton
          size="small"
          onClick={() => window.open(buildViewUrl(page, pages), '_blank', 'noopener')}
          aria-label="view"
          data-href={buildViewUrl(page, pages)}
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
      <PageTreeList
        parentId={page.id}
        pages={pages}
        getChildren={getChildren}
        onEdit={onEdit}
        onDelete={onDelete}
        onReorder={onReorder}
        dragOverId={dragOverId}
        setDragOverId={setDragOverId}
        depth={depth + 1}
      />
    </li>
  );
}
