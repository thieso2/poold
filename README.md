# poold

`poold` is the pool-side daemon for Pooly. It runs next to the Intex spa, talks to the spa over its TCP JSON protocol, stores short-term state in SQLite, owns local schedules, and exposes a small authenticated HTTP API over Tailscale.

The module builds two binaries:

- `poold`: daemon for OpenWrt/Linux.
- `poolctl`: CLI for macOS/Linux clients on the same Tailscale network.

## Development

```sh
mise run poold:test
mise run poold:run
mise run poolctl -- status
```

Defaults:

- HTTP: `127.0.0.1:8090`
- Pool TCP: `127.0.0.1:8990`
- SQLite: `./var/poold.db`
- Token: `dev-token`

Set these in the environment or with `poold` flags:

- `POOLD_LISTEN_ADDR`
- `POOLD_POOL_ADDR`
- `POOLD_DB_PATH`
- `POOLD_TOKEN`
- `POOLD_TIMEZONE`
- `POOLD_HEATING_RATE_C_PER_HOUR`
- `POOLD_READINESS_BUFFER`
- `POOLD_POLL_STARTUP_INTERVAL` default `10s`
- `POOLD_POLL_IDLE_INTERVAL` default `10m`
- `POOLD_POLL_STABLE_INTERVAL` default `5m`
- `POOLD_POLL_ACTIVE_INTERVAL` default `1m`
- `POOLD_POLL_ERROR_MIN_INTERVAL` default `30s`
- `POOLD_POLL_ERROR_MAX_INTERVAL` default `5m`
- `POOLD_COMMAND_CONFIRM_DELAY` default `10s`
- `POOLD_EVENT_HEARTBEAT` default `30m`

`POOLD_POLL_INTERVAL` is still accepted as a compatibility alias for the stable interval. `poold` wakes immediately for desired-state or plan updates, confirms commands after `POOLD_COMMAND_CONFIRM_DELAY`, and caps sleep at the next known plan boundary.

`poolctl watch` prints compact event lines by default. Use `poolctl watch --json` to see the full event JSON stream.

## Builds

```sh
mise run poold:build:darwin-arm64
mise run poold:build:linux-amd64
mise run poold:build:openwrt-mips
mise run poold:build:all
```

The OpenWrt target is:

```sh
GOOS=linux GOARCH=mips GOMIPS=softfloat CGO_ENABLED=0
```

Artifacts are written under `services/poold/dist/`.

## API

All endpoints require `Authorization: Bearer <token>`.

- `GET /health`
- `GET /status`
- `GET /events?after=<id>&limit=<n>`
- `GET /events/stream`
- `GET /desired-state`
- `PUT /desired-state`
- `GET /plans`
- `PUT /plans`
- `POST /commands`

Example command:

```json
{
  "capability": "heater",
  "state": true,
  "source": "poolctl"
}
```

Example ready-by plan:

```json
{
  "id": "saturday-ready",
  "type": "ready_by",
  "enabled": true,
  "target_temp": 36,
  "at": "2026-05-09T08:30:00+02:00"
}
```

## OpenWrt

Copy `dist/openwrt-mips/poold` to `/usr/bin/poold`, copy `packaging/openwrt/init.d/poold` to `/etc/init.d/poold`, then set at least:

```sh
export POOLD_TOKEN='replace-me'
export POOLD_LISTEN_ADDR='100.x.y.z:8090'
export POOLD_POOL_ADDR='192.168.x.y:8990'
```

The init script uses `/var/lib/poold/poold.db` and `procd` respawn by default.
