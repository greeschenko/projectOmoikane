"use client";

import { useState, useEffect } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import {
  Toolbar, Drawer,
  List, ListItem, ListItemButton, ListItemText, ListItemIcon, Box, Badge,
  useMediaQuery, useTheme,
} from "@mui/material";
import DashboardIcon from "@mui/icons-material/Dashboard";
import PeopleIcon from "@mui/icons-material/People";
import ArticleIcon from "@mui/icons-material/Article";
import BookIcon from "@mui/icons-material/Book";
import MailIcon from "@mui/icons-material/Mail";
import CollectionsIcon from "@mui/icons-material/Collections";
import SettingsIcon from "@mui/icons-material/Settings";
import DeleteSweepIcon from "@mui/icons-material/DeleteSweep";
import HistoryIcon from "@mui/icons-material/History";
import AdminAppBar from "./AdminAppBar";

const navItems = [
  { label: "Dashboard", href: "/admin", icon: <DashboardIcon /> },
  { label: "Users", href: "/admin/users", icon: <PeopleIcon /> },
  { label: "Pages", href: "/admin/pages", icon: <ArticleIcon /> },
  { label: "Blog", href: "/admin/blog", icon: <BookIcon /> },
  { label: "Messages", href: "/admin/messages", icon: <MailIcon /> },
  { label: "Contacts", href: "/admin/contacts", icon: <MailIcon /> },
  { label: "Media", href: "/admin/media", icon: <CollectionsIcon /> },
  { label: "Trash", href: "/admin/trash", icon: <DeleteSweepIcon /> },
  { label: "Audit Log", href: "/admin/audit-logs", icon: <HistoryIcon /> },
  { label: "Settings", href: "/admin/settings", icon: <SettingsIcon /> },
];

export default function AdminLayout({
  children,
  session,
}: {
  children: React.ReactNode;
  session: { userId: string; role: string };
}) {
  const pathname = usePathname();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const [mobileOpen, setMobileOpen] = useState(false);
  const [trashCount, setTrashCount] = useState(0);

  useEffect(() => {
    const fetchTrashCount = async () => {
      try {
        const res = await fetch("/api/trash/count");
        if (res.ok) {
          const data = await res.json();
          setTrashCount(data.count ?? 0);
        }
      } catch {
        // ignore
      }
    };
    fetchTrashCount();
    const interval = setInterval(fetchTrashCount, 30000);
    const onTrashChanged = () => fetchTrashCount();
    window.addEventListener("trash-changed", onTrashChanged);
    return () => {
      clearInterval(interval);
      window.removeEventListener("trash-changed", onTrashChanged);
    };
  }, []);

  const sidebarContent = (
    <List>
      {navItems.map((item) => (
        <ListItem key={item.href} disablePadding sx={{ pl: item.indent ? 2 : 0 }}>
          <ListItemButton
            component={Link}
            href={item.href}
            selected={pathname === item.href}
            onClick={() => setMobileOpen(false)}
          >
            <ListItemIcon>
              {item.label === "Trash" ? (
                <Badge badgeContent={trashCount} color="error">
                  {item.icon}
                </Badge>
              ) : (
                item.icon
              )}
            </ListItemIcon>
            <ListItemText primary={item.label} />
          </ListItemButton>
        </ListItem>
      ))}
    </List>
  );

  const toggleMobile = () => setMobileOpen(!mobileOpen);

  return (
    <Box sx={{ display: "flex", flexDirection: "column", minHeight: "100vh" }}>
      <AdminAppBar session={session} onMenuToggle={isMobile ? toggleMobile : undefined} />
      <Box sx={{ display: "flex", flex: 1 }}>
        <Box
          component="main"
          sx={{ flexGrow: 1, p: 3, mt: 0, order: { md: 1 } }}
        >
          {children}
        </Box>
        {isMobile ? (
          <Drawer open={mobileOpen} onClose={() => setMobileOpen(false)}>
            <Box sx={{ width: 250 }} role="navigation">
              {sidebarContent}
            </Box>
          </Drawer>
        ) : (
          <Drawer variant="permanent" sx={{ width: 240, flexShrink: 0, order: { md: -1 } }}>
            <Toolbar />
            <Box role="navigation">{sidebarContent}</Box>
          </Drawer>
        )}
      </Box>
    </Box>
  );
}
