"use client";

import { useState, useEffect, useCallback } from "react";
import {
  IconButton, Badge, Menu, MenuItem, Typography, Box, Divider, Button,
} from "@mui/material";
import NotificationsIcon from "@mui/icons-material/Notifications";

interface MessageItem {
  id: string;
  title: string;
  content: string;
  createdAt: string;
  readBy: string[];
}

export default function MessageWidget({ sessionUserId }: { sessionUserId: string }) {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [messages, setMessages] = useState<MessageItem[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);

  const fetchMessages = useCallback(async () => {
    try {
      const res = await fetch("/api/messages");
      if (res.ok) {
        const data = await res.json();
        setMessages(data.messages || []);
        setUnreadCount(data.unreadCount || 0);
      }
    } catch {}
  }, []);

  useEffect(() => {
    fetchMessages();
  }, [fetchMessages]);

  async function handleMarkAsRead(messageId: string) {
    await fetch(`/api/messages/${messageId}/read`, { method: "POST" });
    fetchMessages();
  }

  async function handleMarkAllAsRead() {
    await fetch("/api/messages/read-all", { method: "POST" });
    fetchMessages();
  }

  return (
    <>
      <IconButton
        color="inherit"
        aria-label="notifications"
        onClick={(e) => setAnchorEl(e.currentTarget)}
      >
        <Badge badgeContent={unreadCount} color="error">
          <NotificationsIcon />
        </Badge>
      </IconButton>
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
        slotProps={{ paper: { sx: { width: 320, maxHeight: 400 } } }}
      >
        <Box sx={{ px: 2, py: 1, display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <Typography variant="subtitle2">Messages</Typography>
          {unreadCount > 0 && (
            <Button size="small" onClick={handleMarkAllAsRead}>
              Mark all as read
            </Button>
          )}
        </Box>
        <Divider />
        {messages.length === 0 ? (
          <MenuItem disabled>
            <Typography variant="body2" color="text.secondary">No messages</Typography>
          </MenuItem>
        ) : (
          messages.map((msg) => {
            const isUnread = !msg.readBy.includes(sessionUserId);
            return (
              <MenuItem
                key={msg.id}
                onClick={() => isUnread && handleMarkAsRead(msg.id)}
                sx={{ flexDirection: "column", alignItems: "flex-start", opacity: isUnread ? 1 : 0.6 }}
              >
                <Typography variant="body2" sx={{ fontWeight: isUnread ? "bold" : "normal" }}>
                  {msg.title}
                </Typography>
                {msg.content && (
                  <Typography variant="caption" color="text.secondary">
                    {msg.content}
                  </Typography>
                )}
              </MenuItem>
            );
          })
        )}
      </Menu>
    </>
  );
}
