"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import {
  Container, Typography, Button, TextField, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Box, Grid, FormControlLabel, Switch, Select, MenuItem, InputLabel, FormControl,
  IconButton, CircularProgress, Checkbox, Chip,
} from "@mui/material";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import PreviewIcon from "@mui/icons-material/Preview";
import DragIndicatorIcon from "@mui/icons-material/DragIndicator";
import ArrowUpwardIcon from "@mui/icons-material/ArrowUpward";
import ArrowDownwardIcon from "@mui/icons-material/ArrowDownward";
import RichTextEditor from "@/components/RichTextEditor";

interface Page {
  id: number;
  title: string;
  slug: string;
  content: string;
  metaTitle?: string;
  metaDescription?: string;
  metaKeywords?: string;
  parentId: number | null;
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
  const [dragOverId, setDragOverId] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [bulkAction, setBulkAction] = useState<string | null>(null);

  const fetchPages = useCallback(async () => {
    setPagesLoading(true);
    const res = await fetch("/api/pages");
    if (res.ok) setPages(await res.json());
    setPagesLoading(false);
  }, []);

  useEffect(() => { fetchPages(); }, [fetchPages]);

  function getChildren(parentId: number | null): Page[] {
    return pages
      .filter((p) => p.parentId === parentId)
      .sort((a, b) => a.sortOrder - b.sortOrder);
  }

  const rootPages = getChildren(null);

  async function handleReorder(parentId: number | null, pageIds: number[]) {
    await fetch("/api/pages/reorder", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ parentId, pageIds }),
    });
    fetchPages();
  }

  function openCreate(parentId?: number) {
    setEditingPage(null);
    setFormData({ title: "", slug: "", content: "", metaTitle: "", metaDescription: "", metaKeywords: "", parentId: parentId != null ? String(parentId) : "", status: "draft", inMenu: false });
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
      parentId: page.parentId != null ? String(page.parentId) : "",
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
    const body = { ...formData, parentId: formData.parentId ? Number(formData.parentId) : null };
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

  function toggleSelect(id: number) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id); else next.add(id);
    setSelectedIds(next);
  }

  async function handleBulkAction() {
    if (!bulkAction || selectedIds.size === 0) return;
    await fetch("/api/pages/batch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: bulkAction, ids: [...selectedIds] }),
    });
    setBulkAction(null);
    setSelectedIds(new Set());
    fetchPages();
  }

  return (
    <Container>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
        <Typography variant="h4" component="h1">Pages</Typography>
        <Button variant="contained" onClick={() => openCreate()}>New Page</Button>
      </Box>

      {selectedIds.size > 0 && (
        <Box sx={{ mb: 2, display: "flex", alignItems: "center", gap: 1 }}>
          <Typography variant="body2">{selectedIds.size} selected</Typography>
          <Button size="small" variant="outlined" onClick={() => setBulkAction("publish")}>Publish</Button>
          <Button size="small" variant="outlined" onClick={() => setBulkAction("draft")}>Draft</Button>
          <Button size="small" variant="outlined" color="error" onClick={() => setBulkAction("delete")}>Delete Selected</Button>
          <Button size="small" onClick={() => setSelectedIds(new Set())}>Clear</Button>
        </Box>
      )}

      <Paper sx={{ p: 2 }}>
        {pagesLoading ? (
          <Box sx={{ display: "flex", justifyContent: "center", py: 8 }}>
            <CircularProgress aria-label="Loading" />
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
            selectedIds={selectedIds}
            onToggleSelect={toggleSelect}
          />
        )}
      </Paper>

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="lg" fullWidth>
        <DialogTitle>{editingPage ? "Edit Page" : "New Page"}</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0 }}>
            <Grid size={{ xs: 12, md: 8.4 }}>
              <RichTextEditor
                value={formData.content}
                onChange={(html) => setFormData({ ...formData, content: html })}
                error={!!formErrors.content}
                helperText={formErrors.content}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3.6 }}>
              <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5 }}>
                <TextField label="Title" fullWidth margin="dense" value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  error={!!formErrors.title} helperText={formErrors.title} required />
                <TextField label="Slug" fullWidth margin="dense" value={formData.slug}
                  onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                  error={!!formErrors.slug} helperText={formErrors.slug} required />
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
                        <MenuItem key={p.id} value={String(p.id)}>{p.title}</MenuItem>
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
                />
              </Box>
            </Grid>
          </Grid>
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

      <Dialog open={!!bulkAction} onClose={() => setBulkAction(null)}>
        <DialogTitle>Confirm Bulk Action</DialogTitle>
        <DialogContent>
          <Typography>
            {bulkAction === "delete" && `Delete ${selectedIds.size} page(s)? They will be moved to trash.`}
            {bulkAction === "publish" && `Publish ${selectedIds.size} page(s)?`}
            {bulkAction === "draft" && `Set ${selectedIds.size} page(s) to draft?`}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setBulkAction(null)}>Cancel</Button>
          <Button onClick={handleBulkAction} variant="contained" color={bulkAction === "delete" ? "error" : "primary"}>
            Confirm
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}

