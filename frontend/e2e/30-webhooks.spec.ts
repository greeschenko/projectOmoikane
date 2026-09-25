import { test, expect, type Page } from "@playwright/test";
import { isMobile, loginAsAdmin, waitForHydration } from "./helpers";

// The delivery pipeline target names are byte-identical in docker-compose and
// in the Helm chart (webhook-sink Service), so the same URLs work for both
// gate environments. 8099 is an unpublished port -> ECONNREFUSED.
const SINK_URL = "http://webhook-sink:8091/hook";
const DEAD_URL = "http://webhook-sink:8099/hook";

const POST_EVENT = "org.omoikane.content.post.published.v1";

interface Delivery {
  id: number;
  subscriptionId: number;
  eventId: string;
  eventType: string;
  payload: string;
  attempts: number;
  httpStatus: number;
  status: string;
  error: string;
}

async function createSubscriptionViaApi(page: Page, url: string, eventType = POST_EVENT) {
  const res = await page.request.post("/api/webhooks", {
    data: { eventType, url },
  });
  expect(res.ok()).toBeTruthy();
  const body = (await res.json()) as { id: number; active: boolean; secret?: string };
  expect(body.active).toBe(true);
  return body;
}

async function publishPost(page: Page, title: string) {
  const res = await page.request.post("/api/blog/posts", {
    data: {
      title,
      slug: `phase35-${Date.now()}-${Math.floor(Math.random() * 100000)}`,
      content: "<p>webhook gate post</p>",
      status: "published",
    },
  });
  expect(res.ok()).toBeTruthy();
}

// Polls the webhooks deliveries API until a delivery matching `predicate`
// exists, then returns it (Kafka pipeline is async, run once in production).
async function pollDelivery(page: Page, subscriptionId: number, predicate: (d: Delivery) => boolean): Promise<Delivery> {
  let found: Delivery | null = null;
  await expect
    .poll(
      async () => {
        const r = await page.request.get("/api/webhooks/deliveries", {
          params: { subscriptionId: String(subscriptionId) },
        });
        if (!r.ok()) return false;
        const body = (await r.json()) as { deliveries: Delivery[] };
        found = (body.deliveries ?? []).find(predicate) ?? null;
        return found !== null;
      },
      { timeout: 30000, intervals: [1000] }
    )
    .toBe(true);
  return found!;
}

test.describe("Admin Webhooks", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("sidebar has Webhooks nav link", async ({ page }) => {
    test.skip(isMobile(), "Sidebar is collapsed behind hamburger on mobile");
    await expect(page.getByRole("link", { name: /webhooks/i })).toBeVisible();
  });

  test("webhooks page loads and shows heading + tab bar", async ({ page }) => {
    await page.goto("/admin/webhooks");
    await expect(page.getByRole("heading", { name: /webhooks/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /subscriptions/i })).toBeVisible();
    await expect(page.getByRole("tab", { name: /delivery log/i })).toBeVisible();
  });

  test("can create a subscription through the UI and it appears in the list", async ({ page }) => {
    await page.goto("/admin/webhooks");
    await waitForHydration(page, "main");
    const newBtn = page.getByRole("button", { name: /new subscription/i });
    await newBtn.click();
    // Guard against the hydration race (see AGENTS.md): retry the click once.
    await expect(page.getByLabel("Event Type")).toBeVisible({ timeout: 3000 }).catch(async () => {
      await newBtn.click();
      await expect(page.getByLabel("Event Type")).toBeVisible({ timeout: 15000 });
    });
    await page.getByLabel("Destination URL").fill(SINK_URL);
    await page.getByRole("button", { name: /create/i }).last().click();
    // The one-time HMAC secret alert appears and the row is listed.
    // ("shown only once" disambiguates from the dialog's form label while the
    // closed dialog is still exit-transitioning in the DOM.)
    await expect(page.getByText(/HMAC secret \(shown only once\)/i)).toBeVisible();
    await expect(page.getByText(SINK_URL)).toBeVisible();
    // List shows the event type label ("User Registered" is the default).
    // exact: the actions cell ("Test ping User Registered subscription"
    // aria-label) and the explainer paragraph would substring-match otherwise.
    await expect(page.getByRole("cell", { name: "User Registered", exact: true })).toBeVisible();
  });

  // Phase 35 gate: proves the FULL pipeline — post.published flows outbox ->
  // relay -> Kafka -> webhooks consumer (group "webhooks") -> pending delivery
  // -> pump HTTP POST to webhook-sink with HMAC signature -> delivered row.
  test("publishing a blog post delivers a signed webhook via Kafka", async ({ page }) => {
    const sub = await createSubscriptionViaApi(page, SINK_URL);
    const title = `Phase35 Webhook ${Date.now()}`;
    await publishPost(page, title);

    const d = await pollDelivery(
      page,
      sub.id,
      (x) => x.status === "delivered" && x.payload.includes(title)
    );
    expect(d.httpStatus).toBe(200);
    expect(d.attempts).toBeGreaterThanOrEqual(1);
    expect(d.eventType).toBe(POST_EVENT);
  });

  // Failure semantics: ECONNREFUSED deliveries retry with exponential backoff
  // (attempts climb; status stays "failed" until the row expires after
  // WEBHOOKS_MAX_ATTEMPTS). We assert the retry behavior, not the (long)
  // terminal state.
  test("failing delivery URL retries with backoff", async ({ page }) => {
    const sub = await createSubscriptionViaApi(page, DEAD_URL);
    await publishPost(page, `Phase35 Failure ${Date.now()}`);

    const d = await pollDelivery(page, sub.id, (x) => x.attempts >= 2 && x.status === "failed");
    expect(d.httpStatus).toBe(0);
    expect(d.error).toContain("request failed");
  });

  // The test-ping route enqueues a synthetic ping delivery that runs through
  // the same pump; both the deliveries API and the Delivery Log tab surface it.
  test("test ping produces a delivered ping delivery", async ({ page }) => {
    const sub = await createSubscriptionViaApi(page, SINK_URL);
    const res = await page.request.post(`/api/webhooks/${sub.id}/test`);
    expect(res.ok()).toBeTruthy();
    const ping = await res.json();
    expect(ping.status).toBe("pending");
    expect(ping.eventType).toBe("org.omoikane.webhooks.ping.v1");

    const d = await pollDelivery(
      page,
      sub.id,
      (x) => x.eventType === "org.omoikane.webhooks.ping.v1" && x.status === "delivered"
    );
    expect(d.httpStatus).toBe(200);

    // The Delivery Log tab renders the delivered ping row.
    await page.goto("/admin/webhooks");
    await waitForHydration(page, "main");
    await page.getByRole("tab", { name: /delivery log/i }).click();
    await expect(page.getByText(/delivered/i).first()).toBeVisible();
  });
});