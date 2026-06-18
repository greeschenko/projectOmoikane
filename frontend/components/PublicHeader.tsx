"use client";

import { useState } from "react";
import Link from "next/link";
import {
  AppBar, Toolbar, Typography, Button,
  IconButton, Menu, MenuItem, Avatar, Box,
} from "@mui/material";
import MainMenu from "./MainMenu";
import MessageWidget from "./MessageWidget";

export default function PublicHeader({
  session,
}: {
  session: { userId: string; role: string } | null;
}) {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  async function handleLogout() {
    setAnchorEl(null);
    await fetch("/api/auth/logout", { method: "POST" });
    window.location.href = "/";
  }

  return (
    <AppBar position="static" sx={{ bgcolor: "background.paper", color: "text.primary" }} aria-label="Public header">
      <Toolbar>
        <Typography variant="h6" sx={{ flexShrink: 0, mr: 2 }}>
          <Link href="/" style={{ color: "inherit", textDecoration: "none" }}>
            Omoikane
          </Link>
        </Typography>
        <MainMenu />
        <Box sx={{ flexGrow: 1 }} />
        {session ? (
          <>
            <MessageWidget sessionUserId={session.userId} />
            <IconButton
              color="inherit"
              onClick={(e) => setAnchorEl(e.currentTarget)}
              aria-label="user menu"
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
              {session.role === "admin" && (
                <MenuItem
                  component={Link}
                  href="/admin"
                  onClick={() => setAnchorEl(null)}
                >
                  Admin Panel
                </MenuItem>
              )}
              <MenuItem onClick={handleLogout}>Exit</MenuItem>
            </Menu>
          </>
        ) : (
          <Button color="inherit" component={Link} href="/login">
            Login
          </Button>
        )}
      </Toolbar>
    </AppBar>
  );
}
