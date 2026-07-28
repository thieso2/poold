const { test, expect } = require("@playwright/test");
const AxeBuilder = require("@axe-core/playwright").default;

const observedState = {
  power: true, filter: true, heater: false, jets: false, bubbles: false, target_temp: 36
};

function representation(overrides = {}) {
  return {
    control: "automatic",
    control_revision: "revision-1",
    observed: {
      observation_id: 41,
      observed_at: new Date().toISOString(),
      connected: true,
      state: { ...observedState }
    },
    session: null,
    ...overrides
  };
}

function session(state, outcomes = {}) {
  const all = {};
  for (const field of ["power", "filter", "heater", "jets", "bubbles", "target_temp"]) {
    all[field] = outcomes[field] || { state: "confirmed" };
  }
  return {
    state,
    duration: "30m",
    started_at: new Date().toISOString(),
    expires_at: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
    intended: { ...observedState, heater: true },
    outcomes: all
  };
}

async function tabTo(page, selector) {
  for (let attempt = 0; attempt < 40; attempt++) {
    await page.keyboard.press("Tab");
    if (await page.locator(selector).evaluate(element => element === document.activeElement)) return;
  }
  throw new Error(`Could not reach ${selector} through keyboard Tab order`);
}

async function openDashboard(page, initial = representation()) {
  let current = initial;
  let nextPutMode = "";
  const requests = [];
  await page.addInitScript(() => localStorage.setItem("poold.token", "browser-token"));
  await page.route("**/*", async route => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/manual-session") {
      requests.push({ method: request.method(), body: request.postDataJSON?.(), headers: request.headers() });
      if (request.method() === "PUT") {
        if (nextPutMode === "offline") {
          nextPutMode = "";
          return route.fulfill({
            status: 503,
            contentType: "application/json",
            body: JSON.stringify({ error: { code: "pool_unreachable", message: "Pool is offline." } })
          });
        }
        if (nextPutMode === "conflict") {
          nextPutMode = "";
          current = representation({
            control: "manual",
            control_revision: "competing-revision",
            session: session("active")
          });
          return route.fulfill({
            status: 409,
            contentType: "application/json",
            body: JSON.stringify({ error: {
              code: "control_changed",
              message: "Control changed after this draft was opened.",
              current
            } })
          });
        }
        if (nextPutMode === "lost") {
          nextPutMode = "";
          return route.abort("connectionreset");
        }
        current = representation({
          control: "manual",
          control_revision: "revision-2",
          session: session("applying", { heater: { state: "pending" } })
        });
        return route.fulfill({ status: 202, contentType: "application/json", body: JSON.stringify(current) });
      }
      if (request.method() === "DELETE") {
        current = representation({ control_revision: "revision-3" });
        return route.fulfill({ contentType: "application/json", body: JSON.stringify(current) });
      }
      return route.fulfill({ contentType: "application/json", body: JSON.stringify(current) });
    }
    if (path === "/manual-session/retry") {
      requests.push({ method: request.method(), body: request.postDataJSON?.(), headers: request.headers() });
      current = { ...current, session: session("applying", { heater: { state: "pending" } }) };
      return route.fulfill({ status: 202, contentType: "application/json", body: JSON.stringify(current) });
    }
    if (path === "/status") {
      return route.fulfill({ contentType: "application/json", body: JSON.stringify({
        connected: true, observed_at: new Date().toISOString(), current_temp: 35, preset_temp: 36
      }) });
    }
    if (request.resourceType() === "document" || path.endsWith(".svg") || path.endsWith(".png")) {
      return route.continue();
    }
    return route.fulfill({ status: 404, contentType: "application/json", body: "{}" });
  });
  await page.goto("/");
  await expect(page.locator("#manualControl")).toBeEnabled();
  return {
    requests,
    setCurrent(value) { current = value; },
    failNextPut(mode) { nextPutMode = mode; }
  };
}

