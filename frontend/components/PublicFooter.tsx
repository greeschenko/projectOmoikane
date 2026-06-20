"use client";

import { useState, useEffect } from "react";
import { Box, Typography } from "@mui/material";

export default function PublicFooter() {
  const [siteName, setSiteName] = useState("Omoikane");

  useEffect(() => {
    fetch("/api/settings")
      .then((res) => res.json())
      .then((data) => {
        if (data.siteName) setSiteName(data.siteName);
      })
      .catch(() => {});
  }, []);

  return (
    <Box
      component="footer"
      sx={{ bgcolor: "grey.100", py: 3, mt: "auto", textAlign: "center" }}
    >
      <Typography variant="body2" color="text.secondary">
        &copy; {new Date().getFullYear()} {siteName}
      </Typography>
    </Box>
  );
}
