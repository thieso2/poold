# poold Deployment

## Target

The current OpenWrt deployment target is:

```sh
ssh -i ~/.ssh/poold_openwrt_ed25519 -o IdentitiesOnly=yes root@<gl-mt3000-ip>
```

Resolve the current `gl-mt3000` Tailscale address with `tailscale status` from a Tailscale-enabled host. Do not commit the resolved `100.x.y.z` address.

Observed from this environment:

- The router responds to ICMP over the Tailscale path.
- SSH reaches a Dropbear server on port 22.
- The router accepts the local passwordless key at `~/.ssh/poold_openwrt_ed25519`.

The router runs OpenWrt on `aarch64_cortex-a53`, so deploy the Linux arm64 daemon.

## Build

Use Go 1.26.1. If `go` is not on `PATH`, the repo can be built with `mise`:

```sh
mkdir -p dist/openwrt-arm64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  mise x go@1.26.1 -- go build -trimpath -ldflags='-s -w' \
  -o dist/openwrt-arm64/poold ./cmd/poold
```

Run tests before deploying:

```sh
mise x go@1.26.1 -- go test ./...
```

## Copy

OpenWrt does not provide `sftp-server`, so use legacy scp mode:

```sh
scp -O -i ~/.ssh/poold_openwrt_ed25519 -o IdentitiesOnly=yes \
  dist/openwrt-arm64/poold root@<gl-mt3000-ip>:/tmp/poold.new
scp -O -i ~/.ssh/poold_openwrt_ed25519 -o IdentitiesOnly=yes \
  packaging/openwrt/init.d/poold root@<gl-mt3000-ip>:/tmp/poold.init.new
```

## Install

Install atomically enough to avoid replacing the running binary with a partial copy:

```sh
ssh -i ~/.ssh/poold_openwrt_ed25519 -o IdentitiesOnly=yes root@<gl-mt3000-ip> '
  set -eu
  /etc/init.d/poold stop || true
  cp /tmp/poold.new /usr/bin/poold
  cp /tmp/poold.init.new /etc/init.d/poold
  chmod 0755 /usr/bin/poold /etc/init.d/poold
  /etc/init.d/poold enable
  /etc/init.d/poold restart
'
```

Router-specific values belong in `/etc/poold.env` on the device and must not be committed. The init script sources that file for `POOLD_TOKEN`, `POOLD_LISTEN_ADDR`, `POOLD_POOL_ADDR`, and related settings. The production SQLite path is `/data/poold.db`.

## Verify

```sh
ssh -i ~/.ssh/poold_openwrt_ed25519 -o IdentitiesOnly=yes root@<gl-mt3000-ip> '/etc/init.d/poold status'
ssh -i ~/.ssh/poold_openwrt_ed25519 -o IdentitiesOnly=yes root@<gl-mt3000-ip> 'logread -e poold | tail -80'
ssh -i ~/.ssh/poold_openwrt_ed25519 -o IdentitiesOnly=yes root@<gl-mt3000-ip> 'pgrep -af poold'
```

If the public hostname is in use, Cloudflare Tunnel should route `pool.tc42.uk` to the local poold origin on the router.
