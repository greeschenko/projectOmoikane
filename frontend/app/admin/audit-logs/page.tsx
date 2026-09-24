"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Table, TableBody,
  TableCell, TableContainer, TableHead, TableRow, Paper,
  Box, Chip, Tabs, Tab, CircularProgress, TextField, InputAdornment,
} from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";

interface AuditLogEntry {
  id: number;
  userId: number;
  userName: string;
  action: string;
  entityType: string;
  entityId: number;
  detail: string;
  ip: string;
  userAgent: string;
  createdAt: string;
}

const ACTION_COLORS: Record<string, "success" | "error" | "warning" | "info" | "primary" | "secondary" | "default"> = {
  create: "success",
  update: "info",
  delete: "error",
  login: "primary",
  logout: "secondary",
  restore: "warning",
  batch_delete: "error",
  batch_update: "info",
  // Phase 32: event-driven audit rows (Kafka backbone) reuse the palette;
  // 'delete'/'restore' stay for future fine-grained CRUD events.
  register: "success",
  publish: "success",
  upload: "info",
  contact: "warning",
};

// MUI `info` (#0288d1) and `warning` (#ed6c02) fail WCAG AA 4.5:1 with white
// text; darken with the 900 shades so the action chips pass the a11y gate.
const ACTION_BG: Record<string, string> = {
  update: "#01579b", // lightBlue 900
  batch_update: "#01579b",
  upload: "#01579b",
  restore: "#bf360c", // deepOrange 900
  contact: "#bf360c",
};

const ENTITY_TYPES = ["all", "user", "page", "post", "media", "contact", "message", "tag", "category", "settings"];

export default function AdminAuditLog() {
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [filterEntity, setFilterEntity] = useState("all");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(0);
  const pageSize = 50;

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams();
    if (filterEntity !== "all") params.set("entity", filterEntity);
    if (search) params.set("search", search);
    params.set("limit", String(pageSize));
    params.set("offset", String(page * pageSize));

    try {
      const res = await fetch(`/api/audit-logs?${params.toString()}`);
      if (res.ok) {
        const data = await res.json();
        setLogs(data.logs || []);
        setTotal(data.total || 0);
      }
    } catch {
      // ignore
    }
    setLoading(false);
  }, [filterEntity, search, page]);

  useEffect(() => { fetchLogs(); }, [fetchLogs]);

  useEffect(() => { setPage(0); }, [filterEntity, search]);

  return (
    <Container>
      <Typography variant="h4" component="h1" sx={{ mb: 2 }}>
        Audit Log
      </Typography>

      <Box sx={{ display: "flex", gap: 2, mb: 2, alignItems: "center" }}>
        <TextField
          size="small"
          placeholder="Search user or detail..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          sx={{ minWidth: 250 }}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            },
          }}
        />
        <Typography variant="body2" color="text.secondary">
          {total} entries
        </Typography>
      </Box>

      <Tabs
        value={filterEntity}
        onChange={(_, v) => setFilterEntity(v)}
        variant="scrollable"
        scrollButtons="auto"
        sx={{ mb: 2 }}
      >
        {ENTITY_TYPES.map((e) => (
          <Tab key={e} value={e} label={e === "all" ? "All" : e.charAt(0).toUpperCase() + e.slice(1)} />
        ))}
      </Tabs>

      {loading ? (
        <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
          <CircularProgress aria-label="Loading" />
        </Box>
      ) : logs.length === 0 ? (
        <Typography color="text.secondary" sx={{ textAlign: "center", py: 4 }}>
          No audit log entries found
        </Typography>
      ) : (
        <TableContainer component={Paper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Time</TableCell>
                <TableCell>User</TableCell>
                <TableCell>Action</TableCell>
                <TableCell>Entity</TableCell>
                <TableCell>Detail</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell sx={{ whiteSpace: "nowrap" }}>
                    {new Date(log.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>{log.userName || "system"}</TableCell>
                  <TableCell>
                    <Chip
                      label={log.action}
                      color={ACTION_COLORS[log.action] || "default"}
                      size="small"
                      sx={ACTION_BG[log.action] ? { bgcolor: ACTION_BG[log.action] } : undefined}
                    />
                  </TableCell>
                  <TableCell>
                    {log.entityType}
                    {log.entityId > 0 && ` #${log.entityId}`}
                  </TableCell>
                  <TableCell sx={{ maxWidth: 300, overflow: "hidden", textOverflow: "ellipsis" }}>
                    {log.detail || "-"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {total > pageSize && (
        <Box sx={{ display: "flex", justifyContent: "center", gap: 1, mt: 2 }}>
          <button
            disabled={page === 0}
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            style={{ padding: "6px 12px", cursor: page === 0 ? "default" : "pointer" }}
          >
            Previous
          </button>
          <Typography variant="body2" sx={{ alignSelf: "center" }}>
            Page {page + 1} of {Math.ceil(total / pageSize)}
          </Typography>
          <button
            disabled={(page + 1) * pageSize >= total}
            onClick={() => setPage((p) => p + 1)}
            style={{ padding: "6px 12px", cursor: (page + 1) * pageSize >= total ? "default" : "pointer" }}
          >
            Next
          </button>
        </Box>
      )}
    </Container>
  );
}
