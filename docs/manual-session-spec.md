# Manual Session Specification

Status: accepted
Applies to: `poold` HTTP API, persistence, reconciliation, and embedded dashboard

This document is the normative implementation and acceptance specification for
manual pool control. It supersedes the legacy control-mode API, ad-hoc command
API, `manual_override` plans, and `poolctl`.

## Product contract

Pool control has two user-visible states:

- **Automatic**: schedules own the pool. The dashboard shows live state, but
  feature and temperature controls are disabled.
- **Manual session**: one complete intended pool state owns control for a
  selected duration. The server reconciles the physical pool toward that state.

Changing the dashboard selector from Automatic to Manual only opens a local
draft. It must not contact the mutation API or affect the pool. The server
changes ownership only when the user presses Apply.

There is at most one Manual session. A new session or edit replaces its complete
intent and restarts its selected duration. Clearing or expiring it returns
ownership to Automatic and immediately reconciles the applicable schedule.

## Language

- **Observed state**: the most recent complete state returned by the pool.
- **Intended state**: the complete state a Manual session asks poold to hold.
- **Draft**: local, unsubmitted dashboard state. A draft has no server effect.
- **Control revision**: an opaque, durable token identifying the current control
  intent. Clients compare it for optimistic concurrency; they must not parse it.
- **Observation ID**: a durable identifier for an observed state snapshot.
- **Command outcome**: progress for one intended field: `pending`, `confirmed`,
  or `failed`.

## Complete state

Every intended and observed controllable state contains all of these fields:

```json
{
  "power": true,
  "filter": true,
  "heater": false,
  "jets": false,
  "bubbles": false,
  "target_temp": 38
}
```

`target_temp` is an integer Celsius value from 10 through 40 inclusive.
Sanitizer is not part of the product or API.

Valid states obey all of these constraints:

- A disabled power field requires filter, heater, jets, and bubbles to be off.
- Any enabled feature requires power to be on.
- Heater requires filter to be on.
- A disabled filter requires heater to be off.

The server rejects contradictory complete states. It does not silently add
dependencies or normalize intent.

## HTTP API

All endpoints require the existing bearer-token authentication. All responses
use `Content-Type: application/json` and `Cache-Control: no-store`.

### Representation

`GET /manual-session` always returns `200 OK` with the current complete control
representation:

```json
{
  "control": "automatic",
  "control_revision": "opaque-revision",
  "observed": {
    "observation_id": 417,
    "observed_at": "2026-07-28T14:20:00Z",
    "connected": true,
    "state": {
      "power": true,
      "filter": true,
      "heater": false,
      "jets": false,
      "bubbles": false,
      "target_temp": 38
    }
  },
  "session": null
}
```

When control is manual, `control` is `manual` and `session` is:

```json
{
  "state": "applying",
  "duration": "30m",
  "started_at": "2026-07-28T14:20:03Z",
  "expires_at": "2026-07-28T14:50:03Z",
  "intended": {
    "power": true,
    "filter": true,
    "heater": true,
    "jets": false,
    "bubbles": false,
    "target_temp": 38
  },
  "outcomes": {
    "power": {"state": "confirmed"},
    "filter": {"state": "confirmed"},
    "heater": {"state": "pending"},
    "jets": {"state": "confirmed"},
    "bubbles": {"state": "confirmed"},
    "target_temp": {"state": "confirmed"}
  }
}
```

`session.state` is:

- `applying` while one or more outcomes are pending and none have failed.
- `active` when every outcome is confirmed.
- `degraded` when one or more outcomes have failed.

A failed outcome also contains stable `code` and human-readable `message`
strings. The top-level `observed` object is the Manual session's current
observed state and is updated as commands and polling produce new snapshots.

For `until_off`, `expires_at` is `null`. For timed sessions it is an RFC 3339
timestamp calculated from the committed `started_at`.

### Create or replace

`PUT /manual-session` requires an `Idempotency-Key` header and this body:

```json
{
  "expected_control_revision": "opaque-revision",
  "base_observation_id": 417,
  "duration": "30m",
  "intended": {
    "power": true,
    "filter": true,
    "heater": true,
    "jets": false,
    "bubbles": false,
    "target_temp": 38
  }
}
```

Allowed durations are `10m`, `30m`, `60m`, `2h`, and `until_off`.

