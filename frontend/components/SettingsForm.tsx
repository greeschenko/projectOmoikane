"use client";

import { useState, useEffect } from "react";
import {
  Container, Typography, TextField, Button, Box, Alert, Paper, Tabs, Tab,
} from "@mui/material";

export default function SettingsForm() {
  const [tabIndex, setTabIndex] = useState(0);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [avatar, setAvatar] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [profileSuccess, setProfileSuccess] = useState(false);
  const [passwordSuccess, setPasswordSuccess] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/settings/profile")
      .then((res) => res.json())
      .then((data) => {
        if (data.name) setName(data.name);
        if (data.email) setEmail(data.email);
        if (data.avatar) setAvatar(data.avatar);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function validatePassword() {
    const errors: Record<string, string> = {};
    if (!currentPassword) errors.currentPassword = "Current password is required";
    if (!newPassword || newPassword.length < 6) errors.newPassword = "Password must be at least 6 characters";
    if (newPassword !== confirmPassword) errors.confirmPassword = "Passwords do not match";
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function handleProfileSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setProfileSuccess(false);
    if (!name.trim()) {
      setError("Name cannot be empty");
      return;
    }
    const res = await fetch("/api/settings/profile", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, email, avatar }),
    });
    if (!res.ok) {
      const data = await res.json();
      setError(data.error || "Failed to update profile");
      return;
    }
    setProfileSuccess(true);
  }

  async function handlePasswordSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPasswordSuccess(false);
    if (!validatePassword()) return;

    const res = await fetch("/api/settings/password", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ currentPassword, newPassword }),
    });
    if (!res.ok) {
      const data = await res.json();
      setError(data.error || "Failed to update password");
      return;
    }
    setPasswordSuccess(true);
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
  }

  function handleAvatarUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => setAvatar(reader.result as string);
    reader.readAsDataURL(file);
  }

  if (loading) return <Container maxWidth="md" sx={{ mt: 4 }}><Typography>Loading...</Typography></Container>;

  return (
    <Container maxWidth="md" sx={{ mt: 4 }}>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Paper sx={{ p: 2 }}>
        <Box sx={{ display: "flex", gap: 3 }}>
          <Tabs
            orientation="vertical"
            value={tabIndex}
            onChange={(_, v) => setTabIndex(v)}
            sx={{ borderRight: 1, borderColor: "divider", minWidth: 160 }}
          >
            <Tab label="Profile" />
            <Tab label="Password" />
            <Tab label="Avatar" />
          </Tabs>

          <Box sx={{ flexGrow: 1, p: 1 }}>
            {tabIndex === 0 && (
              <Box>
                <Typography variant="h5" gutterBottom>Profile</Typography>
                {profileSuccess && (
                  <Alert severity="success" sx={{ mb: 2 }}>Profile updated successfully</Alert>
                )}
                <Box component="form" onSubmit={handleProfileSubmit} noValidate>
                  <TextField
                    label="Name"
                    fullWidth
                    margin="normal"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                  />
                  <TextField
                    label="Email"
                    type="email"
                    fullWidth
                    margin="normal"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                  />
                  <Button type="submit" variant="contained" sx={{ mt: 2 }}>
                    Save Changes
                  </Button>
                </Box>
              </Box>
            )}

            {tabIndex === 1 && (
              <Box>
                <Typography variant="h5" gutterBottom>Change Password</Typography>
                {passwordSuccess && (
                  <Alert severity="success" sx={{ mb: 2 }}>Password changed successfully</Alert>
                )}
                <Box component="form" onSubmit={handlePasswordSubmit} noValidate>
                  <TextField
                    label="Current Password"
                    type="password"
                    fullWidth
                    margin="normal"
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                    error={!!fieldErrors.currentPassword}
                    helperText={fieldErrors.currentPassword}
                    required
                  />
                  <TextField
                    label="New Password"
                    type="password"
                    fullWidth
                    margin="normal"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    error={!!fieldErrors.newPassword}
                    helperText={fieldErrors.newPassword}
                    required
                  />
                  <TextField
                    label="Confirm New Password"
                    type="password"
                    fullWidth
                    margin="normal"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    error={!!fieldErrors.confirmPassword}
                    helperText={fieldErrors.confirmPassword}
                    required
                  />
                  <Button type="submit" variant="contained" sx={{ mt: 2 }}>
                    Change Password
                  </Button>
                </Box>
              </Box>
            )}

            {tabIndex === 2 && (
              <Box>
                <Typography variant="h5" gutterBottom>Avatar</Typography>
                <Box sx={{ mt: 2, mb: 1 }}>
                  <Button variant="outlined" component="label" aria-label="upload avatar">
                    {avatar ? "Change Avatar" : "Upload Avatar"}
                    <input type="file" hidden accept="image/*" onChange={handleAvatarUpload} />
                  </Button>
                  {avatar && (
                    <Box sx={{ mt: 2 }}>
                      <img src={avatar} alt="Avatar preview" style={{ maxHeight: 64, maxWidth: 64, borderRadius: "50%" }} />
                    </Box>
                  )}
                </Box>
              </Box>
            )}
          </Box>
        </Box>
      </Paper>
    </Container>
  );
}
