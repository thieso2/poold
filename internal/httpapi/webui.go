package httpapi

import "net/http"

func (a *API) handleWebUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(webUIHTML))
}

const webUIHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover">
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
  background: var(--accent);
  color: #fff;
  display: grid;
  place-items: center;
  font-weight: 800;
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
.controls {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
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
  grid-template-columns: 72px 1fr;
  align-items: start;
}
.activity time {
  color: var(--muted);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
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
.tabs {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
}
.tabs button.active {
  background: #172126;
  border-color: #172126;
  color: #fff;
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
}
</style>
</head>
<body>
<main class="app">
  <header class="topbar">
    <div class="brand">
      <div class="mark">P</div>
      <div>
        <h1>Pooly Control</h1>
        <p class="muted" id="subline">Pool daemon</p>
      </div>
    </div>
    <div class="top-actions">
      <button id="editToken">Token</button>
      <button id="refresh">Refresh</button>
    </div>
  </header>

  <section class="tokenbar" id="tokenbar">
    <input id="token" type="password" autocomplete="current-password" placeholder="Bearer token">
    <button class="primary" id="saveToken">Save</button>
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
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>Controls</h2>
        <span class="badge" id="busy">Ready</span>
      </div>
      <div class="controls" id="controls"></div>
      <div class="temp-row">
        <button id="tempDown">-</button>
        <label>Target <input id="tempInput" type="number" min="10" max="40" step="1"></label>
        <button id="tempUp">+</button>
      </div>
      <button class="primary" id="sendTemp">Set Temperature</button>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>Desired State</h2>
        <button class="primary" id="saveDesired">Save</button>
      </div>
      <div class="forms" id="desiredForm"></div>
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
      <div class="panel-head">
        <h2>Activity</h2>
        <div class="tabs" style="width:min(340px,100%)">
          <button class="active" data-activity="events">Events</button>
          <button data-activity="polls">Polls</button>
          <button data-activity="commands">Commands</button>
        </div>
      </div>
      <div class="activity-list" id="activity"></div>
    </section>
  </div>
  <div class="toast" id="toast"></div>
</main>

<script>
var caps = ["power", "filter", "heater", "jets", "bubbles", "sanitizer"];
var capLabels = {power:"Power", filter:"Filter", heater:"Heater", jets:"Jets", bubbles:"Bubbles", sanitizer:"Sanitizer"};
var days = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
var state = {
  token: localStorage.getItem("poold.token") || "",
  status: null,
  desired: {},
  plans: [],
  events: [],
  polls: [],
  commands: [],
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
    loadDesired(),
    loadPlans(),
    loadEvents(),
    loadPolls()
  ]).finally(function() {
    setBusy(false);
    renderAll();
  });
}

function loadStatus() {
  return api("/status").then(function(status) {
    state.status = status;
  }).catch(function(err) {
    toast("Status: " + err.message, "bad");
  });
}

function loadDesired() {
  return api("/desired-state").then(function(desired) {
    state.desired = desired || {};
  }).catch(function(err) {
    if (err.status === 401) {
      state.token = "";
      localStorage.removeItem("poold.token");
      updateTokenUI();
    }
    toast("Desired state: " + err.message, "bad");
  });
}

function loadPlans() {
  return api("/plans").then(function(data) {
    state.plans = data.plans || [];
  }).catch(function(err) {
    toast("Plans: " + err.message, "bad");
  });
}

function loadEvents() {
  return api("/events?latest=1&limit=12").then(function(data) {
    state.events = data.events || [];
    state.commands = state.events.filter(function(event) {
      return event.type === "command" || event.type === "command_error" || event.type === "scheduler";
    });
  }).catch(function() {});
}

function loadPolls() {
  return api("/observations?latest=1&limit=12").then(function(data) {
    state.polls = data.observations || [];
  }).catch(function() {});
}

function renderAll() {
  renderStatus();
  renderControls();
  renderDesired();
  renderPlans();
  renderActivity();
}

function renderLivePanels() {
  renderStatus();
  renderControls();
  renderActivity();
}

function renderStatus() {
  var status = state.status || {};
  var unit = status.unit || "°C";
  var current = status.current_temp == null ? "--" : status.current_temp + unit;
  $("currentTemp").textContent = current;
  $("targetTemp").textContent = "Target " + (status.preset_temp || "--") + unit;
  $("tempInput").value = status.preset_temp || "";
  $("observedAt").textContent = status.observed_at ? formatTime(status.observed_at) : "--";
  $("errorCode").textContent = status.error_code || "None";
  $("subline").textContent = status.connected ? "Connected " + formatAge(status.observed_at) : "Pool daemon";
  $("connected").textContent = status.connected ? "Connected" : state.token ? "Disconnected" : "Token";
  $("connected").className = status.connected ? "badge ok" : state.token ? "badge bad" : "badge warn";
  var active = activeCaps(status);
  $("stateBadge").textContent = active.length ? active.map(title).join(", ") : "Idle";
  $("stateBadge").className = active.length ? "badge ok" : "badge";
}

