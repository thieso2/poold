# ECharts History Implementation Plan

## Goal

Replace the inline SVG History renderer with ECharts, add a dedicated `/history` full-screen page, and keep the existing timeline API contract.

## Decisions

- CDN: `https://cdn.jsdelivr.net/npm/echarts@6.0.0/dist/echarts.min.js`.
- Dashboard preview: default `24h`, measured mode, tooltip enabled, no zoom, no range/mode controls.
- Full-screen page: `/history`, public HTML shell, visible Dashboard button linking to `/`, fixed ranges, measured/predicted toggle, tooltip, and data zoom.
- Auth: HTML shells are public; `/dashboard/timeline` remains bearer-token protected.
- Failure state: if ECharts is unavailable, show a chart-library unavailable message.

## Work Plan

1. Add `/history` to the HTTP router and public web path list.
2. Change the web UI renderer so the same embedded HTML can render either dashboard or full-screen History mode.
3. Add the pinned ECharts script to the HTML shell.
4. Replace `timelineSVG` rendering with ECharts option creation and chart lifecycle code.
5. Keep helper functions that still support labels, time formatting, lanes, annotations, and metadata; remove SVG-only helpers.
6. Make the dashboard History panel compact and link to `/history` through a header action.
7. Make `/history` fill the viewport, hide non-History dashboard panels, and include a Dashboard back button.
8. Add handler tests for `/history` and CDN inclusion.
9. Run `gofmt`, `go test ./...`, and a JavaScript syntax check extracted from `webui.go`.

## Risks

- ECharts is CDN-loaded, so a network or CDN issue can blank the chart. The UI must detect this and show an explicit unavailable state.
- The embedded JavaScript is large and not typechecked by Go. A syntax check should be part of verification.
- Full-screen iPhone layout needs stable heights; the History page should use viewport-relative chart sizing rather than relying on SVG aspect ratio.