`base_observation_id` is required when Automatic control is being replaced. The
server performs a synchronous fresh pool read before committing:

- If the pool cannot be read, it returns `503 pool_unreachable` and makes no
  change.
- If any controllable field differs from the base observation, it returns
  `409 observed_state_changed` and makes no change.
- Current water temperature and observation time do not cause this conflict.

When editing an existing Manual session, `base_observation_id` must be omitted.
An edit may be accepted while the pool is offline because the current intent,
not a potentially stale observation, is its base.

On success, the server commits the complete session and its expiry in one
database transaction, returns `202 Accepted` with the complete representation,
and performs physical commands asynchronously.

### Retry failed outcomes

`POST /manual-session/retry` requires an `Idempotency-Key` header and:

```json
{
  "expected_control_revision": "opaque-revision"
}
```

It changes every failed outcome to pending and starts reconciliation. It does
not change intent, control revision, start time, or expiry. It returns
`202 Accepted` with the complete representation. Calling it without any failed
outcomes is successful and returns the unchanged representation.

### Return to Automatic

`DELETE /manual-session` requires an `Idempotency-Key` header and:

```json
{
  "expected_control_revision": "opaque-revision"
}
```

It commits Automatic ownership first, advances the control revision, returns
`200 OK` with the complete Automatic representation, and asynchronously
reconciles the applicable schedule. It remains available when the pool is
offline.

### Errors

Errors use this envelope:

```json
{
  "error": {
    "code": "control_changed",
    "message": "Control changed after this draft was opened.",
    "changed_fields": [],
    "violations": [],
    "current": {}
  }
}
```

- `400 invalid_request`: malformed JSON, missing required data, unknown fields,
  invalid duration, or target temperature outside its range.
- `401 unauthorized`: missing or invalid bearer token.
- `409 control_changed`: `expected_control_revision` is stale. `current`
  contains the complete current representation.
- `409 observed_state_changed`: a new Automatic-session draft is based on
  changed controllable fields. `changed_fields` names them and `current`
  contains the complete current representation.
- `409 idempotency_key_reused`: the same key was previously used for a
  materially different operation.
- `422 invalid_state`: complete intent violates a dependency.
  `violations` contains stable dependency codes.
- `503 pool_unreachable`: the mandatory fresh read for a new Manual session
  failed.
- `500 persistence_error`: the server could not durably commit the operation.

An error never partially changes server ownership or intent.

## Concurrency and idempotency

The durable control revision changes only when control intent changes:

- creating a Manual session;
- replacing or editing its intent or duration;
- clearing a Manual session;
- expiry returning ownership to Automatic.

Observation updates, outcome progress, retries, and reconnects do not change the
control revision.

Each mutation atomically compares `expected_control_revision` with the durable
current value. A stale edit cannot recreate a session that expired or was
cleared, and a stale worker cannot write outcomes for a replaced session.

Idempotency keys and their result representations are persisted before a
successful response. Repeating the same operation with the same key returns its
original status and representation, including after restart or a lost response.
Reusing the key with different method, target, expected revision, duration, or
intent returns `409 idempotency_key_reused`. Keys may be retained indefinitely;
storage may compact them only after 30 days.

Dashboard drafts are local and lock-free. Each tab stores its draft in
`sessionStorage`; tabs do not overwrite one another's drafts.

On either concurrency conflict, the dashboard preserves explicitly edited
fields, rebases them onto the returned current representation, recalculates
dependencies, visibly marks the rebased draft for review, and requires a new
Apply. It never automatically resubmits.

## Persistence

The database contains:

- one durable control-revision record, including while Automatic;
- a `manual_session` table constrained to zero or one row, holding the complete
  intent, lifecycle state, duration, start and expiry times, and owning control
  revision;
- durable per-field outcomes associated with the owning revision;
- durable idempotency records containing the operation fingerprint, response
  status, and complete response representation;
- ordinary observations, commands, and events as before.

The transaction that creates or replaces a session writes its intent, expiry,
initial outcomes, new control revision, idempotency record, and
`manual_session.created` or `manual_session.replaced` event together. The
transaction that clears or expires a session removes it, advances revision,
writes its idempotency record when applicable, and writes the lifecycle event
together.

