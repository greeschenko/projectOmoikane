"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Table, TableBody,
  TableCell, TableContainer, TableHead, TableRow, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, Box, Chip, Tabs, Tab, CircularProgress,
} from "@mui/material";

interface TrashItem {
  id: number;
  title: string;
  entity: string;
  deletedAt: string;
}

const ENTITY_LABELS: Record<string, string> = {
  page: "Page",
  user: "User",
  post: "Post",
  media: "Media",
  contact: "Contact",
  message: "Message",
  tag: "Tag",
  category: "Category",
};

const ENTITY_COLORS: Record<string, "primary" | "success" | "warning" | "info" | "error" | "secondary" | "default"> = {
  page: "primary",
  user: "success",
  post: "info",
  media: "warning",
  contact: "error",
  message: "secondary",
  tag: "default",
  category: "default",
};

const ALL_ENTITIES = ["all", "page", "user", "post", "media", "contact", "message", "tag", "category"];

export default function AdminTrash() {
  const [items, setItems] = useState<TrashItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [filterEntity, setFilterEntity] = useState("all");
  const [restoreTarget, setRestoreTarget] = useState<TrashItem | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<TrashItem | null>(null);

  const fetchTrash = useCallback(async () => {
    setLoading(true);
    const res = await fetch("/api/trash");
    if (res.ok) setItems(await res.json());
    setLoading(false);
  }, []);

  useEffect(() => { fetchTrash(); }, [fetchTrash]);

  const filtered = filterEntity === "all" ? items : items.filter((i) => i.entity === filterEntity);

  async function handleRestore() {
    if (!restoreTarget) return;
    await fetch(`/api/trash/${restoreTarget.entity}/${restoreTarget.id}/restore`, { method: "POST" });
    setRestoreTarget(null);
    fetchTrash();
    window.dispatchEvent(new Event("trash-changed"));
  }

  async function handleHardDelete() {
    if (!deleteTarget) return;
    await fetch(`/api/trash/${deleteTarget.entity}/${deleteTarget.id}`, { method: "DELETE" });
    setDeleteTarget(null);
    fetchTrash();
    window.dispatchEvent(new Event("trash-changed"));
  }

  return (
    <Container>
      <Typography variant="h4" component="h1" sx={{ mb: 2 }}>
        Trash
      </Typography>

      <Tabs
        value={filterEntity}
        onChange={(_, v) => setFilterEntity(v)}
        variant="scrollable"
        scrollButtons="auto"
        sx={{ mb: 2 }}
      >
        {ALL_ENTITIES.map((e) => (
          <Tab key={e} value={e} label={e === "all" ? "All" : ENTITY_LABELS[e] || e} />
        ))}
      </Tabs>

      {loading ? (
        <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
          <CircularProgress />
        </Box>
      ) : filtered.length === 0 ? (
        <Typography color="text.secondary" sx={{ textAlign: "center", py: 4 }}>
          Trash is empty
        </Typography>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Title</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Deleted</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filtered.map((item) => (
                <TableRow key={`${item.entity}-${item.id}`}>
                  <TableCell>{item.title}</TableCell>
                  <TableCell>
                    <Chip
                      label={ENTITY_LABELS[item.entity] || item.entity}
                      color={ENTITY_COLORS[item.entity] || "default"}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{new Date(item.deletedAt).toLocaleString()}</TableCell>
                  <TableCell>
                    <Button size="small" onClick={() => setRestoreTarget(item)} sx={{ mr: 1 }}>
                      Restore
                    </Button>
                    <Button size="small" color="error" onClick={() => setDeleteTarget(item)}>
                      Delete Forever
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Dialog open={!!restoreTarget} onClose={() => setRestoreTarget(null)}>
        <DialogTitle>Restore Item</DialogTitle>
        <DialogContent>
          <Typography>
            Restore &ldquo;{restoreTarget?.title}&rdquo;?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRestoreTarget(null)}>Cancel</Button>
          <Button onClick={handleRestore} variant="contained">Restore</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Permanently Delete</DialogTitle>
        <DialogContent>
          <Typography>
            Permanently delete &ldquo;{deleteTarget?.title}&rdquo;? This cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button onClick={handleHardDelete} variant="contained" color="error">Delete</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
