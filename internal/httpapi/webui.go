package httpapi

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"strings"
	"sync"
)

func (a *API) handleWebUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(webUIHTML("dashboard")))
}

func (a *API) handleHistoryUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/history" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(webUIHTML("history")))
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

func webUIHTML(page string) string {
	if page != "history" {
		page = "dashboard"
	}
	return strings.Replace(webUIHTMLTemplate, "__POOLD_PAGE__", page, 1)
}

const eChartsCDN = "https://cdn.jsdelivr.net/npm/echarts@6.0.0/dist/echarts.min.js"

const webUIHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover">
<meta name="theme-color" content="#007c89">
<link rel="icon" type="image/svg+xml" href="/favicon.svg">
<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png">
<script src="https://cdn.jsdelivr.net/npm/echarts@6.0.0/dist/echarts.min.js"></script>
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
  --soft: #eef6f8;
  --field: #fbfdfe;
  --shadow: 0 14px 34px rgba(23, 33, 38, .09);
  --shadow-soft: 0 6px 18px rgba(23, 33, 38, .06);
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
button, .button-link {
  min-height: 42px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #fff;
  color: var(--text);
  font-weight: 700;
  transition: background .18s ease, border-color .18s ease, box-shadow .18s ease, transform .18s ease;
}
.button-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 12px;
  text-decoration: none;
}
button.primary, .button-link.primary {
  border-color: var(--accent);
  background: var(--accent);
  color: #fff;
}
button:hover:not(:disabled), .button-link:hover {
  border-color: rgba(0, 124, 137, .42);
  box-shadow: var(--shadow-soft);
}
button:active:not(:disabled), .button-link:active {
  transform: translateY(1px);
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
  background: var(--field);
  color: var(--text);
  font-size: 16px;
  padding: 9px 10px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, .72);
  transition: background .18s ease, border-color .18s ease, box-shadow .18s ease;
}
input:hover, select:hover {
  border-color: #b8cbd2;
  background: #fff;
}
input:focus, select:focus {
  outline: 0;
  border-color: var(--accent);
  background: #fff;
  box-shadow: 0 0 0 3px rgba(0, 124, 137, .14);
}
select {
  appearance: none;
  padding-right: 36px;
  background-image: linear-gradient(45deg, transparent 50%, #5d6b73 50%), linear-gradient(135deg, #5d6b73 50%, transparent 50%);
  background-position: calc(100% - 18px) 18px, calc(100% - 13px) 18px;
  background-size: 5px 5px, 5px 5px;
  background-repeat: no-repeat;
}
input[type="datetime-local"], input[type="time"] {
  font-variant-numeric: tabular-nums;
  letter-spacing: 0;
  background: linear-gradient(180deg, #fff, #f8fbfc);
}
input[type="datetime-local"]::-webkit-calendar-picker-indicator,
input[type="time"]::-webkit-calendar-picker-indicator {
  border-radius: 7px;
  padding: 5px;
  background-color: #edf6f8;
  cursor: pointer;
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
body[data-page="history"] {
  background: #fff;
}
body[data-page="history"] .app {
  width: 100%;
  min-height: 100dvh;
  display: grid;
  grid-template-rows: auto auto 1fr auto;
  padding: max(12px, env(safe-area-inset-top)) max(12px, env(safe-area-inset-right)) max(12px, env(safe-area-inset-bottom)) max(12px, env(safe-area-inset-left));
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
.manual-actions.permanent {
  grid-template-columns: 1fr;
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
.timeline-panel {
  min-height: 430px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, .96), rgba(255, 255, 255, 1)),
    radial-gradient(circle at 20% 0%, rgba(0, 124, 137, .08), transparent 34%);
}
.timeline-title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.timeline-toolbar {
  display: grid;
  gap: 10px;
}
.timeline-controls {
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
}
.timeline-controls .tabs {
  padding: 4px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: #f2f7f9;
  gap: 4px;
}
.timeline-controls .tabs:first-child {
  grid-template-columns: repeat(5, minmax(0, 1fr));
}
.timeline-controls .tabs:last-child {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.timeline-controls .tabs button {
  min-height: 36px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  box-shadow: none;
  color: var(--muted);
}
.timeline-controls .tabs button.active {
  background: #fff;
  color: var(--text);
  box-shadow: 0 3px 10px rgba(23, 33, 38, .08);
}
.timeline-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  min-height: 27px;
}
.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 27px;
  padding: 4px 8px;
  border: 1px solid #e3ebef;
  border-radius: 999px;
  background: #fbfdfe;
  color: var(--muted);
  font-size: 12px;
  font-weight: 750;
}
.legend-swatch {
  width: 18px;
  height: 3px;
  border-radius: 999px;
  background: var(--accent);
  flex: 0 0 auto;
}
.legend-swatch.dot {
  width: 8px;
  height: 8px;
}
.legend-swatch.dash {
  background: repeating-linear-gradient(90deg, #8f6f2a 0 6px, transparent 6px 10px);
}
.legend-swatch.band {
  width: 16px;
  height: 9px;
  border-radius: 3px;
}
.timeline-chart {
  margin-top: 12px;
  min-height: 330px;
  border: 1px solid #e3ebef;
  border-radius: 8px;
  padding: 8px;
  background: linear-gradient(180deg, #fff, #fbfdfe);
}
.timeline-canvas {
  width: 100%;
  height: 330px;
}
.timeline-meta {
  color: var(--muted);
  font-size: 12px;
}
.timeline-empty {
  color: var(--muted);
  padding: 18px 0;
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
.history-only {
  display: none !important;
}
body[data-page="history"] .dashboard-only {
  display: none !important;
}
body[data-page="history"] .history-only {
  display: initial !important;
}
body[data-page="history"] .button-link.history-only {
  display: inline-flex !important;
}
body[data-page="history"] .timeline-controls.history-only {
  display: grid !important;
}
body[data-page="history"] .grid {
  display: block;
}
body[data-page="history"] .timeline-panel {
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  border: 0;
  box-shadow: none;
  padding: 0;
}
body[data-page="history"] .timeline-toolbar {
  margin-bottom: 8px;
}
body[data-page="history"] .timeline-chart {
  flex: 1 1 auto;
  min-height: 0;
  height: min(760px, calc(100dvh - 220px));
  padding-top: 8px;
}
body[data-page="history"] .timeline-canvas {
  height: 100%;
  min-height: 460px;
}
.hidden {
  display: none !important;
}
@media (min-width: 760px) {
  .app { padding: 20px; }
  .grid { grid-template-columns: 1.05fr .95fr; align-items: start; }
  .span-2 { grid-column: span 2; }
  .wide-only { display: inline; }
  .controls { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .row.three { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .timeline-controls { grid-template-columns: 1.4fr .8fr; align-items: start; }
  .activity-tabs { grid-template-columns: repeat(5, minmax(0, 1fr)); width: min(560px, 100%); }
}
</style>
</head>
<body data-page="__POOLD_PAGE__">
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
      <a class="button-link primary history-only" href="/">Dashboard</a>
      <button id="editToken">Token</button>
      <button class="dashboard-only" id="settingsToggle">Settings</button>
      <button id="refresh">Refresh</button>
    </div>
  </header>

  <section class="tokenbar" id="tokenbar">
    <input id="token" type="password" autocomplete="current-password" placeholder="Bearer token">
    <button class="primary" id="saveToken">Save</button>
  </section>

  <section class="panel settings-panel dashboard-only" id="settingsPanel">
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
    <section class="panel dashboard-only">
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

    <section class="panel dashboard-only">
      <div class="panel-head">
        <h2>Controls</h2>
        <span class="badge" id="busy">Ready</span>
      </div>
      <div class="manual-strip" id="manualStrip">
        <div>
          <strong id="manualTitle">Manual control</strong>
          <span id="manualDetail">No override active</span>
        </div>
        <div class="manual-actions" id="manualActions">
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

    <section class="panel dashboard-only">
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

    <section class="panel span-2 timeline-panel">
      <div class="panel-head">
        <div class="timeline-title">
          <h2>History</h2>
          <span class="badge" id="timelineBadge">24h</span>
        </div>
        <a class="button-link dashboard-only" href="/history">Open</a>
      </div>
      <div class="timeline-toolbar">
        <div class="timeline-controls history-only">
          <div class="tabs">
            <button data-timeline-range="6h">6h</button>
            <button class="active" data-timeline-range="24h">24h</button>
            <button data-timeline-range="3d">3d</button>
            <button data-timeline-range="7d">7d</button>
            <button data-timeline-range="14d">14d</button>
          </div>
          <div class="tabs">
            <button class="active" data-timeline-mode="measured">Measured</button>
            <button data-timeline-mode="predicted">Predicted</button>
          </div>
        </div>
        <div class="timeline-meta" id="timelineMeta">No history loaded</div>
        <div class="timeline-legend" id="timelineLegend"></div>
      </div>
      <div class="timeline-chart" id="timelineChart"></div>
    </section>

    <section class="panel span-2 dashboard-only">
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
var pageMode = document.body.dataset.page || "dashboard";
var isHistoryPage = pageMode === "history";
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
  timeline: null,
  timelineRange: "24h",
  timelineMode: "measured",
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
  if (isHistoryPage) {
    setBusy(true);
    loadTimeline().finally(function() {
      setBusy(false);
      renderTimeline();
    });
    return;
  }
  setBusy(true);
  Promise.all([
    loadStatus(),
    loadWeather(),
    loadTimeline(),
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

function loadTimeline() {
  return api("/dashboard/timeline?range=" + encodeURIComponent(state.timelineRange)).then(function(data) {
    state.timeline = data || null;
  }).catch(function(err) {
    state.timeline = null;
    toast("History: " + err.message, "bad");
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
  if (isHistoryPage) {
    renderTimeline();
    return;
  }
  renderStatus();
  renderWeather();
  renderSettings();
  renderManual();
  renderControls();
  renderPlans();
  renderTimeline();
  renderActivity();
}

function renderLivePanels() {
  if (isHistoryPage) {
    renderTimeline();
    return;
  }
  renderStatus();
  renderWeather();
  renderSettings();
  renderManual();
  renderControls();
  renderTimeline();
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
  var actions = $("manualActions");
  strip.classList.toggle("active", !!plan);
  var timed = !!(plan && plan.expires_at);
  actions.classList.toggle("permanent", !!plan && !timed);
  $("manualMinus").classList.toggle("hidden", !!plan && !timed);
  $("manualPlus").classList.toggle("hidden", !!plan && !timed);
  $("manualPermanent").classList.toggle("hidden", !!plan && !timed);
  $("manualMinus").disabled = state.pending;
  $("manualPlus").disabled = state.pending;
  $("manualPermanent").disabled = state.pending || (!!plan && !timed);
  if (plan) {
    $("manualTitle").textContent = manualTitle(plan.desired_state || {});
    $("manualDetail").textContent = manualSummary(plan.desired_state || {}) + " · " + manualDurationLabel(plan);
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
    item.innerHTML = "<div class=\"plan-main\"><div><h3>" + escapeHTML(plan.name || plan.id) + "</h3><p class=\"muted\">" + escapeHTML(describePlan(plan)) + "</p></div><span class=\"badge " + (plan.enabled ? "ok" : "") + "\">" + (plan.enabled ? "Active" : "Paused") + "</span></div><div class=\"plan-actions\"><label>Active <select data-active=\"" + escapeHTML(plan.id) + "\"><option value=\"true\"" + (plan.enabled ? " selected" : "") + ">Yes</option><option value=\"false\"" + (!plan.enabled ? " selected" : "") + ">No</option></select></label><button class=\"danger\" data-delete=\"" + escapeHTML(plan.id) + "\">Delete</button></div>";
    list.appendChild(item);
  });
  view.appendChild(list);
  qsa("[data-active]").forEach(function(select) {
    select.onchange = function() {
      var enabled = select.value === "true";
      updatePlans(state.plans.map(function(plan) {
        if (plan.id === select.dataset.active) return Object.assign({}, plan, {enabled: enabled});
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
  view.innerHTML = "<div class=\"row three\"><label>Name <input id=\"readyName\" value=\"Ready by\"></label><label>Target <input id=\"readyTemp\" type=\"number\" min=\"10\" max=\"40\" step=\"1\" value=\"36\"></label><label>Active <select id=\"readyEnabled\"><option value=\"true\">Yes</option><option value=\"false\">No</option></select></label></div><div class=\"row two\"><label>Mode <select id=\"readyMode\"><option value=\"once\">Once</option><option value=\"cron\">Repeating</option></select></label><label id=\"readyAtWrap\">At <input id=\"readyAt\" type=\"datetime-local\"></label><label id=\"readyTimeWrap\" class=\"hidden\">At <input id=\"readyTime\" type=\"time\" value=\"08:30\"></label></div><div class=\"days hidden\" id=\"readyDays\"></div><button class=\"primary\" id=\"addReady\">Add Ready Plan</button>";
  $("readyAt").value = localDateTime(new Date(Date.now() + 24 * 60 * 60 * 1000));
  var dayWrap = $("readyDays");
  days.forEach(function(day) {
    var button = document.createElement("button");
    button.className = "day active";
    button.textContent = day.slice(0, 1).toUpperCase();
    button.dataset.day = day;
    button.onclick = function() { button.classList.toggle("active"); };
    dayWrap.appendChild(button);
  });
  $("readyMode").onchange = updateReadyMode;
  updateReadyMode();
  $("addReady").onclick = function() {
    var plan = {
      id: "ready-by-" + Date.now(),
      type: "ready_by",
      name: $("readyName").value || "Ready by",
      enabled: $("readyEnabled").value === "true",
      target_temp: Number($("readyTemp").value || 36)
    };
    if ($("readyMode").value === "cron") {
      var cron = readyCron();
      if (!cron) return toast("Ready time is required", "bad");
      plan.cron = cron;
    } else {
      var at = $("readyAt").value;
      if (!at) return toast("Ready time is required", "bad");
      plan.at = new Date(at).toISOString();
    }
    updatePlans(state.plans.concat([plan]));
  };
}

function updateReadyMode() {
  var repeating = $("readyMode").value === "cron";
  $("readyAtWrap").classList.toggle("hidden", repeating);
  $("readyTimeWrap").classList.toggle("hidden", !repeating);
  $("readyDays").classList.toggle("hidden", !repeating);
}

function readyCron() {
  var time = $("readyTime").value;
  if (!time || time.indexOf(":") < 0) return "";
  var parts = time.split(":");
  var selectedDays = qsa("#readyDays .day.active").map(function(button) { return button.dataset.day; });
  var dayField = "*";
  if (selectedDays.length > 0 && selectedDays.length < days.length) {
    dayField = selectedDays.map(cronDay).join(",");
  }
  return Number(parts[1]) + " " + Number(parts[0]) + " * * " + dayField;
}

function cronDay(day) {
  return {sun: 0, mon: 1, tue: 2, wed: 3, thu: 4, fri: 5, sat: 6}[day];
}

function renderWindowForm(view) {
  view.innerHTML = "<div class=\"row three\"><label>Name <input id=\"windowName\" value=\"Filter window\"></label><label>Capability <select id=\"windowCap\"><option value=\"filter\">Filter</option><option value=\"heater\">Heater</option><option value=\"jets\">Jets</option><option value=\"bubbles\">Bubbles</option><option value=\"sanitizer\">Sanitizer</option></select></label><label>Active <select id=\"windowEnabled\"><option value=\"true\">Yes</option><option value=\"false\">No</option></select></label></div><div class=\"row two\"><label>From <input id=\"windowFrom\" type=\"time\" value=\"02:00\"></label><label>To <input id=\"windowTo\" type=\"time\" value=\"04:00\"></label></div><div class=\"days\" id=\"windowDays\"></div><button class=\"primary\" id=\"addWindow\">Add Window</button>";
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

var timelineChartInstance = null;

function renderTimeline() {
  qsa("[data-timeline-range]").forEach(function(button) {
    button.classList.toggle("active", button.dataset.timelineRange === state.timelineRange);
  });
  qsa("[data-timeline-mode]").forEach(function(button) {
    button.classList.toggle("active", button.dataset.timelineMode === state.timelineMode);
  });
  var chart = $("timelineChart");
  if (!state.token) {
    $("timelineBadge").textContent = "Token";
    $("timelineMeta").textContent = "History locked";
    renderTimelineLegend(null);
    renderTimelineMessage(chart, "No history loaded");
    return;
  }
  var data = state.timeline;
  if (!data || !data.from || !data.to) {
    $("timelineBadge").textContent = state.timelineRange;
    $("timelineMeta").textContent = "No history loaded";
    renderTimelineLegend(null);
    renderTimelineMessage(chart, "No history loaded");
    return;
  }
  if (typeof echarts === "undefined") {
    $("timelineBadge").textContent = data.range || state.timelineRange;
    $("timelineMeta").textContent = "Chart library unavailable";
    renderTimelineLegend(data);
    renderTimelineMessage(chart, "Chart library unavailable");
    return;
  }
  $("timelineBadge").textContent = data.range || state.timelineRange;
  $("timelineMeta").textContent = timelineMeta(data);
  renderTimelineLegend(data);
  renderTimelineChart(chart, data);
}

function timelineMeta(data) {
  var model = data.model || {};
  var parts = [
    formatDateTime(data.from) + " to " + formatDateTime(data.to),
    "bucket " + formatDurationSeconds(data.bucket_seconds || 0),
    state.timelineMode === "predicted" ? model.heating_model + ", " + model.cooling_model : "measured"
  ];
  if (data.warnings && data.warnings.length) parts.push(data.warnings.length + " warning" + (data.warnings.length === 1 ? "" : "s"));
  return parts.join(" · ");
}

function renderTimelineLegend(data) {
  var box = $("timelineLegend");
  if (!box) return;
  if (!data || !data.from || !data.to) {
    box.innerHTML = "";
    return;
  }
  var items = [
    {label: state.timelineMode === "predicted" ? "Pool predicted" : "Pool measured", color: state.timelineMode === "predicted" ? "#235ea8" : "#007c89"},
    {label: "Outside", color: "#8aa1a8"},
    {label: "Target", kind: "dash"}
  ];
  if (state.timelineMode === "predicted") {
    items.push({label: "Measured", color: "#172126", kind: "dot"});
    items.push({label: "Correction", color: "#d97904", kind: "dot"});
  }
  if ((data.annotations || []).length) {
    items.push({label: "Command/plan", color: "#172126", kind: "dot"});
  }
  timelineFeatureLegendItems(data.feature_spans || []).forEach(function(item) {
    items.push(item);
  });
  box.innerHTML = items.map(function(item) {
    var kind = item.kind || "line";
    var style = item.color ? " style=\"background:" + item.color + "\"" : "";
    return "<span class=\"legend-item\"><i class=\"legend-swatch " + kind + "\"" + style + "></i>" + escapeHTML(item.label) + "</span>";
  }).join("");
}

function timelineFeatureLegendItems(spans) {
  var colors = timelineFeatureColors();
  var lanes = timelineLanes(spans);
  return lanes.filter(function(lane) {
    return spans.some(function(span) { return lane === "connected" ? span.connected === false : !!span[lane]; });
  }).map(function(lane) {
    return {label: timelineLaneLabel(lane), color: colors[lane], kind: "band"};
  });
}

function renderTimelineMessage(chart, message) {
  disposeTimelineChart();
  chart.innerHTML = "<div class=\"timeline-empty\">" + escapeHTML(message) + "</div>";
}

function renderTimelineChart(chart, data) {
  disposeTimelineChart();
  chart.innerHTML = "";
  var canvas = document.createElement("div");
  canvas.className = "timeline-canvas";
  chart.appendChild(canvas);
  timelineChartInstance = echarts.init(canvas, null, {renderer: "canvas"});
  timelineChartInstance.setOption(timelineOption(data), true);
  setTimeout(function() {
    if (timelineChartInstance) timelineChartInstance.resize();
  }, 0);
}

function disposeTimelineChart() {
  if (timelineChartInstance) {
    timelineChartInstance.dispose();
    timelineChartInstance = null;
  }
}

function timelineOption(data) {
  var from = new Date(data.from).getTime();
  var to = new Date(data.to).getTime();
  var modePoints = state.timelineMode === "predicted" ? (data.predicted || []) : (data.measured || []);
  var measured = data.measured || [];
  var target = data.target || [];
  var values = timelineValues(modePoints, measured, target);
  if (!values.length || !Number.isFinite(from) || !Number.isFinite(to) || to <= from) {
    return {title: {text: "No timeline data", left: "center", top: "middle", textStyle: {fontSize: 14, color: "#5d6b73", fontWeight: 600}}};
  }
  var min = Math.floor(Math.min.apply(null, values)) - 1;
  var max = Math.ceil(Math.max.apply(null, values)) + 1;
  if (min === max) {
    min -= 1;
    max += 1;
  }
  var lanes = timelineLanes(data.feature_spans || []);
  var laneGridHeight = Math.max(58, lanes.length * 24 + 14);
  var gridBottom = laneGridHeight + (isHistoryPage ? 58 : 34);
  var option = {
    animation: false,
    grid: [
      {left: 48, right: isHistoryPage ? 28 : 12, top: isHistoryPage ? 42 : 16, bottom: gridBottom},
      {left: 48, right: isHistoryPage ? 28 : 12, height: laneGridHeight, bottom: isHistoryPage ? 42 : 12}
    ],
    legend: {show: false},
    tooltip: {
      trigger: "axis",
      confine: true,
      axisPointer: {type: "cross"},
      formatter: timelineTooltip
    },
    axisPointer: {link: [{xAxisIndex: [0, 1]}]},
    xAxis: [
      {type: "time", min: from, max: to, axisLabel: {color: "#5d6b73"}, axisLine: {lineStyle: {color: "#d8e1e5"}}, splitLine: {show: true, lineStyle: {color: "#eef3f5"}}},
      {type: "time", min: from, max: to, gridIndex: 1, axisLabel: {show: isHistoryPage, color: "#5d6b73"}, axisLine: {lineStyle: {color: "#d8e1e5"}}, splitLine: {show: false}}
    ],
    yAxis: [
      {type: "value", min: min, max: max, axisLabel: {formatter: "{value}°", color: "#5d6b73"}, axisLine: {show: false}, splitLine: {lineStyle: {color: "#e6edf0"}}},
      {type: "category", gridIndex: 1, data: lanes.map(timelineLaneLabel), inverse: true, axisTick: {show: false}, axisLine: {show: false}, axisLabel: {color: "#5d6b73", fontSize: 12}, splitLine: {show: true, lineStyle: {color: "#eef3f5"}}}
    ],
    series: timelineSeries(data, lanes, min, max)
  };
  if (isHistoryPage) {
    option.dataZoom = [
      {type: "inside", xAxisIndex: [0, 1], filterMode: "none"},
      {type: "slider", xAxisIndex: [0, 1], filterMode: "none", bottom: 6, height: 24, borderColor: "#d8e1e5"}
    ];
  }
  return option;
}

function timelineSeries(data, lanes, min, max) {
  var measured = data.measured || [];
  var modePoints = state.timelineMode === "predicted" ? (data.predicted || []) : measured;
  var linePoints = modePoints.filter(function(point) { return point.kind !== "correction"; });
  var series = [
    {
      name: "Outside",
      type: "line",
      data: timelineLineData(measured, "outside_temp_c"),
      showSymbol: false,
      connectNulls: false,
      lineStyle: {color: "#8aa1a8", width: 2, opacity: .75},
      itemStyle: {color: "#8aa1a8"},
      emphasis: {focus: "series"}
    },
    {
      name: "Target",
      type: "line",
      data: timelineTargetData(data.target || []),
      showSymbol: false,
      step: "end",
      lineStyle: {color: "#8f6f2a", width: 2, type: "dashed"},
      itemStyle: {color: "#8f6f2a"},
      emphasis: {focus: "series"}
    },
    {
      name: state.timelineMode === "predicted" ? "Pool predicted" : "Pool measured",
      type: "line",
      data: timelineLineData(linePoints, "pool_temp"),
      showSymbol: false,
      connectNulls: false,
      lineStyle: {color: state.timelineMode === "predicted" ? "#235ea8" : "#007c89", width: 3},
      itemStyle: {color: state.timelineMode === "predicted" ? "#235ea8" : "#007c89"},
      emphasis: {focus: "series"}
    }
  ];
  if (state.timelineMode === "predicted") {
    series.push({
      name: "Measured anchors",
      type: "scatter",
      data: timelineLineData(measured, "pool_temp"),
      symbolSize: isHistoryPage ? 6 : 4,
      itemStyle: {color: "#172126"}
    });
    series.push({
      name: "Corrections",
      type: "scatter",
      data: timelineLineData((data.predicted || []).filter(function(point) { return point.kind === "correction"; }), "pool_temp"),
      symbolSize: isHistoryPage ? 10 : 7,
      itemStyle: {color: "#d97904"}
    });
  }
  var annotations = timelineAnnotationData(data.annotations || [], max);
  if (annotations.length) {
    series.push({
      name: "Annotations",
      type: "scatter",
      data: annotations,
      symbol: "pin",
      symbolSize: isHistoryPage ? 18 : 13,
      itemStyle: {color: "#172126"},
      tooltip: {trigger: "item", formatter: timelineItemTooltip}
    });
  }
  var featureData = timelineFeatureData(data.feature_spans || [], lanes);
  if (featureData.length) {
    series.push({
      name: "Features",
      type: "custom",
      xAxisIndex: 1,
      yAxisIndex: 1,
      clip: true,
      data: featureData,
      renderItem: renderTimelineFeature,
      tooltip: {trigger: "item", formatter: timelineItemTooltip}
    });
  }
  return series;
}

function timelineValues(points, measured, target) {
  var values = [];
  points.forEach(function(point) {
    if (point.pool_temp != null) values.push(Number(point.pool_temp));
  });
  measured.forEach(function(point) {
    if (point.outside_temp_c != null) values.push(Number(point.outside_temp_c));
  });
  target.forEach(function(point) {
    if (point.target_temp != null) values.push(Number(point.target_temp));
  });
  return values.filter(Number.isFinite);
}

function timelineLineData(points, field) {
  return points.map(function(point) {
    var value = point[field];
    if (value == null || !Number.isFinite(Number(value))) {
      return null;
    }
    return [new Date(point.t).getTime(), Number(value), point.confidence == null ? null : Number(point.confidence), point.model || "", point.kind || ""];
  }).filter(function(point) { return point && Number.isFinite(point[0]); });
}

function timelineTargetData(points) {
  return points.map(function(point) {
    if (point.target_temp == null) {
      return null;
    }
    return [new Date(point.t).getTime(), Number(point.target_temp)];
  }).filter(function(point) { return point && Number.isFinite(point[0]) && Number.isFinite(point[1]); });
}

function timelineLanes(spans) {
  var lanes = ["power", "filter", "heater"];
  ["jets", "bubbles", "sanitizer"].forEach(function(cap) {
    if (spans.some(function(span) { return !!span[cap]; })) lanes.push(cap);
  });
  if (spans.some(function(span) { return span.connected === false; })) lanes.push("connected");
  return lanes;
}

function timelineLaneLabel(lane) {
  return title(lane === "connected" ? "offline" : lane);
}

function timelineFeatureData(spans, lanes) {
  var colors = timelineFeatureColors();
  var data = [];
  lanes.forEach(function(lane, index) {
    spans.forEach(function(span) {
      var active = lane === "connected" ? span.connected === false : !!span[lane];
      if (!active) return;
      var start = new Date(span.from).getTime();
      var end = new Date(span.to).getTime();
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return;
      var label = timelineLaneLabel(lane);
      data.push({
        name: label + " · " + formatDateTime(span.from) + " to " + formatDateTime(span.to),
        value: [start, end, index],
        itemStyle: {color: colors[lane], opacity: .82}
      });
    });
  });
  return data;
}

function timelineFeatureColors() {
  return {power:"#7a8790", filter:"#00a6b2", heater:"#d97904", jets:"#235ea8", bubbles:"#7c4dff", sanitizer:"#1d7f45", connected:"#b42318"};
}

function timelineAnnotationData(annotations, y) {
  return annotations.map(function(annotation) {
    var t = new Date(annotation.t).getTime();
    if (!Number.isFinite(t)) return null;
    return {
      name: annotation.label || "Annotation",
      value: [t, y, annotation.detail || "", annotation.source || ""]
    };
  }).filter(Boolean);
}

function renderTimelineFeature(params, api) {
  var start = api.coord([api.value(0), api.value(2)]);
  var end = api.coord([api.value(1), api.value(2)]);
  var laneSize = api.size([0, 1]);
  var height = Math.max(8, laneSize[1] * .58);
  var shape = echarts.graphic.clipRectByRect({
    x: start[0],
    y: start[1] - height / 2,
    width: Math.max(1, end[0] - start[0]),
    height: height
  }, {
    x: params.coordSys.x,
    y: params.coordSys.y,
    width: params.coordSys.width,
    height: params.coordSys.height
  });
  if (!shape) return;
  return {type: "rect", shape: shape, style: api.style()};
}

function timelineTooltip(params) {
  if (!Array.isArray(params)) return timelineItemTooltip(params);
  var time = null;
  var rows = [];
  params.forEach(function(param) {
    if (!param || !param.value) return;
    if (param.seriesType === "custom") {
      rows.push(param.marker + escapeHTML(param.name || "Feature"));
      return;
    }
    var value = param.value;
    if (time == null && value[0] != null) time = value[0];
    if (param.seriesName === "Annotations") {
      rows.push(param.marker + escapeHTML(param.name || "Annotation") + (value[2] ? " · " + escapeHTML(value[2]) : ""));
      return;
    }
    if (value[1] == null || !Number.isFinite(Number(value[1]))) return;
    var suffix = param.seriesName === "Outside" || param.seriesName.indexOf("Pool") === 0 || param.seriesName === "Target" || param.seriesName === "Measured anchors" || param.seriesName === "Corrections" ? "°" : "";
    var detail = "";
    if (value[2] != null && Number.isFinite(Number(value[2]))) detail = " · confidence " + Math.round(Number(value[2]) * 100) + "%";
    if (value[3]) detail += " · " + escapeHTML(value[3]);
    rows.push(param.marker + escapeHTML(param.seriesName) + ": " + Number(value[1]).toFixed(1) + suffix + detail);
  });
  if (time == null) return rows.join("<br>");
  return "<strong>" + formatDateTime(time) + "</strong><br>" + rows.join("<br>");
}

function timelineItemTooltip(param) {
  if (!param) return "";
  if (param.seriesType === "custom") return escapeHTML(param.name || "Feature");
  var value = param.value || [];
  if (param.seriesName === "Annotations") {
    return "<strong>" + formatDateTime(value[0]) + "</strong><br>" + escapeHTML(param.name || "Annotation") + (value[2] ? "<br>" + escapeHTML(value[2]) : "");
  }
  if (value[1] == null) return escapeHTML(param.name || "");
  return "<strong>" + formatDateTime(value[0]) + "</strong><br>" + escapeHTML(param.seriesName || "") + ": " + Number(value[1]).toFixed(1) + "°";
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
  if (!plan.expires_at) return;
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
  var permanentPlan = {
    id: manualPlanID,
    type: "manual_override",
    name: manualTitle(manualDesired),
    enabled: true,
    desired_state: manualDesired
  };
  runAction(function() {
    return api("/plans", {method: "PUT", body: JSON.stringify({
      plans: state.plans.filter(function(existing) { return !isReservedManualPlan(existing); }).concat([permanentPlan])
    })}).then(function(data) {
      state.plans = data.plans || [];
      state.manualDraft = null;
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

function runAction(action, message) {
  if (!state.token) return toast("Token required", "bad");
  setBusy(true);
  action().then(function() {
    toast(message, "ok");
    return Promise.all([loadStatus(), loadWeather(), loadTimeline(), loadPlans(), loadActivities()]);
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
  if (isReservedManualPlan(plan)) return manualSummary(plan.desired_state || {}) + " · " + manualDurationLabel(plan);
  if (plan.type === "ready_by") return (plan.target_temp || "--") + "° by " + readyScheduleLabel(plan);
  if (plan.type === "time_window") return title(plan.capability) + " " + plan.from + "-" + plan.to + (plan.days && plan.days.length ? " · " + plan.days.join(", ") : "");
  if (plan.type === "manual_override") return plan.expires_at ? "Until " + formatDateTime(plan.expires_at) : "Permanent";
  return title(plan.type);
}

function readyScheduleLabel(plan) {
  if (plan.cron) return describeCron(plan.cron);
  return formatDateTime(plan.at);
}

function describeCron(cron) {
  var fields = String(cron || "").trim().split(/\s+/);
  if (fields.length !== 5) return cron || "--";
  if (!isSimpleCronNumber(fields[0]) || !isSimpleCronNumber(fields[1])) return cron;
  var label = pad2(Number(fields[1])) + ":" + pad2(Number(fields[0]));
  if (fields[2] !== "*" || fields[3] !== "*") return label + " cron " + cron;
  return label + " " + describeCronDays(fields[4]);
}

function isSimpleCronNumber(value) {
  return /^\d+$/.test(value);
}

function describeCronDays(field) {
  if (!field || field === "*") return "daily";
  var labels = {0: "sun", 1: "mon", 2: "tue", 3: "wed", 4: "thu", 5: "fri", 6: "sat", 7: "sun"};
  var parts = field.split(",");
  if (parts.length === 7) return "daily";
  return parts.map(function(part) { return labels[part] || part; }).join(", ");
}

function pad2(value) {
  return String(value).padStart(2, "0");
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
      (!plan.expires_at || new Date(plan.expires_at).getTime() > now);
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

function manualDurationLabel(plan) {
  return plan && plan.expires_at ? remainingTime(plan.expires_at) + " left" : "Permanent";
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
qsa("[data-timeline-range]").forEach(function(button) {
  button.onclick = function() {
    state.timelineRange = button.dataset.timelineRange;
    setBusy(true);
    loadTimeline().finally(function() {
      setBusy(false);
      renderTimeline();
    });
  };
});
qsa("[data-timeline-mode]").forEach(function(button) {
  button.onclick = function() {
    state.timelineMode = button.dataset.timelineMode;
    renderTimeline();
  };
});
window.addEventListener("resize", function() {
  if (timelineChartInstance) timelineChartInstance.resize();
});

updateTokenUI();
renderAll();
loadAll();
setInterval(function() {
  if (!isHistoryPage && state.token && !state.pending) Promise.all([loadStatus(), loadWeather(), loadActivities()]).then(renderLivePanels);
}, 30000);
setInterval(function() {
  if (state.token && !state.pending) loadTimeline().then(renderTimeline);
}, 60000);
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
