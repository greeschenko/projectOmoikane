"use client";

import { useState } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import {
  AppBar, Toolbar, IconButton, Typography, Drawer,
  List, ListItem, ListItemButton, ListItemText, Box,
  useMediaQuery, useTheme,
} from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";

const navItems = [
  { label: "Dashboard", href: "/admin" },
  { label: "Users", href: "/admin/users" },
  { label: "Pages", href: "/admin/pages" },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const [mobileOpen, setMobileOpen] = useState(false);

  const sidebarContent = (
    <List>
      {navItems.map((item) => (
        <ListItem key={item.href} disablePadding>
          <ListItemButton
            component={Link}
            href={item.href}
            selected={pathname === item.href}
            onClick={() => setMobileOpen(false)}
          >
            <ListItemText primary={item.label} />
          </ListItemButton>
        </ListItem>
      ))}
    </List>
  );

  return (
    <Box sx={{ display: "flex" }}>
      {isMobile && (
        <AppBar position="fixed">
          <Toolbar>
            <IconButton
              color="inherit"
              edge="start"
              aria-label="toggle sidebar"
              onClick={() => setMobileOpen(!mobileOpen)}
            >
              <MenuIcon />
            </IconButton>
            <Typography variant="h6" sx={{ ml: 2 }}>
              Admin
            </Typography>
          </Toolbar>
        </AppBar>
      )}
      {isMobile ? (
        <Drawer open={mobileOpen} onClose={() => setMobileOpen(false)}>
          <Box sx={{ width: 250 }} role="navigation">
            {sidebarContent}
          </Box>
        </Drawer>
      ) : (
        <Drawer variant="permanent" sx={{ width: 240, flexShrink: 0 }}>
          <Toolbar />
          <Box role="navigation">{sidebarContent}</Box>
        </Drawer>
      )}
      <Box
        component="main"
        sx={{ flexGrow: 1, p: 3, mt: isMobile ? 8 : 0 }}
      >
        {children}
      </Box>
    </Box>
  );
}
