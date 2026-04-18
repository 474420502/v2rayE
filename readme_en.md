# v2rayE

[中文说明](./README.md)

[![ui](image/README/ui.webp)
](./readme_en.md)

v2rayE is a Linux-first local proxy control plane that unifies profile management, TUN VPN routing, desktop proxy integration, subscriptions, and terminal interaction into one executable.

The first release baseline is now centered on:

- one unified entry binary: `./v2raye`
- one local HTTP API service mode: `./v2raye --server`
- one default terminal UI mode: `./v2raye`
- one Debian `.deb` packaging and release path

This project is closer to a local proxy workstation and TUN/VPN control console than to a browser-first web panel.

## v0.1.3 highlights

`v0.1.3` focuses on stabilizing the terminal-first path and making the TUI behave correctly under real terminal constraints:

- guarded TUI background startup so the app no longer gets stuck on `initial load`
- cleaned up Network summary and routing-preset lock interactions to remove re-entrant deadlock paths
- introduced `localProxyMode` so explicit local HTTP/SOCKS traffic can choose `follow-routing` or `force-proxy`
- moved major TUI pages onto one responsive layout system so compact and narrow terminals no longer depend on a fixed wide-screen two-column layout
- refined focus flow and grouping so both read-only and editable panels are reachable by keyboard in visual order

## What the first release includes

- Unified executable with TUI as default mode and backend service via `--server`
- Local HTTP API for TUI and scripts
- Profile and subscription management
- Profile selection, single delay tests, and batch delay tests
- Linux desktop system proxy integration
- Xray TUN mode
- One-click TUN diagnostics and repair
- Direct traffic bypass fix to prevent `direct` traffic from being recaptured by TUN
- Dual-stack TUN policy routing when an IPv6 default route exists
- `tun-health-check.sh` operational validation script
- Debian packaging script
- GitHub Actions workflow that builds `.deb` packages and attaches them to GitHub Releases on tag push

## Good fit for

- users who want to manage proxies directly from a Linux terminal
- users who want TUI, backend API, systemd service, and Debian packaging in one flow
- users who need TUN mode with stronger diagnostics and repairability
- users who want profile management, subscriptions, desktop proxy, and TUN routing under one local control plane

## Repository layout

- `backend-go/`: Go backend, TUI, and unified entrypoint
- `scripts/`: build, launch, health-check, and Debian packaging scripts
- `docs/`: design notes, systemd unit, migration records
- `dist/`: local build output directory for generated packages

## Quick start

### 1. Build locally

```bash
./scripts/build.sh
```

Output:

- `./v2raye`

### 2. Start the TUI

```bash
./v2raye
```

This is the default mode.

### 3. Start backend service mode

```bash
./v2raye --server
```

Default listen address:

- `127.0.0.1:18000`

### 4. Start the VPN workflow

```bash
./scripts/vpn-up.sh
```

This script will:

- ensure the backend is running
- start the core
- apply system proxy
- check core status
- check network availability

### 5. Run the TUN health check

```bash
sudo ./scripts/tun-health-check.sh
```

The check covers:

- API availability
- core running state
- TUN takeover state
- IPv4 policy routing integrity
- `fwmark -> main` direct bypass rule
- IPv6 policy routing integrity when an IPv6 default route exists

## Debian package

### Build a local `.deb`

```bash
./scripts/build-deb.sh 0.1.3
```

Expected output:

```bash
dist/v2raye_0.1.3_amd64.deb
```

### Install

```bash
sudo apt install ./dist/v2raye_0.1.3_amd64.deb
```

### Remove

```bash
sudo apt remove v2raye
sudo apt purge v2raye
```

Installed layout:

- binary: `/usr/lib/v2raye/v2raye`
- global command: `/usr/bin/v2raye`
- systemd unit: `/usr/lib/systemd/system/v2raye-server.service`
- runtime data directory: `/opt/v2rayE`

## systemd deployment

The repository already includes a unit file:

- `docs/systemd/v2raye-server.service`

Manual setup:

