# Vacation Heating Spec

## Goal

Pooly should make the pool warm every morning during a vacation window, prioritizing reliability over energy efficiency. The first implementation should intentionally start heating too early rather than risk being late.

## User Experience

Add a vacation/morning-ready plan screen with:

- Enabled toggle.
- Start date and end date.
- Ready time, default `08:30`.
- Target temperature, default `36°C`.
- Reliability mode: `Conservative`, `Balanced`, `Efficient`; vacation defaults to `Conservative`.
- Preview: next ready time, estimated heating start, learned safe rate, safety buffer, and confidence.
- Last result: e.g. `Ready 42m early`, `Late by 18m`, or `No data yet`.

Dashboard should show an active card when relevant:

```text
Morning Ready
36°C by 08:30
Heating starts around 22:15
Conservative model · 0.60°C/h · 90m buffer
```

## Plan Model

Introduce a recurring ready-by plan:

```json
{
  "id": "vacation-morning",
  "type": "recurring_ready_by",
  "enabled": true,
  "from_date": "2026-05-10",
  "to_date": "2026-05-17",
  "time": "08:30",
  "target_temp": 36,
  "days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
  "reliability": "conservative"
}
```

Manual overrides still take precedence. Time-window plans and base desired state remain lower priority.

## Conservative Start Calculation

Temperatures are only reported in full degrees, so treat readings as ranges. For conservative scheduling:

```text
effective_current = current_temp - 0.5°C
effective_target = target_temp + 0.5°C
required_delta = effective_target - effective_current
start_at = ready_at - required_delta / safe_rate - safety_buffer
```

If `current_temp=30`, `target_temp=36`, `safe_rate=0.60°C/h`, and `buffer=90m`:

```text
required_delta = 7°C
heat_time = 11h40m
start_at = 19:20 previous evening
```

## Learning Heating Rate

Create a `heating_sessions` table. Record sessions when heater, filter, and power are on and temperature increases.

Suggested schema:

```sql
CREATE TABLE heating_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  started_at TEXT NOT NULL,
  ended_at TEXT NOT NULL,
  start_temp INTEGER NOT NULL,
  end_temp INTEGER NOT NULL,
  target_temp INTEGER NOT NULL,
  duration_minutes INTEGER NOT NULL,
  observed_gain_c INTEGER NOT NULL,
  lower_bound_gain_c REAL NOT NULL,
  rate_c_per_hour REAL NOT NULL,
  lower_bound_rate_c_per_hour REAL NOT NULL,
  weather_observation_id INTEGER,
  notes_json BLOB
);
```

Because readings are full degrees, calculate a conservative lower bound:

```text
observed_gain = end_temp - start_temp
lower_bound_gain = max(0, observed_gain - 1.0)
lower_bound_rate = lower_bound_gain / duration_hours
```

Only use sessions with:

- At least `2°C` observed gain.
- At least `90m` duration.
- No long connectivity gaps.
- Heater on for most of the session.

Prefer sessions with `3°C+` gain for high confidence.

## Safe Rate Selection

For `Conservative` mode:

- If fewer than 3 trusted sessions exist, use default `0.55°C/h`.
- Otherwise use a slow percentile of trusted lower-bound rates, e.g. 20th percentile.
- Cap upward changes slowly.
- Apply a minimum safety buffer of `90m`.

For `Balanced`:

- Use median lower-bound rate.
- Minimum buffer `60m`.

For `Efficient`:

- Use median observed/lower-bound blend.
- Minimum buffer `30m`.

Late results should immediately reduce the safe rate and increase buffer. Early results should adjust slowly.

## Weather Integration

Weather is recorded separately as raw OpenWeatherMap JSON. Heating sessions should link to the closest recent weather observation, but the first model can ignore weather for scheduling.

Future model inputs:

- Outside temperature.
- Cloud cover.
- Wind speed.
- Overnight low.
- Previous day sun/cloud conditions.

## Success Criteria

- Vacation plan can cover a date range and repeat every morning.
- Scheduler starts heating without user action.
- Morning target is reached by ready time in conservative mode.
- UI clearly shows next start time and model confidence.
- System records enough heating-session data to improve future estimates.