test("control layout remains usable and geometrically sound", async ({ page }, testInfo) => {
  await openDashboard(page);
  const viewport = page.viewportSize();
  const overflow = await page.evaluate(() => Array.from(document.querySelectorAll("body *"))
    .filter(element => {
      const rect = element.getBoundingClientRect();
      return rect.left < -0.5 || rect.right > document.documentElement.clientWidth + 0.5;
    })
    .map(element => ({ tag: element.tagName, id: element.id, className: element.className })));
  expect(overflow).toEqual([]);
  await expect(page.locator('meta[name="viewport"]')).toHaveAttribute("content", /viewport-fit=cover/);
  await expect(page.locator('meta[name="viewport"]')).not.toHaveAttribute("content", /user-scalable=no|maximum-scale=1/);
  await expect(page.locator('meta[name="apple-mobile-web-app-capable"]')).toHaveAttribute("content", "yes");

  // Automatic shows the summary and no switches at all.
  await expect(page.locator("[data-manual-cap]")).toHaveCount(0);
  await expect(page.locator(".pc-minis .pc-mini")).toHaveCount(5);
  await expect(page.locator(".pc-now-temp")).toContainText("35");

  await page.locator("#manualControl").click();
  const tiles = page.locator(".pc-cap");
  await expect(tiles).toHaveCount(5);
  const boxes = await tiles.evaluateAll(elements => elements.map(element => {
    const r = element.getBoundingClientRect();
    return { x: r.x, y: r.y, width: r.width, height: r.height };
  }));
  const scrollHeight = await page.evaluate(() => document.documentElement.scrollHeight);
  for (const box of boxes) {
    expect(box.width).toBeGreaterThanOrEqual(44);
    expect(box.height).toBeGreaterThanOrEqual(44);
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(viewport.width);
    expect(box.y).toBeGreaterThanOrEqual(0);
    expect(box.y + box.height).toBeLessThanOrEqual(scrollHeight);
  }
  // One single row: every tile shares a top edge.
  const tops = new Set(boxes.map(box => Math.round(box.y)));
  expect(tops.size).toBe(1);
  for (let i = 0; i < boxes.length; i++) for (let j = i + 1; j < boxes.length; j++) {
    const overlap = boxes[i].x < boxes[j].x + boxes[j].width &&
      boxes[i].x + boxes[i].width > boxes[j].x &&
      boxes[i].y < boxes[j].y + boxes[j].height &&
      boxes[i].y + boxes[i].height > boxes[j].y;
    expect(overlap).toBe(false);
  }
  for (const box of await page.locator("[data-manual-duration]").evaluateAll(els => els.map(e => e.getBoundingClientRect()))) {
    expect(box.height).toBeGreaterThanOrEqual(44);
  }
  await page.screenshot({ path: testInfo.outputPath("accepted-dashboard.png"), fullPage: true });
});

test("accessible semantics, focus, announcements, and reduced motion", async ({ page }) => {
  await openDashboard(page);
  await expect(page.locator("#manualSessionStatus")).toHaveAttribute("aria-live", "polite");

  await page.locator("#manualControl").click();
  const controls = page.locator("[data-manual-cap]");
  await expect(controls).toHaveCount(5);
  for (const control of await controls.all()) {
    await expect(control).toHaveAccessibleName(/Power|Filter|Heater|Jets|Bubbles/);
    await expect(control).toHaveAttribute("aria-pressed", /true|false/);
    await expect(control).toHaveAttribute("title", /Power|Filter|Heater|Jets|Bubbles/);
    await expect(control).toBeEnabled();
  }
  const heater = page.locator('[data-manual-cap="heater"]');
  await heater.focus();
  await page.keyboard.press("Tab");
  await page.keyboard.press("Shift+Tab");
  expect(await heater.evaluate(element => getComputedStyle(element).outlineStyle !== "none" ||
    getComputedStyle(element).boxShadow !== "none")).toBe(true);
  await expect(page.locator("#manualSessionStatus")).toContainText("Draft saved");

  // The icon carries its own name and state without any separate help text.
  await expect(heater).toHaveAccessibleName(/Heater (on|off)/);

  await page.emulateMedia({ reducedMotion: "reduce" });
  expect(await page.evaluate(() => {
    const svg = document.querySelector('[data-manual-cap="filter"] .spin');
    return getComputedStyle(svg).animationName;
  })).toBe("none");

  const accessibility = await new AxeBuilder({ page })
    .include("#poolControl")
    .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
    .analyze();
  expect(accessibility.violations).toEqual([]);
});

