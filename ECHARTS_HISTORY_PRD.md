# ECharts History PRD

## Problem Statement

The dashboard currently renders History with a hand-built inline SVG. It works for a first slice, but it is hard to inspect on mobile, has limited tooltip behavior, and will become expensive to maintain as Pooly adds richer history interactions. The iPhone full-screen web-app flow also needs a dedicated History page with its own navigation, because browser chrome and browser history are not reliable in that mode.

## Solution

Replace the current History renderer with Apache ECharts loaded from a pinned CDN URL. The dashboard keeps a compact 24 hour History preview. A dedicated `/history` page provides the full-screen chart, a visible Dashboard back button, range controls, a measured/predicted toggle, ECharts tooltips, and zoom controls.

The existing `/dashboard/timeline` API remains the data contract. HTML pages stay public so the token entry flow works, while the timeline JSON remains bearer-token protected.

## User Stories

1. As a pool owner, I want the dashboard History panel to show the last 24 hours by default, so that I can quickly see recent pool behavior.
2. As a pool owner, I want a full-screen History page, so that I can inspect temperature and equipment behavior without the rest of the dashboard crowding the chart.
3. As an iPhone full-screen web-app user, I want a visible Dashboard button, so that I can return from History without relying on browser chrome.
4. As a pool owner, I want the full-screen chart to offer fixed ranges, so that I can switch between recent and longer history quickly.
5. As a pool owner, I want to toggle measured and predicted pool temperature, so that I can compare real observations with the learned model.
6. As a pool owner, I want outside temperature and target temperature visible with the pool temperature, so that I can understand heating and cooling context.
7. As a pool owner, I want pool features shown as lanes, so that I can correlate filter, heater, power, jets, bubbles, sanitizer, and offline periods with temperature changes.
8. As a pool owner, I want command and plan annotations visible on the chart, so that I can understand why state changes happened.
9. As a pool owner, I want tooltips on the chart, so that I can inspect exact values without reading raw activity logs.
10. As a dashboard user, I want the compact preview to stay lightweight, so that the main control screen remains fast and uncluttered.
11. As a dashboard user, I want the preview header to navigate to full-screen History, so that tooltip inspection on the preview does not accidentally navigate away.
12. As an operator, I want the CDN dependency pinned, so that chart behavior does not change unexpectedly without a deploy.
13. As an operator, I want a clear unavailable message if the CDN fails, so that the rest of the dashboard remains usable.

## Implementation Decisions

- Load Apache ECharts from `https://cdn.jsdelivr.net/npm/echarts@6.0.0/dist/echarts.min.js`.
- Do not vendor ECharts in this slice.
- Serve a new public `/history` HTML route from the daemon.
- Keep `/dashboard/timeline` authenticated and unchanged.
- Reuse the existing timeline response shape for measured points, predicted points, target points, feature spans, annotations, confidence, and model metadata.
- Render the dashboard History panel as a compact ECharts preview with the default `24h` measured view and no range or mode controls.
- Render the full-screen History page with fixed ranges: `6h`, `24h`, `3d`, `7d`, and `14d`.
- Render the full-screen History page with `Measured` and `Predicted` mode controls.
- Use one ECharts instance per History chart.
- Use synchronized time axes inside one chart: a main temperature grid and compact lower feature-lane grids.
- Enable tooltip behavior in both preview and full-screen views.
- Enable ECharts data zoom only in the full-screen view.
- Provide a normal `/` Dashboard link as the full-screen back button.
- Show an explicit chart-library unavailable state when ECharts is not loaded.

## Testing Decisions

- Test externally visible HTTP behavior rather than ECharts internals.
- Cover that `/history` is public and serves the full-screen History shell.
- Cover that `/` still serves the dashboard shell and includes the pinned ECharts CDN URL.
- Preserve the existing timeline API tests because the renderer continues to use that API contract.
- Run `go test ./...` after implementation.
- Run a JavaScript syntax check against the embedded dashboard script after implementation.

## Out of Scope

- Custom date/time picking.
- Vendored or offline fallback ECharts assets.
- Changing the timeline JSON contract.
- Changing poll intervals or the learned prediction model.
- Deploying this change unless explicitly requested after implementation.

## Further Notes

This is a renderer and navigation change. It should not change scheduler behavior, database schema, polling behavior, or plan semantics.
