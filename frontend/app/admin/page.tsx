"use client";

import { Container, Typography } from "@mui/material";

export default function AdminDashboard() {
  return (
    <Container>
      <Typography variant="h4" component="h1" gutterBottom>
        Dashboard
      </Typography>
      <Typography variant="body1" color="text.secondary">
        Welcome to the admin panel.
      </Typography>
    </Container>
  );
}
