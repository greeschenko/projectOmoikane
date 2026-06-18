"use client";

import { useState, useEffect } from "react";
import { Container, Typography, Box, Paper, Grid } from "@mui/material";
import PeopleIcon from "@mui/icons-material/People";
import ArticleIcon from "@mui/icons-material/Article";

export default function AdminDashboard() {
  const [stats, setStats] = useState<{
    userCount: number;
    pageCount: number;
    recentRegistrations: { date: string; count: number }[];
  } | null>(null);

  useEffect(() => {
    fetch("/api/dashboard/stats")
      .then((r) => r.json())
      .then(setStats)
      .catch(() => {});
  }, []);

  const maxCount = stats
    ? Math.max(...stats.recentRegistrations.map((r) => r.count), 1)
    : 1;

  return (
    <Container>
      <Typography variant="h4" component="h1" gutterBottom>
        Dashboard
      </Typography>
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Paper sx={{ p: 3, display: "flex", alignItems: "center", gap: 2 }}>
            <PeopleIcon color="primary" sx={{ fontSize: 48 }} />
            <Box>
              <Typography variant="h3">{stats?.userCount ?? "—"}</Typography>
              <Typography color="text.secondary">Users</Typography>
            </Box>
          </Paper>
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Paper sx={{ p: 3, display: "flex", alignItems: "center", gap: 2 }}>
            <ArticleIcon color="primary" sx={{ fontSize: 48 }} />
            <Box>
              <Typography variant="h3">{stats?.pageCount ?? "—"}</Typography>
              <Typography color="text.secondary">Pages</Typography>
            </Box>
          </Paper>
        </Grid>
      </Grid>

      <Typography variant="h5" gutterBottom>
        Registrations (Last 7 Days)
      </Typography>
      <Paper sx={{ p: 3, mb: 4 }}>
        {stats ? (
          <Box sx={{ display: "flex", alignItems: "flex-end", gap: 2, minHeight: 200 }}>
            {stats.recentRegistrations.map((r) => (
              <Box
                key={r.date}
                sx={{
                  flex: 1,
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                }}
              >
                <Typography variant="caption" sx={{ mb: 0.5 }}>
                  {r.count}
                </Typography>
                <Box
                  sx={{
                    width: "100%",
                    maxWidth: 40,
                    height: `${Math.max((r.count / maxCount) * 150, 4)}px`,
                    bgcolor: "primary.main",
                    borderRadius: 1,
                  }}
                />
                <Typography variant="caption" sx={{ mt: 0.5 }}>
                  {new Date(r.date + "T00:00:00").toLocaleDateString(undefined, {
                    weekday: "short",
                  })}
                </Typography>
              </Box>
            ))}
          </Box>
        ) : (
          <Typography color="text.secondary">Loading...</Typography>
        )}
      </Paper>
    </Container>
  );
}
