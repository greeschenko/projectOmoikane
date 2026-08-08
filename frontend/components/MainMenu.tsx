"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Box, Button, Menu, MenuItem, IconButton, Drawer, List, ListItem, ListItemButton, ListItemText, useMediaQuery, useTheme } from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";

interface MenuPage {
  id: string;
  title: string;
  slug: string;
  parentId: string | null;
}

export default function MainMenu() {
  const [pages, setPages] = useState<MenuPage[]>([]);
  const [blogEnabled, setBlogEnabled] = useState(true);
  const [anchorEl, setAnchorEl] = useState<{ [key: string]: HTMLElement | null }>({});
  const [mobileOpen, setMobileOpen] = useState(false);
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));

  useEffect(() => {
    fetch("/api/pages?menu=true")
      .then((r) => r.json())
      .then(setPages)
      .catch(() => {});
    fetch("/api/settings")
      .then((r) => r.json())
      .then((data) => {
        if (data.blogEnabled !== undefined) setBlogEnabled(data.blogEnabled);
      })
      .catch(() => {});
  }, []);

  const rootPages = pages.filter((p) => !p.parentId);

  function getChildren(parentId: string): MenuPage[] {
    return pages.filter((p) => p.parentId === parentId);
  }

  function buildPath(page: MenuPage): string {
    const segments: string[] = [page.slug];
    let current = page;
    while (current.parentId) {
      const parent = pages.find((p) => p.id === current.parentId);
      if (!parent) break;
      segments.unshift(parent.slug);
      current = parent;
    }
    return "/pages/" + segments.join("/");
  }

  const hasAnyContent = blogEnabled || pages.length > 0;
  if (!hasAnyContent) return null;

  if (isMobile) {
    return (
      <>
        <IconButton
          color="inherit"
          aria-label="Open menu"
          aria-expanded={mobileOpen}
          aria-controls="mobile-navigation-menu"
          onClick={() => setMobileOpen(true)}
        >
          <MenuIcon />
        </IconButton>
        <Drawer open={mobileOpen} onClose={() => setMobileOpen(false)}>
          <Box id="mobile-navigation-menu" sx={{ width: 250 }} role="navigation" aria-label="Main navigation">
            <List>
              {blogEnabled && (
                <ListItem disablePadding>
                  <ListItemButton component={Link} href="/blog" onClick={() => setMobileOpen(false)}>
                    <ListItemText primary="Blog" />
                  </ListItemButton>
                </ListItem>
              )}
              <ListItem disablePadding>
                <ListItemButton component={Link} href="/contact" onClick={() => setMobileOpen(false)}>
                  <ListItemText primary="Contact" />
                </ListItemButton>
              </ListItem>
              {rootPages.map((page) => (
                <ListItem key={page.id} disablePadding>
                  <ListItemButton
                    component={Link}
                    href={buildPath(page)}
                    onClick={() => setMobileOpen(false)}
                  >
                    <ListItemText primary={page.title} />
                  </ListItemButton>
                </ListItem>
              ))}
            </List>
          </Box>
        </Drawer>
      </>
    );
  }

  return (
    <Box sx={{ display: "flex", alignItems: "center", gap: 1 }} role="navigation">
      {blogEnabled && (
        <Button color="inherit" component={Link} href="/blog">
          Blog
        </Button>
      )}
      <Button color="inherit" component={Link} href="/contact">
        Contact
      </Button>
      {rootPages.map((page) => {
        const children = getChildren(page.id);
        if (children.length > 0) {
          return (
            <Box key={page.id}>
              <Button
                color="inherit"
                onClick={(e) => setAnchorEl({ ...anchorEl, [page.id]: e.currentTarget })}
              >
                {page.title} ▼
              </Button>
              <Menu
                anchorEl={anchorEl[page.id]}
                open={Boolean(anchorEl[page.id])}
                onClose={() => setAnchorEl({ ...anchorEl, [page.id]: null })}
              >
                <MenuItem
                  component={Link}
                  href={buildPath(page)}
                  onClick={() => setAnchorEl({ ...anchorEl, [page.id]: null })}
                >
                  {page.title}
                </MenuItem>
                {children.map((child) => (
                  <MenuItem
                    key={child.id}
                    component={Link}
                    href={buildPath(child)}
                    onClick={() => setAnchorEl({ ...anchorEl, [page.id]: null })}
                  >
                    {child.title}
                  </MenuItem>
                ))}
              </Menu>
            </Box>
          );
        }
        return (
          <Button key={page.id} color="inherit" component={Link} href={buildPath(page)}>
            {page.title}
          </Button>
        );
      })}
    </Box>
  );
}
