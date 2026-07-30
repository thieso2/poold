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
  color-scheme: dark;
  --bg: #0b1013;
  --panel: #151d21;
  --panel-hi: #1b2429;
  --text: #dce8eb;
  --muted: #8ba0a8;
  --line: #26323a;
  --accent: #3fe3d0;
  --accent-strong: #7defdf;
  --ok: #4fd39a;
  --warn: #ff9a3c;
  --bad: #ff5f4d;
  --cool: #6aa8e8;
  --soft: #16232a;
  --field: #101a1e;
  --ink-on-accent: #05191c;
  --shadow: 0 18px 40px rgba(0, 0, 0, .45);
  --shadow-soft: 0 6px 18px rgba(0, 0, 0, .3);
}
* { box-sizing: border-box; }
/* Real data carries unbreakable strings — control revisions, idempotency keys,
   raw spa frames — which would otherwise push the page wider than the phone. */
html { overflow-x: clip; }
.panel, .panel-head, .pc, .pc-window, .pc-next-row, .pc-next-row > *, .line, .say,
.settings-group, .plans-add, .plan, .plan-main, .activity-item, .activity-list > * {
  min-width: 0;
}
body {
  overflow-wrap: break-word;
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
  background: var(--panel-hi);
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
  color: var(--ink-on-accent);
}
button:hover:not(:disabled), .button-link:hover {
  border-color: rgba(63, 227, 208, .45);
  box-shadow: var(--shadow-soft);
}
button:active:not(:disabled), .button-link:active {
  transform: translateY(1px);
}
button.danger {
  border-color: rgba(255, 95, 77, .5);
  color: var(--bad);
}
button:disabled {
  opacity: .55;
}
button:focus-visible, .button-link:focus-visible,
input:focus-visible, select:focus-visible {
  outline: 3px solid var(--accent);
  outline-offset: 3px;
  box-shadow: 0 0 0 5px rgba(11, 16, 19, .9);
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
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, .05);
  transition: background .18s ease, border-color .18s ease, box-shadow .18s ease;
}
input:hover, select:hover {
  border-color: #3a4a53;
  background: var(--panel-hi);
}
input:focus, select:focus {
  outline: 0;
  border-color: var(--accent);
  background: var(--panel-hi);
  box-shadow: 0 0 0 3px rgba(63, 227, 208, .18);
}
select {
  appearance: none;
  padding-right: 36px;
  background-image: linear-gradient(45deg, transparent 50%, var(--muted) 50%), linear-gradient(135deg, var(--muted) 50%, transparent 50%);
  background-position: calc(100% - 18px) 18px, calc(100% - 13px) 18px;
  background-size: 5px 5px, 5px 5px;
  background-repeat: no-repeat;
}
input[type="datetime-local"], input[type="time"] {
  font-variant-numeric: tabular-nums;
  letter-spacing: 0;
  background: linear-gradient(180deg, var(--panel-hi), var(--panel));
}
input[type="datetime-local"]::-webkit-calendar-picker-indicator,
input[type="time"]::-webkit-calendar-picker-indicator {
  border-radius: 7px;
  padding: 5px;
  background-color: var(--soft);
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
  width: min(660px, 100%);
  margin: 0 auto;
  padding: max(14px, env(safe-area-inset-top)) max(14px, env(safe-area-inset-right)) max(14px, env(safe-area-inset-bottom)) max(14px, env(safe-area-inset-left));
}
body[data-page="history"] {
  background: var(--bg);
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
  justify-content: flex-start;
  gap: 10px;
  padding: 2px 0 12px;
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
#settingsToggle {
  display: grid;
  place-items: center;
  width: 36px;
  min-height: 36px;
  padding: 0;
  border-radius: 999px;
  background: transparent;
  border-color: transparent;
  color: var(--muted);
}
#settingsToggle:hover:not(:disabled) {
  border-color: var(--line);
  background: var(--panel-hi);
  color: var(--text);
  box-shadow: none;
}
#settingsToggle[aria-expanded="true"] {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--panel-hi);
}
#settingsToggle svg {
  width: 19px;
  height: 19px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.settings-sub {
  margin: -4px 0 14px;
  font-size: 12.5px;
  color: var(--muted);
}
.settings-group {
  display: grid;
  gap: 9px;
  padding-top: 14px;
  border-top: 1px solid var(--line);
}
.settings-group:first-of-type { padding-top: 0; border-top: 0; }
.settings-label {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: .18em;
  text-transform: uppercase;
  color: var(--muted);
}
.settings-acts { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
@media (max-width: 480px) { .settings-acts { grid-template-columns: 1fr; } }
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
.badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 28px;
  padding: 4px 9px;
  border-radius: 999px;
  background: var(--soft);
  color: var(--muted);
  font-weight: 800;
  font-size: 12px;
}
.badge.ok { background: rgba(79, 211, 154, .14); color: var(--ok); }
.badge.bad { background: rgba(255, 95, 77, .16); color: var(--bad); }
.badge.warn { background: rgba(255, 154, 60, .16); color: var(--warn); }
/* ---- pool control: the unit ---- */
.pc {
  --case: #131a1e; --case-hi: #1d262b; --key: #1e262b; --key-edge: #2c383e;
  --led: #3fe3d0; --led-dim: rgba(63,227,208,.16);
  --warm: #ff9a3c; --warm-dim: rgba(255,154,60,.16);
  --fault: #ff5f4d;
  --etch: #6d7f88; --etch-hi: #a3b6bf;
  --mono: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  display: block;
  margin: 0 -14px -14px;
  padding: 14px;
  border-radius: 0 0 7px 7px;
  background: linear-gradient(180deg, var(--case-hi), var(--case));
  color: var(--etch-hi);
}
.pc button { font-family: inherit; }
.pc .etch { font-size: 9.5px; font-weight: 700; letter-spacing: .22em; text-transform: uppercase; color: var(--etch); }

