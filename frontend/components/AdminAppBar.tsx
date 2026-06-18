"use client";

import { useState } from "react";
import Link from "next/link";
import {
  AppBar, Toolbar, Typography, IconButton, Avatar,
  Menu, MenuItem, Box,
} from "@mui/material";
import MessageWidget from "./MessageWidget";

export default function AdminAppBar({
  session,
}: {
  session: { userId: string; role: string };
}) {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  async function handleLogout() {
    setAnchorEl(null);
    await fetch("/api/auth/logout", { method: "POST" });
    window.location.href = "/";
  }

  return (
    <AppBar position="sticky" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }} aria-label="Admin header">
      <Toolbar>
        <Typography variant="h6" sx={{ flexGrow: 1 }}>
          Omoikane Admin
        </Typography>
        <MessageWidget sessionUserId={session.userId} />
        <IconButton
          color="inherit"
          onClick={(e) => setAnchorEl(e.currentTarget)}
          aria-label="account"
        >
          <Avatar sx={{ width: 32, height: 32, bgcolor: "secondary.main" }}>
            {session.role[0].toUpperCase()}
          </Avatar>
        </IconButton>
        <Menu
          anchorEl={anchorEl}
          open={Boolean(anchorEl)}
          onClose={() => setAnchorEl(null)}
        >
          <MenuItem
            component={Link}
            href="/settings"
            onClick={() => setAnchorEl(null)}
          >
            Settings
          </MenuItem>
          <MenuItem onClick={handleLogout}>Exit</MenuItem>
        </Menu>
      </Toolbar>
    </AppBar>
  );
}