test("keyboard-only Manual-session lifecycle", async ({ page }) => {
  const harness = await openDashboard(page);
  await page.locator("body").focus();
  await tabTo(page, "#manualControl");
  await page.keyboard.press("Enter");
  await expect(page.locator("#manualSessionStatus")).toContainText("Draft saved");
  expect(harness.requests.filter(request => request.method !== "GET")).toHaveLength(0);

  await tabTo(page, '[data-manual-cap="power"]');
  await page.keyboard.press("Space");
  await expect(page.locator('[data-manual-cap="power"]')).toHaveAttribute("aria-pressed", "false");

  // Each duration is its own submit; no separate Apply control exists.
  await expect(page.locator("#manualSessionApply")).toHaveCount(0);
  await tabTo(page, '[data-manual-duration="30m"]');
  await page.keyboard.press("Enter");
  await expect(page.locator("#manualSessionStatus")).toContainText("Applying");
  expect(harness.requests.filter(request => request.method === "PUT")).toHaveLength(1);
  expect(harness.requests.filter(request => request.method === "PUT").at(-1).body.duration).toBe("30m");

  harness.setCurrent(representation({
    control: "manual",
    control_revision: "revision-2",
    session: session("degraded", { heater: { state: "failed", code: "spa_error", message: "Heater failed" } })
  }));
  await tabTo(page, "#refresh");
  await page.keyboard.press("Enter");
  await expect(page.locator("#manualSessionStatus")).toContainText("degraded");
  await tabTo(page, '[data-manual-act="retry"]');
  await page.keyboard.press("Enter");
  await expect.poll(() => harness.requests.filter(request => request.method === "POST").length).toBe(1);
  await expect(page.locator("#manualSessionStatus")).toContainText("Applying");

  harness.setCurrent(representation({ control: "manual", control_revision: "revision-2", session: session("active") }));
  await tabTo(page, "#refresh");
  await page.keyboard.press("Enter");
  await expect(page.locator("#manualSessionStatus")).toContainText("active");
  await tabTo(page, "#automaticControl");
  await page.keyboard.press("Enter");
  await expect.poll(() => harness.requests.filter(request => request.method === "DELETE").length).toBe(1);
  await expect(page.locator("#automaticControl")).toHaveAttribute("aria-pressed", "true");
});

