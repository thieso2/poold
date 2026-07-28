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
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="theme-color" content="#007c89">
<meta name="apple-mobile-web-app-capable" content="yes">
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
  min-height: 44px;
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
button:focus-visible, .button-link:focus-visible,
input:focus-visible, select:focus-visible {
  outline: 3px solid #172126;
  outline-offset: 3px;
  box-shadow: 0 0 0 5px #fff;
}
input, select {
  width: 100%;
  min-height: 44px;
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
  padding: max(14px, env(safe-area-inset-top)) max(14px, env(safe-area-inset-right)) max(14px, env(safe-area-inset-bottom)) max(14px, env(safe-area-inset-left));
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
/* ---- pool control: ownership, equipment, target, apply ---- */
.pc { --live: #00727e; --live-deep: #005661; --live-soft: #e2f1f2; --live-line: #93c8cd;
  --manual: #a55f00; --manual-ink: #8a4f00; --manual-soft: #fdf2e2; --manual-line: #e8bf80;
  --data: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  display: grid; gap: 10px; }
.pc-owner { display: grid; grid-template-columns: 1fr 1fr; gap: 9px; }
.pc-own {
  display: grid; gap: 3px; align-content: start; text-align: left;
  padding: 12px 13px 13px; border: 1.5px solid var(--line); border-radius: 12px; background: var(--panel);
}
.pc-own-head { display: flex; align-items: center; gap: 8px; }
.pc-own-head svg { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 1.85; stroke-linecap: round; stroke-linejoin: round; }
.pc-own-name { font-family: var(--data); font-size: 12.5px; font-weight: 700; letter-spacing: .11em; }
.pc-own p { margin: 0; color: var(--muted); font-size: 12px; line-height: 1.35; }
.pc-own-flag {
  justify-self: start; margin-top: 6px; padding: 3px 8px; border-radius: 5px;
  font-family: var(--data); font-size: 9.5px; font-weight: 700; letter-spacing: .1em;
  background: #edf2f4; color: var(--muted);
}
.pc-own[aria-pressed="true"] { color: #fff; }
.pc-own.auto[aria-pressed="true"] { border-color: var(--live-deep); background: var(--live); }
.pc-own.man[aria-pressed="true"] { border-color: var(--manual); background: var(--manual); }
.pc-own[aria-pressed="true"] p { color: rgba(255, 255, 255, .87); }
.pc-own[aria-pressed="true"] .pc-own-flag { background: rgba(255, 255, 255, .2); color: #fff; }
.pc-own:disabled { opacity: .5; }

.pc-card { border: 1px solid var(--line); border-radius: 12px; background: var(--panel); overflow: hidden; }
.pc-head {
  display: flex; align-items: baseline; justify-content: space-between; gap: 10px;
  padding: 10px 13px; border-bottom: 1px solid var(--line); background: #f4f8f9;
}
.pc-head h3 { margin: 0; font-family: var(--data); font-size: 11px; font-weight: 700; letter-spacing: .12em; color: var(--muted); }
.pc-head span { font-size: 11.5px; color: var(--muted); }
.pc-lock { padding: 8px 13px; border-bottom: 1px solid var(--line); background: #f4f8f9; font-size: 12px; color: var(--muted); }

.pc-temp { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 13px 0; }
.pc-big { font-family: var(--data); font-size: 40px; font-weight: 700; line-height: 1; letter-spacing: -.035em; font-variant-numeric: tabular-nums; }
.pc-big small { font-size: 12px; font-weight: 700; letter-spacing: .02em; color: var(--muted); margin-left: 6px; }
.pc-temp-say { text-align: right; }
.pc-temp-say strong { display: block; font-size: 13px; }
.pc-temp-say span { display: block; margin-top: 1px; font-size: 11.5px; color: var(--muted); }

.pc-row { display: grid; grid-template-columns: repeat(5, 1fr); gap: 8px; padding: 12px 13px 13px; }
.pc-cap {
  position: relative; display: grid; place-items: center; height: 64px; padding: 0;
  border: 1.5px solid var(--line); border-radius: 12px; background: #f4f8f9; color: #6d8288;
}
.pc-cap svg { width: 29px; height: 29px; fill: none; stroke: currentColor; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
.pc-cap[aria-pressed="true"] { border-color: var(--live-deep); background: var(--live); color: #fff; }
.pc-cap:not(:disabled):hover { border-color: #6d8288; }
.pc-cap:disabled { opacity: .8; }
.pc-flag {
  position: absolute; top: -5px; right: -5px; width: 13px; height: 13px; border-radius: 50%;
  background: var(--manual); border: 2.5px solid var(--panel);
}
.pc-cap.dep .pc-flag { background: #6d8288; }
.pc-why { padding: 0 13px 12px; margin: 0; font-size: 12px; font-weight: 600; color: var(--manual); }

.pc-target { display: grid; grid-template-columns: 1fr auto; gap: 12px; align-items: center; padding: 12px 13px; }
.pc-target-note { font-size: 12.5px; color: var(--muted); }
.pc-target-note.changed { color: var(--manual); font-weight: 600; }
.pc-step { display: grid; grid-template-columns: 44px auto 44px; gap: 8px; align-items: center; }
.pc-step button { height: 44px; padding: 0; border-radius: 10px; font-size: 20px; font-weight: 700; }
.pc-val { display: grid; gap: 1px; justify-items: center; min-width: 70px; }
.pc-val b { font-family: var(--data); font-size: 25px; font-weight: 700; font-variant-numeric: tabular-nums; letter-spacing: -.02em; }
.pc-val.changed b { color: var(--manual); }
.pc-val span { font-family: var(--data); font-size: 9px; letter-spacing: .12em; color: var(--muted); }

.pc-apply {
  display: grid; gap: 10px; padding: 12px 13px; border: 1.5px solid var(--manual-line);
  border-radius: 12px; background: var(--manual-soft);
}
.pc-apply-top { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; }
.pc-apply-top strong { font-size: 13.5px; }
.pc-apply-top span { font-family: var(--data); font-size: 10.5px; letter-spacing: .08em; color: var(--manual-ink); }
.pc-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.pc-chip { padding: 4px 9px; border: 1px solid var(--manual-line); border-radius: 999px; background: var(--panel); font-size: 12px; font-weight: 600; }
.pc-chip i { font-style: normal; color: var(--muted); font-weight: 400; }
.pc-note { margin: 0; font-size: 12px; color: var(--text); }
.pc-label { font-family: var(--data); font-size: 9.5px; font-weight: 700; letter-spacing: .13em; color: var(--manual-ink); }
.pc-durations { display: grid; grid-template-columns: repeat(auto-fit, minmax(76px, 1fr)); gap: 7px; margin-top: 6px; }
.pc-durations button {
  min-height: 44px; padding: 6px 4px; border: 1.5px solid var(--manual); border-radius: 10px;
  background: var(--manual); color: #fff; font-size: 13px; font-weight: 700;
}
.pc-durations button:disabled { border-color: var(--manual-line); background: var(--panel); color: var(--muted); opacity: .75; }
.pc-confirm { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }

.pc-session-top { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 13px; }
.pc-state { display: grid; gap: 3px; }
.pc-state b { display: flex; align-items: center; gap: 8px; font-family: var(--data); font-size: 12.5px; font-weight: 700; letter-spacing: .1em; }
.pc-state b i { width: 9px; height: 9px; border-radius: 50%; background: currentColor; }
.pc-state span { font-size: 12.5px; color: var(--muted); }
.pc-state.applying b { color: #9a6100; }
.pc-state.active b { color: var(--ok, #1d7f45); }
.pc-state.degraded b { color: #b3261e; }
.pc-clock { font-family: var(--data); font-size: 25px; font-weight: 700; font-variant-numeric: tabular-nums; text-align: right; }
.pc-clock small { display: block; font-size: 9px; font-weight: 700; letter-spacing: .12em; color: var(--muted); }
.pc-outcomes { display: grid; grid-template-columns: repeat(6, 1fr); gap: 8px; padding: 12px 13px; border-top: 1px solid var(--line); }
.pc-out {
  position: relative; display: grid; place-items: center; height: 62px;
  border: 1.5px solid var(--line); border-radius: 12px; background: #f4f8f9; color: #6d8288;
}
.pc-out svg { width: 28px; height: 28px; fill: none; stroke: currentColor; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
.pc-out.on { border-color: var(--live-deep); background: var(--live); color: #fff; }
.pc-out-temp { display: grid; gap: 1px; justify-items: center; color: var(--text); }
.pc-out-temp b { font-family: var(--data); font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; letter-spacing: -.02em; }
.pc-out-temp span { font-family: var(--data); font-size: 8px; letter-spacing: .1em; color: var(--muted); }
.pc-out i {
  position: absolute; top: -5px; right: -5px; width: 14px; height: 14px; border-radius: 50%;
  border: 2.5px solid var(--panel); background: var(--ok, #1d7f45);
}
.pc-out.confirmed i { display: none; }
.pc-out.pending i { background: #9a6100; animation: pcPulse 1.1s ease-in-out infinite; }
.pc-out.failed { border-color: #b3261e; background: #fdeeec; color: #b3261e; }
.pc-out.failed i { background: #b3261e; }
@keyframes pcPulse { 0%, 100% { opacity: 1; } 50% { opacity: .3; } }
.pc-fail-note { margin: 0; padding: 0 13px 12px; font-size: 12.5px; color: #b3261e; }
.pc-progress { margin: 0; padding: 0 13px 12px; font-size: 12.5px; color: var(--muted); }
.pc-session-actions { display: flex; flex-wrap: wrap; gap: 8px; padding: 11px 13px; border-top: 1px solid var(--line); background: #f4f8f9; }
.pc-session-actions button { min-height: 38px; padding: 7px 12px; font-size: 13px; }

.pc-now { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 13px; }
.pc-now-lead { display: flex; align-items: baseline; gap: 11px; min-width: 0; }
.pc-now-temp { font-family: var(--data); font-size: 27px; font-weight: 700; line-height: 1; letter-spacing: -.03em; font-variant-numeric: tabular-nums; }
.pc-now-say strong { display: block; font-size: 13px; }
.pc-now-say span { display: block; font-size: 11.5px; color: var(--muted); }
.pc-minis { display: flex; gap: 6px; }
.pc-mini {
  display: grid; place-items: center; width: 32px; height: 32px; flex: 0 0 auto;
  border: 1.5px solid var(--line); border-radius: 9px; background: #f4f8f9; color: #6d8288;
}
.pc-mini svg { width: 19px; height: 19px; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.pc-mini.on { border-color: var(--live-deep); background: var(--live); color: #fff; }
.pc-next { position: relative; display: grid; grid-template-columns: 72px 1fr; gap: 12px; padding: 11px 13px; border-top: 1px solid var(--line); }
.pc-next.soon::before { content: ""; position: absolute; left: 0; top: 0; bottom: 0; width: 3px; background: var(--live); }
.pc-next.soon { background: #f2fafb; }
.pc-when { font-family: var(--data); font-size: 14px; font-weight: 700; font-variant-numeric: tabular-nums; }
.pc-rel { display: block; font-family: var(--data); font-size: 10px; color: var(--muted); }
.pc-next strong { display: block; font-size: 13px; }
.pc-tags { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 5px; }
.pc-tag { display: inline-flex; align-items: center; gap: 5px; padding: 3px 9px; border: 1px solid var(--line); border-radius: 999px; background: var(--panel); font-size: 11.5px; font-weight: 600; color: var(--muted); }
.pc-tag::before { content: ""; width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.pc-tag.up { border-color: var(--live-line); background: var(--live-soft); color: var(--live); }
.pc-after { margin: 5px 0 0; font-size: 11.5px; color: var(--muted); }
.pc-foot { padding: 10px 13px; border-top: 1px solid var(--line); background: #f4f8f9; font-size: 12px; color: var(--muted); }

@media (prefers-reduced-motion: no-preference) {
  [aria-pressed="true"] .jet-stream, .pc-mini.on .jet-stream, .pc-out.on .jet-stream { animation: pcThrust 1.5s ease-out infinite; }
  [aria-pressed="true"] .jet-stream:nth-of-type(2), .pc-mini.on .jet-stream:nth-of-type(2), .pc-out.on .jet-stream:nth-of-type(2) { animation-delay: .18s; }
  [aria-pressed="true"] .jet-stream:nth-of-type(3), .pc-mini.on .jet-stream:nth-of-type(3), .pc-out.on .jet-stream:nth-of-type(3) { animation-delay: .36s; }
  [aria-pressed="true"] .bub, .pc-mini.on .bub, .pc-out.on .bub { animation: pcRise 2.4s ease-in-out infinite; }
  [aria-pressed="true"] .bub:nth-of-type(2), .pc-mini.on .bub:nth-of-type(2), .pc-out.on .bub:nth-of-type(2) { animation-delay: .5s; }
  [aria-pressed="true"] .bub:nth-of-type(3), .pc-mini.on .bub:nth-of-type(3), .pc-out.on .bub:nth-of-type(3) { animation-delay: 1s; }
  [aria-pressed="true"] .bub:nth-of-type(4), .pc-mini.on .bub:nth-of-type(4), .pc-out.on .bub:nth-of-type(4) { animation-delay: 1.5s; }
  [aria-pressed="true"] .spin, .pc-mini.on .spin, .pc-out.on .spin { transform-box: view-box; transform-origin: 12px 12px; animation: pcSpin 2.6s linear infinite; }
  [aria-pressed="true"] .flame, .pc-mini.on .flame, .pc-out.on .flame { transform-box: view-box; transform-origin: 12px 21px; animation: pcFlicker 1.5s ease-in-out infinite; }
  [aria-pressed="true"] .pwr, .pc-mini.on .pwr, .pc-out.on .pwr { transform-box: view-box; transform-origin: 12px 12px; animation: pcBreathe 2.8s ease-in-out infinite; }
}
@keyframes pcThrust { 0% { opacity: .25; transform: translateX(-1.5px); } 45% { opacity: 1; transform: translateX(0); } 100% { opacity: .25; transform: translateX(1.5px); } }
@keyframes pcRise { 0% { opacity: 0; transform: translateY(2.5px); } 25% { opacity: 1; } 75% { opacity: .9; } 100% { opacity: 0; transform: translateY(-3.5px); } }
@keyframes pcSpin { to { transform: rotate(360deg); } }
@keyframes pcFlicker { 0%, 100% { transform: scale(1, 1); } 30% { transform: scale(1.07, .93); } 62% { transform: scale(.95, 1.06); } }
@keyframes pcBreathe { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: .58; transform: scale(1.07); } }
@media (max-width: 560px) {
  .pc-owner { grid-template-columns: 1fr; }
  .pc-row { gap: 6px; padding: 11px; }
  .pc-cap { height: 58px; }
  .pc-cap svg { width: 26px; height: 26px; }
}
/* Keep the single icon row at a 44px touch target on the narrowest phones. */
@media (max-width: 380px) {
  .pc-outcomes { grid-template-columns: repeat(3, 1fr); }
  .pc-row { gap: 4px; padding: 10px 8px; }
  .pc-temp { padding: 10px 8px 0; }
  .pc-cap svg { width: 24px; height: 24px; }
}
.visually-hidden {
  position: absolute !important;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
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
@media (max-width: 430px) {
  .topbar {
    flex-wrap: wrap;
  }
  .top-actions {
    width: 100%;
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    scroll-behavior: auto !important;
    transition-duration: 0s !important;
    animation-duration: 0s !important;
    animation-iteration-count: 1 !important;
  }
}
@media (min-width: 760px) {
  .app {
    padding: max(20px, env(safe-area-inset-top)) max(20px, env(safe-area-inset-right)) max(20px, env(safe-area-inset-bottom)) max(20px, env(safe-area-inset-left));
  }
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
      <div class="pc" id="poolControl"></div>
      <p class="visually-hidden" id="manualSessionStatus" role="status" aria-live="polite" aria-atomic="true"></p>
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
  <div class="toast" id="toast" role="status" aria-live="polite" aria-atomic="true"></div>
</main>

<script>
var pageMode = document.body.dataset.page || "dashboard";
var isHistoryPage = pageMode === "history";
var caps = ["power", "filter", "heater", "jets", "bubbles", "sanitizer"];
var capLabels = {power:"Power", filter:"Filter", heater:"Heater", jets:"Jets", bubbles:"Bubbles", sanitizer:"Sanitizer"};
var days = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
var activityPageSize = 12;
var activityKeys = ["events", "polls", "commands", "plan_executions", "heating_sessions"];
var state = {
  token: localStorage.getItem("poold.token") || "",
  status: null,
  poolControlRepresentation: null,
  manualSessionDraft: readManualSessionDraft(),
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
  settingsOpen: false,
  planView: "plans",
  editPlanId: null,
  activityView: "events",
  pending: false,
  pendingCount: 0
};

function $(id) { return document.getElementById(id); }
function qsa(selector) { return Array.prototype.slice.call(document.querySelectorAll(selector)); }
function boolText(value) { return value ? "On" : "Off"; }
function title(value) { return (value || "").replace(/_/g, " ").replace(/\b\w/g, function(c) { return c.toUpperCase(); }); }

function readManualSessionDraft() {
  try {
    var draft = JSON.parse(sessionStorage.getItem("poold.manualSessionDraft") || "null");
    return draft && draft.intended ? draft : null;
  } catch (_) {
    return null;
  }
}

function saveManualSessionDraft() {
  if (state.manualSessionDraft) {
    sessionStorage.setItem("poold.manualSessionDraft", JSON.stringify(state.manualSessionDraft));
  } else {
    sessionStorage.removeItem("poold.manualSessionDraft");
  }
}

function manualSessionObserved() {
  var representation = state.poolControlRepresentation;
  return representation && representation.observed ? representation.observed : null;
}

function manualSessionObservationIsStale(observed) {
  if (!observed || !observed.observed_at) return true;
  return Date.now() - new Date(observed.observed_at).getTime() > 90000;
}

function startManualSessionDraft() {
  var observed = manualSessionObserved();
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  if (!session && (!observed || !observed.connected || manualSessionObservationIsStale(observed))) return;
  var base = session ? session.intended : observed.state;
  state.manualSessionDraft = {
    duration: "30m",
    base: Object.assign({}, base),
    intended: Object.assign({}, base),
    expected_control_revision: state.poolControlRepresentation.control_revision,
    explicit: {},
    dependencies: {},
    dirty: false
  };
  if (session) state.manualSessionDraft.duration = session.duration;
  if (!session) state.manualSessionDraft.base_observation_id = observed.observation_id;
  saveManualSessionDraft();
  renderPoolControl();
}

function discardManualSessionDraft() {
  pendingDiscard = false;
  state.manualSessionDraft = null;
  saveManualSessionDraft();
  renderPoolControl();
}

function stageManualSessionCapability(cap) {
  var draft = state.manualSessionDraft;
  if (!draft) return;
  var intended = draft.intended;
  var value = !intended[cap];
  draft.explicit = draft.explicit || {};
  draft.dependencies = draft.dependencies || {};
  draft.explicit[cap] = true;
  delete draft.dependencies[cap];
  intended[cap] = value;
  if (cap !== "power" && value && !intended.power) {
    markManualSessionDependency(draft, "power", true, "feature_power");
  }
  if (cap === "heater" && value && !intended.filter) {
    markManualSessionDependency(draft, "filter", true, "heater_filter");
  }
  if (cap === "power" && !value) {
    ["filter", "heater", "jets", "bubbles"].forEach(function(field) {
      markManualSessionDependency(draft, field, false, "power_off");
    });
  }
  if (cap === "filter" && !value && intended.heater) {
    markManualSessionDependency(draft, "heater", false, "filter_off");
  }
  reconcileManualSessionProvenance(draft);
  draft.dirty = true;
  delete draft.idempotency_key;
  saveManualSessionDraft();
  renderControls();
}

function markManualSessionDependency(draft, field, value, reason) {
  if (draft.intended[field] === value) return;
  draft.intended[field] = value;
  draft.dependencies = draft.dependencies || {};
  draft.explicit = draft.explicit || {};
  draft.dependencies[field] = reason;
  delete draft.explicit[field];
}

function reconcileManualSessionProvenance(draft) {
  var base = draft.base || {};
  draft.explicit = draft.explicit || {};
  draft.dependencies = draft.dependencies || {};
  Object.keys(draft.dependencies).forEach(function(field) {
    var reason = draft.dependencies[field];
    if ((reason === "heater_filter" && !draft.intended.heater) ||
        (reason === "filter_off" && draft.intended.filter) ||
        (reason === "power_off" && draft.intended.power)) {
      draft.intended[field] = base[field];
      delete draft.dependencies[field];
    }
  });
  if (draft.dependencies.power === "feature_power" &&
      !draft.intended.filter && !draft.intended.heater &&
      !draft.intended.jets && !draft.intended.bubbles) {
    draft.intended.power = base.power;
    delete draft.dependencies.power;
  }
  ["power", "filter", "heater", "jets", "bubbles", "target_temp"].forEach(function(field) {
    if (draft.intended[field] === base[field]) {
      delete draft.explicit[field];
      delete draft.dependencies[field];
    } else if (draft.explicit[field]) {
      delete draft.dependencies[field];
    }
  });
}

function rebaseManualSessionDraft(current) {
  var draft = state.manualSessionDraft;
  if (!draft || !current) return;
  var currentBase = current.session ? current.session.intended :
    current.observed ? current.observed.state : null;
  if (!currentBase) return;
  var edited = {};
  Object.keys(draft.explicit || {}).forEach(function(field) {
    edited[field] = draft.intended[field];
  });
  draft.base = Object.assign({}, currentBase);
  draft.intended = Object.assign({}, currentBase, edited);
  draft.expected_control_revision = current.control_revision;
  if (current.session) {
    delete draft.base_observation_id;
  } else if (current.observed) {
    draft.base_observation_id = current.observed.observation_id;
  }
  draft.dependencies = {};
  if (draft.explicit.power && !draft.intended.power) {
    ["filter", "heater", "jets", "bubbles"].forEach(function(field) {
      if (draft.intended[field]) {
        draft.intended[field] = false;
        if (!draft.explicit[field]) draft.dependencies[field] = "power_off";
      }
    });
  }
  if (draft.explicit.filter && !draft.intended.filter && draft.intended.heater) {
    draft.intended.heater = false;
    if (!draft.explicit.heater) draft.dependencies.heater = "filter_off";
  }
  if ((draft.intended.filter || draft.intended.heater || draft.intended.jets || draft.intended.bubbles) &&
      !draft.intended.power) {
    draft.intended.power = true;
    if (!draft.explicit.power) draft.dependencies.power = "feature_power";
  }
  if (draft.intended.heater && !draft.intended.filter) {
    draft.intended.filter = true;
    if (!draft.explicit.filter) draft.dependencies.filter = "heater_filter";
  }
  draft.review_required = true;
  draft.dirty = true;
  delete draft.idempotency_key;
  saveManualSessionDraft();
}

function setBusy(value, message) {
  state.pendingCount = Math.max(0, state.pendingCount + (value ? 1 : -1));
  state.pending = state.pendingCount > 0;
  $("busy").textContent = state.pending ? "Working" : "Ready";
  $("busy").className = state.pending ? "badge warn" : "badge ok";
  if (value && message) toast(message, "busy");
}

function toast(message, kind) {
  var box = $("toast");
  box.textContent = message;
  box.style.background = kind === "bad" ? "#b42318" : kind === "ok" ? "#1d7f45" : "#172126";
  box.classList.add("show");
  clearTimeout(toast.timer);
  if (kind !== "busy") {
    toast.timer = setTimeout(function() { box.classList.remove("show"); }, 2800);
  }
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
        var detail = body && body.error;
        var message = detail && detail.message ? detail.message : detail || resp.status + " " + resp.statusText;
        var error = new Error(message);
        error.status = resp.status;
        error.detail = detail;
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
    loadStatus().then(loadPoolControl),
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

function loadPoolControl() {
  return api("/manual-session").then(function(representation) {
    var previous = state.poolControlRepresentation;
    state.poolControlRepresentation = representation;
    if (previous && previous.session && representation && !representation.session &&
        previous.session.expires_at &&
        new Date(previous.session.expires_at).getTime() <= Date.now()) {
      toast("Manual session ended. Automatic control resumed.", "ok");
    }
    if (state.manualSessionDraft && representation && !representation.session && representation.observed &&
        (!state.manualSessionDraft.expected_control_revision || !state.manualSessionDraft.base_observation_id)) {
      state.manualSessionDraft.expected_control_revision = representation.control_revision;
      state.manualSessionDraft.base_observation_id = representation.observed.observation_id;
      state.manualSessionDraft.base = Object.assign({}, representation.observed.state);
      state.manualSessionDraft.explicit = state.manualSessionDraft.explicit || {};
      state.manualSessionDraft.dependencies = state.manualSessionDraft.dependencies || {};
      saveManualSessionDraft();
    }
    if (representation && representation.session && representation.session.state === "applying") {
      scheduleManualSessionRefresh();
    } else if (representation && representation.session && representation.session.state === "active") {
      scheduleManualSessionClock();
    }
  }).catch(function(err) {
    state.poolControlRepresentation = null;
    toast("Pool control: " + err.message, "bad");
  });
}

var manualSessionRefreshTimer = null;
var manualSessionClockTimer = null;
function scheduleManualSessionRefresh() {
  clearTimeout(manualSessionRefreshTimer);
  manualSessionRefreshTimer = setTimeout(function() {
    loadPoolControl().then(function() {
      renderPoolControl();
    });
  }, 800);
}

function scheduleManualSessionClock() {
  clearTimeout(manualSessionClockTimer);
  manualSessionClockTimer = setTimeout(function() {
    loadPoolControl().then(function() {
      renderPoolControl();
    });
  }, 30000);
}

function manualSessionIdempotencyKey() {
  if (window.crypto && typeof window.crypto.randomUUID === "function") return window.crypto.randomUUID();
  return "manual-" + Date.now() + "-" + Math.random().toString(16).slice(2);
}

function manualSessionClearAttempt(revision) {
  var attempt = null;
  try {
    attempt = JSON.parse(sessionStorage.getItem("poold.manualSessionClear") || "null");
  } catch (_) {}
  if (!attempt || attempt.expected_control_revision !== revision || !attempt.idempotency_key) {
    attempt = {
      expected_control_revision: revision,
      idempotency_key: manualSessionIdempotencyKey()
    };
    sessionStorage.setItem("poold.manualSessionClear", JSON.stringify(attempt));
  }
  return attempt;
}

function manualSessionRetryAttempt(revision) {
  var attempt = null;
  try {
    attempt = JSON.parse(sessionStorage.getItem("poold.manualSessionRetry") || "null");
  } catch (_) {}
  if (!attempt || attempt.expected_control_revision !== revision || !attempt.idempotency_key) {
    attempt = {
      expected_control_revision: revision,
      idempotency_key: manualSessionIdempotencyKey()
    };
    sessionStorage.setItem("poold.manualSessionRetry", JSON.stringify(attempt));
  }
  return attempt;
}

function applyManualSessionDraft() {
  var draft = state.manualSessionDraft;
  if (!draft || state.pending) return;
  if (!draft.idempotency_key) {
    draft.idempotency_key = manualSessionIdempotencyKey();
    saveManualSessionDraft();
  }
  var body = {
    expected_control_revision: draft.expected_control_revision,
    base_observation_id: draft.base_observation_id,
    duration: draft.duration,
    intended: draft.intended
  };
  setBusy(true, "Applying Manual session");
  api("/manual-session", {
    method: "PUT",
    headers: {"Idempotency-Key": draft.idempotency_key},
    body: JSON.stringify(body)
  }).then(function(representation) {
    state.poolControlRepresentation = representation;
    state.manualSessionDraft = null;
    saveManualSessionDraft();
    toast("Manual session committed and applying.", "ok");
    scheduleManualSessionRefresh();
  }).catch(function(err) {
    if (err.detail && (err.detail.code === "control_changed" ||
        err.detail.code === "observed_state_changed") && err.detail.current) {
      state.poolControlRepresentation = err.detail.current;
      rebaseManualSessionDraft(err.detail.current);
      toast("Control changed. Your edits were rebased; review and Apply again.", "bad");
    } else {
      toast("Manual session: " + err.message, "bad");
    }
  }).finally(function() {
    setBusy(false);
    renderPoolControl();
  });
}

function endManualSession() {
  var representation = state.poolControlRepresentation;
  if (!representation || !representation.session || state.pending) return;
  var attempt = manualSessionClearAttempt(representation.control_revision);
  var body = {
    expected_control_revision: attempt.expected_control_revision
  };
  setBusy(true, "Returning to Automatic control");
  api("/manual-session", {
    method: "DELETE",
    headers: {"Idempotency-Key": attempt.idempotency_key},
    body: JSON.stringify(body)
  }).then(function(representation) {
    sessionStorage.removeItem("poold.manualSessionClear");
    state.poolControlRepresentation = representation;
    state.manualSessionDraft = null;
    saveManualSessionDraft();
    renderPoolControl();
    toast("Automatic control resumed.", "ok");
    scheduleManualSessionRefresh();
  }).catch(function(err) {
    toast("Manual session: " + err.message, "bad");
  }).finally(function() {
    setBusy(false);
    renderPoolControl();
  });
}

function retryManualSession() {
  var representation = state.poolControlRepresentation;
  if (!representation || !representation.session || state.pending) return;
  var attempt = manualSessionRetryAttempt(representation.control_revision);
  setBusy(true, "Retrying failed Manual session fields");
  api("/manual-session/retry", {
    method: "POST",
    headers: {"Idempotency-Key": attempt.idempotency_key},
    body: JSON.stringify({expected_control_revision: attempt.expected_control_revision})
  }).then(function(current) {
    sessionStorage.removeItem("poold.manualSessionRetry");
    state.poolControlRepresentation = current;
    toast("Manual session retry started.", "ok");
    scheduleManualSessionRefresh();
  }).catch(function(err) {
    toast("Manual session retry: " + err.message, "bad");
  }).finally(function() {
    setBusy(false);
    renderPoolControl();
  });
}

function selectAutomaticControl() {
  if (state.poolControlRepresentation && state.poolControlRepresentation.session) {
    endManualSession();
    return;
  }
  discardManualSessionDraft();
}


function loadPlans() {
  return api("/plans").then(function(data) {
    state.plans = data.plans || [];
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
  renderPoolControl();
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
  renderPoolControl();
  renderTimeline();
  renderActivity();
}

function renderStatus() {
  var status = state.status || {};
  var unit = status.unit || "°C";
  var current = status.current_temp == null ? "--" : status.current_temp + unit;
  $("currentTemp").textContent = current;
  $("targetTemp").textContent = "Target " + (status.preset_temp || "--") + unit;
  $("observedAt").textContent = status.observed_at ? formatTime(status.observed_at) : "--";
  $("errorCode").textContent = status.error_code || "None";
  $("subline").textContent = status.connected ? "Connected " + formatAge(status.observed_at) : "Pool daemon";
  $("connected").textContent = status.connected ? "Connected" : state.token ? "Disconnected" : "Token";
  $("connected").className = status.connected ? "badge ok" : state.token ? "badge bad" : "badge warn";
  if (state.poolControlRepresentation && state.poolControlRepresentation.control === "manual") {
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

var manualCaps = ["power", "filter", "heater", "jets", "bubbles"];
var manualCapSubtitles = {
  power: "Master supply for everything below",
  filter: "Circulation pump",
  heater: "Needs the filter running",
  jets: "Massage jets",
  bubbles: "Air blower"
};
var manualCapIcons = {
  power: '<svg viewBox="0 0 24 24"><g class="pwr"><path d="M12 3v9"/><path d="M6.6 6.6a7.6 7.6 0 1 0 10.8 0"/></g></svg>',
  filter: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="1.7"/><g class="spin"><path d="M12 10.3c2.3-1.5 4.1-1.7 5.5-.7"/><path d="M13.5 12.9c.5 2.7-.1 4.4-1.5 5.3"/><path d="M10.5 12.9c-2.9-1.2-4-2.7-4-4.6"/></g></svg>',
  heater: '<svg viewBox="0 0 24 24"><g class="flame"><path d="M12 21c3.6 0 6-2.3 6-5.4 0-3.6-3-4.6-2.4-8.2C13.3 8.5 13 10.6 13 12c-1.3-.6-1.7-2.4-1.4-4.4C9.1 9.4 6 11.7 6 15.6 6 18.7 8.4 21 12 21Z"/></g></svg>',
  jets: '<svg viewBox="0 0 24 24"><path class="jet-stream" d="M3 7h9"/><path class="jet-stream" d="M9.6 4.6 12 7l-2.4 2.4"/><path class="jet-stream" d="M3 12h13"/><path class="jet-stream" d="M13.6 9.6 16 12l-2.4 2.4"/><path class="jet-stream" d="M3 17h9"/><path class="jet-stream" d="M9.6 14.6 12 17l-2.4 2.4"/></svg>',
  bubbles: '<svg viewBox="0 0 24 24"><path d="M2.8 5.2c1.6-1.4 3.2-1.4 4.8 0s3.2 1.4 4.8 0 3.2-1.4 4.8 0 3.2 1.4 4.8 0"/><circle class="bub" cx="7.4" cy="17.6" r="2.3"/><circle class="bub" cx="13.6" cy="18.6" r="1.5"/><circle class="bub" cx="15.8" cy="12.6" r="2.7"/><circle class="bub" cx="8.8" cy="10.6" r="1.7"/></svg>'
};
var manualDurations = [["10m", "10 min"], ["30m", "30 min"], ["60m", "60 min"], ["2h", "2 hours"], ["until_off", "Until off"]];
var pendingDiscard = false;

function durationLabel(key) {
  for (var i = 0; i < manualDurations.length; i++) {
    if (manualDurations[i][0] === key) return manualDurations[i][1];
  }
  return key;
}

function manualBaseState() {
  var draft = state.manualSessionDraft;
  if (draft) return draft.base || {};
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  if (session) return session.intended || {};
  var observed = manualSessionObserved();
  return observed ? observed.state : {};
}

function manualDisplayedState() {
  var draft = state.manualSessionDraft;
  if (draft) return draft.intended;
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  if (session) return session.intended;
  var observed = manualSessionObserved();
  return observed ? observed.state : {};
}

function manualChangedFields() {
  var draft = state.manualSessionDraft;
  if (!draft) return [];
  var base = draft.base || {};
  return manualCaps.concat(["target_temp"]).filter(function(field) {
    return draft.intended[field] !== base[field];
  });
}

function ownershipMarkup() {
  var draft = state.manualSessionDraft;
  var observed = manualSessionObserved();
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var manual = !!draft || !!session;
  var manualBlocked = !!draft || (!session &&
    (!observed || !observed.connected || manualSessionObservationIsStale(observed)));
  var autoIcon = '<svg viewBox="0 0 24 24"><path d="M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M18.4 5.6l-2.1 2.1M7.7 16.3l-2.1 2.1"/><circle cx="12" cy="12" r="3.6"/></svg>';
  var handIcon = '<svg viewBox="0 0 24 24"><path d="M9 11V4.8a1.4 1.4 0 0 1 2.8 0V11"/><path d="M11.8 10.4V3.6a1.4 1.4 0 0 1 2.8 0V11"/><path d="M14.6 11V5.8a1.4 1.4 0 0 1 2.8 0V14"/><path d="M9 11V9.2a1.4 1.4 0 0 0-2.8 0v5.5c0 3.4 2.4 6.3 6 6.3s5.2-2.4 5.2-5.6"/></svg>';
  return '<div class="pc-owner">' +
    '<button class="pc-own auto" id="automaticControl" aria-pressed="' + (!manual) + '"' + (state.pending ? " disabled" : "") + ">" +
      '<span class="pc-own-head">' + autoIcon + '<span class="pc-own-name">AUTOMATIC</span></span>' +
      "<p>Your schedules run the pool. Controls stay locked.</p>" +
      '<span class="pc-own-flag">' + (manual ? "TAP TO HAND BACK" : "IN CONTROL") + "</span></button>" +
    '<button class="pc-own man" id="manualControl" aria-pressed="' + manual + '"' + (manualBlocked ? " disabled" : "") + ">" +
      '<span class="pc-own-head">' + handIcon + '<span class="pc-own-name">MANUAL</span></span>' +
      "<p>You set the pool yourself and hold it for a chosen time.</p>" +
      '<span class="pc-own-flag">' + (manual ? "IN CONTROL" : "TAP TO TAKE OVER") + "</span></button></div>";
}

function equipmentMarkup() {
  var draft = state.manualSessionDraft;
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var observed = manualSessionObserved();
  var shown = manualDisplayedState();
  var base = manualBaseState();
  var water = currentWaterTemp();
  var say = shown.heater ?
    (water != null && water < shown.target_temp ? "Heating to " + shown.target_temp + "°" : "At temperature") :
    "Heating is off";

  var tiles = manualCaps.map(function(cap) {
    var on = !!shown[cap];
    var explicit = !!(draft && draft.explicit && draft.explicit[cap]);
    var dependency = !!(draft && draft.dependencies && draft.dependencies[cap]);
    var progress = session && session.outcomes && session.outcomes[cap] ? ", " + session.outcomes[cap].state : "";
    return '<button class="pc-cap' + (dependency ? " dep" : "") + '" data-manual-cap="' + cap + '"' +
      ' aria-pressed="' + on + '"' +
      ' title="' + capLabels[cap] + " · " + (on ? "on" : "off") + '"' +
      ' aria-label="' + capLabels[cap] + " " + (on ? "on" : "off") + progress + ", " + manualCapSubtitles[cap] + '"' +
      (draft ? "" : " disabled") + '><span aria-hidden="true">' + manualCapIcons[cap] + "</span>" +
      (explicit || dependency ? '<span class="pc-flag" aria-hidden="true"></span>' : "") + "</button>";
  }).join("");

  var why = "";
  if (draft && draft.dependencies) {
    var reasons = Object.keys(draft.dependencies).map(function(field) {
      var reason = draft.dependencies[field];
      if (reason === "heater_filter") return "Filter switched on — the heater needs it";
      if (reason === "feature_power") return "Power switched on — the feature needs it";
      if (reason === "power_off") return capLabels[field] + " switched off — it follows the power";
      if (reason === "filter_off") return "Heater switched off — it needs the filter";
      return capLabels[field] + " changed automatically";
    });
    var unique = reasons.filter(function(text, index) { return reasons.indexOf(text) === index; });
    if (unique.length) why = '<p class="pc-why">' + escapeHTML(unique.join(" · ")) + "</p>";
  }

  return '<div class="pc-card"><div class="pc-head"><h3>EQUIPMENT</h3><span>' +
    (draft ? "Tap to switch" : "Read-only") + "</span></div>" +
    (draft ? "" : '<div class="pc-lock">' + (session ?
      "Locked while your session holds these settings — use Change settings below." :
      "Locked because your schedules are in charge. Take over with Manual to change anything.") + "</div>") +
    '<div class="pc-temp"><div class="pc-big">' + (water == null ? "--" : water) + "°<small>now</small></div>" +
    '<div class="pc-temp-say"><strong>' + say + "</strong><span>" + observedAgeLabel(observed) + "</span></div></div>" +
    '<div class="pc-row">' + tiles + "</div>" + why + "</div>";
}

function currentWaterTemp() {
  var status = state.status;
  return status && status.current_temp != null ? status.current_temp : null;
}

function observedAgeLabel(observed) {
  if (!observed || !observed.observed_at) return "No reading yet";
  var seconds = Math.max(0, Math.round((Date.now() - new Date(observed.observed_at).getTime()) / 1000));
  if (seconds < 60) return "Read just now";
  var minutes = Math.round(seconds / 60);
  return "Read " + minutes + " min ago";
}

function targetMarkup() {
  var draft = state.manualSessionDraft;
  var shown = manualDisplayedState();
  if (!shown.heater) return "";
  var base = manualBaseState();
  var changed = !!draft && draft.intended.target_temp !== base.target_temp;
  var observed = manualSessionObserved();
  var water = currentWaterTemp();
  var note = changed ? "Was " + base.target_temp + "°" :
    water != null && water < shown.target_temp ? (shown.target_temp - water) + "° to go from where the water is now" :
    "The water is already there";
  var control = draft ?
    '<div class="pc-step"><button data-manual-step="-1" aria-label="Lower target">−</button>' +
      '<span class="pc-val' + (changed ? " changed" : "") + '"><b>' + shown.target_temp + '°</b><span>TARGET</span></span>' +
      '<button data-manual-step="1" aria-label="Raise target">+</button></div>' :
    '<span class="pc-val"><b>' + shown.target_temp + '°</b><span>TARGET</span></span>';
  return '<div class="pc-card"><div class="pc-head"><h3>TARGET TEMPERATURE</h3><span>' +
    (draft ? "The heater stops here" : "Held by your session") + "</span></div>" +
    '<div class="pc-target"><span class="pc-target-note' + (changed ? " changed" : "") + '">' + note + "</span>" +
    control + "</div></div>";
}

function applyMarkup() {
  var draft = state.manualSessionDraft;
  if (!draft) return "";
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var base = draft.base || {};
  var changes = manualChangedFields();
  if (pendingDiscard) {
    return '<div class="pc-apply"><div class="pc-apply-top"><strong>' +
      (changes.length === 1 ? "Throw away this change?" : "Throw away these " + changes.length + " changes?") +
      "</strong></div>" +
      '<p class="pc-note">They were never sent, so the pool is unaffected either way.</p>' +
      '<div class="pc-confirm"><button data-manual-act="keep">Keep editing</button>' +
      '<button data-manual-act="discard">Throw away</button></div></div>';
  }
  var chips = changes.map(function(field) {
    if (field === "target_temp") {
      return '<span class="pc-chip">Target ' + base[field] + "° <i>→</i> " + draft.intended[field] + "°</span>";
    }
    return '<span class="pc-chip">' + capLabels[field] + " " + (base[field] ? "on" : "off") +
      " <i>→</i> " + (draft.intended[field] ? "on" : "off") + "</span>";
  }).join("");
  var canApply = (changes.length > 0 || !!session) && !state.pending;
  var title = changes.length ? changes.length + " change" + (changes.length !== 1 ? "s" : "") + " ready" :
    session ? "No changes — you can still restart the clock" : "No changes yet";
  var label = draft.review_required ? "CONTROL CHANGED — REVIEW, THEN TAP A TIME" :
    canApply ? "TAP A TIME TO APPLY" :
    state.pending ? "WORKING…" : "TAP AN ICON ABOVE TO MAKE A CHANGE";
  return '<div class="pc-apply"><div class="pc-apply-top"><strong>' + title + "</strong>" +
    '<span>' + (session ? "EDITING SESSION" : "NEW SESSION") + "</span></div>" +
    (changes.length ? '<div class="pc-chips">' + chips + "</div>" : "") +
    '<p class="pc-note">The pool has not been told anything yet.' +
    (session ? " Applying restarts the clock." : "") + "</p>" +
    '<div><span class="pc-label">' + label + '</span><div class="pc-durations">' +
    manualDurations.map(function(item) {
      return '<button data-manual-duration="' + item[0] + '"' + (canApply ? "" : " disabled") + ">" + item[1] + "</button>";
    }).join("") + "</div></div></div>";
}

function sessionMarkup() {
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  if (!session) return "";
  var draft = state.manualSessionDraft;
  var word = session.state === "applying" ? "APPLYING" : session.state === "degraded" ? "NEEDS ATTENTION" : "HOLDING";
  var sub = session.state === "applying" ? "Sending your settings to the pool" :
    session.state === "degraded" ? "Part of your session is not running" :
    "The pool is held exactly as you set it";
  var outcomeState = function(field) {
    return ((session.outcomes && session.outcomes[field]) || {state: "pending"}).state;
  };
  var stateWord = {confirmed: "confirmed", pending: "still sending", failed: "not confirmed"};
  var tiles = manualCaps.map(function(cap) {
    var on = !!session.intended[cap];
    var status = outcomeState(cap);
    return '<div class="pc-out ' + status + (on ? " on" : "") + '" role="img" title="' +
      capLabels[cap] + " · " + (on ? "on" : "off") + " · " + stateWord[status] + '"' +
      ' aria-label="' + capLabels[cap] + " " + (on ? "on" : "off") + ", " + stateWord[status] + '">' +
      manualCapIcons[cap] + "<i></i></div>";
  }).join("");
  var targetStatus = outcomeState("target_temp");
  tiles += '<div class="pc-out ' + targetStatus + '" role="img" title="Target ' + session.intended.target_temp +
    "° · " + stateWord[targetStatus] + '" aria-label="Target temperature ' + session.intended.target_temp +
    " degrees, " + stateWord[targetStatus] + '"><span class="pc-out-temp"><b>' + session.intended.target_temp +
    '°</b><span>TARGET</span></span><i></i></div>';

  var note = "";
  if (session.state === "degraded") {
    var failures = manualCaps.concat(["target_temp"]).filter(function(field) {
      return outcomeState(field) === "failed";
    }).map(function(field) {
      var outcome = session.outcomes[field];
      var label = capLabels[field] || "Target temperature";
      return label + " — " + (outcome.message || outcome.code || "not confirmed by the pool");
    });
    if (failures.length) note = '<p class="pc-fail-note">' + escapeHTML(failures.join(" · ")) + "</p>";
  } else if (session.state === "applying") {
    var confirmed = manualCaps.concat(["target_temp"]).filter(function(field) {
      return outcomeState(field) === "confirmed";
    }).length;
    note = '<p class="pc-progress">' + confirmed + " of 6 confirmed by the pool.</p>";
  }

  var clock = session.expires_at ? manualSessionClock(session.expires_at) : "—";
  var clockCap = session.expires_at ? "LEFT" : "UNTIL YOU STOP IT";
  return '<div class="pc-card"><div class="pc-head"><h3>YOUR SESSION</h3><span>' +
    durationLabel(session.duration) + "</span></div>" +
    '<div class="pc-session-top"><div class="pc-state ' + session.state + '"><b><i></i>' + word + "</b>" +
    '<span>' + sub + "</span></div>" +
    '<div class="pc-clock" id="manualSessionClock">' + clock + "<small>" + clockCap + "</small></div></div>" +
    '<div class="pc-outcomes">' + tiles + "</div>" + note +
    '<div class="pc-session-actions">' +
      (session.state === "degraded" ? '<button class="primary" data-manual-act="retry"' + (state.pending ? " disabled" : "") + ">Try the failed fields again</button>" : "") +
      (draft ? '<button data-manual-act="stopEdit">Stop editing</button>' :
        '<button data-manual-act="edit">Change settings</button>') +
      '<button data-manual-act="automatic"' + (state.pending ? " disabled" : "") + ">Hand back to Automatic</button>" +
    "</div></div>";
}

function manualSessionClock(expiresAt) {
  var seconds = Math.max(0, Math.ceil((new Date(expiresAt).getTime() - Date.now()) / 1000));
  var hours = Math.floor(seconds / 3600);
  var minutes = Math.floor((seconds % 3600) / 60);
  var rest = seconds % 60;
  if (hours > 0) return hours + ":" + pad2(minutes) + ":" + pad2(rest);
  return pad2(minutes) + ":" + pad2(rest);
}

function renderPoolControl() {
  var draft = state.manualSessionDraft;
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var observed = manualSessionObserved();
  var handsOn = !!draft || !!session;
  var html = ownershipMarkup();
  if (observed && !observed.connected) {
    html += '<div class="pc-card"><div class="pc-lock">The pool is not answering. ' +
      (session ? "The session you applied still stands and you can hand it back." :
        "You cannot start a manual session until a fresh reading arrives.") + "</div></div>";
  }
  html += handsOn ?
    equipmentMarkup() + targetMarkup() + applyMarkup() + sessionMarkup() :
    automaticSummaryMarkup();
  var root = $("poolControl");
  var restore = focusKeyWithin(root, document.activeElement);
  root.innerHTML = html;
  if (restore) {
    var target = root.querySelector(restore);
    if (target && !target.disabled) target.focus();
  }
  bindPoolControl();
  startManualSessionTick();
  announceManualSession(manualSessionAnnouncement());
}

// Re-rendering replaces the control markup wholesale, which would otherwise drop
// keyboard focus on every poll. Remember which control was focused and restore it.
function focusKeyWithin(root, element) {
  if (!element || !root.contains(element)) return null;
  if (element.id) return "#" + element.id;
  var attributes = ["data-manual-cap", "data-manual-duration", "data-manual-act", "data-manual-step"];
  for (var i = 0; i < attributes.length; i++) {
    var value = element.getAttribute(attributes[i]);
    if (value) return "[" + attributes[i] + '="' + value + '"]';
  }
  return null;
}

var manualSessionTickTimer = null;
function startManualSessionTick() {
  clearInterval(manualSessionTickTimer);
  manualSessionTickTimer = null;
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  if (!session || !session.expires_at) return;
  manualSessionTickTimer = setInterval(function() {
    var element = $("manualSessionClock");
    if (!element || !element.firstChild) {
      clearInterval(manualSessionTickTimer);
      manualSessionTickTimer = null;
      return;
    }
    element.firstChild.nodeValue = manualSessionClock(session.expires_at);
  }, 1000);
}

function manualSessionAnnouncement() {
  var draft = state.manualSessionDraft;
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var observed = manualSessionObserved();
  if (draft && draft.review_required) return "Control changed. Review this rebased draft, then apply again.";
  if (draft) {
    var count = manualChangedFields().length;
    return "Draft saved in this tab: " + count + " change" + (count === 1 ? "" : "s") +
      ". Nothing is sent until you pick how long.";
  }
  if (session && session.state === "applying") {
    var confirmed = Object.keys(session.outcomes || {}).filter(function(field) {
      return session.outcomes[field].state === "confirmed";
    }).length;
    return "Applying manual session: " + confirmed + " of 6 fields confirmed.";
  }
  if (session && session.state === "active") {
    return "Manual session active. " + (session.expires_at ? manualSessionRemaining(session.expires_at) : "Until turned off.");
  }
  if (session && session.state === "degraded") {
    var failed = Object.keys(session.outcomes || {}).filter(function(field) {
      return session.outcomes[field].state === "failed";
    }).map(function(field) { return capLabels[field] || "Target temperature"; });
    return "Manual session degraded. Failed: " + failed.join(", ") + ".";
  }
  if (observed && !observed.connected) return "Pool offline. Starting a manual session is unavailable.";
  if (observed && manualSessionObservationIsStale(observed)) return "Pool observation is stale. Refresh before starting a manual session.";
  return "Schedules and reconciliation govern the pool.";
}

function bindPoolControl() {
  var manual = $("manualControl");
  if (manual) manual.onclick = startManualSessionDraft;
  var automatic = $("automaticControl");
  if (automatic) automatic.onclick = function() {
    if (state.manualSessionDraft && manualChangedFields().length) {
      pendingDiscard = true;
      renderPoolControl();
      return;
    }
    selectAutomaticControl();
  };
  qsa("[data-manual-cap]").forEach(function(button) {
    button.onclick = function() { stageManualSessionCapability(button.dataset.manualCap); };
  });
  qsa("[data-manual-step]").forEach(function(button) {
    button.onclick = function() { stepManualSessionTarget(Number(button.dataset.manualStep)); };
  });
  qsa("[data-manual-duration]").forEach(function(button) {
    button.onclick = function() { applyManualSessionFor(button.dataset.manualDuration); };
  });
  var actions = {
    retry: retryManualSession,
    edit: startManualSessionDraft,
    automatic: selectAutomaticControl,
    keep: function() { pendingDiscard = false; renderPoolControl(); },
    discard: function() {
      pendingDiscard = false;
      state.manualSessionDraft = null;
      saveManualSessionDraft();
      renderPoolControl();
    },
    stopEdit: function() {
      if (manualChangedFields().length) { pendingDiscard = true; renderPoolControl(); return; }
      state.manualSessionDraft = null;
      saveManualSessionDraft();
      renderPoolControl();
    }
  };
  qsa("[data-manual-act]").forEach(function(button) {
    button.onclick = actions[button.dataset.manualAct];
  });
}

function stepManualSessionTarget(delta) {
  var draft = state.manualSessionDraft;
  if (!draft) return;
  var next = Math.min(40, Math.max(10, Number(draft.intended.target_temp) + delta));
  if (next === draft.intended.target_temp) return;
  draft.intended.target_temp = next;
  draft.explicit = draft.explicit || {};
  draft.explicit.target_temp = true;
  reconcileManualSessionProvenance(draft);
  draft.dirty = true;
  delete draft.idempotency_key;
  saveManualSessionDraft();
  renderPoolControl();
}

function applyManualSessionFor(duration) {
  var draft = state.manualSessionDraft;
  if (!draft || state.pending) return;
  if (draft.duration !== duration) {
    draft.duration = duration;
    delete draft.idempotency_key;
  }
  draft.dirty = true;
  saveManualSessionDraft();
  applyManualSessionDraft();
}

function planForecast(limit) {
  var plans = (state.plans || []).filter(function(plan) { return plan.enabled; });
  var observed = manualSessionObserved();
  var start = observed && observed.state ? Object.assign({}, observed.state) : {};
  var events = [];
  var now = new Date();
  var horizon = 3 * 24 * 60;
  plans.forEach(function(plan) {
    if (plan.type === "time_window" && plan.from && plan.to) {
      addWindowEvents(plan, now, horizon, events);
    } else if (plan.type === "ready_by") {
      addReadyEvents(plan, now, horizon, events);
    }
  });
  events.sort(function(a, b) { return a.at - b.at; });
  var forecast = [];
  var running = Object.assign({}, start);
  for (var i = 0; i < events.length && forecast.length < limit; i++) {
    var event = events[i];
    var changes = [];
    Object.keys(event.to).forEach(function(field) {
      if (running[field] !== event.to[field]) changes.push({cap: field, on: event.to[field]});
      running[field] = event.to[field];
    });
    if (event.target != null && running.target_temp !== event.target) {
      changes.push({cap: "target_temp", value: event.target});
      running.target_temp = event.target;
    }
    if (!changes.length) continue;
    forecast.push({
      name: event.name,
      at: event.at,
      changes: changes,
      power: running.power !== false,
      running: manualCaps.filter(function(cap) { return cap !== "power" && running[cap]; }),
      target: running.target_temp
    });
  }
  return forecast;
}

function addWindowEvents(plan, now, horizon, events) {
  var caps = manualCaps.indexOf(plan.capability) >= 0 ? [plan.capability] : [];
  if (!caps.length) return;
  for (var dayOffset = 0; dayOffset <= 3; dayOffset++) {
    var day = new Date(now.getFullYear(), now.getMonth(), now.getDate() + dayOffset);
    if (!planDayAllowed(plan, day)) continue;
    [[plan.from, true], [plan.to, false]].forEach(function(pair) {
      var at = clockOnDay(day, pair[0]);
      if (!at) return;
      var minutes = (at.getTime() - now.getTime()) / 60000;
      if (minutes <= 0 || minutes > horizon) return;
      var to = {};
      to[plan.capability] = pair[1];
      if (pair[1]) to.power = true;
      if (pair[1] && plan.capability === "heater") to.filter = true;
      events.push({
        at: at,
        name: (plan.name || title(plan.capability)) + (pair[1] ? " starts" : " ends"),
        to: to,
        target: pair[1] && plan.target_temp ? plan.target_temp : null
      });
    });
  }
}

function addReadyEvents(plan, now, horizon, events) {
  var times = [];
  if (plan.at) {
    times.push(new Date(plan.at));
  } else if (plan.cron) {
    var fields = String(plan.cron).trim().split(/\s+/);
    if (fields.length === 5 && isSimpleCronNumber(fields[0]) && isSimpleCronNumber(fields[1])) {
      for (var dayOffset = 0; dayOffset <= 3; dayOffset++) {
        var day = new Date(now.getFullYear(), now.getMonth(), now.getDate() + dayOffset);
        if (fields[4] !== "*" && !cronDayAllowed(fields[4], day.getDay())) continue;
        times.push(new Date(day.getFullYear(), day.getMonth(), day.getDate(), Number(fields[1]), Number(fields[0])));
      }
    }
  }
  times.forEach(function(at) {
    var minutes = (at.getTime() - now.getTime()) / 60000;
    if (minutes <= 0 || minutes > horizon) return;
    events.push({
      at: at,
      name: plan.name || "Ready by",
      to: {power: true, filter: true, heater: false},
      target: plan.target_temp || null
    });
  });
}

function planDayAllowed(plan, day) {
  if (!plan.days || !plan.days.length) return true;
  return plan.days.indexOf(days[(day.getDay() + 6) % 7]) >= 0;
}

function cronDayAllowed(field, weekday) {
  return String(field).split(",").some(function(part) {
    return Number(part) === weekday || (weekday === 0 && Number(part) === 7);
  });
}

function clockOnDay(day, clock) {
  var parts = String(clock || "").split(":");
  if (parts.length < 2) return null;
  var hour = Number(parts[0]);
  var minute = Number(parts[1]);
  if (!Number.isInteger(hour) || !Number.isInteger(minute)) return null;
  return new Date(day.getFullYear(), day.getMonth(), day.getDate(), hour, minute);
}

function relativeMinutes(at) {
  var minutes = Math.max(0, Math.round((at.getTime() - Date.now()) / 60000));
  var hours = Math.floor(minutes / 60);
  var rest = minutes % 60;
  if (hours && rest) return "in " + hours + " h " + rest + " min";
  if (hours) return "in " + hours + " h";
  return "in " + rest + " min";
}

function automaticSummaryMarkup() {
  var observed = manualSessionObserved();
  var shown = observed && observed.state ? observed.state : {};
  var water = currentWaterTemp();
  var say;
  if (!shown.power) say = {line: "Everything is off", sub: "The pool is powered down"};
  else if (shown.heater) say = {line: "Heating to " + shown.target_temp + "°",
    sub: water != null && water < shown.target_temp ? (shown.target_temp - water) + "° to go" : "Nearly there"};
  else {
    var extras = ["jets", "bubbles"].filter(function(cap) { return shown[cap]; })
      .map(function(cap) { return capLabels[cap].toLowerCase(); });
    if (extras.length) say = {line: extras.join(" and ") + " running", sub: "Not heating · target " + shown.target_temp + "°"};
    else if (shown.filter) say = {line: "Circulating, not heating", sub: "Target " + shown.target_temp + "° when the schedule calls for it"};
    else say = {line: "Powered, everything idle", sub: "Target " + shown.target_temp + "°"};
  }

  var minis = manualCaps.map(function(cap) {
    var on = !!shown[cap];
    return '<div class="pc-mini' + (on ? " on" : "") + '" role="img" title="' + capLabels[cap] + " · " + (on ? "on" : "off") +
      '" aria-label="' + capLabels[cap] + " " + (on ? "on" : "off") + '">' + manualCapIcons[cap] + "</div>";
  }).join("");

  var forecast = planForecast(3);
  var next = forecast.length ? forecast.map(function(item, index) {
    var tags = item.changes.map(function(change) {
      if (change.cap === "target_temp") return '<span class="pc-tag up">Target ' + change.value + "°</span>";
      return '<span class="pc-tag' + (change.on ? " up" : "") + '">' + capLabels[change.cap] + " " + (change.on ? "on" : "off") + "</span>";
    }).join("");
    var after = !item.power ? "Nothing running after this" :
      item.running.length ? "Then running: " + item.running.map(function(cap) { return capLabels[cap]; }).join(", ") +
        (item.target != null ? " · target " + item.target + "°" : "") :
      "Then idle" + (item.target != null ? " · target " + item.target + "°" : "");
    return '<div class="pc-next' + (index === 0 ? " soon" : "") + '"><div><span class="pc-when">' +
      pad2(item.at.getHours()) + ":" + pad2(item.at.getMinutes()) + '</span><span class="pc-rel">' +
      relativeMinutes(item.at) + "</span></div><div><strong>" + escapeHTML(item.name) + "</strong>" +
      '<div class="pc-tags">' + tags + '</div><p class="pc-after">' + after + "</p></div></div>";
  }).join("") : '<div class="pc-next"><div><span class="pc-when">—</span></div><div><strong>Nothing scheduled</strong>' +
    '<p class="pc-after">No enabled plan changes the pool in the next three days.</p></div></div>';

  return '<div class="pc-card"><div class="pc-head"><h3>RIGHT NOW</h3><span>' +
    observedAgeLabel(observed) + "</span></div>" +
    '<div class="pc-now"><div class="pc-now-lead"><span class="pc-now-temp">' + (water == null ? "--" : water) + "°</span>" +
    '<span class="pc-now-say"><strong>' + say.line + "</strong><span>" + say.sub + "</span></span></div>" +
    '<div class="pc-minis">' + minis + "</div></div>" +
    '<div class="pc-head"><h3>WHAT HAPPENS NEXT</h3><span>From your schedules</span></div>' + next +
    '<div class="pc-foot">Switch to Manual to change any of this yourself.</div></div>';
}

var lastManualSessionAnnouncement = "";
function announceManualSession(message) {
  if (!message || message === lastManualSessionAnnouncement) return;
  lastManualSessionAnnouncement = message;
  $("manualSessionStatus").textContent = message;
}

function renderControls() {
  renderPoolControl();
}

function manualSessionRemaining(expiresAt) {
  var seconds = Math.max(0, Math.ceil((new Date(expiresAt).getTime() - Date.now()) / 1000));
  if (seconds < 60) return seconds + "s remaining.";
  var minutes = Math.ceil(seconds / 60);
  if (minutes < 60) return minutes + "m remaining.";
  var hours = Math.floor(minutes / 60);
  var remainder = minutes % 60;
  return hours + "h" + (remainder ? " " + remainder + "m" : "") + " remaining.";
}

function renderPlans() {
  qsa("[data-view]").forEach(function(button) {
    button.classList.toggle("active", button.dataset.view === state.planView);
  });
  var view = $("plansView");
  view.innerHTML = "";
  if (state.planView === "edit") {
    var plan = state.plans.find(function(p) { return p.id === state.editPlanId; });
    if (plan) return renderEditForm(view, plan);
    state.planView = "plans";
  }
  if (state.planView === "plans") return renderPlanList(view);
  if (state.planView === "ready") return renderReadyForm(view);
  renderWindowForm(view);
}

function renderPlanList(view) {
  var visiblePlans = state.plans;
  if (!visiblePlans.length) {
    view.innerHTML = "<p class=\"muted\">No plans</p>";
    return;
  }
  var list = document.createElement("div");
  list.className = "plan-list";
  visiblePlans.forEach(function(plan) {
    var item = document.createElement("div");
    item.className = "plan";
    item.innerHTML = "<div class=\"plan-main\"><div><h3>" + escapeHTML(plan.name || plan.id) + "</h3><p class=\"muted\">" + escapeHTML(describePlan(plan)) + "</p></div><span class=\"badge " + (plan.enabled ? "ok" : "") + "\">" + (plan.enabled ? "Active" : "Paused") + "</span></div><div class=\"plan-actions\"><label>Active <select data-active=\"" + escapeHTML(plan.id) + "\"><option value=\"true\"" + (plan.enabled ? " selected" : "") + ">Yes</option><option value=\"false\"" + (!plan.enabled ? " selected" : "") + ">No</option></select></label><button data-edit=\"" + escapeHTML(plan.id) + "\">Edit</button></div>";
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
  qsa("[data-edit]").forEach(function(button) {
    button.onclick = function() {
      state.editPlanId = button.dataset.edit;
      state.planView = "edit";
      renderPlans();
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

function renderEditForm(view, plan) {
  if (plan.type === "ready_by") return renderEditReadyForm(view, plan);
  if (plan.type === "time_window") return renderEditWindowForm(view, plan);
  view.innerHTML = "<p class=\"muted\">This plan type cannot be edited here.</p><button id=\"editBack\">Back</button>";
  $("editBack").onclick = function() { state.planView = "plans"; renderPlans(); };
}

function renderEditReadyForm(view, plan) {
  var isRepeating = !!plan.cron;
  var atValue = "";
  var timeValue = "08:30";
  var allDaysSelected = true;
  var selectedDayNames = [];
  if (!isRepeating && plan.at) {
    atValue = localDateTime(new Date(plan.at));
  }
  if (isRepeating && plan.cron) {
    var cronFields = plan.cron.trim().split(/\s+/);
    if (cronFields.length >= 2) {
      timeValue = pad2(Number(cronFields[1])) + ":" + pad2(Number(cronFields[0]));
    }
    if (cronFields.length === 5 && cronFields[4] !== "*") {
      allDaysSelected = false;
      var numToDay = {0: "sun", 1: "mon", 2: "tue", 3: "wed", 4: "thu", 5: "fri", 6: "sat"};
      selectedDayNames = cronFields[4].split(",").map(function(n) { return numToDay[Number(n)]; }).filter(Boolean);
    }
  }
  view.innerHTML =
    "<button id=\"editBack\" style=\"margin-bottom:10px\">← Back</button>" +
    "<div class=\"row three\"><label>Name <input id=\"editName\" value=\"" + escapeHTML(plan.name || "") + "\"></label>" +
    "<label>Target <input id=\"editTemp\" type=\"number\" min=\"10\" max=\"40\" step=\"1\" value=\"" + (plan.target_temp || 36) + "\"></label>" +
    "<label>Active <select id=\"editEnabled\"><option value=\"true\"" + (plan.enabled ? " selected" : "") + ">Yes</option><option value=\"false\"" + (!plan.enabled ? " selected" : "") + ">No</option></select></label></div>" +
    "<div class=\"row two\"><label>Mode <select id=\"editMode\"><option value=\"once\"" + (!isRepeating ? " selected" : "") + ">Once</option><option value=\"cron\"" + (isRepeating ? " selected" : "") + ">Repeating</option></select></label>" +
    "<label id=\"editAtWrap\"" + (isRepeating ? " class=\"hidden\"" : "") + ">At <input id=\"editAt\" type=\"datetime-local\" value=\"" + escapeHTML(atValue) + "\"></label>" +
    "<label id=\"editTimeWrap\"" + (!isRepeating ? " class=\"hidden\"" : "") + ">At <input id=\"editTime\" type=\"time\" value=\"" + escapeHTML(timeValue) + "\"></label></div>" +
    "<div class=\"days" + (!isRepeating ? " hidden" : "") + "\" id=\"editDays\"></div>" +
    "<div class=\"row two\" style=\"margin-top:12px\"><button class=\"primary\" id=\"editSave\">Save</button><button class=\"danger\" id=\"editDelete\">Delete</button></div>";
  var dayWrap = $("editDays");
  days.forEach(function(day) {
    var btn = document.createElement("button");
    btn.className = "day" + (allDaysSelected || selectedDayNames.indexOf(day) >= 0 ? " active" : "");
    btn.textContent = day.slice(0, 1).toUpperCase();
    btn.dataset.day = day;
    btn.onclick = function() { btn.classList.toggle("active"); };
    dayWrap.appendChild(btn);
  });
  $("editMode").onchange = function() {
    var repeating = $("editMode").value === "cron";
    $("editAtWrap").classList.toggle("hidden", repeating);
    $("editTimeWrap").classList.toggle("hidden", !repeating);
    $("editDays").classList.toggle("hidden", !repeating);
  };
  $("editBack").onclick = function() { state.planView = "plans"; renderPlans(); };
  $("editDelete").onclick = function() {
    var planId = plan.id;
    state.planView = "plans";
    updatePlans(state.plans.filter(function(p) { return p.id !== planId; }));
  };
  $("editSave").onclick = function() {
    var updated = {id: plan.id, type: plan.type, created_at: plan.created_at,
      name: $("editName").value || plan.name,
      enabled: $("editEnabled").value === "true",
      target_temp: Number($("editTemp").value || 36)
    };
    if ($("editMode").value === "cron") {
      var t = $("editTime").value;
      if (!t || t.indexOf(":") < 0) return toast("Ready time is required", "bad");
      var parts = t.split(":");
      var selDays = qsa("#editDays .day.active").map(function(b) { return b.dataset.day; });
      var dayField = "*";
      if (selDays.length > 0 && selDays.length < days.length) dayField = selDays.map(cronDay).join(",");
      updated.cron = Number(parts[1]) + " " + Number(parts[0]) + " * * " + dayField;
    } else {
      var at = $("editAt").value;
      if (!at) return toast("Ready time is required", "bad");
      updated.at = new Date(at).toISOString();
    }
    state.planView = "plans";
    updatePlans(state.plans.map(function(p) { return p.id === plan.id ? updated : p; }));
  };
}

function renderEditWindowForm(view, plan) {
  var windowCaps = ["filter", "heater", "jets", "bubbles", "sanitizer"];
  view.innerHTML =
    "<button id=\"editBack\" style=\"margin-bottom:10px\">← Back</button>" +
    "<div class=\"row three\">" +
    "<label>Name <input id=\"editName\" value=\"" + escapeHTML(plan.name || "") + "\"></label>" +
    "<label>Capability <select id=\"editCap\">" +
    windowCaps.map(function(c) { return "<option value=\"" + c + "\"" + (plan.capability === c ? " selected" : "") + ">" + title(c) + "</option>"; }).join("") +
    "</select></label>" +
    "<label>Active <select id=\"editEnabled\"><option value=\"true\"" + (plan.enabled ? " selected" : "") + ">Yes</option><option value=\"false\"" + (!plan.enabled ? " selected" : "") + ">No</option></select></label>" +
    "</div>" +
    "<div class=\"row two\"><label>From <input id=\"editFrom\" type=\"time\" value=\"" + escapeHTML(plan.from || "02:00") + "\"></label>" +
    "<label>To <input id=\"editTo\" type=\"time\" value=\"" + escapeHTML(plan.to || "04:00") + "\"></label></div>" +
    "<div class=\"days\" id=\"editDays\"></div>" +
    "<div class=\"row two\" style=\"margin-top:12px\"><button class=\"primary\" id=\"editSave\">Save</button><button class=\"danger\" id=\"editDelete\">Delete</button></div>";
  var planDays = plan.days || [];
  var dayWrap = $("editDays");
  days.forEach(function(day) {
    var btn = document.createElement("button");
    btn.className = "day" + (planDays.indexOf(day) >= 0 ? " active" : "");
    btn.textContent = day.slice(0, 1).toUpperCase();
    btn.dataset.day = day;
    btn.onclick = function() { btn.classList.toggle("active"); };
    dayWrap.appendChild(btn);
  });
  $("editBack").onclick = function() { state.planView = "plans"; renderPlans(); };
  $("editDelete").onclick = function() {
    var planId = plan.id;
    state.planView = "plans";
    updatePlans(state.plans.filter(function(p) { return p.id !== planId; }));
  };
  $("editSave").onclick = function() {
    var updated = {id: plan.id, type: plan.type, created_at: plan.created_at,
      name: $("editName").value || plan.name,
      enabled: $("editEnabled").value === "true",
      capability: $("editCap").value,
      from: $("editFrom").value,
      to: $("editTo").value,
      days: qsa("#editDays .day.active").map(function(b) { return b.dataset.day; })
    };
    state.planView = "plans";
    updatePlans(state.plans.map(function(p) { return p.id === plan.id ? updated : p; }));
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
  newer.disabled = page <= 0;
  older.disabled = !hasOlder;
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

function runAction(action, message) {
  if (!state.token) return toast("Token required", "bad");
  setBusy(true, "Working...");
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
  if (event.type && event.type.indexOf("manual_session.") === 0) return manualSessionEventLine(event);
  if (event.type === "control_mode" || legacyManualActivity(event)) return "Legacy manual control · " + (event.message || "Historical activity");
  if (event.type === "scheduler" && event.data) return planExecutionLine(event);
  return event.message || "";
}

function manualSessionEventLine(event) {
  var data = event.data || {};
  var transition = event.type.replace("manual_session.", "").replace(/_/g, " ");
  var parts = ["Manual session " + transition];
  var failed = data.failed_fields || [];
  if (!failed.length && data.outcomes) {
    Object.keys(data.outcomes).forEach(function(field) {
      if (data.outcomes[field] && data.outcomes[field].state === "failed") failed.push(field);
    });
  }
  if (failed.length) parts.push("Failed fields: " + failed.map(title).join(", "));
  if (data.capability) parts.push(title(data.capability) + (data.outcome ? " " + data.outcome : ""));
  if (data.code) parts.push(title(data.code));
  return parts.join(" · ");
}

function legacyManualActivity(record) {
  var data = record.data || {};
  var source = String(record.source || data.source || "");
  var kind = String(record.kind || data.kind || "");
  return /manual_override|webui-manual|webui-pause/.test(source + " " + kind);
}

function previousObservationEvent(rows, index) {
  for (var i = index + 1; i < rows.length; i++) {
    if (rows[i].type === "observation" && rows[i].data) return rows[i];
  }
  return null;
}

function commandLine(command) {
  if (legacyManualActivity(command)) return "Legacy manual control · " + title(command.capability);
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
  if (plan.type === "ready_by") return (plan.target_temp || "--") + "° by " + readyScheduleLabel(plan);
  if (plan.type === "time_window") return title(plan.capability) + " " + plan.from + "-" + plan.to + (plan.days && plan.days.length ? " · " + plan.days.join(", ") : "");
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
  if (!isHistoryPage && state.token) Promise.all([loadStatus().then(loadPoolControl), loadWeather(), loadActivities()]).then(renderLivePanels);
}, 30000);
setInterval(function() {
  if (state.token) loadTimeline().then(renderTimeline);
}, 60000);
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
