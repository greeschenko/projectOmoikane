"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Button, Card, CardContent, CardActions,
  Dialog, DialogTitle, DialogContent, DialogActions,
  TextField, Box, Chip, Alert,
} from "@mui/material";

interface Message {
  id: string;
  title: string;
  content: string;
  createdAt: string;
  readBy: string[];
}

export default function AdminMessages() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [formOpen, setFormOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [error, setError] = useState("");

  const fetchMessages = useCallback(async () => {
    const res = await fetch("/api/messages");
    if (res.ok) {
      const data = await res.json();
      setMessages(data.messages ?? []);
    }
  }, []);

  useEffect(() => { fetchMessages(); }, [fetchMessages]);

  async function handleSubmit() {
    if (!title.trim()) { setError("Title is required"); return; }
    setError("");
    const res = await fetch("/api/messages", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title: title.trim(), content: content.trim() }),
    });
    if (res.ok) {
      setFormOpen(false);
      setTitle("");
      setContent("");
      fetchMessages();
    } else {
      setError("Failed to create message");
    }
  }

  async function handleClearAll() {
    await fetch("/api/messages", { method: "DELETE" });
    fetchMessages();
  }

  return (
    <Container>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
        <Typography variant="h4" component="h1">Messages</Typography>
        <Box sx={{ display: "flex", gap: 1 }}>
          <Button variant="outlined" color="error" onClick={handleClearAll}>
            Clear All
          </Button>
          <Button variant="contained" onClick={() => setFormOpen(true)}>
            New Message
          </Button>
        </Box>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {messages.length === 0 ? (
        <Typography color="text.secondary" sx={{ textAlign: "center", py: 4 }}>
          No messages yet
        </Typography>
      ) : (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {messages.map((msg) => (
            <Card key={msg.id} variant="outlined">
              <CardContent>
                <Typography variant="h6">{msg.title}</Typography>
                {msg.content && (
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                    {msg.content}
                  </Typography>
                )}
                <Typography variant="caption" color="text.disabled" sx={{ mt: 1, display: "block" }}>
                  {new Date(msg.createdAt).toLocaleString()} &middot; Read by {msg.readBy.length} user{msg.readBy.length !== 1 ? "s" : ""}
                </Typography>
              </CardContent>
            </Card>
          ))}
        </Box>
      )}

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>New Message</DialogTitle>
        <DialogContent>
          <TextField
            label="Title"
            fullWidth
            margin="dense"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            autoFocus
          />
          <TextField
            label="Content"
            fullWidth
            margin="dense"
            multiline
            rows={4}
            value={content}
            onChange={(e) => setContent(e.target.value)}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => { setFormOpen(false); setError(""); }}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained">Send</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
