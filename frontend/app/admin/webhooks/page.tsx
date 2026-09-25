"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Container, Typography, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Paper, Box, Button, Chip, CircularProgress,
  Dialog, DialogTitle, DialogContent, DialogActions, TextField,
  MenuItem, Alert, IconButton, Switch, FormControlLabel, Tabs, Tab, Tooltip,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import SendIcon from "@mui/icons-material/Send";

interface WebhookSubscription {
  id: number;
  eventType: string;
  url: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

interface WebhookDelivery {
  id: number;
  createdAt: string;
  subscriptionId: number;
  eventId: string;
  eventType: string;
  attempts: number;
  httpStatus: number;
  status: string;
  error: string;
  nextAttemptAt: string;
}

// The 7 live backbone event types (backend/docs/events.md) that subscriptions
// can bind to. Display labels keep the table readable; the full CloudEvents
// type is what the API stores.
const EVENT_TYPES: { value: string; label: string }[] = [
  { value: "org.omoikane.auth.user.registered.v1", label: "User Registered" },
  { value: "org.omoikane.auth.login.v1", label: "Auth Login" },
  { value: "org.omoikane.content.page.published.v1", label: "Page Published" },
  { value: "org.omoikane.content.post.published.v1", label: "Post Published" },
  { value: "org.omoikane.media.uploaded.v1", label: "Media Uploaded" },
  { value: "org.omoikane.messages.contact.received.v1", label: "Contact Received" },
  { value: "org.omoikane.settings.updated.v1", label: "Settings Updated" },
];

function eventTypeLabel(value: string): string {
  return EVENT_TYPES.find((e) => e.value === value)?.label ?? value;
}

const fullEventType = (label: string): string =>
  EVENT_TYPES.find((e) => e.label === label)?.value ?? label;

// Delivery status chips. `failed`/`expired` use dark backgrounds so the white
// label passes WCAG AA (MUI `warning`/`error` at default shades are too light).
const STATUS_COLOR: Record<string, "success" | "error" | "default" | "secondary"> = {
  delivered: "success",
  failed: "error",
  expired: "secondary",
  pending: "default",
};

const STATUS_BG: Record<string, string> = {
  failed: "#b71c1c", // red 900
  expired: "#4a148c", // deepPurple 900
};

const DELIVERY_STATUSES = ["all", "pending", "delivered", "failed", "expired"];

export default function AdminWebhooks() {
  const [tab, setTab] = useState(0);
  const [subs, setSubs] = useState<WebhookSubscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [eventType, setEventType] = useState(EVENT_TYPES[0].label);
  const [url, setUrl] = useState("");
  const [active, setActive] = useState(true);
  const [secret, setSecret] = useState("");
  const [createdSecret, setCreatedSecret] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<WebhookSubscription | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  // Delivery log state.
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([]);
  const [deliveryTotal, setDeliveryTotal] = useState(0);
  const [deliveryLoading, setDeliveryLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState("all");
  const [subFilter, setSubFilter] = useState("all");
  const [deliveryPage, setDeliveryPage] = useState(0);
  const pageSize = 50;

  const fetchSubs = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/webhooks");
      if (res.ok) setSubs(await res.json());
    } catch {
      // ignore
    }
    setLoading(false);
  }, []);

  useEffect(() => { fetchSubs(); }, [fetchSubs]);

  const fetchDeliveries = useCallback(async () => {
    setDeliveryLoading(true);
    const params = new URLSearchParams();
    if (statusFilter !== "all") params.set("status", statusFilter);
    if (subFilter !== "all") params.set("subscriptionId", subFilter);
    params.set("limit", String(pageSize));
    params.set("offset", String(deliveryPage * pageSize));
    try {
      const res = await fetch(`/api/webhooks/deliveries?${params.toString()}`);
      if (res.ok) {
        const data = await res.json();
        setDeliveries(data.deliveries || []);
        setDeliveryTotal(data.total || 0);
      }
    } catch {
      // ignore
    }
    setDeliveryLoading(false);
  }, [statusFilter, subFilter, deliveryPage]);

  useEffect(() => { if (tab === 1) fetchDeliveries(); }, [tab, fetchDeliveries]);
  useEffect(() => { setDeliveryPage(0); }, [statusFilter, subFilter]);

  const handleCreate = async () => {
    setError("");
    if (!url.trim()) {
      setError("Destination URL is required");
      return;
    }
    const res = await fetch("/api/webhooks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        eventType: fullEventType(eventType),
        url: url.trim(),
        active,
        ...(secret.trim() ? { secret: secret.trim() } : {}),
      }),
    });
    if (res.ok) {
      const data = await res.json();
      setCreatedSecret(data.secret ?? null);
      setEventType(EVENT_TYPES[0].label);
      setUrl("");
      setActive(true);
      setSecret("");
      setCreateOpen(false);
      fetchSubs();
    } else {
      const data = await res.json().catch(() => ({}));
      setError(data.error || "Failed to create subscription");
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    const res = await fetch(`/api/webhooks/${deleteTarget.id}`, { method: "DELETE" });
    if (res.ok) {
      setDeleteTarget(null);
      fetchSubs();
      setNotice("Subscription deleted");
    }
  };

  const handleTestPing = async (sub: WebhookSubscription) => {
    const res = await fetch(`/api/webhooks/${sub.id}/test`, { method: "POST" });
    if (res.ok) {
      setNotice(`Test ping sent to ${sub.url}`);
    } else {
      const data = await res.json().catch(() => ({}));
      setNotice(data.error || "Test ping failed");
    }
  };

  const copySecret = () => {
    if (createdSecret) navigator.clipboard?.writeText(createdSecret);
  };

  return (
    <Container>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
        <Typography variant="h4" component="h1">
          Webhooks
        </Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => setCreateOpen(true)}>
          New Subscription
        </Button>
      </Box>

      <Box sx={{ mb: 2 }}>
        <Paper sx={{ p: 2, bgcolor: "action.hover" }} variant="outlined">
          <Typography variant="body1" sx={{ fontWeight: 600, mb: 1 }}>How webhooks work</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            A subscription delivers <strong>one event type</strong> to an HTTP URL. Platform
            events (user registered, auth login, page/post published, media uploaded, contact
            received, settings updated) flow over the Kafka backbone; a delivery worker POSTs
            each event to your URL as a <code>application/cloudevents+json</code> payload.
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Every request carries an <code>X-Omoikane-Signature: sha256=&lt;HMAC&gt;</code> header
            keyed by the subscription&apos;s secret — verify it on your side to authenticate the
            request. The raw secret is shown <strong>only once</strong> at creation. Deliveries
            retry with exponential backoff; after 6 failed attempts a delivery is marked{" "}
            <strong>expired</strong> and stays visible in the delivery log.
          </Typography>
        </Paper>
      </Box>

      {createdSecret && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setCreatedSecret(null)}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 1, flexWrap: "wrap" }}>
            <span>Subscription created — HMAC secret (shown only once):</span>
            <code>{createdSecret}</code>
            <IconButton size="small" onClick={copySecret} aria-label="copy secret">
              <ContentCopyIcon fontSize="small" />
            </IconButton>
          </Box>
        </Alert>
      )}

      {notice && (
        <Alert severity="info" sx={{ mb: 2 }} onClose={() => setNotice("")}>
          {notice}
        </Alert>
      )}

      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2 }}>
        <Tab label="Subscriptions" />
        <Tab label="Delivery Log" />
      </Tabs>

      {tab === 0 && (
        loading ? (
          <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
            <CircularProgress aria-label="Loading" />
          </Box>
        ) : subs.length === 0 ? (
          <Typography color="text.secondary" sx={{ textAlign: "center", py: 4 }}>
            No subscriptions yet. Create one to start receiving events.
          </Typography>
        ) : (
          <TableContainer component={Paper}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Event Type</TableCell>
                  <TableCell>URL</TableCell>
                  <TableCell>Active</TableCell>
                  <TableCell>Created</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {subs.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell>{eventTypeLabel(s.eventType)}</TableCell>
                    <TableCell sx={{ maxWidth: 280, overflow: "hidden", textOverflow: "ellipsis" }}>
                      {s.url}
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={s.active ? "Active" : "Inactive"}
                        color={s.active ? "success" : "default"}
                        size="small"
                      />
                    </TableCell>
                    <TableCell sx={{ whiteSpace: "nowrap" }}>
                      {new Date(s.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell align="right">
                      <Tooltip title="Send test ping (uses the real delivery pipeline)">
                        <IconButton
                          size="small"
                          aria-label={`Test ping ${eventTypeLabel(s.eventType)} subscription`}
                          onClick={() => handleTestPing(s)}
                        >
                          <SendIcon />
                        </IconButton>
                      </Tooltip>
                      <IconButton
                        size="small"
                        color="error"
                        aria-label={`Delete ${eventTypeLabel(s.eventType)} subscription`}
                        onClick={() => setDeleteTarget(s)}
                      >
                        <DeleteIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )
      )}

      {tab === 1 && (
        <>
          <Box sx={{ display: "flex", gap: 2, mb: 2, alignItems: "center" }}>
            <TextField
              select
              size="small"
              label="Status"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              sx={{ minWidth: 140 }}
            >
              {DELIVERY_STATUSES.map((s) => (
                <MenuItem key={s} value={s}>{s === "all" ? "All" : s.charAt(0).toUpperCase() + s.slice(1)}</MenuItem>
              ))}
            </TextField>
            <TextField
              select
              size="small"
              label="Subscription"
              value={subFilter}
              onChange={(e) => setSubFilter(e.target.value)}
              sx={{ minWidth: 180 }}
            >
              <MenuItem value="all">All</MenuItem>
              {subs.map((s) => (
                <MenuItem key={s.id} value={String(s.id)}>
                  {s.id} — {eventTypeLabel(s.eventType)}
                </MenuItem>
              ))}
            </TextField>
            <Typography variant="body2" color="text.secondary">
              {deliveryTotal} deliveries
            </Typography>
          </Box>

          {deliveryLoading ? (
            <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
              <CircularProgress aria-label="Loading" />
            </Box>
          ) : deliveries.length === 0 ? (
            <Typography color="text.secondary" sx={{ textAlign: "center", py: 4 }}>
              No deliveries found
            </Typography>
          ) : (
            <TableContainer component={Paper}>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Time</TableCell>
                    <TableCell>Event Type</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Attempts</TableCell>
                    <TableCell>HTTP</TableCell>
                    <TableCell>Error</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {deliveries.map((d) => (
                    <TableRow key={d.id}>
                      <TableCell sx={{ whiteSpace: "nowrap" }}>
                        {new Date(d.createdAt).toLocaleString()}
                      </TableCell>
                      <TableCell>{eventTypeLabel(d.eventType)}</TableCell>
                      <TableCell>
                        <Chip
                          label={d.status}
                          color={STATUS_COLOR[d.status] || "default"}
                          size="small"
                          sx={STATUS_BG[d.status] ? { bgcolor: STATUS_BG[d.status] } : undefined}
                        />
                      </TableCell>
                      <TableCell>{d.attempts}</TableCell>
                      <TableCell>{d.httpStatus || "-"}</TableCell>
                      <TableCell sx={{ maxWidth: 260, overflow: "hidden", textOverflow: "ellipsis" }}>
                        {d.error || "-"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}

          {deliveryTotal > pageSize && (
            <Box sx={{ display: "flex", justifyContent: "center", gap: 1, mt: 2 }}>
              <button
                disabled={deliveryPage === 0}
                onClick={() => setDeliveryPage((p) => Math.max(0, p - 1))}
                style={{ padding: "6px 12px", cursor: deliveryPage === 0 ? "default" : "pointer" }}
              >
                Previous
              </button>
              <Typography variant="body2" sx={{ alignSelf: "center" }}>
                Page {deliveryPage + 1} of {Math.ceil(deliveryTotal / pageSize)}
              </Typography>
              <button
                disabled={(deliveryPage + 1) * pageSize >= deliveryTotal}
                onClick={() => setDeliveryPage((p) => p + 1)}
                style={{ padding: "6px 12px", cursor: (deliveryPage + 1) * pageSize >= deliveryTotal ? "default" : "pointer" }}
              >
                Next
              </button>
            </Box>
          )}
        </>
      )}

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)}>
        <DialogTitle>New Webhook Subscription</DialogTitle>
        <DialogContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          <TextField
            select
            label="Event Type"
            value={eventType}
            onChange={(e) => setEventType(e.target.value)}
            fullWidth
            margin="normal"
            autoFocus
            slotProps={{ select: { "aria-label": "Event Type" } }}
          >
            {EVENT_TYPES.map((e) => (
              <MenuItem key={e.value} value={e.label}>{e.label}</MenuItem>
            ))}
          </TextField>
          <TextField
            label="Destination URL"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://example.com/hooks/omoikane"
            fullWidth
            margin="normal"
            slotProps={{
              htmlInput: { "aria-label": "Destination URL" },
            }}
          />
          <FormControlLabel
            control={
              <Switch checked={active} onChange={(e) => setActive(e.target.checked)} />
            }
            label="Active"
            sx={{ mt: 1 }}
          />
          <TextField
            label="HMAC Secret (optional — generated if empty)"
            value={secret}
            onChange={(e) => setSecret(e.target.value)}
            fullWidth
            margin="normal"
            slotProps={{
              htmlInput: { "aria-label": "HMAC Secret" },
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCreate}>Create</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Subscription</DialogTitle>
        <DialogContent>
          Delete the <strong>{deleteTarget ? eventTypeLabel(deleteTarget.eventType) : ""}</strong>{" "}
          subscription to <strong>{deleteTarget?.url}</strong>? Past deliveries stay in the
          delivery log.
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={handleDelete}>Delete</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}