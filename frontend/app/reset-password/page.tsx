"use client";

import { Suspense, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
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

function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") || "";

  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");

    if (!token) {
      setError("Invalid reset link");
      return;
    }

    if (password.length < 6) {
      setError("Password must be at least 6 characters");
      return;
    }

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    const res = await fetch("/api/auth/reset-password", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token, password }),
    });

    if (!res.ok) {
      const data = await res.json();
      setError(data.error || "Reset failed");
      return;
    }

    setSuccess(true);
  }

  if (!token) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        Invalid reset link. No token provided.
      </Alert>
    );
  }

  if (success) {
    return (
      <>
        <Alert severity="success" sx={{ mb: 2 }}>
          Password reset successfully!
        </Alert>
        <Box sx={{ textAlign: "center" }}>
          <MuiLink component={Link} href="/login" variant="body2">
            Sign in with your new password
          </MuiLink>
        </Box>
      </>
    );
  }

  return (
    <Box component="form" onSubmit={handleSubmit} noValidate>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <TextField
        label="New Password"
        type="password"
        fullWidth
        margin="normal"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
      />
      <TextField
        label="Confirm Password"
        type="password"
        fullWidth
        margin="normal"
        value={confirmPassword}
        onChange={(e) => setConfirmPassword(e.target.value)}
        required
      />
      <Button type="submit" variant="contained" fullWidth sx={{ mt: 2 }}>
        Reset Password
      </Button>
    </Box>
  );
}

export default function ResetPasswordPage() {
  return (
    <Container maxWidth="xs" sx={{ mt: 8 }}>
      <Paper sx={{ p: 4 }}>
        <Typography variant="h5" gutterBottom>
          Reset Password
        </Typography>
        <Suspense fallback={<div>Loading...</div>}>
          <ResetPasswordForm />
        </Suspense>
        <Box sx={{ mt: 2, textAlign: "center" }}>
          <MuiLink component={Link} href="/login" variant="body2">
            Back to Sign In
          </MuiLink>
        </Box>
      </Paper>
    </Container>
  );
}
