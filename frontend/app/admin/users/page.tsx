"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Button, TextField, Table, TableBody,
  TableCell, TableContainer, TableHead, TableRow, Paper,
  Dialog, DialogTitle, DialogContent, DialogActions,
  Box, Select, MenuItem, FormControl, InputLabel, Alert, Chip,
  TableSortLabel, Checkbox,
} from "@mui/material";

interface User {
  id: string;
  name: string;
  email: string;
  role: string;
  status: "active" | "banned";
  createdAt: string;
}

export default function AdminUsers() {
  const [users, setUsers] = useState<User[]>([]);
  const [filter, setFilter] = useState("");
  const [sortField, setSortField] = useState<keyof User>("name");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");
  const [formOpen, setFormOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null);
  const [formData, setFormData] = useState({ name: "", email: "", password: "", confirmPassword: "", role: "user", status: "active" });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [bulkAction, setBulkAction] = useState<string | null>(null);

  const fetchUsers = useCallback(async () => {
    const res = await fetch("/api/users");
    if (res.ok) setUsers(await res.json());
  }, []);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  function handleSort(field: keyof User) {
    if (sortField === field) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  }

  const filtered = users
    .filter((u) =>
      u.name.toLowerCase().includes(filter.toLowerCase()) ||
      u.email.toLowerCase().includes(filter.toLowerCase())
    )
    .sort((a, b) => {
      const aVal = String(a[sortField] || "").toLowerCase();
      const bVal = String(b[sortField] || "").toLowerCase();
      return sortDir === "asc" ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
    });

  const allFilteredSelected = filtered.length > 0 && filtered.every((u) => selectedIds.has(u.id));

  function toggleSelectAll() {
    if (allFilteredSelected) {
      setSelectedIds(new Set([...selectedIds].filter((id) => !filtered.some((u) => u.id === id))));
    } else {
      const next = new Set(selectedIds);
      filtered.forEach((u) => next.add(u.id));
      setSelectedIds(next);
    }
  }

  function toggleSelect(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id); else next.add(id);
    setSelectedIds(next);
  }

  function openCreate() {
    setEditingUser(null);
    setFormData({ name: "", email: "", password: "", confirmPassword: "", role: "user", status: "active" });
    setFormErrors({});
    setFormOpen(true);
  }

  function openEdit(user: User) {
    setEditingUser(user);
    setFormData({ name: user.name, email: user.email, password: "", confirmPassword: "", role: user.role, status: user.status || "active" });
    setFormErrors({});
    setFormOpen(true);
  }

  function validateForm() {
    const errors: Record<string, string> = {};
    const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!formData.name.trim()) errors.name = "Name is required";
    if (!emailPattern.test(formData.email)) errors.email = "Please enter a valid email address";
    if (!editingUser) {
      if (!formData.password) errors.password = "Password is required";
      if (formData.password !== formData.confirmPassword) errors.password = "Passwords do not match";
      if (formData.password && formData.password.length < 6) errors.password = "Password must be at least 6 characters";
    } else if (formData.password) {
      if (formData.password !== formData.confirmPassword) errors.password = "Passwords do not match";
    }
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function handleSubmit() {
    if (!validateForm()) return;
    const url = editingUser ? `/api/users/${editingUser.id}` : "/api/users";
    const method = editingUser ? "PUT" : "POST";
    const body: Record<string, string> = { name: formData.name, email: formData.email, role: formData.role, status: formData.status };
    if (formData.password) body.password = formData.password;
    const res = await fetch(url, { method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
    if (res.ok) {
      setFormOpen(false);
      fetchUsers();
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return;
    await fetch(`/api/users/${deleteTarget.id}`, { method: "DELETE" });
    setDeleteTarget(null);
    fetchUsers();
  }

  async function handleBulkAction() {
    if (!bulkAction || selectedIds.size === 0) return;
    await fetch("/api/users/batch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: bulkAction, ids: [...selectedIds] }),
    });
    setBulkAction(null);
    setSelectedIds(new Set());
    fetchUsers();
  }

  return (
    <Container>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
        <Typography variant="h4" component="h1">Users</Typography>
        <Button variant="contained" onClick={openCreate}>New User</Button>
      </Box>
      <TextField
        label="Search users"
        placeholder="Search users..."
        size="small"
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
        sx={{ mb: 2, width: 300 }}
      />
      {selectedIds.size > 0 && (
        <Box sx={{ mb: 2, display: "flex", alignItems: "center", gap: 1 }}>
          <Typography variant="body2">{selectedIds.size} selected</Typography>
          <Button size="small" variant="outlined" onClick={() => setBulkAction("delete")}>Delete Selected</Button>
          <Button size="small" variant="outlined" onClick={() => setBulkAction("ban")}>Ban</Button>
          <Button size="small" variant="outlined" onClick={() => setBulkAction("activate")}>Activate</Button>
          <Button size="small" onClick={() => setSelectedIds(new Set())}>Clear</Button>
        </Box>
      )}
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell padding="checkbox">
                <Checkbox
                  indeterminate={selectedIds.size > 0 && !allFilteredSelected}
                  checked={allFilteredSelected}
                  onChange={toggleSelectAll}
                  slotProps={{ input: { "aria-label": "Select all users" } }}
                />
              </TableCell>
              {(["name", "email", "role", "status", "createdAt"] as const).map((field) => (
                <TableCell
                  key={field}
                  aria-sort={sortField === field ? (sortDir === "asc" ? "ascending" : "descending") : undefined}
                  sx={{ cursor: "pointer" }}
                  onClick={() => handleSort(field)}
                >
                  <TableSortLabel
                    active={sortField === field}
                    direction={sortField === field ? sortDir : "asc"}
                  >
                    {field.charAt(0).toUpperCase() + field.slice(1)}
                  </TableSortLabel>
                </TableCell>
              ))}
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center" sx={{ py: 4, color: "text.secondary" }}>
                  No users found
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((user) => (
              <TableRow key={user.id} selected={selectedIds.has(user.id)}>
                <TableCell padding="checkbox">
                  <Checkbox checked={selectedIds.has(user.id)} onChange={() => toggleSelect(user.id)} slotProps={{ input: { "aria-label": `Select ${user.name}` } }} />
                </TableCell>
                <TableCell>{user.name}</TableCell>
                <TableCell>{user.email}</TableCell>
                <TableCell>{user.role}</TableCell>
                <TableCell>
                  <Chip
                    label={user.status || "active"}
                    color={user.status === "banned" ? "error" : "success"}
                    size="small"
                  />
                </TableCell>
                <TableCell>{new Date(user.createdAt).toLocaleDateString()}</TableCell>
                <TableCell>
                  <Button size="small" onClick={() => openEdit(user)}>Edit</Button>
                  <Button size="small" color="error" onClick={() => setDeleteTarget(user)}>Delete</Button>
                </TableCell>
              </TableRow>
            )))
          }</TableBody>
        </Table>
      </TableContainer>

      <Dialog open={formOpen} onClose={() => setFormOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editingUser ? "Edit User" : "New User"}</DialogTitle>
        <DialogContent>
          {formErrors._general && <Alert severity="error" sx={{ mb: 2 }}>{formErrors._general}</Alert>}
          <TextField label="Name" fullWidth margin="dense" value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            error={!!formErrors.name} helperText={formErrors.name} required />
          <TextField label="Email" type="email" fullWidth margin="dense" value={formData.email}
            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
            error={!!formErrors.email} helperText={formErrors.email} required />
          <TextField label="Password" type="password" fullWidth margin="dense" value={formData.password}
            onChange={(e) => setFormData({ ...formData, password: e.target.value })}
            error={!!formErrors.password} helperText={formErrors.password}
            required={!editingUser} />
          <TextField label="Confirm Password" type="password" fullWidth margin="dense" value={formData.confirmPassword}
            onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
            required={!editingUser} />
          <FormControl fullWidth margin="dense">
            <InputLabel>Role</InputLabel>
            <Select label="Role" value={formData.role}
              onChange={(e) => setFormData({ ...formData, role: e.target.value })}>
              <MenuItem value="admin">admin</MenuItem>
              <MenuItem value="user">user</MenuItem>
            </Select>
          </FormControl>
          {editingUser && (
            <FormControl fullWidth margin="dense">
              <InputLabel>Status</InputLabel>
              <Select label="Status" value={formData.status || "active"}
                onChange={(e) => setFormData({ ...formData, status: e.target.value })}>
                <MenuItem value="active">active</MenuItem>
                <MenuItem value="banned">banned</MenuItem>
              </Select>
            </FormControl>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setFormOpen(false)}>Cancel</Button>
          <Button onClick={handleSubmit} variant="contained">{editingUser ? "Save" : "Create"}</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <Typography>Are you sure you want to delete {deleteTarget?.name}?</Typography>
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
            {bulkAction === "delete" && `Delete ${selectedIds.size} user(s)? They will be moved to trash.`}
            {bulkAction === "ban" && `Ban ${selectedIds.size} user(s)?`}
            {bulkAction === "activate" && `Activate ${selectedIds.size} user(s)?`}
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
