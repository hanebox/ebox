# ebox

A personal-server PaaS panel in the spirit of Coolify: the control plane for the
host it runs on. Single static Go binary, embedded React UI, no external
dependencies beyond Docker and Traefik.

## Stack

- **Backend** — [Goravel](https://www.goravel.dev) v1.18 (Go), SQLite (pure-Go
  driver, WAL), in-process queue workers and SSE hub, Docker Engine API client
  plus the Docker CLI for streamed builds.
- **Frontend** — React 19 + TypeScript, Vite, Tailwind CSS v4,
  [Base UI](https://base-ui.com), TanStack Router + Query, xterm.js, CodeMirror.
- **Runtime** — the panel runs as a systemd service on the host; Traefik runs as
  a container and routes domains to app containers via Docker labels.

## Development

Requires Go 1.26+, Node 22+ (or bun), and Docker for anything beyond the
foundation milestone.

```bash
make dev          # migrations + backend on :8000 + vite dev server on :5173
make test         # go tests + vitest
make lint         # go vet + tsc
make build        # web bundle -> embedded assets -> dist/ebox
```

The installer creates the admin account during setup. If that is skipped
(`--no-admin` or no terminal), the first boot serves a setup wizard at
`/setup` that creates it instead. There are no default credentials.

## Install

On a Linux host with systemd and Docker:

```bash
curl -fsSL https://raw.githubusercontent.com/buildandbuild/ebox/main/installer/install.sh | sudo bash
```

The installer checks the host first — Docker CLI, daemon and compose plugin,
systemd, free ports, disk space, network, git and ssh — and prints a checklist
with fix hints for anything missing. It then creates the `ebox` user, generates
`APP_KEY`, installs the binary to `/usr/local/bin/ebox`, writes
`/etc/ebox/ebox.env` and the systemd unit, and asks for the admin account:
display name, email and a password of at least 10 characters.

The account is created through the panel's local API right after the service
starts, so the setup wizard is never left open to the network. If there is no
terminal to prompt on, or `--no-admin` is passed, the wizard at `/setup`
remains available instead.

| Flag | Purpose |
| --- | --- |
| `--version v0.5.0` | install a specific release |
| `--binary ./dist/ebox` | install a locally built binary |
| `--host 0.0.0.0` | address the panel binds to |
| `--port 9000` | panel port |
| `--data-dir /srv/ebox` | data, deployments and backups |
| `--no-start` | install without starting the service |
| `--no-admin` | skip admin account creation and finish at `/setup` |
| `--admin-name`, `--admin-email`, `--admin-password` | create the account without prompting |
| `--no-color` | plain output (also honours `NO_COLOR`) |
| `--force` | no confirmations; continue past failed checks |

Unattended install (flags are also the escape hatch when there is no TTY):

```bash
curl -fsSL https://raw.githubusercontent.com/buildandbuild/ebox/main/installer/install.sh | sudo bash -s -- \
  --admin-name "Admin" --admin-email you@example.com --admin-password "$EBOX_ADMIN_PASSWORD"
```

The panel updates itself from the **Settings → About → Updates** card: it
downloads the release asset, verifies the published checksum, swaps the binary
and restarts through systemd. A rollback restores the previous binary.

```bash
sudo bash installer/uninstall.sh          # remove ebox, keep data
sudo bash installer/uninstall.sh --purge  # remove everything, including data
```

## Layout

```
backend/   Goravel application (module `ebox`)
  app/http/controllers, app/http/middleware
  app/models, app/services/{account,crypto,metrics,realtime,settings,system}
  database/migrations
  web/     go:embed target for the built frontend
web/       Vite + React SPA
installer/ systemd unit and install/uninstall scripts
scripts/   dev / build helpers
```

## Milestones

- **M0 (done)** — foundation: auth, sessions + CSRF, live host metrics, SSE
  event stream, settings, design system.
- **M1 (done)** — projects, GitHub App sources, repo/branch pickers.
- **M2 (done)** — applications and deployments (Dockerfile, Compose, Static,
  Image) with live build logs, env vars, domains.
- **M3 (done)** — browser terminals, container logs, proxy management,
  dashboards.
- **M4 (done)** — one-click databases (Postgres, MySQL, MariaDB, Redis, MongoDB)
  and compose services, plus a compose build pack for applications.
- **M5 (done)** — notifications (email, Discord, Slack, Telegram, webhooks),
  scheduled backups with restore, a systemd installer and self-update.
