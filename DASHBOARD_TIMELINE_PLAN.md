# Dashboard Timeline Plan

## Goal

Add a dashboard history graph that shows pool temperature, predicted pool temperature, target temperature, outside temperature, pool feature states, and operational annotations over selectable time ranges.

## Locked Decisions

- Ship as one slice: backend API, learned prediction, confidence, annotations, and dashboard SVG rendering together.
- Add a dedicated authenticated endpoint: `GET /dashboard/timeline`.
- Support fixed ranges first: `6h`, `24h`, `3d`, `7d`, `14d`. Default is `24h`.
- Render a full-width History panel above Activity.
- Use inline SVG in the embedded dashboard; no chart library or frontend build step.
- Dashboard mode toggle: `Measured` / `Predicted`.
- Predicted mode keeps measured points visible as anchors.
- Target temperature is a separate step series and stays visible by default.
- Active features render as compact lanes below the temperature chart.
- Include command and plan/manual annotations as grouped vertical markers.
- Timeline API requires bearer auth.

## Data Source

Use existing SQLite data:

- `observations`: pool temperature, target temperature, feature booleans, connected state, observation spans, weather snapshots.
- `commands`: command annotations.
- `events`: scheduler and plan/manual annotations.

The store query must include one observation before the selected range so feature states and step values are correct at the left edge. Spans are clipped to the selected range in the response.

## API Shape

```json
{
  "from": "2026-05-08T00:00:00Z",
  "to": "2026-05-09T00:00:00Z",
  "range": "24h",
  "generated_at": "2026-05-09T00:00:05Z",
  "bucket_seconds": 300,
  "unit": "C",
  "weather_available": true,
  "measured": [],
  "predicted": [],
  "target": [],
  "feature_spans": [],
  "annotations": [],
  "model": {},
  "warnings": []
}
```

Measured points use the last known measured value per bucket, not averages. Predicted points include `kind` and model source. Feature spans are merged and clipped. Annotations are top-level, tooltip-ready objects.

## Prediction Model

Learn rates from all retained observations, not just the selected range.

- Heating model: global learned heating rate from heater-on periods where temperature rose.
- Cooling model: outside-temperature bucketed learned cooling loss rate from heater-off periods where temperature stayed flat or fell.
- Buckets: `<10`, `10-20`, `20-30`, `>=30`, `unknown`.
- Minimum evidence before trusting a learned rate: 3 samples and 2 hours of observed duration.
- Fallback order for cooling: bucket learned rate, global learned rate, configured fallback.
- Heating fallback: `POOLD_HEATING_RATE_C_PER_HOUR`.
- Cooling fallback: `POOLD_COOLING_RATE_C_PER_HOUR`, default `0.10`.
- Ignore learning samples with disconnected state, missing temps, duration under 10 minutes, or jets/bubbles active.
- Flat heater-off samples count as `0 C/hour`; flat heater-on samples are ignored.
- Prediction is display/API only; scheduler control logic is unchanged.

## Confidence

- Measured confidence means freshness/staleness, not sensor accuracy.
- Predicted confidence decays with prediction horizon and model evidence.
- Disconnected or missing temperature forces confidence to `0`.
- Weather staleness remains separate as `weather_age_seconds`.

## Rendering

Defaults:

- Range: `24h`
- Mode: `Measured`
- Visible series: pool temp, target temp, outside temp
- Feature lanes: power, filter, heater; jets/bubbles/sanitizer appear when active in range

Annotations render as vertical markers with grouped tooltips and no always-visible labels.

## Implementation Checklist

- Add timeline response types.
- Add store range queries for observations, commands, and scheduler/plan events.
- Add learned model and timeline builder with unit tests.
- Add `GET /dashboard/timeline` handler and API tests.
- Add dashboard History panel and SVG renderer.
- Add dashboard range/mode controls.
- Add `POOLD_COOLING_RATE_C_PER_HOUR` config and docs.
- Run `go test ./...` and a JavaScript syntax/browser smoke check.
