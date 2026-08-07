"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import {
  AppBar, Toolbar, Typography, IconButton, Avatar,
  Menu, MenuItem, Box,
} from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";
import MessageWidget from "./MessageWidget";

export default function AdminAppBar({
  session,
  onMenuToggle,
}: {
  session: { userId: string; role: string };
  onMenuToggle?: () => void;
}) {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [siteName, setSiteName] = useState("Omoikane");
  const [avatarUrl, setAvatarUrl] = useState("");

  useEffect(() => {
    fetch("/api/settings")
      .then((res) => res.json())
      .then((data) => {
        if (data.siteName) setSiteName(data.siteName);
      })
      .catch(() => {});
    fetch("/api/settings/profile")
      .then((res) => res.json())
      .then((data) => {
        if (data.avatar) setAvatarUrl(data.avatar);
      })
      .catch(() => {});

    const onAvatarChanged = () => {
      fetch("/api/settings/profile")
        .then((res) => res.json())
        .then((data) => {
          if (data.avatar) setAvatarUrl(data.avatar);
        })
        .catch(() => {});
    };
    window.addEventListener("avatar-changed", onAvatarChanged);
    return () => window.removeEventListener("avatar-changed", onAvatarChanged);
  }, []);

  async function handleLogout() {
    setAnchorEl(null);
    await fetch("/api/auth/logout", { method: "POST" });
    window.location.href = "/";
  }

  return (
    <AppBar position="sticky" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }} aria-label="Admin header">
      <Toolbar>
        {onMenuToggle && (
          <IconButton
            color="inherit"
            edge="start"
            aria-label="toggle sidebar"
            onClick={onMenuToggle}
            sx={{ mr: 1 }}
          >
            <MenuIcon />
          </IconButton>
        )}
        <Typography variant="h6" sx={{ flexGrow: 1 }}>
          {siteName} Admin
        </Typography>
        <MessageWidget sessionUserId={session.userId} />
        <IconButton
          color="inherit"
          onClick={(e) => setAnchorEl(e.currentTarget)}
          aria-label="account"
        >
          <Avatar
            src={avatarUrl || undefined}
            sx={{ width: 32, height: 32, bgcolor: "secondary.main" }}
          >
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
