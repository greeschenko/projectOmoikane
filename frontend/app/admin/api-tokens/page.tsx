"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Paper, Box, Button, Chip, CircularProgress,
  Dialog, DialogTitle, DialogContent, DialogActions, TextField,
  MenuItem, Alert, IconButton,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";

interface ApiToken {
  id: number;
  name: string;
  role: string;
  description: string;
  expiresAt: string | null;
  lastUsedAt: string | null;
  createdAt: string;
}

export default function AdminApiTokens() {
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [role, setRole] = useState("admin");
  const [expiresInDays, setExpiresInDays] = useState("0");
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ApiToken | null>(null);
  const [error, setError] = useState("");

  const fetchTokens = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/api-tokens");
      if (res.ok) {
        setTokens(await res.json());
      }
    } catch {
      // ignore
    }
    setLoading(false);
  }, []);

  useEffect(() => { fetchTokens(); }, [fetchTokens]);

  const handleCreate = async () => {
    setError("");
    if (!name.trim()) {
      setError("Name is required");
      return;
    }
    const res = await fetch("/api/api-tokens", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: name.trim(),
        role,
        expiresInDays: parseInt(expiresInDays, 10) || 0,
        description: "",
      }),
    });
    if (res.ok) {
      const data = await res.json();
      setCreatedToken(data.token);
      setName("");
      setRole("admin");
      setExpiresInDays("0");
      setCreateOpen(false);
      fetchTokens();
    } else {
      const data = await res.json().catch(() => ({}));
      setError(data.error || "Failed to create token");
    }
  };

  const handleRevoke = async () => {
    if (!deleteTarget) return;
    const res = await fetch(`/api/api-tokens/${deleteTarget.id}`, {
      method: "DELETE",
    });
    if (res.ok) {
      setDeleteTarget(null);
      fetchTokens();
    }
  };

  const copyToken = () => {
    if (createdToken) {
      navigator.clipboard?.writeText(createdToken);
    }
  };

  return (
    <Container>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
        <Typography variant="h4" component="h1">
          API Tokens
        </Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => setCreateOpen(true)}>
          New Token
        </Button>
      </Box>
      <Box sx={{ mb: 2 }}>
        <Paper sx={{ p: 2, bgcolor: "action.hover" }} variant="outlined">
          <Typography variant="body1" sx={{ fontWeight: 600, mb: 1 }}>What are API tokens?</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            API tokens let external systems read and write content through the same
            REST API this admin interface uses — a headless CMS setup. Each request
            authenticates with an <code>Authorization: Bearer &lt;token&gt;</code> header
            instead of a login session. A token&apos;s role grants the same permissions as
            that role on a user account.
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Tokens are stored as SHA-256 hashes, so the raw value is shown <strong>only once</strong>
            at creation — copy it somewhere safe. Set a lifetime or pick days = 0 for a token
            that never expires, and revoke any token here to cut off access immediately.
          </Typography>
          <Box component="pre" sx={{ m: 0, p: 1, bgcolor: "background.paper", borderRadius: 1, fontSize: "0.75rem", overflowX: "auto" }}>
{`curl -H "Authorization: Bearer <token>" https://your-site.com/api/blog/posts`}
          </Box>
        </Paper>
      </Box>

      {createdToken && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setCreatedToken(null)}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1, flexWrap: "wrap" }}>
            <span>Token created — copy it now, it is shown only once:</span>
            <code>{createdToken}</code>
            <IconButton size="small" onClick={copyToken} aria-label="copy token">
              <ContentCopyIcon fontSize="small" />
            </IconButton>
          </Box>
        </Alert>
      )}

      {loading ? (
        <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
          <CircularProgress aria-label="Loading" />
        </Box>
      ) : tokens.length === 0 ? (
        <Typography color="text.secondary" sx={{ textAlign: "center", py: 4 }}>
          No API tokens yet
        </Typography>
      ) : (
        <TableContainer component={Paper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Role</TableCell>
                <TableCell>Expires</TableCell>
                <TableCell>Last Used</TableCell>
                <TableCell>Created</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {tokens.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>{t.name}</TableCell>
                  <TableCell>
                    <Chip label={t.role} color={t.role === "admin" ? "primary" : "default"} size="small" />
                  </TableCell>
                  <TableCell>{t.expiresAt ? new Date(t.expiresAt).toLocaleString() : "Never"}</TableCell>
                  <TableCell>{t.lastUsedAt ? new Date(t.lastUsedAt).toLocaleString() : "Never"}</TableCell>
                  <TableCell>{new Date(t.createdAt).toLocaleString()}</TableCell>
                  <TableCell align="right">
                    <IconButton size="small" color="error" aria-label={`Revoke ${t.name}`} onClick={() => setDeleteTarget(t)}>
                      <DeleteIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)}>
        <DialogTitle>Create API Token</DialogTitle>
        <DialogContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          <TextField
            label="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            fullWidth
            margin="normal"
            autoFocus
          />
          <TextField
            select
            label="Role"
            value={role}
            onChange={(e) => setRole(e.target.value)}
            fullWidth
            margin="normal"
          >
            <MenuItem value="admin">Admin</MenuItem>
            <MenuItem value="user">User</MenuItem>
          </TextField>
          <TextField
            label="Expires in (days, 0 = never)"
            type="number"
            value={expiresInDays}
            onChange={(e) => setExpiresInDays(e.target.value)}
            fullWidth
            margin="normal"
            inputProps={{ min: 0 }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCreate}>Create</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Revoke Token</DialogTitle>
        <DialogContent>
          Revoke <strong>{deleteTarget?.name}</strong>? Any client using this token will lose access immediately.
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={handleRevoke}>Revoke</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