Restart recovery loads the durable owner before running schedules. If a Manual
session remains valid, poold resumes its reconciliation. If its expiry passed
while poold was stopped, poold atomically expires it and resumes Automatic
reconciliation before issuing commands.

## Physical command reconciliation

The Intex protocol has no atomic command that sets the full pool state. Each
feature command changes one device setting and returns a full status snapshot.
Therefore the API is atomic at the ownership-and-intent layer, while physical
convergence is asynchronous and observable.

The worker fences itself with the owning control revision before every physical
command and again before persisting its result. It refreshes status immediately
before any toggle, skips a toggle whose observed field already equals intent,
and records the returned full status after every command.

Minimal dependency order is:

1. Disable heater, jets, and bubbles that should be off.
2. Enable power when required.
3. Enable filter when required.
4. Set target temperature when different.
5. Enable heater, jets, and bubbles that should be on.
6. Disable filter when required.
7. Disable power when required.

Fields already equal to intent become confirmed without a command. A command
failure marks that field failed. A dependent field that cannot safely be
attempted becomes failed with `dependency_not_met`. Unrelated safe commands may
continue. Any failure makes the session degraded.

Ordinary polling continuously compares observed and intended state. Drift
changes the affected confirmed outcome to pending and reconciliation attempts
it once. A failed attempt becomes degraded and waits for explicit Retry. A
successful attempt becomes confirmed. This prevents an uncontrolled retry loop
while keeping drift visible.

Clearing, expiry, or replacement cancels the old worker logically through its
revision fence. Results arriving from an old worker may add historical command
records but cannot change current intent, current outcomes, or current owner.

## Legacy cutover and migration

This is a one-way replacement with no backward compatibility.

At database migration:

- delete the `control_mode` key/value record;
- delete current plans whose kind or source identifies `manual_override`,
  `webui-manual`, or `webui-pause`;
- create the Manual-session, outcome, revision, and idempotency storage;
- begin in Automatic control and reconcile the applicable ordinary schedule.

The migration preserves:

- all historical observations;
- all historical commands;
- all historical events;
- all ordinary schedules and their state.

Historical records from removed behavior render as **Legacy manual control**.
They are retained for history only and never regain executable semantics.

Implementation removes:

- `GET /control-mode` and `PUT /control-mode`;
- the ad-hoc command mutation endpoint;
- creation or execution of `manual_override` plans;
- `ControlMode`, `PlanManualOverride`, and their scheduler branches;
- reserved manual-control plan-source behavior;
- `POOLD_MANUAL_CONTROL_DURATION`;
- the entire `cmd/poolctl` binary and its build/documentation references.

The read-only command history endpoint, desired-state behavior, ordinary plans,
and historical data remain.

Before deploying the migration, copy the production SQLite database to a dated
backup while poold is stopped and verify it opens with SQLite. Rollback means
stopping the new binary, restoring that complete backup, and deploying the
previous binary. There is no down migration.

## Dashboard interaction

The accepted layout is visual Variant C:

- Power is at the geometric center.
- Filter, Heater, Jets, and Bubbles form the surrounding ring.
- Controls are compact icon-only buttons; there is no Sanitizer and no
  “Proposed” label.
- Duration and Apply are visibly associated with the Manual draft.

### Automatic

The dashboard renders live observed values. Feature controls and target
temperature are disabled with native disabled semantics. Selecting Manual
creates a local draft from the displayed observed state, chooses `30m` by
default, and makes no mutation request.

### Draft

Feature and temperature edits update only the draft. The UI automatically adds
required dependencies and visibly distinguishes dependency-added fields from
fields explicitly selected by the user without relying on color alone.

Apply sends one complete `PUT /manual-session`. Cancel or selecting Automatic
discards an untouched draft immediately. Discarding a dirty draft first asks
for confirmation and still sends no server request.

### Applying, active, and degraded

Applying shows per-field progress and disables duplicate Apply. Active shows
the remaining time or “Until turned off.” Editing an active session begins a
local draft seeded from its intended state; Apply replaces the complete intent
and restarts the selected duration.

Degraded keeps the complete intended and observed states visible, identifies
failed fields in text, and offers Retry. Retry sends one retry request for all
failed outcomes.

Selecting Automatic while a Manual session exists sends one DELETE. The UI
reflects the returned Automatic representation immediately, then continues to
show live schedule reconciliation.