```bash
sudo install -d -m 755 /opt/v2rayE
sudo install -d -m 755 /usr/lib/v2raye
sudo install -m 755 ./v2raye /usr/lib/v2raye/v2raye
sudo ln -sf /usr/lib/v2raye/v2raye /usr/bin/v2raye
sudo install -m 644 ./docs/systemd/v2raye-server.service /etc/systemd/system/v2raye-server.service
sudo systemctl daemon-reload
sudo systemctl enable --now v2raye-server
```

Note: with the default configuration, the background service does not force desktop proxy changes at boot. If you want GNOME/KDE proxy integration, explicitly set `systemProxyMode=forced_change` and provide `V2RAYN_DESKTOP_USER` when running under root/systemd.

## Key environment variables

- `V2RAYN_API_ADDR`: backend listen address, default `0.0.0.0:18000`
- `V2RAYN_API_ALLOW_PUBLIC`: whether to allow public internet clients to reach the backend API; disabled by default, which limits access to loopback and LAN clients
- `V2RAYN_DATA_DIR`: data directory, default `/opt/v2rayE`
- `V2RAYN_API_TOKEN`: optional API token
- `V2RAYN_BACKEND_MODE`: backend mode, default `native`
- `V2RAYN_CORE_CMD`: external core command
- `V2RAYN_CORE_CMD_TEMPLATE`: external core command template with placeholders
- `V2RAYN_DESKTOP_USER`: desktop user name to target when the service runs as root/systemd

Boot restore note: when `autoRun` or persisted runtime state requests restore, the backend now waits for a bounded connectivity probe and then retries guarded core restore with backoff, reducing early-boot TUN/proxy failures caused by unstable default routes or DNS.

## TUN status in the first release

The first release closes the main TUN stability gaps:

- wider Linux policy-routing priority window
- critical bypass rules generated first
- dedicated `fwmark` for `direct` traffic to escape via the main table
- physical interface binding for both proxy and direct outbounds
- automatic IPv6 policy routing when an IPv6 default route exists
- backend diagnostics API exposing takeover, direct-bypass, and IPv6 state
- TUI and scripts surfacing the same runtime diagnostics

That means TUN status is no longer judged only by whether the core is running. The project can now verify whether:

- the default route was actually taken over
- direct traffic still has an escape path
- IPv6 is handled correctly in dual-stack environments

## API overview

The first release already exposes a usable local control API, including:

- `/api/health`
- `/api/core/status`
- `/api/core/start`
- `/api/core/stop`
- `/api/core/restart`
- `/api/profiles`
- `/api/subscriptions`
- `/api/network/availability`
- `/api/system-proxy/users`
- `/api/system-proxy/apply`
- `/api/config`
- `/api/routing`
- `/api/routing/diagnostics`
- `/api/routing/tun/repair`
- `/api/routing/hits`
- `/api/events/stream`
- `/api/logs/stream`

## GitHub automatic `.deb` release

The repository now includes:

- `.github/workflows/release-deb.yml`

Release flow:

```bash
git tag v0.1.3
git push origin master
git push origin v0.1.3
```

After the tag is pushed, GitHub Actions will automatically:

- set up Go
- run `go test ./...`
- run `./scripts/build-deb.sh <version>`
- generate `.deb` and `SHA256SUMS`
- create a GitHub Release and upload the artifacts

The workflow also supports manual `workflow_dispatch` runs with an explicit version input.

## Current recommended version

Use:

- `v0.1.3`

It is the current baseline for the unified entrypoint, TUI-first flow, Linux TUN hardening, responsive terminal layout, and Debian release pipeline.

## Current limits

- Linux is the primary target right now
- desktop proxy integration is mainly aimed at Linux desktop sessions
- the automated release currently produces Debian packages only
- the release workflow is tag-based and does not publish nightly builds

## Development validation

Backend validation command:

```bash
cd backend-go
go test ./...
```

## License

The repository does not yet include a dedicated LICENSE file. If this project will be published long-term, adding one is strongly recommended.