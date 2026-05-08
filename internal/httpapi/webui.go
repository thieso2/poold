package httpapi

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"sync"
)

func (a *API) handleWebUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(webUIHTML))
}

func (a *API) handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write([]byte(faviconSVG))
}

func (a *API) handleAppleTouchIcon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(appleTouchIconPNG())
}

var (
	appleTouchIconOnce sync.Once
	appleTouchIconData []byte
)

func appleTouchIconPNG() []byte {
	appleTouchIconOnce.Do(func() {
		appleTouchIconData = renderAppIconPNG(180)
	})
	return appleTouchIconData
}

const webUIHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover">
<meta name="theme-color" content="#007c89">
<link rel="icon" type="image/svg+xml" href="/favicon.svg">
<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png">
<title>Pooly Control</title>
<style>
:root {
  color-scheme: light;
  --bg: #f4f7f8;
  --panel: #ffffff;
  --text: #172126;
  --muted: #5d6b73;
  --line: #d8e1e5;
  --accent: #007c89;
  --accent-strong: #005e67;
  --ok: #1d7f45;
  --warn: #a45f00;
  --bad: #b42318;
  --cool: #235ea8;
  --shadow: 0 12px 32px rgba(23, 33, 38, .10);
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font: 15px/1.45 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  touch-action: pan-x pan-y;
  -webkit-text-size-adjust: 100%;
}
button, input, select {
  font: inherit;
}
button {
  min-height: 42px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #fff;
  color: var(--text);
  font-weight: 700;
}
button.primary {
  border-color: var(--accent);
  background: var(--accent);
  color: #fff;
}
button.danger {
  border-color: #f0b8b3;
  color: var(--bad);
}
button:disabled {
  opacity: .55;
}
input, select {
  width: 100%;
  min-height: 42px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #fff;
  color: var(--text);
  font-size: 16px;
  padding: 9px 10px;
}
label {
  display: grid;
  gap: 6px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}
