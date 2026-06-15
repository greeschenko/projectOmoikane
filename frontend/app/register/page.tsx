"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
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

export default function RegisterPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  function validate() {
    const errors: Record<string, string> = {};
    const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailPattern.test(email)) errors.email = "Please enter a valid email address";
    if (password !== confirmPassword) errors.password = "Passwords do not match";
    if (!password || password.length < 6) errors.password = "Password must be at least 6 characters";
    if (!name.trim()) errors.name = "Name is required";
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!validate()) return;

    const res = await fetch("/api/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, email, password }),
    });
    if (!res.ok) {
      const data = await res.json();
      setError(data.error || "Registration failed");
      return;
    }
    router.push("/login");
  }

  return (
    <Container maxWidth="xs" sx={{ mt: 8 }}>
      <Paper sx={{ p: 4 }}>
        <Typography variant="h5" gutterBottom>
          Create Account
        </Typography>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        <Box component="form" onSubmit={handleSubmit} noValidate>
          <TextField
            label="Name"
            fullWidth
            margin="normal"
            value={name}
            onChange={(e) => setName(e.target.value)}
            error={!!fieldErrors.name}
            helperText={fieldErrors.name}
            required
          />
          <TextField
            label="Email"
            type="email"
            fullWidth
            margin="normal"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            error={!!fieldErrors.email}
            helperText={fieldErrors.email}
            required
          />
          <TextField
            label="Password"
            type="password"
            fullWidth
            margin="normal"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={!!fieldErrors.password}
            helperText={fieldErrors.password}
            required
          />
          <TextField
            label="Confirm Password"
            type="password"
            fullWidth
            margin="normal"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            error={!!fieldErrors.password}
            required
          />
          <Button
            type="submit"
            variant="contained"
            fullWidth
            sx={{ mt: 2 }}
          >
            Sign Up
          </Button>
        </Box>
        <Box sx={{ mt: 2, textAlign: "center" }}>
          <MuiLink component={Link} href="/login" variant="body2">
            Already have an account? Sign in
          </MuiLink>
        </Box>
      </Paper>
    </Container>
  );
}