Expiry returns the interface to Automatic and announces that the session ended.
Offline and stale states are explicit. Starting a new Manual session is
unavailable while known offline; a connectivity loss after drafting preserves
the draft and reports the failed Apply.

## Accessibility and responsive behavior

The dashboard meets WCAG 2.2 AA:

- Every icon control has a programmatic accessible name.
- Focus and long-press help expose each icon's meaning.
- Every interactive target is at least 44 by 44 CSS pixels even when its visible
  icon is smaller.
- Toggle buttons expose `aria-pressed`; unavailable controls use actual disabled
  semantics.
- Keyboard focus is visible and follows a logical order.
- Ownership changes, applying progress, failures, conflicts, retry results, and
  expiry are announced through an appropriate live region.
- State and dependency provenance are never communicated by color alone.
- Text and controls meet AA contrast, and nonessential animation respects
  `prefers-reduced-motion`.

Required automated viewports are:

- Mobile WebKit: 320x568, 390x844, and 430x932 portrait.
- Mobile WebKit: 844x390 landscape.
- Desktop Chromium: 1280x800.

At each viewport there is no horizontal page scrolling, clipped control,
safe-area overlap, or unintended browser zoom. Automated geometry assertions
verify centered Power and non-overlapping ring controls. Accepted-state
screenshots are retained as test artifacts.

The final device smoke test uses iPhone Safari over the Tailnet, launched from
Add to Home Screen in full-screen mode.

## Observability

No metrics stack is required. The implementation provides:

- durable `manual_session` events for create, replace, applying, active,
  degraded, retry, clear, expiry, recovery, conflict, and stale-worker discard;
- structured logs for the same lifecycle and recovery decisions;
- command events containing the owning control revision and capability outcome;
- a dashboard history rendering that makes the session transition and failed
  fields understandable.

Logs and events must not contain bearer tokens, idempotency keys, raw
authorization headers, or other sensitive headers.

## Acceptance scenarios

Every requirement above is covered by one or more scenarios below. Server
scenarios run black-box through HTTP against a real temporary SQLite database
and a controllable fake spa. Lower-level unit tests supplement, but do not
replace, these scenarios.

### API and lifecycle

- [ ] **API-01** GET in Automatic returns the complete no-store representation,
  observation, opaque revision, and null session.
- [ ] **API-02** A valid PUT performs a fresh read, atomically commits ownership,
  returns 202 applying, and later reaches active.
- [ ] **API-03** PUT accepts exactly the four durations and calculates the
  corresponding expiry, including null for `until_off`.
- [ ] **API-04** Malformed, incomplete, unknown, out-of-range, or contradictory
  intent returns the specified 400 or 422 without mutation.
- [ ] **API-05** Editing a session replaces complete intent and restarts duration
  without requiring a fresh pool read.
- [ ] **API-06** Retry resets all failed outcomes without changing intent,
  revision, or expiry.
- [ ] **API-07** DELETE commits Automatic while offline and schedule
  reconciliation resumes asynchronously.
- [ ] **API-08** Authentication and every documented error status and envelope
  are contract-tested.
- [ ] **LIFE-01** Timed expiry atomically returns to Automatic and reconciles the
  applicable schedule.
- [ ] **LIFE-02** `until_off` survives time passage and ends only through DELETE
  or replacement.
- [ ] **LIFE-03** Polling detects drift, attempts it once, and exposes confirmed,
  pending, failed, applying, active, and degraded transitions.
- [ ] **LIFE-04** Partial failure preserves full intended and observed state,
  identifies failed fields, and succeeds through explicit Retry.

### Concurrency, crash safety, and recovery

- [ ] **CONC-01** Stale revision on PUT, retry, and DELETE returns
  `control_changed` with current representation and no mutation.
- [ ] **CONC-02** A changed controllable base observation returns
  `observed_state_changed`; current temperature/time-only changes do not.
- [ ] **CONC-03** Two dashboards race: exactly one intent wins, the loser rebases
  explicit edits, recalculates dependencies, and requires review and Apply.
- [ ] **CONC-04** Expiry advances revision so a stale edit cannot resurrect the
  expired session.
- [ ] **CONC-05** Repeating an identical mutation and idempotency key after a
  lost response returns the original result without duplicate toggles.
