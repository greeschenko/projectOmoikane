"use client";

import { Button, Container, Typography, Box } from "@mui/material";

export default function HomePage() {
  return (
    <Container maxWidth="sm" sx={{ mt: 8, textAlign: "center" }}>
      <Typography variant="h3" component="h1" gutterBottom>
        projectOmoikane
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Welcome to your Next.js app with Material UI.
      </Typography>
      <Box sx={{ display: "flex", gap: 2, justifyContent: "center" }}>
        <Button variant="contained">Get Started</Button>
        <Button variant="outlined">Learn More</Button>
      </Box>
      <Box sx={{ mt: 4, display: "flex", gap: 3, justifyContent: "center" }}>
        <Typography
          component="a"
          href="https://opencode.ai/docs"
          target="_blank"
          rel="noopener"
          color="primary"
          sx={{ textDecoration: "underline", cursor: "pointer" }}
        >
          Documentation
        </Typography>
        <Typography
          component="a"
          href="https://github.com/anomalyco/opencode"
          target="_blank"
          rel="noopener"
          color="primary"
          sx={{ textDecoration: "underline", cursor: "pointer" }}
        >
          GitHub
        </Typography>
      </Box>
    </Container>
  );
}