function renderControls() {
  var wrap = $("controls");
  var status = state.status || {};
  wrap.innerHTML = "";
  caps.forEach(function(cap) {
    var button = document.createElement("button");
    button.className = "control" + (status[cap] ? " on" : "");
    button.innerHTML = "<span><strong>" + capLabels[cap] + "</strong><br><span class=\"muted\">" + boolText(!!status[cap]) + "</span></span><i class=\"dot\"></i>";
    button.onclick = function() { commandBool(cap, !status[cap]); };
    wrap.appendChild(button);
  });
}

function renderDesired() {
  var form = $("desiredForm");
  form.innerHTML = "";
  caps.forEach(function(cap) {
    var field = document.createElement("div");
    field.className = "row";
    field.innerHTML = "<label>" + capLabels[cap] + "<div class=\"segments\" data-desired=\"" + cap + "\"><button data-value=\"unset\">Any</button><button data-value=\"false\">Off</button><button data-value=\"true\">On</button></div></label>";
    form.appendChild(field);
    var current = state.desired[cap];
    qsa("[data-desired=\"" + cap + "\"] button").forEach(function(button) {
      var active = current == null ? button.dataset.value === "unset" : String(current) === button.dataset.value;
      button.classList.toggle("active", active);
      button.onclick = function() {
        state.desired[cap] = button.dataset.value === "unset" ? undefined : button.dataset.value === "true";
        renderDesired();
      };
    });
  });
  var target = document.createElement("div");
  target.className = "row two";
  target.innerHTML = "<label>Target <input id=\"desiredTarget\" type=\"number\" min=\"10\" max=\"40\" step=\"1\" value=\"" + (state.desired.target_temp == null ? "" : state.desired.target_temp) + "\"></label><button id=\"clearDesiredTemp\">Clear Target</button>";
  form.appendChild(target);
  $("desiredTarget").oninput = function() {
    state.desired.target_temp = this.value === "" ? undefined : Number(this.value);
  };
  $("clearDesiredTemp").onclick = function() {
    state.desired.target_temp = undefined;
    renderDesired();
  };
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
  if (!state.plans.length) {
    view.innerHTML = "<p class=\"muted\">No plans</p>";
    return;
  }
  var list = document.createElement("div");
  list.className = "plan-list";
  state.plans.forEach(function(plan) {
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
  view.innerHTML = "<div class=\"row three\"><label>Name <input id=\"windowName\" value=\"Filter window\"></label><label>Capability <select id=\"windowCap\"><option value=\"filter\">Filter</option><option value=\"heater\">Heater</option><option value=\"jets\">Jets</option><option value=\"bubbles\">Bubbles</option><option value=\"sanitizer\">Sanitizer</option></select></label><label>Enabled <select id=\"windowEnabled\"><option value=\"true\">On</option><option value=\"false\">Off</option></select></label></div><div class=\"row two\"><label>From <input id=\"windowFrom\" type=\"time\" value=\"02:00\"></label><label>To <input id=\"windowTo\" type=\"time\" value=\"04:00\"></label></div><div class=\"days\" id=\"windowDays\"></div><button class=\"primary\" id=\"addWindow\">Add Window</button><hr><div class=\"row two\"><label>Override Until <input id=\"overrideUntil\" type=\"datetime-local\"></label><label>Heater <select id=\"overrideHeater\"><option value=\"unset\">Any</option><option value=\"true\">On</option><option value=\"false\">Off</option></select></label></div><button id=\"addOverride\">Add Manual Override</button>";
  var dayWrap = $("windowDays");
  days.forEach(function(day) {
    var button = document.createElement("button");
    button.className = "day";
    button.textContent = day.slice(0, 1).toUpperCase();
    button.dataset.day = day;
    button.onclick = function() { button.classList.toggle("active"); };
    dayWrap.appendChild(button);
  });
  $("overrideUntil").value = localDateTime(new Date(Date.now() + 60 * 60 * 1000));
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
  $("addOverride").onclick = function() {
    var desired = {};
    var heater = $("overrideHeater").value;
    if (heater !== "unset") desired.heater = heater === "true";
    if (heater === "true") {
      desired.power = true;
      desired.filter = true;
    }
    if (!Object.keys(desired).length) return toast("Choose an override state", "bad");
    var until = $("overrideUntil").value;
    if (!until) return toast("Override time is required", "bad");
    updatePlans(state.plans.concat([{
      id: "override-" + Date.now(),
      type: "manual_override",
      name: "Manual override",
      enabled: true,
      desired_state: desired,
      expires_at: new Date(until).toISOString()
    }]));
  };
}

function renderActivity() {
  qsa("[data-activity]").forEach(function(button) {
    button.classList.toggle("active", button.dataset.activity === state.activityView);
  });
  var list = $("activity");
  var rows = state.activityView === "polls" ? state.polls : state.activityView === "commands" ? state.commands : state.events;
  if (!rows.length) {
    list.innerHTML = "<p class=\"muted\">No activity</p>";
    return;
  }
  list.innerHTML = "";
  rows.forEach(function(row) {
    var item = document.createElement("div");
    item.className = "activity";
    if (state.activityView === "polls") {
      item.innerHTML = "<time>" + formatTime(row.status.observed_at) + "</time><div><strong>Poll #" + row.id + "</strong><span>" + tempLine(row.status) + " · " + activeCaps(row.status).map(title).join(", ") + "</span></div>";
    } else {
      item.innerHTML = "<time>" + formatTime(row.created_at) + "</time><div><strong>" + title(row.type) + " #" + row.id + "</strong><span>" + eventLine(row) + "</span></div>";
    }
    list.appendChild(item);
  });
}

function commandBool(cap, value) {
  runAction(function() {
    return api("/commands", {method: "POST", body: JSON.stringify({capability: cap, state: value, source: "webui"})});
  }, capLabels[cap] + " " + boolText(value));
}

function commandTemp(value) {
  runAction(function() {
    return api("/commands", {method: "POST", body: JSON.stringify({capability: "target_temp", value: Number(value), source: "webui"})});
  }, "Temperature set");
}

function updatePlans(plans) {
  runAction(function() {
    return api("/plans", {method: "PUT", body: JSON.stringify({plans: plans})}).then(function(data) {
      state.plans = data.plans || [];
    });
  }, "Plans saved");
}

function saveDesired() {
  var desired = {};
  caps.forEach(function(cap) {
    if (state.desired[cap] !== undefined) desired[cap] = state.desired[cap];
  });
  if (state.desired.target_temp !== undefined && state.desired.target_temp !== "") {
    desired.target_temp = Number(state.desired.target_temp);
  }
  runAction(function() {
    return api("/desired-state", {method: "PUT", body: JSON.stringify(desired)}).then(function(saved) {
      state.desired = saved || {};
    });
  }, "Desired state saved");
}

function runAction(action, message) {
  if (!state.token) return toast("Token required", "bad");
  setBusy(true);
  action().then(function() {
    toast(message, "ok");
    return Promise.all([loadStatus(), loadDesired(), loadPlans(), loadEvents(), loadPolls()]);
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

function eventLine(event) {
  if (event.type === "observation" && event.data) return tempLine(event.data);
  if (event.type === "status_error" && event.data && event.data.error) return event.data.error;
  if (event.type === "command" && event.data) return title(event.data.capability) + " ok";
  return event.message || "";
}

function describePlan(plan) {
  if (plan.type === "ready_by") return (plan.target_temp || "--") + "° by " + formatDateTime(plan.at);
  if (plan.type === "time_window") return title(plan.capability) + " " + plan.from + "-" + plan.to + (plan.days && plan.days.length ? " · " + plan.days.join(", ") : "");
  if (plan.type === "manual_override") return "Until " + formatDateTime(plan.expires_at);
  return title(plan.type);
}

function formatTime(value) {
  if (!value) return "--";
  return new Date(value).toLocaleTimeString([], {hour: "2-digit", minute: "2-digit", second: "2-digit"});
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

function localDateTime(date) {
  var pad = function(n) { return String(n).padStart(2, "0"); };
  return date.getFullYear() + "-" + pad(date.getMonth() + 1) + "-" + pad(date.getDate()) + "T" + pad(date.getHours()) + ":" + pad(date.getMinutes());
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
$("refresh").onclick = loadAll;
$("reloadPlans").onclick = function() {
  loadPlans().then(renderPlans);
};
$("saveDesired").onclick = saveDesired;
$("sendTemp").onclick = function() { commandTemp($("tempInput").value); };
$("tempDown").onclick = function() { $("tempInput").value = Number($("tempInput").value || 0) - 1; };
$("tempUp").onclick = function() { $("tempInput").value = Number($("tempInput").value || 0) + 1; };
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
  if (state.token && !state.pending) Promise.all([loadStatus(), loadEvents(), loadPolls()]).then(renderLivePanels);
}, 30000);
</script>
</body>
</html>
`