.app {
  width: min(1120px, 100%);
  margin: 0 auto;
  padding: 14px;
}
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0 16px;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
}
.top-actions {
  display: flex;
  gap: 8px;
}
.mark {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 6px 18px rgba(0, 124, 137, .22);
}
.mark img {
  display: block;
  width: 100%;
  height: 100%;
}
h1, h2, h3, p {
  margin: 0;
}
h1 {
  font-size: 22px;
}
h2 {
  font-size: 16px;
}
h3 {
  font-size: 14px;
}
.muted {
  color: var(--muted);
}
.grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}
.panel {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 8px;
  box-shadow: var(--shadow);
  padding: 14px;
}
.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}
.status-hero {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
  align-items: end;
}
.temp {
  font-size: 52px;
  line-height: 1;
  letter-spacing: 0;
  font-weight: 800;
}
.target {
  color: var(--muted);
  font-weight: 700;
}
.badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 28px;
  padding: 4px 9px;
  border-radius: 999px;
  background: #edf4f6;
  color: var(--muted);
  font-weight: 800;
  font-size: 12px;
}
.badge.ok { background: #e8f5ee; color: var(--ok); }
.badge.bad { background: #fdeceb; color: var(--bad); }
.badge.warn { background: #fff3df; color: var(--warn); }
.metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 14px;
}
.metric {
  border-top: 1px solid var(--line);
  padding-top: 10px;
}
.metric span {
  display: block;
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}
.metric strong {
  display: block;
  margin-top: 2px;
  font-size: 15px;
}
.weather-widget {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 4px 10px;
  align-items: center;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--line);
}
.weather-widget strong {
  grid-row: span 2;
  font-size: 28px;
  line-height: 1;
}
.weather-widget span {
  color: var(--muted);
  font-size: 13px;
}
.controls {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.manual-strip {
  display: grid;
  gap: 10px;
  margin-bottom: 12px;
  padding: 11px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #f8fbfc;
}
.manual-strip.active {
  border-color: #f1c27d;
  background: #fff6e8;
}
.manual-strip strong,
.manual-strip span {
  display: block;
}
.manual-strip span {
  color: var(--muted);
  font-size: 13px;
}
.manual-actions {
  display: none;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.manual-strip.active .manual-actions {
  display: grid;
}
.control {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 54px;
  padding: 10px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #fff;
  text-align: left;
}
.control.on {
  border-color: rgba(0, 124, 137, .45);
  background: #e9f6f7;
  color: var(--accent-strong);
}
.dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: #a7b2b8;
  flex: 0 0 auto;
}
.control.on .dot {
  background: var(--accent);
}
.control.manual {
  box-shadow: inset 0 0 0 2px rgba(241, 194, 125, .65);
}
.temp-row {
  display: grid;
  grid-template-columns: 42px 1fr 42px;
  gap: 8px;
  align-items: end;
  margin: 12px 0;
}
.temp-row input {
  text-align: center;
  font-weight: 800;
  font-size: 22px;
}
.forms {
  display: grid;
  gap: 10px;
}
.row {
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
}
.row.two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.row.three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.segments {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  border: 1px solid var(--line);
  border-radius: 8px;
  overflow: hidden;
}
.segments button {
  border: 0;
  border-radius: 0;
  min-height: 40px;
}
.segments button.active {
  background: var(--accent);
  color: #fff;
}
.days {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 5px;
}
.day {
  min-height: 36px;
  padding: 0;
}
.day.active {
  background: var(--cool);
  border-color: var(--cool);
  color: #fff;
}
.plan-list, .activity-list {
  display: grid;
  gap: 8px;
}
.plan, .activity {
  border-top: 1px solid var(--line);
  padding-top: 10px;
  display: grid;
  gap: 7px;
}
.plan:first-child, .activity:first-child {
  border-top: 0;
  padding-top: 0;
}
.plan-main {
  display: flex;
  justify-content: space-between;
  gap: 8px;
}
.plan-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.activity {
  grid-template-columns: 92px 1fr;
  align-items: start;
}
.activity time {
  color: var(--muted);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
  line-height: 1.35;
  padding-top: 2px;
}
.activity strong {
  display: block;
  font-size: 13px;
}
.activity span {
  color: var(--muted);
  font-size: 13px;
}
.activity-head {
  align-items: start;
  flex-wrap: wrap;
}
.tabs {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
}
.activity-tabs {
  width: 100%;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.tabs button.active {
  background: #172126;
  border-color: #172126;
  color: #fff;
}
.pager {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 8px;
  align-items: center;
  margin-top: 12px;
}
.pager span {
  color: var(--muted);
  font-size: 12px;
  font-weight: 800;
  text-align: center;
  text-transform: uppercase;
}
.toast {
  position: sticky;
  bottom: 12px;
  z-index: 5;
  display: none;
  margin-top: 12px;
  padding: 11px 12px;
  border-radius: 8px;
  background: #172126;
  color: #fff;
  box-shadow: var(--shadow);
}
.toast.show {
  display: block;
}
.tokenbar {
  display: none;
  grid-template-columns: 1fr auto;
  gap: 8px;
  margin-bottom: 12px;
}
.tokenbar.show {
  display: grid;
}
.settings-panel {
  display: none;
  margin-bottom: 12px;
}
.settings-panel.show {
  display: block;
}
.wide-only {
  display: none;
}
@media (min-width: 760px) {
  .app { padding: 20px; }
  .grid { grid-template-columns: 1.05fr .95fr; align-items: start; }
  .span-2 { grid-column: span 2; }
  .wide-only { display: inline; }
  .controls { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .row.three { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .activity-tabs { grid-template-columns: repeat(5, minmax(0, 1fr)); width: min(560px, 100%); }
}
</style>
</head>
<body>
<main class="app">
  <header class="topbar">
    <div class="brand">
      <div class="mark"><img src="/favicon.svg" alt=""></div>
      <div>
        <h1>Pooly Control</h1>
        <p class="muted" id="subline">Pool daemon</p>
      </div>
    </div>
    <div class="top-actions">
      <button id="editToken">Token</button>
      <button id="settingsToggle">Settings</button>
      <button id="refresh">Refresh</button>
    </div>
  </header>

  <section class="tokenbar" id="tokenbar">
    <input id="token" type="password" autocomplete="current-password" placeholder="Bearer token">
    <button class="primary" id="saveToken">Save</button>
  </section>

  <section class="panel settings-panel" id="settingsPanel">
    <div class="panel-head">
      <h2>Settings</h2>
      <button id="settingsClose">Close</button>
    </div>
    <div class="row two">
      <label>OpenWeatherMap API Key <input id="weatherApiKey" type="password" autocomplete="off" placeholder="Leave blank to keep saved key"></label>
      <label>Pool Location <input id="weatherLocation" type="text" autocomplete="address-level2" placeholder="Berlin,DE"></label>
    </div>
    <div style="display:grid; gap:8px; margin-top:10px">
      <button class="primary" id="saveWeatherSettings">Save Weather</button>
      <p class="muted" id="weatherSettingsDetail">Weather is not configured.</p>
    </div>
  </section>

  <div class="grid">
    <section class="panel">
      <div class="panel-head">
        <h2>Status</h2>
        <span class="badge" id="connected">Unknown</span>
      </div>
      <div class="status-hero">
        <div>
          <div class="temp" id="currentTemp">--°</div>
          <div class="target" id="targetTemp">Target --°</div>
        </div>
        <span class="badge" id="stateBadge">Idle</span>
      </div>
      <div class="metrics">
        <div class="metric"><span>Observed</span><strong id="observedAt">--</strong></div>
        <div class="metric"><span>Error</span><strong id="errorCode">None</strong></div>
      </div>
      <div class="weather-widget" id="weatherWidget">
        <strong id="weatherTemp">--°</strong>
        <div id="weatherCondition">Weather not configured</div>
        <span id="weatherObserved">Add OpenWeatherMap settings</span>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>Controls</h2>
        <span class="badge" id="busy">Ready</span>
      </div>
      <div class="manual-strip" id="manualStrip">
        <div>
          <strong id="manualTitle">Manual control</strong>
          <span id="manualDetail">No override active</span>
        </div>
        <div class="manual-actions">
          <button id="manualMinus">-30m</button>
          <button id="manualPlus">+30m</button>
          <button class="primary" id="manualPermanent">Make permanent</button>
          <button class="danger" id="manualClear">Clear</button>
        </div>
      </div>
      <div class="controls" id="controls"></div>
      <div class="temp-row">
        <button id="tempDown">-</button>
        <label>Target <input id="tempInput" type="number" min="10" max="40" step="1"></label>
        <button id="tempUp">+</button>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>Plans</h2>
        <button id="reloadPlans">Reload</button>
      </div>
      <div class="tabs">
        <button class="active" data-view="plans">List</button>
        <button data-view="ready">Ready</button>
        <button data-view="window">Window</button>
      </div>
      <div id="plansView" class="forms" style="margin-top:12px"></div>
    </section>

    <section class="panel span-2">
      <div class="panel-head activity-head">
        <h2>Activity</h2>
        <div class="tabs activity-tabs">
          <button class="active" data-activity="events">Events</button>
          <button data-activity="polls">Polls</button>
          <button data-activity="commands">Commands</button>
          <button data-activity="plan_executions">Plans</button>
          <button data-activity="heating_sessions">Heating</button>
        </div>
      </div>
      <div class="activity-list" id="activity"></div>
      <div class="pager" id="activityPager"></div>
    </section>
  </div>
  <div class="toast" id="toast"></div>
</main>

<script>
var caps = ["power", "filter", "heater", "jets", "bubbles", "sanitizer"];
var capLabels = {power:"Power", filter:"Filter", heater:"Heater", jets:"Jets", bubbles:"Bubbles", sanitizer:"Sanitizer"};
var days = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
var manualPlanID = "webui-manual";
var legacyPausePlanID = "webui-pause";
var manualDefaultMinutes = 30;
var activityPageSize = 12;
var activityKeys = ["events", "polls", "commands", "plan_executions", "heating_sessions"];
var tempDebounceTimer = null;
var state = {
  token: localStorage.getItem("poold.token") || "",
  status: null,
  weather: null,
  plans: [],
  events: [],
  polls: [],
  commands: [],
  planExecutions: [],
  heatingSessions: [],
  activityRaw: {},
  activityPages: {
    events: 0,
    polls: 0,
    commands: 0,
    plan_executions: 0,
    heating_sessions: 0
  },
  activityHasOlder: {},
  manualDraft: null,
  settingsOpen: false,
  planView: "plans",
  activityView: "events",
  pending: false
};

function $(id) { return document.getElementById(id); }
function qsa(selector) { return Array.prototype.slice.call(document.querySelectorAll(selector)); }
function boolText(value) { return value ? "On" : "Off"; }
function title(value) { return (value || "").replace(/_/g, " ").replace(/\b\w/g, function(c) { return c.toUpperCase(); }); }

function setBusy(value) {
  state.pending = value;
  $("busy").textContent = value ? "Working" : "Ready";
  $("busy").className = value ? "badge warn" : "badge ok";
  qsa("button").forEach(function(button) {
    if (button.id !== "saveToken") button.disabled = value;
  });
}

function toast(message, kind) {
  var box = $("toast");
  box.textContent = message;
  box.style.background = kind === "bad" ? "#b42318" : kind === "ok" ? "#1d7f45" : "#172126";
  box.classList.add("show");
  clearTimeout(toast.timer);
  toast.timer = setTimeout(function() { box.classList.remove("show"); }, 2800);
}

function updateTokenUI() {
  $("tokenbar").classList.toggle("show", !state.token);
  $("token").value = state.token;
}

function api(path, options) {
  options = options || {};
  var headers = options.headers || {};
  if (state.token) headers.Authorization = "Bearer " + state.token;
  if (options.body && !headers["Content-Type"]) headers["Content-Type"] = "application/json";
  return fetch(path, Object.assign({}, options, {headers: headers})).then(function(resp) {
    return resp.text().then(function(text) {
      var body = text ? JSON.parse(text) : null;
      if (!resp.ok) {
        var message = body && body.error ? body.error : resp.status + " " + resp.statusText;
        var error = new Error(message);
        error.status = resp.status;
        throw error;
      }
      return body;
    });
  });
}

function loadAll() {
  updateTokenUI();
  if (!state.token) {
    renderAll();
    return;
  }
  setBusy(true);
  Promise.all([
    loadStatus(),
    loadWeather(),
    loadPlans(),
    loadActivities()
  ]).finally(function() {
    setBusy(false);
    renderAll();
  });
}

function loadStatus() {
  return api("/status").then(function(status) {
    state.status = status;
  }).catch(function(err) {
    if (err.status === 401) {
      state.token = "";
      localStorage.removeItem("poold.token");
      updateTokenUI();
    }
    toast("Status: " + err.message, "bad");
  });
}

function loadPlans() {
  return api("/plans").then(function(data) {
    state.plans = data.plans || [];
    state.manualDraft = null;
  }).catch(function(err) {
    toast("Plans: " + err.message, "bad");
  });
}

function loadWeather() {
  return api("/weather").then(function(data) {
    state.weather = data || {};
  }).catch(function(err) {
    toast("Weather: " + err.message, "bad");
  });
}

function loadActivities() {
  return Promise.all(activityKeys.map(function(view) { return loadActivity(view); }));
}

function loadActivity(view) {
  var page = state.activityPages[view] || 0;
  var limit = activityPageSize + 1;
  var offset = page * activityPageSize;
  return api(activityPath(view, limit, offset)).then(function(data) {
    var rows = activityRowsFromResponse(view, data || []);
    state.activityRaw[view] = rows;
    state.activityHasOlder[view] = rows.length > activityPageSize;
    setActivityRows(view, rows.slice(0, activityPageSize));
  }).catch(function() {
    state.activityRaw[view] = [];
    state.activityHasOlder[view] = false;
    setActivityRows(view, []);
  });
}

function activityPath(view, limit, offset) {
  var suffix = "limit=" + limit + "&offset=" + offset;
  if (view === "events") return "/events?latest=1&changes=1&" + suffix;
  if (view === "polls") return "/observations?latest=1&" + suffix;
  if (view === "commands") return "/commands?latest=1&" + suffix;
  if (view === "plan_executions") return "/events?latest=1&type=scheduler&" + suffix;
  return "/heating-sessions?latest=1&" + suffix;
}

function activityRowsFromResponse(view, data) {
  if (view === "events" || view === "plan_executions") return data.events || [];
  if (view === "polls") return data.observations || [];
  if (view === "commands") return data.commands || [];
  return data.heating_sessions || [];
}

function setActivityRows(view, rows) {
  if (view === "events") state.events = rows;
  if (view === "polls") state.polls = rows;
  if (view === "commands") state.commands = rows;
  if (view === "plan_executions") state.planExecutions = rows;
  if (view === "heating_sessions") state.heatingSessions = rows;
}

function renderAll() {
  renderStatus();
  renderWeather();
  renderSettings();
  renderManual();
  renderControls();
  renderPlans();
  renderActivity();
}

function renderLivePanels() {
  renderStatus();
  renderWeather();
  renderSettings();
  renderManual();
  renderControls();
  renderActivity();
}

function renderStatus() {
  var status = state.status || {};
  var unit = status.unit || "°C";
  var current = status.current_temp == null ? "--" : status.current_temp + unit;
  $("currentTemp").textContent = current;
  $("targetTemp").textContent = "Target " + (status.preset_temp || "--") + unit;
  if (document.activeElement !== $("tempInput")) {
    $("tempInput").value = status.preset_temp || "";
  }
  $("observedAt").textContent = status.observed_at ? formatTime(status.observed_at) : "--";
  $("errorCode").textContent = status.error_code || "None";
  $("subline").textContent = status.connected ? "Connected " + formatAge(status.observed_at) : "Pool daemon";
  $("connected").textContent = status.connected ? "Connected" : state.token ? "Disconnected" : "Token";
  $("connected").className = status.connected ? "badge ok" : state.token ? "badge bad" : "badge warn";
  var manual = activeManualPlan();
  if (manual) {
    $("stateBadge").textContent = "Manual";
    $("stateBadge").className = "badge warn";
    return;
  }
  var active = activeCaps(status);
  $("stateBadge").textContent = active.length ? active.map(title).join(", ") : "Idle";
  $("stateBadge").className = active.length ? "badge ok" : "badge";
}

function renderWeather() {
  var weather = state.weather || {};
  var latest = weather.latest || {};
  var data = latest.data || {};
  var main = data.main || {};
  var condition = data.weather && data.weather.length ? data.weather[0] : {};
  var temp = typeof main.temp === "number" ? Math.round(main.temp) + "°C" : "--°";
  $("weatherTemp").textContent = temp;
  if (latest.id) {
    $("weatherCondition").textContent = title(condition.description || condition.main || "Weather");
    var cloudText = data.clouds && typeof data.clouds.all === "number" ? " · " + data.clouds.all + "% clouds" : "";
    $("weatherObserved").textContent = weatherLocationLabel(latest.location) + " · " + formatAge(latest.observed_at) + cloudText;
  } else if (weather.settings && weather.settings.api_key_set) {
    $("weatherCondition").textContent = "Waiting for weather";
    $("weatherObserved").textContent = weatherLocationLabel(weather.settings.location);
  } else {
    $("weatherCondition").textContent = "Weather not configured";
    $("weatherObserved").textContent = "Add OpenWeatherMap settings";
  }
}

function renderSettings() {
  $("settingsPanel").classList.toggle("show", state.settingsOpen);
  var settings = state.weather && state.weather.settings ? state.weather.settings : {};
  var location = settings.location || {};
  if (document.activeElement !== $("weatherLocation")) {
    $("weatherLocation").value = location.query || "";
  }
  $("weatherApiKey").placeholder = settings.api_key_set ? "Saved; leave blank to keep" : "OpenWeatherMap API key";
  var detail = [];
  detail.push(settings.api_key_set ? "API key saved" : "API key missing");
  if (location.name) detail.push(weatherLocationLabel(location));
  $("weatherSettingsDetail").textContent = detail.join(" · ");
}

function renderManual() {
  var plan = activeManualPlan();
  var strip = $("manualStrip");
  strip.classList.toggle("active", !!plan);
  if (plan) {
    $("manualTitle").textContent = manualTitle(plan.desired_state || {});
    $("manualDetail").textContent = manualSummary(plan.desired_state || {}) + " · " + remainingTime(plan.expires_at) + " left";
  } else {
    $("manualTitle").textContent = "Manual control";
    $("manualDetail").textContent = "No override active";
  }
}

function renderControls() {
  var wrap = $("controls");
  var status = state.status || {};
  wrap.innerHTML = "";
  caps.forEach(function(cap) {
    var manualValue = manualDesiredValue(cap);
    var displayValue = manualValue === undefined ? !!status[cap] : manualValue;
    var button = document.createElement("button");
    button.className = "control" + (displayValue ? " on" : "") + (manualValue !== undefined ? " manual" : "");
    button.innerHTML = "<span><strong>" + capLabels[cap] + "</strong></span><i class=\"dot\"></i>";
    button.onclick = function() { setManualBool(cap, !displayValue); };
    wrap.appendChild(button);
  });
}

function renderPlans() {
  qsa("[data-view]").forEach(function(button) {
    button.classList.toggle("active", button.dataset.view === state.planView);
  });
  var view = $("plansView");
  view.innerHTML = "";
  if (state.planView === "plans") return renderPlanList(view);
  if (state.planView === "ready") return renderReadyForm(view);
  renderWindowForm(view);
}

function renderPlanList(view) {
  var visiblePlans = state.plans.filter(function(plan) { return !isReservedManualPlan(plan); });
  if (!visiblePlans.length) {
    view.innerHTML = "<p class=\"muted\">No plans</p>";
    return;
  }
  var list = document.createElement("div");
  list.className = "plan-list";
  visiblePlans.forEach(function(plan) {
    var item = document.createElement("div");
    item.className = "plan";
    item.innerHTML = "<div class=\"plan-main\"><div><h3>" + escapeHTML(plan.name || plan.id) + "</h3><p class=\"muted\">" + describePlan(plan) + "</p></div><span class=\"badge " + (plan.enabled ? "ok" : "") + "\">" + (plan.enabled ? "On" : "Off") + "</span></div><div class=\"plan-actions\"><button data-toggle=\"" + plan.id + "\">" + (plan.enabled ? "Disable" : "Enable") + "</button><button class=\"danger\" data-delete=\"" + plan.id + "\">Delete</button></div>";
    list.appendChild(item);
  });
  view.appendChild(list);
  qsa("[data-toggle]").forEach(function(button) {
    button.onclick = function() {
      updatePlans(state.plans.map(function(plan) {
        if (plan.id === button.dataset.toggle) plan.enabled = !plan.enabled;
        return plan;
      }));
    };
  });
  qsa("[data-delete]").forEach(function(button) {
    button.onclick = function() {
      updatePlans(state.plans.filter(function(plan) { return plan.id !== button.dataset.delete; }));
    };
  });
}

function renderReadyForm(view) {
  view.innerHTML = "<div class=\"row two\"><label>Name <input id=\"readyName\" value=\"Ready by\"></label><label>Target <input id=\"readyTemp\" type=\"number\" min=\"10\" max=\"40\" step=\"1\" value=\"36\"></label></div><label>At <input id=\"readyAt\" type=\"datetime-local\"></label><button class=\"primary\" id=\"addReady\">Add Ready Plan</button>";
  $("readyAt").value = localDateTime(new Date(Date.now() + 24 * 60 * 60 * 1000));
  $("addReady").onclick = function() {
    var at = $("readyAt").value;
    if (!at) return toast("Ready time is required", "bad");
    var plan = {
      id: "ready-by-" + Date.now(),
      type: "ready_by",
      name: $("readyName").value || "Ready by",
      enabled: true,
      target_temp: Number($("readyTemp").value || 36),
      at: new Date(at).toISOString()
    };
    updatePlans(state.plans.concat([plan]));
  };
}

function renderWindowForm(view) {
  view.innerHTML = "<div class=\"row three\"><label>Name <input id=\"windowName\" value=\"Filter window\"></label><label>Capability <select id=\"windowCap\"><option value=\"filter\">Filter</option><option value=\"heater\">Heater</option><option value=\"jets\">Jets</option><option value=\"bubbles\">Bubbles</option><option value=\"sanitizer\">Sanitizer</option></select></label><label>Enabled <select id=\"windowEnabled\"><option value=\"true\">On</option><option value=\"false\">Off</option></select></label></div><div class=\"row two\"><label>From <input id=\"windowFrom\" type=\"time\" value=\"02:00\"></label><label>To <input id=\"windowTo\" type=\"time\" value=\"04:00\"></label></div><div class=\"days\" id=\"windowDays\"></div><button class=\"primary\" id=\"addWindow\">Add Window</button>";
  var dayWrap = $("windowDays");
  days.forEach(function(day) {
    var button = document.createElement("button");
    button.className = "day";
    button.textContent = day.slice(0, 1).toUpperCase();
    button.dataset.day = day;
    button.onclick = function() { button.classList.toggle("active"); };
    dayWrap.appendChild(button);
  });
  $("addWindow").onclick = function() {
    var plan = {
      id: "window-" + Date.now(),
      type: "time_window",
      name: $("windowName").value || title($("windowCap").value) + " window",
      enabled: $("windowEnabled").value === "true",
      capability: $("windowCap").value,
      from: $("windowFrom").value,
      to: $("windowTo").value,
      days: qsa("#windowDays .day.active").map(function(button) { return button.dataset.day; })
    };
    updatePlans(state.plans.concat([plan]));
  };
}

function renderActivity() {
  qsa("[data-activity]").forEach(function(button) {
    button.classList.toggle("active", button.dataset.activity === state.activityView);
  });
  var list = $("activity");
  var rows = currentActivityRows();
  var rawRows = state.activityRaw[state.activityView] || rows;
  if (!rows.length) {
    list.innerHTML = "<p class=\"muted\">No activity</p>";
    renderActivityPager();
    return;
  }
  list.innerHTML = "";
  rows.forEach(function(row, index) {
    var item = document.createElement("div");
    item.className = "activity";
    if (state.activityView === "polls") {
      item.innerHTML = "<time>" + formatActivityTime(row.last_observed_at || row.status.observed_at) + "</time><div><strong>Span #" + row.id + "</strong><span>" + observationLine(row, rawRows[index + 1]) + "</span></div>";
    } else if (state.activityView === "commands") {
      item.innerHTML = "<time>" + formatActivityTime(row.completed_at || row.issued_at) + "</time><div><strong>Command #" + row.id + "</strong><span>" + commandLine(row) + "</span></div>";
    } else if (state.activityView === "plan_executions") {
      item.innerHTML = "<time>" + formatActivityTime(row.created_at) + "</time><div><strong>Plan #" + row.id + "</strong><span>" + planExecutionLine(row) + "</span></div>";
    } else if (state.activityView === "heating_sessions") {
      item.innerHTML = "<time>" + formatActivityTime(row.started_at) + "</time><div><strong>Heating #" + row.first_observation_id + "-" + row.last_observation_id + "</strong><span>" + heatingSessionLine(row) + "</span></div>";
    } else {
      item.innerHTML = "<time>" + formatActivityTime(row.created_at) + "</time><div><strong>" + title(row.type) + " #" + row.id + "</strong><span>" + eventLine(row, previousObservationEvent(rawRows, index)) + "</span></div>";
    }
    list.appendChild(item);
  });
  renderActivityPager();
}

function currentActivityRows() {
  if (state.activityView === "polls") return state.polls;
  if (state.activityView === "commands") return state.commands;
  if (state.activityView === "plan_executions") return state.planExecutions;
  if (state.activityView === "heating_sessions") return state.heatingSessions;
  return state.events;
}

function renderActivityPager() {
  var pager = $("activityPager");
  var page = state.activityPages[state.activityView] || 0;
  var hasOlder = !!state.activityHasOlder[state.activityView];
  pager.innerHTML = "<button data-page=\"newer\">Newer</button><span>Page " + (page + 1) + "</span><button data-page=\"older\">Older</button>";
  var newer = pager.querySelector("[data-page=\"newer\"]");
  var older = pager.querySelector("[data-page=\"older\"]");
  newer.disabled = page <= 0 || state.pending;
  older.disabled = !hasOlder || state.pending;
  newer.onclick = function() { changeActivityPage(-1); };
  older.onclick = function() { changeActivityPage(1); };
}

function changeActivityPage(delta) {
  var next = Math.max(0, (state.activityPages[state.activityView] || 0) + delta);
  if (next === state.activityPages[state.activityView]) return;
  state.activityPages[state.activityView] = next;
  setBusy(true);
  loadActivity(state.activityView).finally(function() {
    setBusy(false);
    renderActivity();
  });
}

function updatePlans(plans, message) {
  runAction(function() {
    return api("/plans", {method: "PUT", body: JSON.stringify({plans: plans})}).then(function(data) {
      state.plans = data.plans || [];
    });
  }, message || "Plans saved");
}

function saveWeatherSettings() {
  var payload = {location: $("weatherLocation").value.trim()};
  var apiKey = $("weatherApiKey").value.trim();
  if (apiKey) payload.api_key = apiKey;
  runAction(function() {
    return api("/weather/settings", {method: "PUT", body: JSON.stringify(payload)}).then(function(data) {
      state.weather = data || {};
      $("weatherApiKey").value = "";
    });
  }, "Weather settings saved");
}

function setManualBool(cap, value) {
  var desired = currentManualDesired();
  if (cap === "power" && value === false) {
    desired = {power: false};
  } else {
    if (cap !== "power" && value === true) desired.power = true;
    desired[cap] = value;
    if (cap === "filter" && value === false && desired.heater === true) desired.heater = false;
  }
  updateManualPlan(desired, new Date(Date.now() + manualDefaultMinutes * 60 * 1000), capLabels[cap] + " " + boolText(value));
}

function scheduleManualTemp(value) {
  clearTimeout(tempDebounceTimer);
  tempDebounceTimer = setTimeout(function() {
    setManualTargetTemp(value);
  }, 650);
}

function setManualTargetTemp(value) {
  if (String(value).trim() === "") {
    var cleared = currentManualDesired();
    delete cleared.target_temp;
    if (!Object.keys(cleared).length) return clearManual();
    return updateManualPlan(cleared, new Date(Date.now() + manualDefaultMinutes * 60 * 1000), "Target cleared");
  }
  var target = Number(value);
  if (!Number.isFinite(target) || target < 10 || target > 40) return;
  var desired = currentManualDesired();
  desired.target_temp = target;
  updateManualPlan(desired, new Date(Date.now() + manualDefaultMinutes * 60 * 1000), "Target " + target + "°");
}

function adjustManual(minutes) {
  var plan = activeManualPlan();
  if (!plan) return;
  var expiresAt = new Date(plan.expires_at).getTime() + minutes * 60 * 1000;
  if (expiresAt <= Date.now()) return clearManual();
  updateManualPlan(Object.assign({}, plan.desired_state || {}), new Date(expiresAt), "Manual time updated");
}

function clearManual() {
  state.manualDraft = null;
  updatePlans(state.plans.filter(function(plan) { return !isReservedManualPlan(plan); }), "Manual cleared");
}

function makeManualPermanent() {
  var plan = activeManualPlan();
  if (!plan) return;
  var manualDesired = Object.assign({}, plan.desired_state || {});
  if (!Object.keys(manualDesired).length) return;
  runAction(function() {
    return api("/desired-state").then(function(base) {
      var desired = permanentDesired(base || {}, manualDesired);
      return api("/desired-state", {method: "PUT", body: JSON.stringify(desired)}).then(function() {
        return api("/plans", {method: "PUT", body: JSON.stringify({
          plans: state.plans.filter(function(existing) { return !isReservedManualPlan(existing); })
        })}).then(function(data) {
          state.plans = data.plans || [];
          state.manualDraft = null;
        });
      });
    });
  }, "Manual control made permanent");
}

function updateManualPlan(desired, expiresAt, message) {
  state.manualDraft = Object.assign({}, desired);
  var plan = {
    id: manualPlanID,
    type: "manual_override",
    name: manualTitle(desired),
    enabled: true,
    desired_state: desired,
    expires_at: expiresAt.toISOString()
  };
  updatePlans(state.plans.filter(function(existing) {
    return !isReservedManualPlan(existing);
  }).concat([plan]), message || "Manual control saved");
}

function permanentDesired(base, manual) {
  var desired = Object.assign({}, base || {}, manual || {});
  if (manual.power === false) {
    desired.filter = false;
    desired.heater = false;
    desired.jets = false;
    desired.bubbles = false;
    desired.sanitizer = false;
  }
  if (manual.filter === false) {
    desired.heater = false;
  }
  if (desired.heater === true) {
    desired.power = true;
    desired.filter = true;
  }
  caps.forEach(function(cap) {
    if (cap !== "power" && desired[cap] === true) desired.power = true;
  });
  if (desired.power === false) {
    desired.filter = false;
    desired.heater = false;
    desired.jets = false;
    desired.bubbles = false;
    desired.sanitizer = false;
  }
  return desired;
}

function runAction(action, message) {
  if (!state.token) return toast("Token required", "bad");
  setBusy(true);
  action().then(function() {
    toast(message, "ok");
    return Promise.all([loadStatus(), loadWeather(), loadPlans(), loadActivities()]);
  }).catch(function(err) {
    toast(err.message, "bad");
  }).finally(function() {
    setBusy(false);
    renderAll();
  });
}

function activeCaps(status) {
  return caps.filter(function(cap) { return !!status[cap]; });
}

function tempLine(status) {
  var unit = status.unit || "°C";
  var current = status.current_temp == null ? "--" : status.current_temp + unit;
  return current + " → " + (status.preset_temp || "--") + unit;
}

function observationLine(observation, previous) {
  var parts = [
    formatSpanDuration(observation.first_observed_at, observation.last_observed_at) + " span",
    (observation.observation_count || 1) + " polls",
    statusChangeLine(observation.status || {}, previous && previous.status)
  ];
  return parts.join(" · ");
}

function eventLine(event, previousObservation) {
  if (event.type === "observation" && event.data) return statusChangeLine(event.data, previousObservation && previousObservation.data);
  if (event.type === "status_error" && event.data && event.data.error) return event.data.error;
  if (event.type === "command" && event.data) return commandLine(event.data);
  if (event.type === "command_error" && event.data) return title(event.data.capability) + " failed · " + (event.data.error || event.message || "");
  if (event.type === "scheduler" && event.data) return planExecutionLine(event);
  return event.message || "";
}

function previousObservationEvent(rows, index) {
  for (var i = index + 1; i < rows.length; i++) {
    if (rows[i].type === "observation" && rows[i].data) return rows[i];
  }
  return null;
}

function commandLine(command) {
  var value = commandValueText(command);
  var result = command.success ? "ok" : "failed";
  var parts = [title(command.capability) + (value ? " " + value : ""), result];
  if (command.source) parts.push(command.source);
  if (command.error) parts.push(command.error);
  return parts.join(" · ");
}

function commandValueText(command) {
  if (Object.prototype.hasOwnProperty.call(command, "state") && command.state !== null) return boolText(!!command.state);
  if (command.value !== undefined && command.value !== null) return String(command.value).replace(/^"|"$/g, "");
  return "";
}

function planExecutionLine(event) {
  var data = event.data || {};
  var parts = [];
  parts.push(data.source ? data.source : event.message || "Scheduler");
  if (data.reason) parts.push(data.reason);
  if (data.desired) parts.push(desiredSummary(data.desired));
  return parts.join(" · ");
}

function desiredSummary(desired) {
  var parts = [];
  caps.forEach(function(cap) {
    if (Object.prototype.hasOwnProperty.call(desired, cap)) {
      parts.push(capLabels[cap] + " " + boolText(!!desired[cap]));
    }
  });
  if (desired.target_temp != null) parts.push("Target " + desired.target_temp + "°");
  return parts.length ? parts.join(", ") : "No desired changes";
}

function heatingSessionLine(session) {
  var unit = "°C";
  var temp = formatTempValue(session.start_temp, unit) + " to " + formatTempValue(session.end_temp, unit);
  var parts = [
    formatDurationSeconds(session.duration_seconds) + (session.active ? " active" : ""),
    temp,
    "Target " + (session.target_temp || "--") + unit,
    (session.span_count || 0) + " spans",
    (session.observation_count || 0) + " polls"
  ];
  return parts.join(" · ");
}

function statusChangeLine(status, previous) {
  status = status || {};
  var fields = ["connected", "power", "filter", "heater", "jets", "bubbles", "sanitizer", "current_temp", "preset_temp", "error_code"];
  if (!previous) return "Initial state · " + tempLine(status);
  var unit = status.unit || previous.unit || "°C";
  var changes = [];
  fields.forEach(function(field) {
    var before = statusFieldValue(previous, field);
    var after = statusFieldValue(status, field);
    if (before !== after) {
      changes.push(statusFieldLabel(field) + " " + formatStatusField(field, after, unit) + " from " + formatStatusField(field, before, unit));
    }
  });
  return changes.length ? changes.join(" · ") : "No changed fields";
}

function statusFieldValue(status, field) {
  if (!status) return null;
  if (field === "current_temp") return status.current_temp == null ? null : Number(status.current_temp);
  if (field === "preset_temp") return status.preset_temp == null ? null : Number(status.preset_temp);
  if (field === "error_code") return status.error_code || "";
  return !!status[field];
}

function statusFieldLabel(field) {
  if (field === "current_temp") return "Temp";
  if (field === "preset_temp") return "Target";
  if (field === "error_code") return "Error";
  return capLabels[field] || title(field);
}

function formatStatusField(field, value, unit) {
  if (field === "current_temp" || field === "preset_temp") return formatTempValue(value, unit);
  if (field === "connected") return value ? "Connected" : "Disconnected";
  if (field === "error_code") return value || "None";
  return boolText(!!value);
}

function formatTempValue(value, unit) {
  return value == null ? "--" : value + (unit || "°C");
}

function describePlan(plan) {
  if (isReservedManualPlan(plan)) return manualSummary(plan.desired_state || {}) + " until " + formatDateTime(plan.expires_at);
  if (plan.type === "ready_by") return (plan.target_temp || "--") + "° by " + formatDateTime(plan.at);
  if (plan.type === "time_window") return title(plan.capability) + " " + plan.from + "-" + plan.to + (plan.days && plan.days.length ? " · " + plan.days.join(", ") : "");
  if (plan.type === "manual_override") return "Until " + formatDateTime(plan.expires_at);
  return title(plan.type);
}

function weatherLocationLabel(location) {
  location = location || {};
  var label = location.name || location.query || "Pool location";
  if (location.country) label += ", " + location.country;
  return label;
}

function formatTime(value) {
  if (!value) return "--";
  return new Date(value).toLocaleTimeString([], {hour: "2-digit", minute: "2-digit", second: "2-digit"});
}

function formatActivityTime(value) {
  if (!value) return "--";
  var date = new Date(value);
  var now = new Date();
  var dateOptions = date.getFullYear() === now.getFullYear() ? {month: "short", day: "numeric"} : {year: "numeric", month: "short", day: "numeric"};
  return date.toLocaleDateString([], dateOptions) + "<br>" + date.toLocaleTimeString([], {hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false});
}

function formatDateTime(value) {
  if (!value) return "--";
  return new Date(value).toLocaleString([], {month: "short", day: "numeric", hour: "2-digit", minute: "2-digit"});
}

function formatAge(value) {
  if (!value) return "";
  var seconds = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 1000));
  if (seconds < 60) return seconds + "s ago";
  var minutes = Math.round(seconds / 60);
  if (minutes < 60) return minutes + "m ago";
  return Math.round(minutes / 60) + "h ago";
}

function formatSpanDuration(start, end) {
  if (!start || !end) return "0s";
  var seconds = Math.max(0, Math.round((new Date(end).getTime() - new Date(start).getTime()) / 1000));
  return formatDurationSeconds(seconds);
}

function formatDurationSeconds(seconds) {
  seconds = Math.max(0, Math.round(Number(seconds) || 0));
  if (seconds < 60) return seconds + "s";
  var minutes = Math.round(seconds / 60);
  if (minutes < 60) return minutes + "m";
  var hours = Math.floor(minutes / 60);
  var rest = minutes % 60;
  if (hours < 48) return rest ? hours + "h " + rest + "m" : hours + "h";
  var days = Math.floor(hours / 24);
  var dayHours = hours % 24;
  return dayHours ? days + "d " + dayHours + "h" : days + "d";
}

function localDateTime(date) {
  var pad = function(n) { return String(n).padStart(2, "0"); };
  return date.getFullYear() + "-" + pad(date.getMonth() + 1) + "-" + pad(date.getDate()) + "T" + pad(date.getHours()) + ":" + pad(date.getMinutes());
}

function activeManualPlan() {
  var now = Date.now();
  return state.plans.find(function(plan) {
    return isReservedManualPlan(plan) &&
      plan.enabled &&
      plan.expires_at &&
      new Date(plan.expires_at).getTime() > now;
  });
}

function isReservedManualPlan(plan) {
  return plan && (plan.id === manualPlanID || plan.id === legacyPausePlanID);
}

function currentManualDesired() {
  if (state.manualDraft) return Object.assign({}, state.manualDraft);
  var plan = activeManualPlan();
  return Object.assign({}, plan && plan.desired_state ? plan.desired_state : {});
}

function manualDesiredValue(cap) {
  var plan = activeManualPlan();
  var desired = state.manualDraft || (plan && plan.desired_state ? plan.desired_state : {});
  return Object.prototype.hasOwnProperty.call(desired, cap) ? desired[cap] : undefined;
}

function manualTitle(desired) {
  if (desired && desired.power === false) return "Pool stopped";
  if (desired && desired.heater === true) return "Manual heating";
  return "Manual control";
}

function manualSummary(desired) {
  var parts = [];
  caps.forEach(function(cap) {
    if (Object.prototype.hasOwnProperty.call(desired, cap)) {
      parts.push(capLabels[cap] + " " + boolText(!!desired[cap]));
    }
  });
  if (desired.target_temp != null) parts.push("Target " + desired.target_temp + "°");
  return parts.length ? parts.join(" · ") : "No settings";
}

function remainingTime(value) {
  var seconds = Math.max(0, Math.round((new Date(value).getTime() - Date.now()) / 1000));
  if (seconds < 60) return seconds + "s";
  var minutes = Math.round(seconds / 60);
  if (minutes < 90) return minutes + "m";
  var hours = Math.floor(minutes / 60);
  var rest = minutes % 60;
  return rest ? hours + "h " + rest + "m" : hours + "h";
}

function escapeHTML(value) {
  return String(value || "").replace(/[&<>"']/g, function(ch) {
    return {"&":"&amp;", "<":"&lt;", ">":"&gt;", "\"":"&quot;", "'":"&#39;"}[ch];
  });
}

$("saveToken").onclick = function() {
  state.token = $("token").value.trim();
  localStorage.setItem("poold.token", state.token);
  loadAll();
};
$("editToken").onclick = function() {
  state.token = "";
  localStorage.removeItem("poold.token");
  updateTokenUI();
};
$("settingsToggle").onclick = function() {
  state.settingsOpen = !state.settingsOpen;
  renderSettings();
};
$("settingsClose").onclick = function() {
  state.settingsOpen = false;
  renderSettings();
};
$("refresh").onclick = loadAll;
$("reloadPlans").onclick = function() {
  loadPlans().then(renderPlans);
};
$("saveWeatherSettings").onclick = saveWeatherSettings;
$("manualMinus").onclick = function() { adjustManual(-30); };
$("manualPlus").onclick = function() { adjustManual(30); };
$("manualPermanent").onclick = makeManualPermanent;
$("manualClear").onclick = clearManual;
$("tempInput").oninput = function() { scheduleManualTemp($("tempInput").value); };
$("tempDown").onclick = function() {
  $("tempInput").value = Number($("tempInput").value || 0) - 1;
  scheduleManualTemp($("tempInput").value);
};
$("tempUp").onclick = function() {
  $("tempInput").value = Number($("tempInput").value || 0) + 1;
  scheduleManualTemp($("tempInput").value);
};
qsa("[data-view]").forEach(function(button) {
  button.onclick = function() {
    state.planView = button.dataset.view;
    renderPlans();
  };
});
qsa("[data-activity]").forEach(function(button) {
  button.onclick = function() {
    state.activityView = button.dataset.activity;
    renderActivity();
  };
});

updateTokenUI();
renderAll();
loadAll();
setInterval(function() {
  if (state.token && !state.pending) Promise.all([loadStatus(), loadWeather(), loadActivities()]).then(renderLivePanels);
}, 30000);
setInterval(renderManual, 1000);
</script>
</body>
</html>
`

const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <defs>
    <linearGradient id="bg" x1="10" y1="4" x2="56" y2="60" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#00a6b2"/>
      <stop offset=".55" stop-color="#007c89"/>
      <stop offset="1" stop-color="#235ea8"/>
    </linearGradient>
    <linearGradient id="water" x1="11" y1="38" x2="53" y2="53" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#e9fbff"/>
      <stop offset="1" stop-color="#b8e9f3"/>
    </linearGradient>
  </defs>
  <rect width="64" height="64" rx="14" fill="url(#bg)"/>
  <path d="M15 39c5.7-4.3 11.3-4.3 17 0s11.3 4.3 17 0" fill="none" stroke="url(#water)" stroke-width="5.5" stroke-linecap="round"/>
  <path d="M15 49c5.7-4.3 11.3-4.3 17 0s11.3 4.3 17 0" fill="none" stroke="#dff8fc" stroke-width="4.5" stroke-linecap="round" opacity=".9"/>
  <path d="M24 30c-3.2-3.2-3.2-7 0-10 2.7-2.6 2.9-5.6.5-8" fill="none" stroke="#fff6d8" stroke-width="4.5" stroke-linecap="round"/>
  <path d="M38 30c-3.2-3.2-3.2-7 0-10 2.7-2.6 2.9-5.6.5-8" fill="none" stroke="#fff6d8" stroke-width="4.5" stroke-linecap="round" opacity=".92"/>
  <circle cx="51" cy="15" r="4.5" fill="#fff6d8" opacity=".95"/>
</svg>
`

func renderAppIconPNG(size int) []byte {
	const scale = 4
	large := image.NewRGBA(image.Rect(0, 0, size*scale, size*scale))
	fillIconBackground(large)
	drawWave(large, 39, 4.3, 5.5, color.RGBA{R: 233, G: 251, B: 255, A: 255})
	drawWave(large, 49, 4.1, 4.5, color.RGBA{R: 223, G: 248, B: 252, A: 235})
	drawSteam(large, 24, color.RGBA{R: 255, G: 246, B: 216, A: 255})
	drawSteam(large, 38, color.RGBA{R: 255, G: 246, B: 216, A: 235})
	drawCircle(large, iconCoord(large, 51), iconCoord(large, 15), iconCoord(large, 4.5), color.RGBA{R: 255, G: 246, B: 216, A: 242})

	small := downsampleRGBA(large, scale)
	var buf bytes.Buffer
	if err := png.Encode(&buf, small); err != nil {
		return nil
	}
	return buf.Bytes()
}

func fillIconBackground(img *image.RGBA) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	start := color.RGBA{R: 0, G: 166, B: 178, A: 255}
	mid := color.RGBA{R: 0, G: 124, B: 137, A: 255}
	end := color.RGBA{R: 35, G: 94, B: 168, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			t := float64(x+y) / float64(width+height-2)
			c := mixColor(start, mid, t*2)
			if t > .5 {
				c = mixColor(mid, end, (t-.5)*2)
			}
			img.SetRGBA(x, y, c)
		}
	}
}

func drawWave(img *image.RGBA, baseY, amplitude, stroke float64, c color.RGBA) {
	const samples = 220
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(samples-1)
		x := 15 + t*34
		y := baseY - math.Sin(t*4*math.Pi)*amplitude
		drawCircle(img, iconCoord(img, x), iconCoord(img, y), iconCoord(img, stroke/2), c)
	}
}