- [ ] **CONC-06** Reusing an idempotency key for different content returns the
  documented conflict.
- [ ] **RACE-01** Crash after database commit but before the first command
  recovers the durable intent and completes reconciliation.
- [ ] **RACE-02** Crash midway through commands resumes from fresh observed state
  without toggling already-satisfied fields.
- [ ] **RACE-03** Clear or expiry while applying prevents the old worker from
  mutating current outcomes or issuing its next command.
- [ ] **RACE-04** A new session replacing an old worker fences all old results.
- [ ] **RACE-05** Retry racing an observation update produces one correct final
  outcome per field and no duplicate toggle.
- [ ] **RACE-06** A stale command result may enter history but cannot overwrite
  the final owner, intent, observation, or outcomes.
- [ ] **REC-01** Restart with an unexpired session restores manual ownership and
  resumes reconciliation before schedules run.
- [ ] **REC-02** Restart after elapsed expiry returns to Automatic before issuing
  commands.

### Migration and retained history

- [ ] **MIG-01** A committed legacy database fixture contains active
  `control_mode`, reserved manual plans, ordinary schedules, and historical
  observations, commands, and events.
- [ ] **MIG-02** Migrating that fixture starts Automatic, removes live legacy
  state, preserves ordinary schedules and all history, and labels old records
  “Legacy manual control.”
- [ ] **MIG-03** Removed routes return unavailable, removed plan kinds cannot be
  created or executed, and `poolctl` is not built.
- [ ] **MIG-04** The documented stop, backup, SQLite-open verification, deploy,
  restore, and previous-binary rollback procedure is exercised in a deployment
  smoke test.

### Dashboard, responsive, and accessibility

- [ ] **UI-01** Automatic shows live state with every feature and temperature
  control disabled.
- [ ] **UI-02** Selecting Manual creates an untouched 30-minute local draft and
  sends no mutation request.
- [ ] **UI-03** Editing creates a dirty draft, adds dependencies visibly, and
  sends no mutation until Apply.
- [ ] **UI-04** Apply sends one complete request and renders applying, active,
  active edit, degraded, retry, expired, and Automatic states from server
  representations.
- [ ] **UI-05** Cancel, dirty-draft discard confirmation, offline Apply, stale
  observation, competing dashboard, and lost-response retry have the specified
  request or no-request behavior.
- [ ] **UI-06** Variant C contains only Power, Filter, Heater, Jets, and Bubbles;
  Power is geometrically centered and the ring does not overlap.
- [ ] **RESP-01** All required browser/viewports pass clipping, horizontal
  scrolling, safe-area, zoom, geometry, and screenshot checks.
- [ ] **A11Y-01** Automated accessibility checks cover WCAG 2.2 AA names, roles,
  states, target sizes, contrast, live announcements, and reduced motion.
- [ ] **A11Y-02** A keyboard-only run covers the complete Automatic-to-Manual,
  edit, Apply, degraded, Retry, and return-to-Automatic flow with visible focus.
- [ ] **DEVICE-01** A real iPhone completes the full-screen Add-to-Home-Screen
  flow over the Tailnet without clipping, zoom, or inaccessible controls.

### Observability

- [ ] **OBS-01** Lifecycle, conflict, recovery, retry, expiry, and stale-worker
  cases write the required durable events and structured logs.
- [ ] **OBS-02** Logs and events include revision and capability outcomes but
  contain no token, idempotency key, authorization header, or sensitive header.
- [ ] **OBS-03** Dashboard history makes current Manual-session transitions,
  failures, and retained legacy manual-control records understandable.

Implementation is accepted only when every checkbox is supported by an
automated result or the explicitly required real-device/deployment smoke-test
record.

## Decision and prototype references

- [Wayfinder map](https://github.com/thieso2/poold/issues/1)
- [Legacy cutover decision](https://github.com/thieso2/poold/issues/2)
- [Variant C interaction prototype decision](https://github.com/thieso2/poold/issues/3)
- [Concurrency decision](https://github.com/thieso2/poold/issues/4)
- [Manual-session API decision](https://github.com/thieso2/poold/issues/5)
- [Acceptance-boundary decision](https://github.com/thieso2/poold/issues/6)
- Prototype source: `internal/httpapi/manual_control_prototype.html`
