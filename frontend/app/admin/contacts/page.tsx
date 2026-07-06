"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container,
  Typography,
  Button,
  Card,
  CardContent,
  CardActions,
  Box,
  Chip,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from "@mui/material";

interface ContactMessage {
  id: string;
  name: string;
  email: string;
  subject: string;
  message: string;
  read: boolean;
  createdAt: string;
}

export default function AdminContacts() {
  const [contacts, setContacts] = useState<ContactMessage[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [error, setError] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<ContactMessage | null>(null);

  const fetchContacts = useCallback(async () => {
    const res = await fetch("/api/contacts");
    if (res.ok) {
      const data = await res.json();
      setContacts(data.contacts ?? []);
      setUnreadCount(data.unreadCount ?? 0);
    }
  }, []);

  useEffect(() => { fetchContacts(); }, [fetchContacts]);

  async function handleMarkRead(id: string) {
    const res = await fetch(`/api/contacts/${id}/read`, { method: "POST" });
    if (res.ok) {
      fetchContacts();
    } else {
      setError("Failed to mark as read");
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return;
    const res = await fetch(`/api/contacts/${deleteTarget.id}`, { method: "DELETE" });
    if (res.ok) {
      setDeleteTarget(null);
      fetchContacts();
    } else {
      setError("Failed to delete");
    }
  }

  return (
    <Container>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
        <Typography variant="h4" component="h1">
          Contacts {unreadCount > 0 && <Chip label={`${unreadCount} unread`} color="primary" size="small" sx={{ ml: 1 }} />}
        </Typography>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError("")}>{error}</Alert>}

      {contacts.length === 0 ? (
        <Typography color="text.secondary" sx={{ textAlign: "center", py: 4 }}>
          No contact messages yet
        </Typography>
      ) : (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {contacts.map((msg) => (
            <Card key={msg.id} variant="outlined" sx={{ opacity: msg.read ? 0.7 : 1 }}>
              <CardContent>
                <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                  <Typography variant="h6">
                    {msg.subject || "Contact Form Submission"}
                    {!msg.read && <Chip label="New" color="primary" size="small" sx={{ ml: 1 }} />}
                  </Typography>
                </Box>
                <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                  From: {msg.name} ({msg.email})
                </Typography>
                <Typography variant="body1" sx={{ mt: 2, whiteSpace: "pre-wrap" }}>
                  {msg.message}
                </Typography>
                <Typography variant="caption" color="text.disabled" sx={{ mt: 1, display: "block" }}>
                  {new Date(msg.createdAt).toLocaleString()}
                </Typography>
              </CardContent>
              <CardActions>
                {!msg.read && (
                  <Button size="small" onClick={() => handleMarkRead(msg.id)}>
                    Mark Read
                  </Button>
                )}
                <Button size="small" color="error" onClick={() => setDeleteTarget(msg)}>
                  Delete
                </Button>
              </CardActions>
            </Card>
          ))}
        </Box>
      )}

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Message</DialogTitle>
        <DialogContent>
          <Typography>Delete this message from {deleteTarget?.name}?</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button onClick={confirmDelete} color="error" variant="contained">Delete</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
