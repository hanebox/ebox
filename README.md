# ebox

A control panel for one Linux server. It builds and runs your applications with
Docker, provisions databases and compose stacks, and manages Traefik so your
domains get HTTPS. One Go binary with the React UI embedded, so there is nothing
to install besides Docker.

ebox is built for a single operator on a personal or small server. It is not
multi-tenant hosting.

## What it does

- **Applications.** Deploy from a git repository with a Dockerfile, a compose
  file, a static site build pack, or a prebuilt image. Build logs stream live,
  deployments keep a rollback history, and a push to the tracked branch can
  deploy automatically.
- **Databases.** PostgreSQL, MySQL, MariaDB, Redis and MongoDB with generated
  credentials. Apps reach them over the ebox network on an internal hostname;
  publishing a host port is optional.
- **Compose services.** Run a stack from a repository or a pasted compose file,
  with one service designated as the ingress.
- **Proxy.** Traefik is provisioned from the panel. It terminates TLS, routes
  domains to containers, requests certificates, and supports middlewares like
  basic auth, redirects, custom headers and rate limits.
- **Logs and terminals.** Live container and stack logs with search and
  download, plus browser shells into the host and any container ebox manages.
- **Backups.** Cron-scheduled database dumps and volume archives with retention,
  manual runs, downloads and restore.
- **Notifications.** Deployment, backup, certificate, container and disk events
  sent to email, Discord, Slack, Telegram or a webhook.
- **Host metrics.** CPU, memory, disk, network and load over 20 minutes or 24
  hours, plus Docker disk usage with a prune action.

Sign-in is one admin account protected by sessions and CSRF tokens. API tokens
cover scripts and CI, and changing the password signs out every other device.

On the host, ebox runs as a systemd service under its own `ebox` user, talks to
the Docker socket, and keeps its state (SQLite database, deployments, backups,
Traefik config) in a single data directory.

## Install

The installer needs a Linux host with systemd, Docker with the compose v2
plugin, and an x86_64 or arm64 CPU. Ports 8000 (panel), 80 and 443 (proxy) must
be free; the Traefik API listens on 8080 on localhost.

```bash
curl -fsSL https://raw.githubusercontent.com/buildandbuild/ebox/main/installer/install.sh | sudo bash
```

It checks the host first and prints a pass, warning or error line for each item,
with a fix command where something is missing: OS and architecture, systemd,
curl, sha256sum, openssl, git, ssh, the Docker CLI, daemon, compose plugin and
group, free disk space, memory, GitHub reachability, and the ports it wants to
use.

It then creates the `ebox` user, generates `APP_KEY`, installs the binary to
`/usr/local/bin/ebox`, writes `/etc/ebox/ebox.env` and the systemd unit, starts
the service, and asks for the admin account: display name, email and a password
of at least 10 characters.

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

Unattended install (the flags are also the escape hatch when there is no TTY):

```bash
curl -fsSL https://raw.githubusercontent.com/buildandbuild/ebox/main/installer/install.sh | sudo bash -s -- \
  --admin-name "Admin" --admin-email you@example.com --admin-password "$EBOX_ADMIN_PASSWORD"
```

The account is created through the panel API right after the service starts, so
the setup wizard is never left open to the network. If there is no terminal to
prompt on, or `--no-admin` is passed, the wizard at `/setup` stays available
instead. There are no default credentials.

## Updating

The panel updates itself from **Settings → About → Updates**. It downloads the
release asset, verifies the published checksum, swaps the binary and restarts
through systemd. The previous binary is kept, so the same card can roll back.

To remove ebox, keeping the data or purging it:

```bash
sudo bash installer/uninstall.sh          # remove ebox, keep data
sudo bash installer/uninstall.sh --purge  # remove everything, including data
```

## Configuration

The installer writes `/etc/ebox/ebox.env` with mode 0600. The defaults suit a
normal install; edit the file and restart the service to change them.

| Variable | Purpose |
| --- | --- |
| `APP_KEY` | Encrypts stored secrets. Losing it makes saved credentials unreadable. |
| `APP_HOST`, `APP_PORT` | Address and port the panel binds to. |
| `EBOX_HOST`, `EBOX_PORT` | Public address used in panel URLs and proxy routing. |
| `EBOX_DATA_DIR` | Database, deployments, backups and Traefik config. |
| `EBOX_DOCKER_NETWORK` | Docker network shared by the panel, proxy and app containers. |
| `EBOX_TRAEFIK_API_PORT` | Local port for the Traefik API. |
| `EBOX_UPDATE_REPOSITORY` | GitHub repository used for release checks and self-update. |

## Development

Requires Go 1.26+, bun (or Node 22+) and Docker. The panel needs Docker for
deployments, databases, services and the proxy; the UI and API run without it.

```bash
make dev       # migrations + backend on :8000 + vite dev server on :5173
make build     # web bundle -> embedded assets -> dist/ebox
make test      # go test ./... + vitest
make lint      # go vet + eslint + tsc
make migrate   # run database migrations
```

```
backend/    Goravel application (module `ebox`)
  app/http/         controllers and middleware
  app/models/       models
  app/services/     applications, backups, databases, deployment, docker,
                    proxy, sources, stacks, terminal, ...
  database/         migrations
  web/              go:embed target for the built frontend
web/        Vite + React SPA
installer/  systemd unit, install and uninstall scripts
scripts/    dev and build helpers
```

Built with Goravel v1.18 on Go, SQLite over the pure-Go driver in WAL mode, and
React 19 + TypeScript, Vite, Tailwind CSS v4, Base UI, TanStack Router and
Query, and xterm.js.
