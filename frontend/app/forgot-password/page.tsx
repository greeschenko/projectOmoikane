"use client";

import { useState } from "react";
import Link from "next/link";
import {
  Container,
  Typography,
  TextField,
  Button,
  Box,
  Alert,
  Paper,
  Link as MuiLink,
} from "@mui/material";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSuccess(false);

    const res = await fetch("/api/auth/forgot-password", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email }),
    });
    if (!res.ok) {
      const data = await res.json();
      setError(data.error || "Something went wrong");
      return;
    }
    setSuccess(true);
  }

  return (
    <Container maxWidth="xs" sx={{ mt: 8 }}>
      <Paper sx={{ p: 4 }}>
        <Typography variant="h5" gutterBottom>
          Reset Password
        </Typography>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        {success && (
          <Alert severity="success" sx={{ mb: 2 }}>
            Check your email for a reset link
          </Alert>
        )}
        {!success && (
          <Box component="form" onSubmit={handleSubmit} noValidate>
            <TextField
              label="Email"
              type="email"
              fullWidth
              margin="normal"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
            <Button
              type="submit"
              variant="contained"
              fullWidth
              sx={{ mt: 2 }}
            >
              Send Reset Link
            </Button>
          </Box>
        )}
        <Box sx={{ mt: 2, textAlign: "center" }}>
          <MuiLink component={Link} href="/login" variant="body2">
            Back to Sign In
          </MuiLink>
        </Box>
      </Paper>
    </Container>
  );
}
