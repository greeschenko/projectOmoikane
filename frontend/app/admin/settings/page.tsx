"use client";

import { useState, useEffect } from "react";
import {
  Container, Typography, Paper, TextField, Button, Box, Alert, Tabs, Tab,
} from "@mui/material";
import RichTextEditor from "@/components/RichTextEditor";

export default function AdminSettings() {
  const [tab, setTab] = useState(0);

  const [siteName, setSiteName] = useState("");
  const [tagline, setTagline] = useState("");
  const [logo, setLogo] = useState("");
  const [favicon, setFavicon] = useState("");

  const [resetEmailSubject, setResetEmailSubject] = useState("");
  const [resetEmailBodyHTML, setResetEmailBodyHTML] = useState("");

  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/settings")
      .then((res) => res.json())
      .then((data) => {
        setSiteName(data.siteName || "");
        setTagline(data.tagline || "");
        setLogo(data.logo || "");
        setFavicon(data.favicon || "");
        setResetEmailSubject(data.resetEmailSubject || "");
        setResetEmailBodyHTML(data.resetEmailBodyHTML || "");
      })
      .catch(() => setMessage({ type: "error", text: "Failed to load settings" }))
      .finally(() => setLoading(false));
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMessage(null);
    const body: Record<string, unknown> = { siteName, tagline, logo, favicon };
    body.resetEmailSubject = resetEmailSubject;
    body.resetEmailBodyHTML = resetEmailBodyHTML;
    try {
      const res = await fetch("/api/settings", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (res.ok) {
        setMessage({ type: "success", text: "Settings saved" });
      } else {
        const err = await res.json();
        setMessage({ type: "error", text: err.error || "Failed to save" });
      }
    } catch {
      setMessage({ type: "error", text: "Failed to save settings" });
    }
  }

  function handleLogoUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => setLogo(reader.result as string);
    reader.readAsDataURL(file);
  }

  function handleFaviconUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => setFavicon(reader.result as string);
    reader.readAsDataURL(file);
  }

  if (loading) return <Container maxWidth="md"><Typography>Loading...</Typography></Container>;

  return (
    <Container maxWidth="md">
      <Typography variant="h4" sx={{ mb: 3 }}>Site Settings</Typography>
      {message && <Alert severity={message.type} sx={{ mb: 2 }}>{message.text}</Alert>}
      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2 }}>
        <Tab label="General" />
        <Tab label="Email Templates" />
      </Tabs>
      <Paper sx={{ p: 3 }} component="form" onSubmit={handleSubmit}>
        {tab === 0 && (
          <Box>
            <TextField
              label="Site Name"
              fullWidth
              margin="dense"
              value={siteName}
              onChange={(e) => setSiteName(e.target.value)}
              required
            />
            <TextField
              label="Tagline"
              fullWidth
              margin="dense"
              value={tagline}
              onChange={(e) => setTagline(e.target.value)}
            />
            <Box sx={{ mt: 2, mb: 1 }}>
              <Typography variant="body2" sx={{ mb: 1 }}>Logo</Typography>
              <Button variant="outlined" component="label" aria-label="logo">
                {logo ? "Change Logo" : "Upload Logo"}
                <input type="file" hidden accept="image/*" onChange={handleLogoUpload} />
              </Button>
              {logo && (
                <Box sx={{ mt: 1 }}>
                  <img src={logo} alt="Logo preview" style={{ maxHeight: 60, maxWidth: 200 }} />
                </Box>
              )}
            </Box>
            <Box sx={{ mt: 2, mb: 2 }}>
              <Typography variant="body2" sx={{ mb: 1 }}>Favicon</Typography>
              <Button variant="outlined" component="label" aria-label="favicon">
                {favicon ? "Change Favicon" : "Upload Favicon"}
                <input type="file" hidden accept="image/*" onChange={handleFaviconUpload} />
              </Button>
              {favicon && (
                <Box sx={{ mt: 1 }}>
                  <img src={favicon} alt="Favicon preview" style={{ maxHeight: 32, maxWidth: 32 }} />
                </Box>
              )}
            </Box>
          </Box>
        )}
        {tab === 1 && (
          <Box>
            <TextField
              label="Reset Email Subject"
              fullWidth
              margin="dense"
              value={resetEmailSubject}
              onChange={(e) => setResetEmailSubject(e.target.value)}
            />
            <Typography variant="body2" color="text.secondary" sx={{ mt: 2, mb: 1 }}>
              Reset Email Body (HTML with template variables: {"{{.ResetLink}}"}, {"{{.SiteName}}"}, {"{{.ExpiryHours}}"})
            </Typography>
            <RichTextEditor
              value={resetEmailBodyHTML}
              onChange={(html: string) => setResetEmailBodyHTML(html)}
              minimal
            />
          </Box>
        )}
        <Box sx={{ mt: 3 }}>
          <Button type="submit" variant="contained">Save</Button>
        </Box>
      </Paper>
    </Container>
  );
}