/* readout window */
.pc-window {
  position: relative; padding: 18px 16px 13px; border-radius: 14px;
  background: linear-gradient(180deg, #05090b, #0a1114);
  box-shadow: inset 0 2px 14px rgba(0,0,0,.9), inset 0 0 0 1px #222c31;
  overflow: hidden;
}
.pc-window::after {
  content: ""; position: absolute; inset: 0; pointer-events: none;
  background: repeating-linear-gradient(180deg, rgba(255,255,255,.028) 0 1px, transparent 1px 3px);
}
.pc-rail { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 10px; }
.pc-flag {
  display: inline-flex; align-items: center; gap: 6px;
  font-family: var(--mono); font-size: 9.5px; font-weight: 700; letter-spacing: .14em; color: var(--etch);
}
.pc-flag i { width: 5px; height: 5px; border-radius: 50%; background: #24333a; }
.pc-flag.ok i { background: var(--led); box-shadow: 0 0 6px var(--led); }
.pc-flag.bad { color: var(--fault); }
.pc-flag.bad i { background: var(--fault); box-shadow: 0 0 6px var(--fault); }
.pc-outside {
  display: flex; align-items: baseline; gap: 9px; margin-top: 12px; padding-top: 10px;
  border-top: 1px solid #1a2429;
}
.pc-outside b { font-family: var(--mono); font-size: 17px; font-weight: 700; color: var(--etch-hi); font-variant-numeric: tabular-nums; }
.pc-outside span:not(.etch) {
  font-family: var(--mono); font-size: 9.5px; letter-spacing: .1em; color: var(--etch);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.pc-digits { position: relative; display: flex; align-items: flex-start; justify-content: center; }
.pc-seg { font-family: var(--mono); font-size: 84px; font-weight: 700; line-height: .84; letter-spacing: .04em; font-variant-numeric: tabular-nums; }
.pc-seg.ghost { color: rgba(63,227,208,.075); }
.pc-seg.live { position: absolute; left: 50%; transform: translateX(-50%); color: var(--led); text-shadow: 0 0 22px rgba(63,227,208,.5); }
.pc-deg { align-self: flex-start; margin-top: 7px; font-family: var(--mono); font-size: 19px; color: var(--led); opacity: .8; }
.pc-window-foot { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-top: 13px; }
.pc-leds { display: flex; flex-wrap: wrap; gap: 13px; }
.pc-led { display: flex; align-items: center; gap: 6px; }
.pc-led i { width: 6px; height: 6px; border-radius: 50%; background: #24333a; }
.pc-led.on i { background: var(--led); box-shadow: 0 0 7px var(--led); }
.pc-led.on.warm i { background: var(--warm); box-shadow: 0 0 7px var(--warm); }
.pc-led.blink i { animation: pcBlink 1.05s steps(2, jump-none) infinite; }
.pc-setpoint { font-family: var(--mono); font-size: 12.5px; color: var(--etch-hi); letter-spacing: .06em; white-space: nowrap; }
@keyframes pcBlink { 50% { opacity: .15; } }

/* mode toggle */
.pc-mode {
  position: relative; display: grid; grid-template-columns: 1fr 1fr; align-items: center;
  margin: 14px 0 4px; padding: 4px; border-radius: 12px;
  background: #0c1215; box-shadow: inset 0 2px 8px rgba(0,0,0,.8);
}
.pc-knob {
  position: absolute; top: 4px; bottom: 4px; left: 4px; width: calc(50% - 4px); border-radius: 9px;
  background: linear-gradient(180deg, #38464d, #232e33);
  box-shadow: 0 2px 6px rgba(0,0,0,.6), inset 0 1px 0 rgba(255,255,255,.14);
  transition: transform .22s cubic-bezier(.4,1.3,.5,1);
}
.pc-mode.manual .pc-knob { transform: translateX(100%); }
.pc-mode button {
  position: relative; z-index: 1; min-height: 40px; padding: 10px 4px; border: 0; background: none; cursor: pointer;
  font-size: 10px; font-weight: 700; letter-spacing: .22em; text-transform: uppercase; color: var(--etch);
}
.pc-mode button[aria-pressed="true"] { color: #eaf6f8; }
.pc-mode.manual button[aria-pressed="true"] { color: var(--warm); }
.pc-mode button:disabled { opacity: .45; cursor: not-allowed; }
.pc-legend { margin: 12px 2px 9px; }

/* backlit keys */
.pc-keys { display: grid; grid-template-columns: repeat(5, 1fr); gap: 8px; }
.pc-key {
  position: relative; display: grid; justify-items: center; gap: 6px;
  min-height: 64px; padding: 12px 3px 9px; border: 1px solid var(--key-edge); border-radius: 13px;
  background: linear-gradient(180deg, #263137, var(--key));
  box-shadow: 0 3px 0 #0c1114, 0 6px 12px rgba(0,0,0,.5), inset 0 1px 0 rgba(255,255,255,.09);
  color: #7d9098; cursor: pointer;
  transition: box-shadow .12s, background .12s, color .12s, transform .07s;
}
.pc-key svg { width: 25px; height: 25px; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.pc-key span { font-size: 8px; font-weight: 700; letter-spacing: .13em; text-transform: uppercase; }
.pc-key:active:not(:disabled) { transform: translateY(2px); box-shadow: 0 1px 0 #0c1114, inset 0 1px 0 rgba(255,255,255,.06); }
.pc-key[aria-pressed="true"] {
  color: var(--led); border-color: rgba(63,227,208,.5);
  background: linear-gradient(180deg, #1d3a3c, #14282b);
  box-shadow: 0 3px 0 #0c1114, 0 0 20px var(--led-dim), inset 0 0 16px rgba(63,227,208,.14), inset 0 1px 0 rgba(255,255,255,.08);
}
.pc-key.warm[aria-pressed="true"] {
  color: var(--warm); border-color: rgba(255,154,60,.5);
  background: linear-gradient(180deg, #3b2a17, #26190d);
  box-shadow: 0 3px 0 #0c1114, 0 0 20px var(--warm-dim), inset 0 0 16px rgba(255,154,60,.14), inset 0 1px 0 rgba(255,255,255,.08);
}
.pc-key:disabled { opacity: .45; cursor: not-allowed; }
.pc-key i.stat { position: absolute; top: 6px; right: 7px; width: 5px; height: 5px; border-radius: 50%; background: transparent; }
.pc-key.pending i.stat { background: var(--warm); box-shadow: 0 0 6px var(--warm); animation: pcBlink 1.05s steps(2, jump-none) infinite; }
.pc-key.failed { border-color: var(--fault); }
.pc-key.failed i.stat { background: var(--fault); box-shadow: 0 0 6px var(--fault); }
.pc-key i.edit { position: absolute; top: 6px; left: 7px; width: 5px; height: 5px; border-radius: 50%; background: var(--warm); }
.pc-key.dep i.edit { background: var(--etch); }
.pc-why { margin: 10px 2px 0; font-size: 11.5px; color: var(--warm); }

/* setpoint rocker */
.pc-rocker { display: grid; grid-template-columns: 54px 1fr 54px; gap: 8px; margin-top: 9px; }
.pc-rocker button {
  min-height: 52px; border: 1px solid var(--key-edge); border-radius: 12px; cursor: pointer;
  background: linear-gradient(180deg, #263137, var(--key));
  box-shadow: 0 3px 0 #0c1114, inset 0 1px 0 rgba(255,255,255,.09);
  color: var(--etch-hi); font-size: 21px; font-weight: 700;
}
.pc-rocker button:active { transform: translateY(2px); box-shadow: 0 1px 0 #0c1114; }
.pc-setwin {
  display: grid; place-items: center; gap: 2px; border-radius: 12px;
  background: #05090b; box-shadow: inset 0 2px 10px rgba(0,0,0,.85), inset 0 0 0 1px #222c31;
}
.pc-setwin b { font-family: var(--mono); font-size: 26px; color: var(--warm); text-shadow: 0 0 14px rgba(255,154,60,.4); }

/* commit */
.pc-commit { margin-top: 14px; padding-top: 13px; border-top: 1px dashed #2a353b; }
.pc-holds { display: grid; grid-template-columns: repeat(5, 1fr); gap: 7px; margin-top: 9px; }
.pc-holds button {
  min-height: 46px; padding: 12px 2px; border: 1px solid #45372a; border-radius: 11px; cursor: pointer;
  background: linear-gradient(180deg, #3a2b1a, #241a10); color: var(--warm);
  font-size: 11.5px; font-weight: 700; letter-spacing: .03em;
  box-shadow: 0 3px 0 #0c1114, inset 0 1px 0 rgba(255,255,255,.07);
}
.pc-holds button:active:not(:disabled) { transform: translateY(2px); box-shadow: 0 1px 0 #0c1114; }
.pc-holds button:disabled { border-color: #29343a; background: #171e22; color: #55656d; box-shadow: none; cursor: not-allowed; }
.pc-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 9px; }
.pc-chip { padding: 3px 8px; border: 1px solid #45372a; border-radius: 999px; font-family: var(--mono); font-size: 10.5px; color: var(--warm); }

/* session */
.pc-timer { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; }
.pc-timer b { font-family: var(--mono); font-size: 29px; font-weight: 700; color: var(--led); letter-spacing: .03em; font-variant-numeric: tabular-nums; }
.pc-acts { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-top: 11px; }
.pc-acts button {
  min-height: 44px; padding: 11px; border: 1px solid var(--key-edge); border-radius: 11px; cursor: pointer;
  background: var(--key); color: var(--etch-hi);
  font-size: 10.5px; font-weight: 700; letter-spacing: .13em; text-transform: uppercase;
}
.pc-acts button.hot { border-color: #45372a; background: linear-gradient(180deg, #3a2b1a, #241a10); color: var(--warm); }
.pc-acts button:disabled { opacity: .4; cursor: not-allowed; }
.pc-alert { margin: 11px 2px 0; font-size: 12px; color: var(--fault); }

/* schedule ahead */
.pc-next { margin-top: 13px; border-top: 1px dashed #2a353b; padding-top: 6px; }
.pc-next-row { display: grid; grid-template-columns: 58px 1fr auto; gap: 10px; align-items: baseline; padding: 8px 2px; border-bottom: 1px solid #202a2f; }
.pc-next-row:last-child { border-bottom: 0; }
.pc-next-row time { font-family: var(--mono); font-size: 13px; color: var(--etch-hi); font-variant-numeric: tabular-nums; }
.pc-next-row b { font-size: 12px; font-weight: 600; color: var(--etch-hi); }
.pc-next-row b em { font-style: normal; color: var(--led); }
.pc-next-row b em.warm { color: var(--warm); }
.pc-next-row small { font-family: var(--mono); font-size: 10px; color: var(--etch); white-space: nowrap; }
.pc-next-row p { margin: 2px 0 0; font-size: 11px; color: var(--etch); }
.pc-offline { margin: 12px 0 0; padding: 10px 12px; border: 1px solid rgba(255,95,77,.4); border-radius: 10px; background: rgba(255,95,77,.08); font-size: 12px; color: #ffb3aa; }
.pc button:focus-visible { outline: 2px solid var(--led); outline-offset: 3px; box-shadow: none; }

@media (prefers-reduced-motion: no-preference) {
  .pc-key[aria-pressed="true"] .jet-stream, .pc-led.on .jet-stream { animation: pcThrust 1.5s ease-out infinite; }
  .pc-key[aria-pressed="true"] .jet-stream:nth-of-type(2) { animation-delay: .18s; }
  .pc-key[aria-pressed="true"] .jet-stream:nth-of-type(3) { animation-delay: .36s; }
  .pc-key[aria-pressed="true"] .jet-stream:nth-of-type(4) { animation-delay: .1s; }
  .pc-key[aria-pressed="true"] .jet-stream:nth-of-type(5) { animation-delay: .28s; }
  .pc-key[aria-pressed="true"] .jet-stream:nth-of-type(6) { animation-delay: .46s; }
  .pc-key[aria-pressed="true"] .bub { animation: pcRise 2.4s ease-in-out infinite; }
  .pc-key[aria-pressed="true"] .bub:nth-of-type(2) { animation-delay: .5s; }
  .pc-key[aria-pressed="true"] .bub:nth-of-type(3) { animation-delay: 1s; }
  .pc-key[aria-pressed="true"] .bub:nth-of-type(4) { animation-delay: 1.5s; }
  .pc-key[aria-pressed="true"] .spin { transform-box: view-box; transform-origin: 12px 12px; animation: pcSpin 2.6s linear infinite; }
  .pc-key[aria-pressed="true"] .flame { transform-box: view-box; transform-origin: 12px 21px; animation: pcFlicker 1.5s ease-in-out infinite; }
  .pc-key[aria-pressed="true"] .pwr { transform-box: view-box; transform-origin: 12px 12px; animation: pcBreathe 2.8s ease-in-out infinite; }
}
@keyframes pcThrust { 0% { opacity: .25; transform: translateX(-1.5px); } 45% { opacity: 1; transform: translateX(0); } 100% { opacity: .25; transform: translateX(1.5px); } }
@keyframes pcRise { 0% { opacity: 0; transform: translateY(2.5px); } 25% { opacity: 1; } 75% { opacity: .9; } 100% { opacity: 0; transform: translateY(-3.5px); } }
@keyframes pcSpin { to { transform: rotate(360deg); } }
@keyframes pcFlicker { 0%, 100% { transform: scale(1, 1); } 30% { transform: scale(1.07, .93); } 62% { transform: scale(.95, 1.06); } }
@keyframes pcBreathe { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: .58; transform: scale(1.07); } }

@media (max-width: 380px) {
  .pc-keys { gap: 5px; }
  .pc-key { padding: 10px 2px 8px; }
  .pc-key svg { width: 22px; height: 22px; }
  .pc-key span { font-size: 7px; letter-spacing: .08em; }
  .pc-holds { gap: 5px; }
  .pc-holds button { font-size: 10.5px; padding: 12px 1px; }
  .pc-seg { font-size: 70px; }
  .pc-leds { gap: 9px; }
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
  color: var(--ink-on-accent);
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
  color: #06121f;
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
/* Activity as a ship's log: timestamp rail, LEDs, small-caps sources */
.al-day {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 10px; font-weight: 700; letter-spacing: .16em; text-transform: uppercase;
  color: var(--muted);
  padding: 14px 0 8px 86px;
}
.al-entry {
  display: grid; grid-template-columns: 58px 20px minmax(0, 1fr); gap: 0 8px;
  padding: 7px 0; position: relative;
  --alc: var(--muted);
}
.al-entry + .al-entry::before, .al-day + .al-entry::before {
  content: ""; position: absolute; left: 67.5px; top: -14px; height: 18px; width: 1px;
  background: var(--line);
}
.al-entry time {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 12px; line-height: 1.6; color: var(--muted); text-align: right;
  font-variant-numeric: tabular-nums;
}
.al-led {
  width: 8px; height: 8px; border-radius: 50%; margin: 6px auto 0;
  background: var(--alc);
  box-shadow: 0 0 7px color-mix(in srgb, var(--alc) 60%, transparent);
}
.al-led.hollow { background: transparent; border: 1.5px solid var(--alc); box-shadow: none; }
.al-msg { font-size: 14px; color: var(--muted); min-width: 0; overflow-wrap: break-word; margin: 0; }
.al-msg b { color: var(--text); font-weight: 600; }
.al-msg .al-src {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 10px; font-weight: 700; letter-spacing: .1em; text-transform: uppercase;
  color: color-mix(in srgb, var(--alc) 80%, var(--muted));
  margin-right: 7px;
}
.al-msg .al-id { color: color-mix(in srgb, var(--muted) 55%, transparent); font-size: 12px; }
.al-msg .al-mult {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 11px; font-weight: 700; color: var(--muted);
  border: 1px solid var(--line); border-radius: 999px; padding: 2px 7px; margin-left: 7px;
  white-space: nowrap;
}
.al-filter { --alc: #00b7c4; }
.al-heater { --alc: #f08c1a; }
.al-jets { --alc: #4d8fe0; }
.al-bubbles { --alc: #9d7bff; }
.al-power { --alc: #9aa8b0; }
.al-sanitizer { --alc: #35b878; }
.al-plan { --alc: var(--accent); }
.al-manual { --alc: var(--cool); }
.al-bad { --alc: var(--bad); }
.al-dim { --alc: #5b6d75; }
@media (max-width: 560px) {
  .al-entry { grid-template-columns: 46px 16px minmax(0, 1fr); }
  .al-entry + .al-entry::before, .al-day + .al-entry::before { left: 53.5px; }
  .al-day { padding-left: 70px; }
}
.activity-head {
  align-items: start;
  flex-wrap: wrap;
}
.timeline-panel {
  min-height: 430px;
  background:
    linear-gradient(180deg, var(--panel-hi), var(--panel)),
    radial-gradient(circle at 20% 0%, rgba(63, 227, 208, .10), transparent 34%);
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
  background: var(--soft);
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
  background: var(--panel-hi);
  color: var(--text);
  box-shadow: 0 3px 10px rgba(0, 0, 0, .35);
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
  border: 1px solid var(--line);
  border-radius: 999px;
  background: var(--field);
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
  background: repeating-linear-gradient(90deg, #b5852f 0 6px, transparent 6px 10px);
}
.legend-swatch.band {
  width: 16px;
  height: 9px;
  border-radius: 3px;
}
.timeline-chart {
  margin-top: 12px;
  min-height: 330px;
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 8px;
  background: linear-gradient(180deg, var(--panel-hi), var(--field));
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
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  border-bottom: 1px solid var(--line);
  margin-bottom: 6px;
}
.activity-tabs button {
  min-height: 0;
  border: 0;
  border-radius: 0;
  background: none;
  box-shadow: none;
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 11px; font-weight: 700; letter-spacing: .1em; text-transform: uppercase;
  color: var(--muted);
  padding: 8px 12px 10px;
  position: relative;
}
.activity-tabs button:hover { color: var(--text); background: none; }
.tabs.activity-tabs button.active {
  background: none; border: 0; color: var(--text);
}
.tabs.activity-tabs button.active::after {
  content: ""; position: absolute; left: 10px; right: 10px; bottom: -1px; height: 2px;
  background: var(--accent); border-radius: 1px;
  box-shadow: 0 0 8px var(--accent);
}
.tabs button.active {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--ink-on-accent);
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
  background: var(--accent);
  color: var(--ink-on-accent);
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
  .wide-only { display: inline; }
  .controls { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .row.three { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .activity-tabs { grid-template-columns: repeat(5, minmax(0, 1fr)); width: min(560px, 100%); }
}

/* ---- Plans as backlit instrument tags ---- */
#plansView { display: grid; gap: 12px; }
.tagplan {
  --pc: #00b7c4;
  position: relative;
  background: linear-gradient(180deg, var(--panel-hi), var(--panel));
  border: 1px solid var(--line);
  border-left: 3px solid var(--pc);
  border-radius: 12px;
  padding: 14px 16px 12px 18px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 4px 14px;
  overflow: hidden;
}
.tagplan::before {
  content: "";
  position: absolute; inset: 0 auto 0 0; width: 120px;
  background: linear-gradient(90deg, color-mix(in srgb, var(--pc) 9%, transparent), transparent);
  pointer-events: none;
}
.tagplan.kind-heater { --pc: #f08c1a; }
.tagplan.kind-jets { --pc: #4d8fe0; }
.tagplan.kind-bubbles { --pc: #9d7bff; }
.tagplan.paused { border-left-color: var(--line); }
.tagplan.paused::before { display: none; }
.tp-caption {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 10px; font-weight: 700; letter-spacing: .16em; text-transform: uppercase;
  color: color-mix(in srgb, var(--pc) 75%, var(--muted));
  margin-bottom: 5px;
}
.tagplan.paused .tp-caption { color: var(--muted); }
.tp-readout {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 27px; font-weight: 600; line-height: 1.15; letter-spacing: .01em;
  font-variant-numeric: tabular-nums;
  color: var(--text);
  text-shadow: 0 0 18px color-mix(in srgb, var(--pc) 35%, transparent);
  background: none; border: 0; min-height: 0; padding: 0; text-align: left;
  cursor: pointer;
}
.tagplan.paused .tp-readout { color: var(--muted); text-shadow: none; }
.tp-readout small { font-size: 14px; font-weight: 500; color: var(--muted); letter-spacing: .03em; }
.tp-say { grid-column: 1 / -1; margin: 8px 0 0; font-size: 13px; color: var(--muted); }
.tagplan.editing .tp-say { display: none; }
.tp-keys { display: flex; gap: 7px; align-self: start; }
.tp-key {
  min-width: 34px; min-height: 34px; padding: 0 8px;
  border: 1px solid var(--line); border-radius: 9px;
  background: linear-gradient(180deg, var(--panel-hi), var(--bg));
  color: var(--muted); font-size: 13px; line-height: 1;
  display: inline-flex; align-items: center; justify-content: center;
  box-shadow: 0 2px 0 rgba(0, 0, 0, .4);
}
.tp-key:active { transform: translateY(1px); box-shadow: 0 1px 0 rgba(0, 0, 0, .4); }
.tp-key:hover { color: var(--text); border-color: var(--muted); background: linear-gradient(180deg, var(--panel-hi), var(--bg)); }
.tp-key.tp-del:hover { color: var(--bad); border-color: color-mix(in srgb, var(--bad) 50%, var(--line)); }
.tp-key.tp-done {
  color: var(--pc);
  border-color: color-mix(in srgb, var(--pc) 45%, var(--line));
  font-weight: 700;
}
.tp-set {
  grid-column: 1 / -1;
  display: flex; flex-wrap: wrap; gap: 12px 18px;
  margin-top: 10px; padding-top: 12px;
  border-top: 1px dashed var(--line);
}
.tp-group { display: grid; gap: 6px; align-content: start; }
.tp-label {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 10px; font-weight: 700; letter-spacing: .14em; text-transform: uppercase;
  color: var(--muted);
}
.tp-stepper { display: flex; align-items: center; gap: 6px; }
.tp-val {
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 19px; font-weight: 600; color: var(--text);
  font-variant-numeric: tabular-nums;
  min-width: 72px; text-align: center;
  text-shadow: 0 0 14px color-mix(in srgb, var(--pc) 40%, transparent);
}
.tp-daykeys { display: flex; flex-wrap: wrap; gap: 4px; }
.tp-daykey {
  min-width: 32px; min-height: 32px; padding: 0 6px;
  border: 1px solid var(--line); border-radius: 8px;
  background: linear-gradient(180deg, var(--panel-hi), var(--bg));
  color: var(--muted);
  font-family: var(--mono, ui-monospace, Menlo, monospace);
  font-size: 11px; font-weight: 700;
  display: inline-flex; align-items: center; justify-content: center;
  box-shadow: 0 2px 0 rgba(0, 0, 0, .35);
}
.tp-daykey[aria-pressed="true"] {
  color: var(--pc);
  border-color: color-mix(in srgb, var(--pc) 55%, var(--line));
  box-shadow: 0 0 10px color-mix(in srgb, var(--pc) 25%, transparent), 0 2px 0 rgba(0, 0, 0, .35);
}
.tp-daykey.tp-cap { font-family: inherit; font-weight: 600; text-transform: none; letter-spacing: 0; }
.plans-empty { color: var(--muted); font-size: 14px; }
@media (max-width: 560px) {
  .tp-readout { font-size: 22px; }
}
</style>
</head>
<body data-page="__POOLD_PAGE__">
<main class="app">
  <div class="topbar history-only">
    <a class="button-link" href="/">Back to the pool</a>
  </div>

  <section class="tokenbar" id="tokenbar">
    <input id="token" type="password" autocomplete="current-password" placeholder="Bearer token">
    <button class="primary" id="saveToken">Save</button>
  </section>

  <section class="panel settings-panel" id="settingsPanel">
    <div class="panel-head">
      <h2>Settings</h2>
      <button id="settingsClose">Close</button>
    </div>
    <p class="settings-sub" id="subline">Pool daemon</p>
    <div class="settings-group">
      <span class="settings-label">Weather</span>
      <div class="row two">
        <label>OpenWeatherMap API key <input id="weatherApiKey" type="password" autocomplete="off" placeholder="Leave blank to keep saved key"></label>
        <label>Where the pool is <input id="weatherLocation" type="text" autocomplete="address-level2" placeholder="Berlin,DE"></label>
      </div>
      <button class="primary" id="saveWeatherSettings">Save weather settings</button>
      <p class="muted" id="weatherSettingsDetail">Weather is not configured.</p>
    </div>
    <div class="settings-group">
      <span class="settings-label">This device</span>
      <div class="settings-acts">
        <button id="refresh">Reload everything</button>
        <button id="editToken">Forget the access token</button>
      </div>
      <p class="muted">Reloading pulls fresh status, plans and history. Forgetting the token signs this browser out.</p>
    </div>
  </section>

  <div class="grid">
    <section class="panel dashboard-only">
      <div class="panel-head">
        <h2>Pool</h2>
        <div class="head-actions">
          <span class="badge" id="busy">Ready</span>
      <button id="settingsToggle" aria-label="Settings" aria-expanded="false" title="Settings">
        <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3.4"/><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.6 1.6 0 0 0-1.8-.3 1.6 1.6 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1A1.6 1.6 0 0 0 9 19.4a1.6 1.6 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.6 1.6 0 0 0 .3-1.8 1.6 1.6 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1A1.6 1.6 0 0 0 4.6 9a1.6 1.6 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.6 1.6 0 0 0 1.8.3H9a1.6 1.6 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.6 1.6 0 0 0 1 1.5 1.6 1.6 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.6 1.6 0 0 0-.3 1.8V9a1.6 1.6 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.6 1.6 0 0 0-1.5 1Z"/></svg>
      </button>
        </div>
      </div>
      <div class="pc" id="poolControl"></div>
      <p class="visually-hidden" id="manualSessionStatus" role="status" aria-live="polite" aria-atomic="true"></p>
    </section>

    <section class="panel dashboard-only">
      <div class="panel-head">
        <h2>Plans</h2>
        <span class="badge" id="plansCount">0 running</span>
      </div>
      <div id="plansView"></div>
      <div class="plans-add">
        <span class="settings-label">Add another</span>
        <button data-add-plan="window">Run <b>filter</b> from <b>06:00</b> for <b>2 hours</b>, every day.</button>
        <button data-add-plan="ready">Have the water at <b>36°</b> by <b>18:30</b>, every day.</button>
      </div>
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
  box.style.background = kind === "bad" ? "#7a2018" : kind === "ok" ? "#12503a" : "#1d262b";
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
  $("subline").textContent = status.connected ? "Connected " + formatAge(status.observed_at) : "Pool daemon";
}


function renderWeather() {
  renderPoolControl();
}


function renderSettings() {
  $("settingsPanel").classList.toggle("show", state.settingsOpen);
  $("settingsToggle").setAttribute("aria-expanded", String(state.settingsOpen));
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
  var blocked = !!draft || (!session &&
    (!observed || !observed.connected || manualSessionObservationIsStale(observed)));
  return '<div class="pc-mode ' + (manual ? "manual" : "auto") + '"><span class="pc-knob"></span>' +
    '<button id="automaticControl" aria-pressed="' + (!manual) + '"' + (state.pending ? " disabled" : "") + ">Auto</button>" +
    '<button id="manualControl" aria-pressed="' + manual + '"' + (blocked ? " disabled" : "") + ">Manual</button></div>";
}

function readoutMarkup() {
  var observed = manualSessionObserved();
  var shown = manualDisplayedState();
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var water = currentWaterTemp();
  var value = water == null ? "--" : String(water);
  var leds = manualCaps.map(function (cap) {
    var on = !!shown[cap];
    var sending = session && session.outcomes && session.outcomes[cap] &&
      session.outcomes[cap].state === "pending";
    return '<span class="pc-led' + (on ? " on" : "") + (cap === "heater" ? " warm" : "") +
      (sending ? " blink" : "") + '"><i></i><span class="etch">' + capLabels[cap] + "</span></span>";
  }).join("");
  var status = state.status || {};
  var linked = !!status.connected;
  var fault = status.error_code && status.error_code !== "None" ? status.error_code : null;

  var rail = '<div class="pc-rail">' +
    '<span class="pc-flag' + (linked ? " ok" : " bad") + '"><i></i>' +
    (linked ? "LINK " + (status.observed_at ? formatTime(status.observed_at) : "--") : "NO LINK") + "</span>" +
    '<span class="pc-flag' + (fault ? " bad" : "") + '"><i></i>ERR ' +
    (fault ? escapeHTML(fault) : "NONE") + "</span></div>";

  return '<div class="pc-window">' + rail + '<div class="pc-digits">' +
    '<span class="pc-seg ghost" aria-hidden="true">' + value.replace(/./g, "8") + "</span>" +
    '<span class="pc-seg live">' + value + '</span><span class="pc-deg">°C</span></div>' +
    '<div class="pc-window-foot"><div class="pc-leds">' + leds + "</div>" +
    '<span class="pc-setpoint">SET ' + shown.target_temp + "°</span></div>" +
    outsideMarkup() + "</div>";
}

// The weather reading belongs on the instrument: it is the other temperature
// that explains what the heater is up against.
function outsideMarkup() {
  var weather = state.weather || {};
  var latest = weather.latest || {};
  var data = latest.data || {};
  var main = data.main || {};
  var condition = data.weather && data.weather.length ? data.weather[0] : {};
  var line;
  if (latest.id) {
    var clouds = data.clouds && typeof data.clouds.all === "number" ? " · " + data.clouds.all + "% CLOUD" : "";
    line = '<b>' + (typeof main.temp === "number" ? Math.round(main.temp) : "--") + "°</b>" +
      "<span>" + escapeHTML((title(condition.description || condition.main || "Outside")).toUpperCase()) +
      " · " + escapeHTML(weatherLocationLabel(latest.location).toUpperCase()) + clouds + "</span>";
  } else if (weather.settings && weather.settings.api_key_set) {
    line = "<b>--°</b><span>WAITING FOR WEATHER</span>";
  } else {
    line = "<b>--°</b><span>OUTSIDE NOT CONFIGURED</span>";
  }
  return '<div class="pc-outside"><span class="etch">Out</span>' + line + "</div>";
}

function equipmentMarkup() {
  var draft = state.manualSessionDraft;
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var shown = manualDisplayedState();
  var keys = manualCaps.map(function (cap) {
    var on = !!shown[cap];
    var outcome = session && session.outcomes && session.outcomes[cap] ? session.outcomes[cap].state : "confirmed";
    var explicit = !!(draft && draft.explicit && draft.explicit[cap]);
    var dependency = !!(draft && draft.dependencies && draft.dependencies[cap]);
    return '<button class="pc-key' + (cap === "heater" ? " warm" : "") +
      (outcome === "pending" ? " pending" : "") + (outcome === "failed" ? " failed" : "") +
      (dependency ? " dep" : "") + '" data-manual-cap="' + cap + '" aria-pressed="' + on + '"' +
      ' title="' + capLabels[cap] + " · " + (on ? "on" : "off") + '"' +
      ' aria-label="' + capLabels[cap] + " " + (on ? "on" : "off") +
      (session ? ", " + outcome : "") + ", " + manualCapSubtitles[cap] + '"' +
      (draft ? "" : " disabled") + '><i class="stat"></i>' +
      (explicit || dependency ? '<i class="edit"></i>' : "") +
      manualCapIcons[cap] + "<span>" + capLabels[cap] + "</span></button>";
  }).join("");

  var why = "";
  if (draft && draft.dependencies) {
    var reasons = Object.keys(draft.dependencies).map(function (field) {
      var reason = draft.dependencies[field];
      if (reason === "heater_filter") return "Filter came on — the heater needs it";
      if (reason === "feature_power") return "Power came on — the feature needs it";
      if (reason === "power_off") return capLabels[field] + " went off — it follows the power";
      if (reason === "filter_off") return "Heater went off — it needs the filter";
      return capLabels[field] + " changed automatically";
    });
    var unique = reasons.filter(function (t, i) { return reasons.indexOf(t) === i; });
    if (unique.length) why = '<p class="pc-why">' + escapeHTML(unique.join(" · ")) + "</p>";
  }
  return '<div class="pc-keys">' + keys + "</div>" + why;
}

function currentWaterTemp() {
  var status = state.status;
  return status && status.current_temp != null ? status.current_temp : null;
}

function observedAgeLabel(observed) {
  if (!observed || !observed.observed_at) return "No reading";
  var seconds = Math.max(0, Math.round((Date.now() - new Date(observed.observed_at).getTime()) / 1000));
  if (seconds < 60) return "Linked · just now";
  return "Linked · " + Math.round(seconds / 60) + " min ago";
}

function targetMarkup() {
  var draft = state.manualSessionDraft;
  var shown = manualDisplayedState();
  if (!shown.heater || !draft) return "";
  return '<div class="pc-rocker"><button data-manual-step="-1" aria-label="Lower setpoint">–</button>' +
    '<span class="pc-setwin"><b>' + shown.target_temp + '°</b><span class="etch">Setpoint</span></span>' +
    '<button data-manual-step="1" aria-label="Raise setpoint">+</button></div>';
}

function applyMarkup() {
  var draft = state.manualSessionDraft;
  if (!draft) return "";
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var base = draft.base || {};
  var changes = manualChangedFields();
  if (pendingDiscard) {
    return '<div class="pc-commit"><span class="etch">' +
      (changes.length === 1 ? "Throw away this change?" : "Throw away these " + changes.length + " changes?") +
      '</span><p class="pc-why" style="color:var(--etch-hi)">Nothing was sent, so the pool is unaffected either way.</p>' +
      '<div class="pc-acts"><button data-manual-act="keep">Keep editing</button>' +
      '<button data-manual-act="discard">Throw away</button></div></div>';
  }
  var chips = changes.map(function (field) {
    if (field === "target_temp") return '<span class="pc-chip">SET ' + base[field] + "°→" + draft.intended[field] + "°</span>";
    return '<span class="pc-chip">' + capLabels[field].toUpperCase() + " " + (draft.intended[field] ? "ON" : "OFF") + "</span>";
  }).join("");
  var canApply = (changes.length > 0 || !!session) && !state.pending;
  var label = draft.review_required ? "Control changed — review, then run for" :
    canApply ? "Run for" : state.pending ? "Working…" : "Press a key above to change something";
  return '<div class="pc-commit">' + (chips ? '<div class="pc-chips">' + chips + "</div>" : "") +
    '<span class="etch" style="display:block;margin-top:' + (chips ? "10px" : "0") + '">' + label + "</span>" +
    '<div class="pc-holds">' + manualDurations.map(function (item) {
      return '<button data-manual-duration="' + item[0] + '"' + (canApply ? "" : " disabled") + ">" +
        item[1].replace(" min", "m").replace("60m", "1h").replace("2 hours", "2h").replace("Until off", "Hold") + "</button>";
    }).join("") + "</div></div>";
}

function sessionMarkup() {
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  if (!session) return "";
  var draft = state.manualSessionDraft;
  var failed = manualCaps.concat(["target_temp"]).filter(function (field) {
    return session.outcomes && session.outcomes[field] && session.outcomes[field].state === "failed";
  });
  var clock = session.expires_at ? manualSessionClock(session.expires_at) : "——";
  var alert = failed.length ? '<p class="pc-alert">' +
    escapeHTML((capLabels[failed[0]] || "Target temperature") + " — " +
      ((session.outcomes[failed[0]].message || session.outcomes[failed[0]].code) || "no acknowledgement")) + "</p>" : "";
  return '<div class="pc-commit"><div class="pc-timer"><span class="etch">' +
    (session.expires_at ? "Time left" : "No time limit") + '</span>' +
    '<b id="manualSessionClock">' + clock + "</b></div>" + alert +
    '<div class="pc-acts">' +
      (failed.length ? '<button class="hot" data-manual-act="retry"' + (state.pending ? " disabled" : "") + ">Retry</button>" : "") +
      (draft ? '<button data-manual-act="stopEdit">Stop editing</button>' :
        '<button data-manual-act="edit">Change</button>') +
      '<button data-manual-act="automatic"' + (state.pending ? " disabled" : "") + ">Back to auto</button>" +
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
  // The readout is the constant: it reads the pool whoever is in charge.
  var html = readoutMarkup() + ownershipMarkup() +
    '<p class="etch pc-legend">' + manualLegend() + "</p>";
  if (observed && !observed.connected) {
    html += '<p class="pc-offline">The pool is not answering. ' +
      (session ? "The session you applied still stands and you can hand it back." :
        "You cannot start a manual session until a fresh reading arrives.") + "</p>";
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

function manualLegend() {
  var draft = state.manualSessionDraft;
  var session = state.poolControlRepresentation && state.poolControlRepresentation.session;
  var observed = manualSessionObserved();
  if (draft && draft.review_required) return "Control changed — review, then run it again";
  if (draft) return "Set the keys, then choose how long";
  if (session && session.state === "applying") return "Sending to the pool";
  if (session && session.state === "degraded") return "One output did not answer";
  if (session) return "You have the pool";
  if (observed && !observed.connected) return "Pool offline";
  if (observed && manualSessionObservationIsStale(observed)) return "Reading is stale";
  return "Schedules hold the pool";
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
    if (plan.type === "time_window" && plan.start && plan.duration_minutes) {
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
    var begin = clockOnDay(day, plan.start);
    if (!begin) continue;
    var end = new Date(begin.getTime() + plan.duration_minutes * 60000);
    [[begin, true], [end, false]].forEach(function(pair) {
      var at = pair[0];
      var minutes = (at.getTime() - now.getTime()) / 60000;
      if (minutes <= 0 || minutes > horizon) return;
      var to = {};
      to[plan.capability] = pair[1];
      if (pair[1]) to.power = true;
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

// In Automatic the readout and its LEDs already state what the pool is doing,
// so this only has to answer what happens next.
function automaticSummaryMarkup() {
  var forecast = planForecast(3);
  var rows = forecast.length ? forecast.map(function (item) {
    var what = item.changes.map(function (change) {
      if (change.cap === "target_temp") return '<em class="warm">SET ' + change.value + "\u00b0</em>";
      return '<em' + (change.cap === "heater" ? ' class="warm"' : (change.on ? "" : ' style="color:var(--etch)"')) +
        ">" + capLabels[change.cap].toUpperCase() + " " + (change.on ? "ON" : "OFF") + "</em>";
    }).join(" · ");
    var after = !item.power ? "nothing runs after this" :
      item.running.length ? "then " + item.running.map(function (c) { return capLabels[c].toLowerCase(); }).join(" + ") +
        (item.target != null ? " to " + item.target + "\u00b0" : "") :
      "then idle";
    return '<div class="pc-next-row"><time>' + pad2(item.at.getHours()) + ":" + pad2(item.at.getMinutes()) +
      "</time><div><b>" + what + '</b><p>' + escapeHTML(item.name) + " \u00b7 " + after + "</p></div>" +
      "<small>" + relativeMinutes(item.at).replace("in ", "") + "</small></div>";
  }).join("") : '<div class="pc-next-row"><time>--:--</time><div><b>Nothing scheduled</b>' +
    '<p>No enabled plan changes the pool in the next three days.</p></div><small>—</small></div>';

  return '<div class="pc-next"><span class="etch" style="display:block;margin:6px 2px 2px">Next</span>' +
    rows + "</div>";
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

/* ---- Plans read as sentences; every value is a slot you tap ---- */
var CAP_PHRASE = {filter: "filter", jets: "the jets", bubbles: "the air"};
var DAY_FULL = {mon: "Monday", tue: "Tuesday", wed: "Wednesday", thu: "Thursday",
  fri: "Friday", sat: "Saturday", sun: "Sunday"};
var DAY_SHORT = {mon: "Mon", tue: "Tue", wed: "Wed", thu: "Thu", fri: "Fri", sat: "Sat", sun: "Sun"};
var WEEKDAYS = ["mon", "tue", "wed", "thu", "fri"];
var WEEKEND = ["sat", "sun"];
var planEditing = null;
var planSaveTimer = null;
var planDirty = false;

function planDays(plan) {
  if (plan.type === "time_window") return plan.days && plan.days.length ? plan.days : days.slice();
  var fields = String(plan.cron || "").trim().split(/\s+/);
  if (fields.length !== 5 || fields[4] === "*") return days.slice();
  var byNumber = {0: "sun", 1: "mon", 2: "tue", 3: "wed", 4: "thu", 5: "fri", 6: "sat"};
  return fields[4].split(",").map(function(n) { return byNumber[Number(n)]; }).filter(Boolean);
}

function planTime(plan) {
  if (plan.type === "time_window") return plan.start || "00:00";
  var fields = String(plan.cron || "").trim().split(/\s+/);
  if (fields.length >= 2 && /^\d+$/.test(fields[0]) && /^\d+$/.test(fields[1])) {
    return pad2(Number(fields[1])) + ":" + pad2(Number(fields[0]));
  }
  return plan.at ? pad2(new Date(plan.at).getHours()) + ":" + pad2(new Date(plan.at).getMinutes()) : "08:30";
}

function cronFor(time, dayList) {
  var parts = time.split(":");
  var toNumber = {sun: 0, mon: 1, tue: 2, wed: 3, thu: 4, fri: 5, sat: 6};
  var dow = dayList.length === 7 ? "*" : dayList.map(function(d) { return toNumber[d]; }).sort().join(",");
  return Number(parts[1]) + " " + Number(parts[0]) + " * * " + (dow || "*");
}

function daysPhrase(list) {
  if (!list.length) return "no days";
  if (list.length === 7) return "every day";
  var covers = function(set) { return set.every(function(d) { return list.indexOf(d) >= 0; }); };
  if (list.length === 5 && covers(WEEKDAYS)) return "on weekdays";
  if (list.length === 2 && covers(WEEKEND)) return "at the weekend";
  if (list.length === 1) return "on " + DAY_FULL[list[0]] + "s";
  var ordered = days.filter(function(d) { return list.indexOf(d) >= 0; })
    .map(function(d) { return DAY_SHORT[d]; });
  return "on " + ordered.slice(0, -1).join(", ") + " and " + ordered[ordered.length - 1];
}

function planKind(plan) {
  return plan.type === "ready_by" ? "heater" : normalizePlanCap(plan.capability);
}

function durShort(minutes) {
  minutes = Number(minutes) || 0;
  if (minutes < 60) return minutes + "m";
  if (minutes % 60 === 0) return (minutes / 60) + " h";
  return Math.floor(minutes / 60) + ":" + pad2(minutes % 60) + " h";
}

function onceDay(date) {
  var names = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  return names[date.getDay()] + " " + date.getDate() + "." + (date.getMonth() + 1) + ".";
}

function planCaption(plan) {
  var d = daysPhrase(planDays(plan)).replace("every day", "daily");
  if (plan.type === "ready_by") {
    return "heater · ready by · " + (!plan.cron && plan.at ? "once" : d);
  }
  return planKind(plan) + " · window · " + d;
}

function planReadout(plan) {
  if (plan.type === "time_window") {
    return escapeHTML(plan.start || "00:00") + " <small>for</small> " + durShort(plan.duration_minutes || 120);
  }
  var temp = (plan.target_temp || 36) + "°";
  if (!plan.cron && plan.at) {
    var at = new Date(plan.at);
    return temp + " <small>by</small> " + onceDay(at) + " " + pad2(at.getHours()) + ":" + pad2(at.getMinutes());
  }
  return temp + " <small>by</small> " + planTime(plan);
}

function windowDurationPhrase(minutes) {
  minutes = Number(minutes) || 0;
  if (minutes < 60) return minutes + (minutes === 1 ? " minute" : " minutes");
  if (minutes % 60 === 0) {
    var hours = minutes / 60;
    return hours + (hours === 1 ? " hour" : " hours");
  }
  return Math.floor(minutes / 60) + "h " + (minutes % 60) + "m";
}

function planSentence(plan) {
  if (plan.type === "time_window") {
    var cap = normalizePlanCap(plan.capability);
    return "Run " + (CAP_PHRASE[cap] || cap) + " from " + (plan.start || "00:00") +
      " for " + windowDurationPhrase(plan.duration_minutes || 120) + ", " +
      daysPhrase(planDays(plan)) + ".";
  }
  if (plan.type === "ready_by") {
    if (!plan.cron && plan.at) {
      return "Have the water at " + (plan.target_temp || 36) + "° by " + formatDateTime(plan.at) + ", once.";
    }
    return "Have the water at " + (plan.target_temp || 36) + "° by " + planTime(plan) + ", " +
      daysPhrase(planDays(plan)) + ".";
  }
  return describePlan(plan);
}

function normalizePlanCap(capability) {
  var cap = String(capability || "filter").toLowerCase();
  return CAP_PHRASE[cap] ? cap : "filter";
}

function planStepper(plan, label, value, field) {
  return '<div class="tp-group"><span class="tp-label">' + label + "</span>" +
    '<div class="tp-stepper">' +
    '<button class="tp-key" data-plan-step="-1" data-field="' + field + '" data-plan="' + escapeHTML(plan.id) + '" aria-label="' + label + ' down">−</button>' +
    '<span class="tp-val">' + value + "</span>" +
    '<button class="tp-key" data-plan-step="1" data-field="' + field + '" data-plan="' + escapeHTML(plan.id) + '" aria-label="' + label + ' up">＋</button>' +
    "</div></div>";
}

function planSetMode(plan) {
  var h = "";
  if (plan.type === "time_window") {
    h += '<div class="tp-group"><span class="tp-label">Run what</span><div class="tp-daykeys">' +
      Object.keys(CAP_PHRASE).map(function(cap) {
        return '<button class="tp-daykey tp-cap" data-plan-cap="' + cap + '" data-plan="' + escapeHTML(plan.id) +
          '" aria-pressed="' + (normalizePlanCap(plan.capability) === cap) + '">' + CAP_PHRASE[cap].replace("the ", "") + "</button>";
      }).join("") + "</div></div>";
    h += planStepper(plan, "Start", escapeHTML(plan.start || "00:00"), "start");
    h += planStepper(plan, "Run for", durShort(plan.duration_minutes || 120), "duration");
  } else if (!plan.cron && plan.at) {
    var at = new Date(plan.at);
    h += planStepper(plan, "Water at", (plan.target_temp || 36) + "°", "temp");
    h += planStepper(plan, "Day", onceDay(at), "at-day");
    h += planStepper(plan, "Ready by", pad2(at.getHours()) + ":" + pad2(at.getMinutes()), "at-time");
  } else {
    h += planStepper(plan, "Water at", (plan.target_temp || 36) + "°", "temp");
    h += planStepper(plan, "Ready by", planTime(plan), "time");
  }
  if (plan.type === "time_window" || plan.cron || !plan.at) {
    var current = planDays(plan);
    h += '<div class="tp-group"><span class="tp-label">Days</span><div class="tp-daykeys">' +
      days.map(function(day) {
        return '<button class="tp-daykey" data-plan-day="' + day + '" data-plan="' + escapeHTML(plan.id) +
          '" aria-pressed="' + (current.indexOf(day) >= 0) + '">' + DAY_SHORT[day].slice(0, 2) + "</button>";
      }).join("") + "</div></div>";
  }
  return '<div class="tp-set">' + h + "</div>";
}

function renderPlans() {
  var view = $("plansView");
  if (!state.plans.length) {
    view.innerHTML = '<p class="plans-empty">No plans yet, so the pool only does what you tell it by hand. ' +
      "Add one below and it will look after itself.</p>";
  } else {
    view.innerHTML = state.plans.map(function(plan) {
      var editing = planEditing === plan.id;
      var id = escapeHTML(plan.id);
      var name = escapeHTML(plan.name || plan.id);
      var keys = editing
        ? '<button class="tp-key tp-done" data-plan-done="' + id + '" aria-label="Done setting ' + name + '">✓</button>'
        : '<button class="tp-key" data-plan-edit="' + id + '" aria-label="Set ' + name + '">⚙</button>' +
          '<button class="tp-key" data-plan-toggle="' + id + '" aria-pressed="' + !plan.enabled + '" aria-label="' +
          (plan.enabled ? "Pause" : "Resume") + " " + name + '">' + (plan.enabled ? "❙❙" : "▶") + "</button>" +
          '<button class="tp-key tp-del" data-plan-delete="' + id + '" aria-label="Delete ' + name + '">✕</button>';
      return '<div class="tagplan kind-' + planKind(plan) + (plan.enabled ? "" : " paused") + (editing ? " editing" : "") + '">' +
        "<div>" +
        '<div class="tp-caption">' + escapeHTML(planCaption(plan)) + (plan.enabled ? "" : " · paused") + "</div>" +
        '<button class="tp-readout" data-plan-edit="' + id + '" aria-label="Set ' + name + '">' + planReadout(plan) + "</button>" +
        "</div>" +
        '<div class="tp-keys">' + keys + "</div>" +
        '<p class="tp-say">' + escapeHTML(planSentence(plan)) + "</p>" +
        (editing ? planSetMode(plan) : "") +
        "</div>";
    }).join("");
  }
  var running = state.plans.filter(function(p) { return p.enabled; }).length;
  $("plansCount").textContent = running + " running · " + state.plans.length + " total";
  bindPlans();
}

function planById(id) {
  return state.plans.filter(function(plan) { return plan.id === id; })[0];
}

function patchPlan(id, changes) {
  state.plans = state.plans.map(function(plan) {
    return plan.id === id ? Object.assign({}, plan, changes) : plan;
  });
  planDirty = true;
  clearTimeout(planSaveTimer);
  planSaveTimer = setTimeout(flushPlanSave, 900);
  renderPlans();
}

function flushPlanSave() {
  clearTimeout(planSaveTimer);
  if (!planDirty) return;
  planDirty = false;
  updatePlans(state.plans, "Plan updated");
}

function closePlanSetMode() {
  if (planEditing == null) return;
  planEditing = null;
  flushPlanSave();
  renderPlans();
}

function shiftClock(value, deltaMinutes) {
  var parts = String(value || "00:00").split(":");
  var minutes = ((Number(parts[0]) * 60 + Number(parts[1]) + deltaMinutes) % 1440 + 1440) % 1440;
  return pad2(Math.floor(minutes / 60)) + ":" + pad2(minutes % 60);
}

function stepPlan(plan, field, direction) {
  if (field === "temp") {
    var temp = Math.max(20, Math.min(40, (plan.target_temp || 36) + direction));
    patchPlan(plan.id, {target_temp: temp});
  } else if (field === "start") {
    patchPlan(plan.id, {start: shiftClock(plan.start, direction * 15)});
  } else if (field === "duration") {
    var duration = Math.max(30, Math.min(1440, (plan.duration_minutes || 120) + direction * 30));
    patchPlan(plan.id, {duration_minutes: duration});
  } else if (field === "time") {
    patchPlan(plan.id, {cron: cronFor(shiftClock(planTime(plan), direction * 15), planDays(plan)), at: null});
  } else if (field === "at-time" || field === "at-day") {
    var at = new Date(plan.at);
    at.setTime(at.getTime() + direction * (field === "at-day" ? 24 * 60 : 15) * 60000);
    patchPlan(plan.id, {at: at.toISOString()});
  }
}

function bindPlans() {
  qsa("[data-plan-edit]").forEach(function(button) {
    button.onclick = function(event) {
      event.stopPropagation();
      if (planEditing && planEditing !== button.dataset.planEdit) flushPlanSave();
      planEditing = button.dataset.planEdit;
      renderPlans();
    };
  });
  qsa("[data-plan-done]").forEach(function(button) {
    button.onclick = function(event) {
      event.stopPropagation();
      closePlanSetMode();
    };
  });
  qsa("[data-plan-toggle]").forEach(function(button) {
    button.onclick = function(event) {
      event.stopPropagation();
      var plan = planById(button.dataset.planToggle);
      updatePlans(state.plans.map(function(p) {
        return p.id === button.dataset.planToggle ? Object.assign({}, p, {enabled: !p.enabled}) : p;
      }), plan && plan.enabled ? "Plan paused" : "Plan resumed");
    };
  });
  qsa("[data-plan-delete]").forEach(function(button) {
    button.onclick = function(event) {
      event.stopPropagation();
      if (planEditing === button.dataset.planDelete) planEditing = null;
      updatePlans(state.plans.filter(function(plan) { return plan.id !== button.dataset.planDelete; }), "Plan deleted");
    };
  });
  qsa("[data-plan-step]").forEach(function(button) {
    button.onclick = function(event) {
      event.stopPropagation();
      var plan = planById(button.dataset.plan);
      if (plan) stepPlan(plan, button.dataset.field, Number(button.dataset.planStep));
    };
  });
  qsa("[data-plan-cap]").forEach(function(button) {
    button.onclick = function(event) {
      event.stopPropagation();
      patchPlan(button.dataset.plan, {capability: button.dataset.planCap});
    };
  });
  qsa("[data-plan-day]").forEach(function(button) {
    button.onclick = function(event) {
      event.stopPropagation();
      var plan = planById(button.dataset.plan);
      if (!plan) return;
      var current = planDays(plan);
      var index = current.indexOf(button.dataset.planDay);
      if (index >= 0) current.splice(index, 1); else current.push(button.dataset.planDay);
      current = days.filter(function(d) { return current.indexOf(d) >= 0; });
      if (plan.type === "time_window") patchPlan(plan.id, {days: current});
      else patchPlan(plan.id, {cron: cronFor(planTime(plan), current), at: null});
    };
  });
}

function addPlan(kind) {
  var plan = kind === "window"
    ? {id: "window-" + Date.now(), type: "time_window", name: "Filter window", enabled: true,
       capability: "filter", start: "06:00", duration_minutes: 120, days: days.slice()}
    : {id: "ready-" + Date.now(), type: "ready_by", name: "Ready by", enabled: true,
       target_temp: 36, cron: cronFor("18:30", days.slice())};
  updatePlans(state.plans.concat([plan]));
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
    items.push({label: "Measured", color: "#dce8eb", kind: "dot"});
    items.push({label: "Correction", color: "#ff9a3c", kind: "dot"});
  }
  if ((data.annotations || []).length) {
    items.push({label: "Command/plan", color: "#dce8eb", kind: "dot"});
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
    return {title: {text: "No timeline data", left: "center", top: "middle", textStyle: {fontSize: 14, color: "#8ba0a8", fontWeight: 600}}};
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
      {type: "time", min: from, max: to, axisLabel: {color: "#8ba0a8"}, axisLine: {lineStyle: {color: "#2b3840"}}, splitLine: {show: true, lineStyle: {color: "#1e2a31"}}},
      {type: "time", min: from, max: to, gridIndex: 1, axisLabel: {show: isHistoryPage, color: "#8ba0a8"}, axisLine: {lineStyle: {color: "#2b3840"}}, splitLine: {show: false}}
    ],
    yAxis: [
      {type: "value", min: min, max: max, axisLabel: {formatter: "{value}°", color: "#8ba0a8"}, axisLine: {show: false}, splitLine: {lineStyle: {color: "#1e2a31"}}},
      {type: "category", gridIndex: 1, data: lanes.map(timelineLaneLabel), inverse: true, axisTick: {show: false}, axisLine: {show: false}, axisLabel: {color: "#8ba0a8", fontSize: 12}, splitLine: {show: true, lineStyle: {color: "#1e2a31"}}}
    ],
    series: timelineSeries(data, lanes, min, max)
  };
  if (isHistoryPage) {
    option.dataZoom = [
      {type: "inside", xAxisIndex: [0, 1], filterMode: "none"},
      {type: "slider", xAxisIndex: [0, 1], filterMode: "none", bottom: 6, height: 24, borderColor: "#2b3840"}
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
      lineStyle: {color: "#b5852f", width: 2, type: "dashed"},
      itemStyle: {color: "#b5852f"},
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
      itemStyle: {color: "#dce8eb"}
    });
    series.push({
      name: "Corrections",
      type: "scatter",
      data: timelineLineData((data.predicted || []).filter(function(point) { return point.kind === "correction"; }), "pool_temp"),
      symbolSize: isHistoryPage ? 10 : 7,
      itemStyle: {color: "#ff9a3c"}
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
      itemStyle: {color: "#dce8eb"},
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

var CAP_LED = {filter: "filter", heater: "heater", jets: "jets", bubbles: "bubbles", power: "power", sanitizer: "sanitizer"};

function schedulerLed(event) {
  var desired = (event.data || {}).desired || {};
  if (desired.heater) return "heater";
  if (desired.jets) return "jets";
  if (desired.bubbles) return "bubbles";
  if (desired.filter) return "filter";
  return "plan";
}

// One log entry per activity row: which LED it lights, who wrote it, what it says.
function activityEntry(view, row, rawRows, index) {
  if (view === "polls") {
    return {ts: row.last_observed_at || (row.status || {}).observed_at, src: "POLL", led: "dim", hollow: true,
      text: observationLine(row, rawRows[index + 1]), id: row.id};
  }
  if (view === "commands") {
    return {ts: row.completed_at || row.issued_at, src: "CMD",
      led: row.success ? (CAP_LED[row.capability] || "dim") : "bad",
      text: commandLine(row), id: row.id};
  }
  if (view === "plan_executions") {
    return {ts: row.created_at, src: "SCHED", led: schedulerLed(row), text: planExecutionLine(row), id: row.id};
  }
  if (view === "heating_sessions") {
    return {ts: row.started_at, src: "HEAT", led: "heater", text: heatingSessionLine(row),
      id: row.first_observation_id + "-" + row.last_observation_id};
  }
  var type = String(row.type || "");
  var entry = {ts: row.created_at, src: "SYS", led: "dim",
    text: eventLine(row, previousObservationEvent(rawRows, index)), id: row.id};
  if (type === "scheduler") { entry.src = "SCHED"; entry.led = schedulerLed(row); }
  else if (type === "command") { entry.src = "CMD"; entry.led = CAP_LED[(row.data || {}).capability] || "dim"; }
  else if (type === "command_error" || type === "status_error") { entry.src = "ERR"; entry.led = "bad"; }
  else if (type === "plans") { entry.src = "PLAN"; entry.led = "plan"; }
  else if (type === "observation") { entry.src = "POLL"; entry.led = "dim"; entry.hollow = true; }
  else if (type.indexOf("manual_session.") === 0) { entry.src = "MANUAL"; entry.led = "manual"; }
  return entry;
}

function activityDayLabel(date) {
  var today = new Date();
  var yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1);
  if (date.toDateString() === today.toDateString()) return "Today";
  if (date.toDateString() === yesterday.toDateString()) return "Yesterday";
  var options = date.getFullYear() === today.getFullYear()
    ? {weekday: "short", month: "short", day: "numeric"}
    : {year: "numeric", month: "short", day: "numeric"};
  return date.toLocaleDateString([], options);
}

function activityEntryHTML(entry, count) {
  var main = entry.text || "";
  var rest = "";
  var split = main.indexOf(" · ");
  if (split > 0) { rest = main.slice(split); main = main.slice(0, split); }
  return '<div class="al-entry al-' + entry.led + '">' +
    "<time>" + escapeHTML(entry.clock) + "</time>" +
    '<span class="al-led' + (entry.hollow ? " hollow" : "") + '"></span>' +
    '<p class="al-msg"><span class="al-src">' + entry.src + "</span><b>" + escapeHTML(main) + "</b>" +
    escapeHTML(rest) +
    (count > 1 ? '<span class="al-mult">×' + count + "</span>" : " ") +
    '<span class="al-id">#' + escapeHTML(String(entry.id)) + "</span></p></div>";
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
  var entries = rows.map(function(row, index) {
    var entry = activityEntry(state.activityView, row, rawRows, index);
    var date = entry.ts ? new Date(entry.ts) : null;
    entry.day = date ? activityDayLabel(date) : "";
    entry.clock = date ? date.toLocaleTimeString([], {hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false}) : "--";
    return entry;
  });
  var html = "", lastDay = null, i = 0;
  while (i < entries.length) {
    var entry = entries[i];
    if (entry.day !== lastDay) { html += '<div class="al-day">' + escapeHTML(entry.day) + "</div>"; lastDay = entry.day; }
    // Collapse runs of identical messages from the same source into one ×N line.
    var count = 1;
    while (i + count < entries.length &&
      entries[i + count].day === entry.day &&
      entries[i + count].src === entry.src &&
      entries[i + count].text === entry.text) count++;
    html += activityEntryHTML(entry, count);
    i += count;
  }
  list.innerHTML = html;
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
  if (plan.type === "time_window") return title(plan.capability) + " " + plan.start + " for " + windowDurationPhrase(plan.duration_minutes) + (plan.days && plan.days.length ? " · " + plan.days.join(", ") : "");
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
document.addEventListener("click", function(event) {
  if (planEditing != null && !event.target.closest(".tagplan")) closePlanSetMode();
});
$("saveWeatherSettings").onclick = saveWeatherSettings;
qsa("[data-add-plan]").forEach(function(button) {
  button.onclick = function() { addPlan(button.dataset.addPlan); };
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