function PageTreeList({
  parentId, pages, getChildren, onEdit, onDelete, onReorder, dragOverId, setDragOverId, depth, selectedIds, onToggleSelect,
}: {
  parentId: number | null;
  pages: Page[];
  getChildren: (pid: number | null) => Page[];
  onEdit: (p: Page) => void;
  onDelete: (p: Page) => void;
  onReorder: (pid: number | null, ids: number[]) => void;
  dragOverId: number | null;
  setDragOverId: (id: number | null) => void;
  depth: number;
  selectedIds: Set<number>;
  onToggleSelect: (id: number) => void;
}) {
  const children = getChildren(parentId);
  if (children.length === 0) return null;

  return (
    <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
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
          selectedIds={selectedIds}
          onToggleSelect={onToggleSelect}
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
  page, pages, getChildren, depth, onEdit, onDelete, onReorder, dragOverId, setDragOverId, selectedIds, onToggleSelect,
}: {
  page: Page;
  pages: Page[];
  getChildren: (pid: number | null) => Page[];
  depth: number;
  onEdit: (p: Page) => void;
  onDelete: (p: Page) => void;
  onReorder: (pid: number | null, ids: number[]) => void;
  dragOverId: number | null;
  setDragOverId: (id: number | null) => void;
  selectedIds: Set<number>;
  onToggleSelect: (id: number) => void;
}) {
  const [dragging, setDragging] = useState(false);
  const dragCleanupRef = useRef<(() => void) | null>(null);

  // Pointer-based drag: listeners attach synchronously in pointerdown (not via
  // an effect) so fast drags never race React's render. The dragged row is
  // highlighted via dragOverId; on release the siblings are reordered.
  function handlePointerDown(e: React.PointerEvent<HTMLSpanElement>) {
    if (e.pointerType === "mouse" && e.button !== 0) return;
    e.preventDefault();
    setDragging(true);
    setDragOverId(page.id);

    const rowAt = (clientX: number, clientY: number): number | null => {
      const el = document.elementFromPoint(clientX, clientY);
      const row = el?.closest?.("[data-page-id]") as HTMLElement | null;
      const raw = row ? row.getAttribute("data-page-id") : null;
      const parsed = raw !== null ? Number(raw) : NaN;
      return Number.isFinite(parsed) ? parsed : null;
    };

    const onMove = (ev: PointerEvent) => {
      const id = rowAt(ev.clientX, ev.clientY);
      if (id !== null) setDragOverId(id);
    };

    const onUp = (ev: PointerEvent) => {
      const targetId = rowAt(ev.clientX, ev.clientY);
      const finalTarget = targetId !== null && targetId !== page.id ? targetId : null;
      cleanup();
      if (finalTarget === null) return;

      const siblings = getChildren(page.parentId);
      const draggedPage = pages.find((p) => p.id === page.id);
      if (!draggedPage) return;
      const reordered = siblings.filter((p) => p.id !== page.id);
      const dropIndex = reordered.findIndex((p) => p.id === finalTarget);
      if (dropIndex < 0) return;
      reordered.splice(dropIndex, 0, draggedPage);
      onReorder(page.parentId, reordered.map((p) => p.id));
    };

    const onCancel = () => cleanup();

    function cleanup() {
      setDragging(false);
      setDragOverId(null);
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
      window.removeEventListener("pointercancel", onCancel);
      dragCleanupRef.current = null;
    }

    dragCleanupRef.current = cleanup;
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    window.addEventListener("pointercancel", onCancel);
  }

  // Ensure listeners are removed if the row unmounts mid-drag.
  useEffect(() => {
    return () => {
      dragCleanupRef.current?.();
    };
  }, [setDragOverId]);

  const isDragOver = dragOverId === page.id;

  function handleMove(direction: "up" | "down") {
    const siblings = getChildren(page.parentId);
    const reordered = [...siblings];
    const index = reordered.findIndex((p) => p.id === page.id);
    const swap = direction === "up" ? index - 1 : index + 1;
    if (index < 0 || swap < 0 || swap >= reordered.length) return;
    [reordered[index], reordered[swap]] = [reordered[swap], reordered[index]];
    onReorder(page.parentId, reordered.map((p) => p.id));
  }

  return (
    <li>
      <Box
        data-page-id={page.id}
        sx={{
          display: "flex", alignItems: "center", gap: 1, py: 0.25, pl: depth * 12,
          bgcolor: selectedIds.has(page.id) ? "action.selected" : dragging ? "action.hover" : isDragOver ? "action.selected" : "transparent",
          borderTop: isDragOver ? 2 : 0,
          borderColor: "primary.main",
        }}
      >
        <Checkbox
          checked={selectedIds.has(page.id)}
          onChange={() => onToggleSelect(page.id)}
          size="small"
          slotProps={{ input: { "aria-label": `Select ${page.title} for bulk delete` } }}
        />
        <Box
          component="span"
          sx={{
            display: "inline-flex",
            alignItems: "center",
            cursor: "grab",
            touchAction: "none",
            "&:active": { cursor: "grabbing" },
            opacity: dragging ? 0.6 : 1,
          }}
          onPointerDown={handlePointerDown}
        >
          <DragIndicatorIcon fontSize="small" color="disabled" data-testid="DragIndicatorIcon" />
        </Box>
        <IconButton size="small" onClick={() => handleMove("up")} aria-label={`Move ${page.title} up`}>
          <ArrowUpwardIcon fontSize="small" />
        </IconButton>
        <IconButton size="small" onClick={() => handleMove("down")} aria-label={`Move ${page.title} down`}>
          <ArrowDownwardIcon fontSize="small" />
        </IconButton>
        <Typography sx={{ flexGrow: 1 }}>{page.title}</Typography>
        <Chip
          label={page.status === "published" ? "Published" : "Draft"}
          size="small"
          color={page.status === "published" ? "success" : "default"}
          variant="outlined"
          sx={{ height: 20, fontSize: "0.7rem" }}
        />
        {page.inMenu && (
          <Chip label="Menu" size="small" variant="outlined" sx={{ height: 20, fontSize: "0.7rem" }} />
        )}
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
        selectedIds={selectedIds}
        onToggleSelect={onToggleSelect}
      />
    </li>
  );
}