test("draft, conflict, offline, stale, discard, expiry, and lost-response flows", async ({ page }, testInfo) => {
  test.skip(!testInfo.project.name.startsWith("chromium"), "Lifecycle matrix runs once in desktop Chromium");
  const harness = await openDashboard(page);

  await page.locator("#manualControl").click();
  expect(harness.requests.filter(request => request.method !== "GET")).toHaveLength(0);
  await page.locator("#automaticControl").click();
  await expect(page.locator("#manualControl")).toHaveAttribute("aria-pressed", "false");

  // A dirty draft asks inline before it is thrown away — no browser dialog.
  await page.locator("#manualControl").click();
  await page.locator('[data-manual-cap="heater"]').click();
  await page.locator("#automaticControl").click();
  await expect(page.locator(".pc-apply")).toContainText("Throw away this change?");
  await page.locator('[data-manual-act="keep"]').click();
  await expect(page.locator("#manualControl")).toHaveAttribute("aria-pressed", "true");
  await page.locator("#automaticControl").click();
  await page.locator('[data-manual-act="discard"]').click();
  await expect(page.locator("#manualControl")).toHaveAttribute("aria-pressed", "false");

  await page.locator("#manualControl").click();
  await page.locator('[data-manual-cap="heater"]').click();
  harness.failNextPut("offline");
  await page.locator('[data-manual-duration="30m"]').click();
  await expect(page.locator("#toast")).toContainText("offline");
  await expect(page.locator("#manualControl")).toHaveAttribute("aria-pressed", "true");

  harness.failNextPut("conflict");
  await page.locator('[data-manual-duration="30m"]').click();
  await expect(page.locator("#manualSessionStatus")).toContainText("Review this rebased draft");
  await expect(page.locator(".pc-label")).toContainText("CONTROL CHANGED");

  harness.failNextPut("lost");
  await page.locator('[data-manual-duration="30m"]').click();
  const lostKey = harness.requests.filter(request => request.method === "PUT").at(-1).headers["idempotency-key"];
  await page.locator('[data-manual-duration="30m"]').click();
  const retriedKey = harness.requests.filter(request => request.method === "PUT").at(-1).headers["idempotency-key"];
  expect(retriedKey).toBe(lostKey);

  // Editing a live session: the target stepper is available because the session heats.
  harness.setCurrent(representation({
    control: "manual",
    control_revision: "revision-edit",
    session: session("active")
  }));
  await page.locator("#refresh").click();
  await page.locator('[data-manual-act="edit"]').click();
  await expect(page.locator(".pc-val b")).toHaveText("36°");
  await page.locator('[data-manual-step="1"]').click();
  await expect(page.locator(".pc-val b")).toHaveText("37°");
  await page.locator('[data-manual-duration="10m"]').click();
  const edit = harness.requests.filter(request => request.method === "PUT").at(-1).body;
  expect(edit.base_observation_id).toBeUndefined();
  expect(edit.duration).toBe("10m");
  expect(Object.keys(edit.intended).sort()).toEqual(
    ["bubbles", "filter", "heater", "jets", "power", "target_temp"].sort()
  );
  expect(edit.intended.target_temp).toBe(37);

  harness.setCurrent(representation({
    control: "manual",
    control_revision: "revision-expiring",
    session: { ...session("active"), expires_at: new Date(Date.now() - 1000).toISOString() }
  }));
  const expiryRefreshes = harness.requests.filter(request => request.method === "GET").length;
  await page.locator("#refresh").click();
  await expect.poll(() => harness.requests.filter(request => request.method === "GET").length)
    .toBeGreaterThan(expiryRefreshes);
  await page.evaluate(() => {
    window.__announcements = [];
    new MutationObserver(() => window.__announcements.push(document.querySelector("#toast").textContent))
      .observe(document.querySelector("#toast"), { childList: true, subtree: true });
  });
  harness.setCurrent(representation({ control_revision: "revision-expired" }));
  await page.locator("#refresh").click();
  await expect.poll(() => page.evaluate(() => window.__announcements)).toContain(
    "Manual session ended. Automatic control resumed."
  );

  harness.setCurrent(representation({
    observed: {
      observation_id: 99,
      observed_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
      connected: true,
      state: observedState
    }
  }));
  await page.locator("#refresh").click();
  await expect(page.locator("#manualControl")).toBeDisabled();
  await expect(page.locator("#manualSessionStatus")).toContainText("stale");

  harness.setCurrent(representation({
    observed: {
      observation_id: 100,
      observed_at: new Date().toISOString(),
      connected: false,
      state: observedState
    }
  }));
  await page.locator("#refresh").click();
  await expect(page.locator("#manualControl")).toBeDisabled();
  await expect(page.locator("#manualSessionStatus")).toContainText("offline");
});
