"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Box, Button, Card, CardMedia, CardContent, Typography,
  Dialog, DialogTitle, DialogContent, DialogActions,
  IconButton, Grid, Alert, Chip, CircularProgress, Checkbox, TextField,
} from "@mui/material";
import CloudUploadIcon from "@mui/icons-material/CloudUpload";
import DeleteIcon from "@mui/icons-material/Delete";
import InsertPhotoIcon from "@mui/icons-material/InsertPhoto";
import EditIcon from "@mui/icons-material/Edit";

interface MediaItem {
  id: string;
  filename: string;
  mimeType: string;
  size: number;
  data: string;
  alt?: string;
  url?: string;
  thumbUrl?: string;
  createdAt: string;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function AdminMediaPage() {
  const [media, setMedia] = useState<MediaItem[]>([]);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<File[]>([]);
  const [previews, setPreviews] = useState<string[]>([]);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<MediaItem | null>(null);
  const [editTarget, setEditTarget] = useState<MediaItem | null>(null);
  const [altText, setAltText] = useState("");
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [bulkAction, setBulkAction] = useState<string | null>(null);

  const fetchMedia = useCallback(async () => {
    const res = await fetch("/api/media");
    const data = await res.json();
    setMedia(Array.isArray(data) ? data : (data.media ?? []));
  }, []);

  useEffect(() => { fetchMedia(); }, [fetchMedia]);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []);
    setError("");
    const valid = files.filter((f) => f.size <= 10 * 1024 * 1024);
    if (valid.length < files.length) {
      setError("Some files exceeded 10MB limit and were skipped");
    }
    setSelectedFiles(valid);
    const newPreviews: string[] = [];
    valid.forEach((file) => {
      const reader = new FileReader();
      reader.onload = () => {
        newPreviews.push(reader.result as string);
        if (newPreviews.length === valid.length) setPreviews([...newPreviews]);
      };
      reader.readAsDataURL(file);
    });
    if (valid.length === 0) setPreviews([]);
  };

  const handleUpload = async () => {
    if (selectedFiles.length === 0) return;
    setUploading(true);
    setError("");
    try {
      for (const file of selectedFiles) {
        const formData = new FormData();
        formData.append("file", file);
        const res = await fetch("/api/media", { method: "POST", body: formData });
        if (!res.ok) {
          const data = await res.json();
          throw new Error(data.error ?? "Upload failed");
        }
      }
      setUploadOpen(false);
      setSelectedFiles([]);
      setPreviews([]);
      fetchMedia();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      const res = await fetch(`/api/media/${deleteTarget.id}`, { method: "DELETE" });
      if (!res.ok) throw new Error("Delete failed");
      setDeleteTarget(null);
      fetchMedia();
    } catch {
      setError("Failed to delete media");
    }
  };

  const openEdit = (item: MediaItem) => {
    setEditTarget(item);
    setAltText(item.alt ?? "");
  };

  const handleSaveAlt = async () => {
    if (!editTarget) return;
    try {
      const res = await fetch(`/api/media/${editTarget.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ alt: altText }),
      });
      if (!res.ok) throw new Error("Save failed");
      setEditTarget(null);
      fetchMedia();
    } catch {
      setError("Failed to update alt text");
    }
  };

  async function handleBulkAction() {
    if (!bulkAction || selectedIds.size === 0) return;
    await fetch("/api/media/batch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: bulkAction, ids: [...selectedIds] }),
    });
    setBulkAction(null);
    setSelectedIds(new Set());
    fetchMedia();
  }

  function toggleSelect(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id); else next.add(id);
    setSelectedIds(next);
  }

  return (
    <Box>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
        <Typography variant="h4">Media Library</Typography>
        <Button variant="contained" startIcon={<CloudUploadIcon />} onClick={() => setUploadOpen(true)}>
          Upload
        </Button>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {selectedIds.size > 0 && (
        <Box sx={{ mb: 2, display: "flex", alignItems: "center", gap: 1 }}>
          <Typography variant="body2">{selectedIds.size} selected</Typography>
          <Button size="small" variant="outlined" color="error" onClick={() => setBulkAction("delete")}>
            Delete Selected
          </Button>
          <Button size="small" onClick={() => setSelectedIds(new Set())}>Clear</Button>
        </Box>
      )}

      {media.length === 0 ? (
        <Box sx={{ textAlign: "center", py: 8, color: "text.secondary" }}>
          <InsertPhotoIcon sx={{ fontSize: 64, mb: 2 }} />
          <Typography variant="h6">No media uploaded yet</Typography>
          <Typography variant="body2">Click Upload to add images</Typography>
        </Box>
      ) : (
        <Grid container spacing={2}>
          {media.map((item) => (
            <Grid item xs={6} sm={4} md={3} key={item.id}>
              <Card sx={{ position: "relative", bgcolor: selectedIds.has(item.id) ? "action.selected" : undefined }}>
                <Box sx={{ position: "absolute", top: 4, left: 4, zIndex: 1 }}>
                  <Checkbox
                    checked={selectedIds.has(item.id)}
                    onChange={() => toggleSelect(item.id)}
                    sx={{ bgcolor: "background.paper", "&:hover": { bgcolor: "action.hover" } }}
                  />
                </Box>
                <CardMedia
                  component="img"
                  height="140"
                  image={item.thumbUrl || item.data}
                  alt={item.alt || item.filename}
                  sx={{ objectFit: "cover" }}
                />
                <CardContent sx={{ p: 1, "&:last-child": { pb: 1 } }}>
                  <Typography variant="body2" noWrap title={item.filename}>
                    {item.filename}
                  </Typography>
                  <Typography variant="caption" color="text.secondary" noWrap title={item.alt || "No alt text"}>
                    {item.alt ? `Alt: ${item.alt}` : "No alt text"}
                  </Typography>
                  <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mt: 0.5 }}>
                    <Chip label={formatSize(item.size)} size="small" variant="outlined" />
                    <Box>
                      <IconButton size="small" aria-label={`Edit alt for ${item.filename}`} onClick={() => openEdit(item)}>
                        <EditIcon fontSize="small" />
                      </IconButton>
                      <IconButton size="small" color="error" aria-label="Delete" onClick={() => setDeleteTarget(item)}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {/* Upload Dialog */}
      <Dialog open={uploadOpen} onClose={() => { setUploadOpen(false); setSelectedFiles([]); setPreviews([]); setError(""); }} maxWidth="sm" fullWidth>
        <DialogTitle>Upload Media</DialogTitle>
        <DialogContent>
          <Box sx={{ mt: 2 }}>
            <Button variant="outlined" component="label" startIcon={<CloudUploadIcon />}>
              Choose Files
              <input type="file" hidden accept="image/*" multiple onChange={handleFileSelect} />
            </Button>
            {selectedFiles.length > 0 && (
              <Typography variant="body2" sx={{ mt: 1 }}>
                {selectedFiles.length} file(s) selected
              </Typography>
            )}
            {previews.length > 0 && (
              <Box sx={{ mt: 2, display: "flex", flexWrap: "wrap", gap: 1, maxHeight: 300, overflow: "auto" }}>
                {previews.map((src, i) => (
                  <Box key={i} sx={{ width: 100, height: 100, borderRadius: 1, overflow: "hidden" }}>
                    <img src={src} alt={`Preview ${i + 1}`} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                  </Box>
                ))}
              </Box>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => { setUploadOpen(false); setSelectedFiles([]); setPreviews([]); setError(""); }}>Cancel</Button>
          <Button variant="contained" onClick={handleUpload} disabled={selectedFiles.length === 0 || uploading}>
            {uploading ? <CircularProgress size={24} aria-hidden="true" /> : `Upload ${selectedFiles.length > 0 ? `(${selectedFiles.length})` : ""}`}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation */}
      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Media</DialogTitle>
        <DialogContent>
          <Typography>Delete {deleteTarget?.filename}? This item will be moved to trash. You can restore it later or permanently delete it from the trash page.</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button variant="contained" color="error" onClick={handleDelete}>Delete</Button>
        </DialogActions>
      </Dialog>

      {/* Alt Text Edit Dialog */}
      <Dialog open={!!editTarget} onClose={() => setEditTarget(null)} fullWidth maxWidth="xs">
        <DialogTitle>Edit Alt Text</DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Alt text describes the image for screen readers and SEO.
          </Typography>
          <TextField
            label="Alt text"
            value={altText}
            onChange={(e) => setAltText(e.target.value)}
            fullWidth
            autoFocus
            margin="dense"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditTarget(null)}>Cancel</Button>
          <Button variant="contained" onClick={handleSaveAlt}>Save</Button>
        </DialogActions>
      </Dialog>

      {/* Bulk Action Confirmation */}
      <Dialog open={!!bulkAction} onClose={() => setBulkAction(null)}>
        <DialogTitle>Confirm Bulk Action</DialogTitle>
        <DialogContent>
          <Typography>
            {bulkAction === "delete" && `Delete ${selectedIds.size} item(s)? They will be moved to trash.`}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setBulkAction(null)}>Cancel</Button>
          <Button onClick={handleBulkAction} variant="contained" color="error">Delete</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