func drawSteam(img *image.RGBA, baseX float64, c color.RGBA) {
	const samples = 80
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(samples-1)
		x := baseX + math.Sin(t*2*math.Pi)*1.6
		y := 30 - t*18
		drawCircle(img, iconCoord(img, x), iconCoord(img, y), iconCoord(img, 2.25), c)
	}
}

func drawCircle(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	if radius <= 0 {
		return
	}
	bounds := img.Bounds()
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		for x := cx - radius; x <= cx+radius; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				blendRGBA(img, x, y, c)
			}
		}
	}
}

func blendRGBA(img *image.RGBA, x, y int, c color.RGBA) {
	if c.A == 255 {
		img.SetRGBA(x, y, c)
		return
	}
	dst := img.RGBAAt(x, y)
	a := float64(c.A) / 255
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(c.R)*a + float64(dst.R)*(1-a)),
		G: uint8(float64(c.G)*a + float64(dst.G)*(1-a)),
		B: uint8(float64(c.B)*a + float64(dst.B)*(1-a)),
		A: 255,
	})
}

func downsampleRGBA(src *image.RGBA, scale int) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx()/scale, bounds.Dy()/scale))
	for y := 0; y < dst.Bounds().Dy(); y++ {
		for x := 0; x < dst.Bounds().Dx(); x++ {
			var r, g, b, a int
			for yy := 0; yy < scale; yy++ {
				for xx := 0; xx < scale; xx++ {
					c := src.RGBAAt(x*scale+xx, y*scale+yy)
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					a += int(c.A)
				}
			}
			count := scale * scale
			dst.SetRGBA(x, y, color.RGBA{R: uint8(r / count), G: uint8(g / count), B: uint8(b / count), A: uint8(a / count)})
		}
	}
	return dst
}

func iconCoord(img *image.RGBA, value float64) int {
	return int(math.Round(value / 64 * float64(img.Bounds().Dx())))
}

func mixColor(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: 255,
	}
}
