# muxcorectl

Admin and operator CLI for a running MuxCore (`muxcored`) node. Mirrors the **admin-ui** web dashboard from the terminal — approachable for CLI newcomers, with `--json` and scripting flags for veterans.

Talks to muxcored over gRPC (core SDK client) and discovers module HTTP/gRPC endpoints the same way admin-ui does.

## Install

**From a release** (recommended for household installs):

Download `muxcorectl` for your platform from the [Forgejo releases](https://git.zem.systems/muxcore/muxcorectl-cli/releases) page, or build locally:

```bash
git clone ssh://forgejo@git.zem.systems:2222/muxcore/muxcorectl-cli.git
cd muxcorectl-cli
make build    # writes bin/muxcorectl
```

**From source** (requires Forgejo module access):

```bash
export GOPRIVATE=github.com/Muxcore-Media/*
export GIT_TERMINAL_PROMPT=0
git config --global url."ssh://forgejo@git.zem.systems:2222/muxcore/".insteadOf "https://github.com/Muxcore-Media/"

git clone ssh://forgejo@git.zem.systems:2222/muxcore/muxcorectl-cli.git
cd muxcorectl-cli
make build
```

Requires Go 1.26+. A standalone clone resolves modules from Forgejo — no sibling module checkouts required.

**Umbrella workspace dev:** copy `go.work.example` to `go.work` (gitignored) to overlay local sibling modules.

## Quick start (local laptop stack)

```bash
export MUXCORE_INSECURE_DISABLE_TLS=true
export MUXCORE_MESH_DIAL_LOCAL=true
export MUXCORE_TOKEN="$(cat ../_mvp/run/admin.token)"   # path may vary

muxcorectl --help
muxcorectl health status
muxcorectl modules list
muxcorectl settings list
muxcorectl media libraries
muxcorectl users list
```

## Vault soak stack (homelab)

Mesh gRPC on vault is **local only** (`127.0.0.1:9090`) — run from your workstation via SSH wrapper or copy the binary to vault.

**Wrapper (preferred):**

```bash
../_mvp/scripts/muxcorectl-vault.sh health status
../_mvp/scripts/muxcorectl-vault.sh modules list
../_mvp/scripts/muxcorectl-vault.sh --json settings list
```

**Deploy updated CLI to vault:**

```bash
../_mvp/scripts/deploy-module-to-vault.sh muxcorectl --verify-all
```

**Post-deploy smoke (from umbrella workspace):**

```bash
../_mvp/scripts/smoke-vault-all.sh
```

**SSH one-liner** (same env as `muxcorectl-vault.sh`):

```bash
ssh -6 ender@fd2c:a2fd:5d9e:ab72:9d99:930d:f160:3e95 \
  'export PATH=/mnt/fast-storage/appdata/muxcore/mvp/bin:$PATH \
     MUXCORE_INSECURE_DISABLE_TLS=true MUXCORE_MESH_DIAL_LOCAL=true MUXCORE_GRPC_ADDR=127.0.0.1:9090 \
     MUXCORE_TOKEN=$(cat /mnt/fast-storage/appdata/muxcore/mvp/run/admin.token) \
     && muxcorectl health status'
```

Full deploy/smoke reference: workspace [`AGENTS.md`](../AGENTS.md) and [`README-UMBRELLA.md`](../README-UMBRELLA.md).

## Global flags

| Flag | Env | Purpose |
|------|-----|---------|
| `--addr` | `MUXCORE_GRPC_ADDR` | muxcored gRPC address (default `127.0.0.1:9090`) |
| `--insecure` | `MUXCORE_INSECURE_DISABLE_TLS` | Disable TLS for local dev |
| `--token` | `MUXCORE_TOKEN` / `MUXCORE_ADMIN_TOKEN` | Bearer token for protected RPCs and HTTP module APIs |
| `--token-file` | `MUXCORE_TOKEN_FILE` | Read bearer token from file instead of argv |
| `--timeout` | — | Per-RPC timeout (default `15s`) |
| `--json` | — | Machine-readable JSON output |
| `--quiet` | — | Suppress success messages |
| `--yes` | — | Skip confirmation prompts |

Optional module URL overrides (when discovery is unavailable):

| Env | Module |
|-----|--------|
| `MUXCORE_AUTH_URL` | auth-local HTTP base |
| `MUXCORE_REQUEST_URL` | request-media HTTP base |
| `MUXCORE_PLAYBACK_MONITOR_URL` | playback-monitor HTTP base |
| `ADMIN_UI_PARENTAL_FILE` | Per-user parental controls JSON |
| `ADMIN_UI_BRANDING_FILE` | Branding settings file |
| `ADMIN_UI_NETWORKING_FILE` | Published URL / proxy settings |
| `ADMIN_UI_DATA_DIR` | Root for CLI/admin-ui JSON state (default `~/.muxcore/admin-ui`) |

## Shell completion

```bash
muxcorectl completion bash > /etc/bash_completion.d/muxcorectl
muxcorectl completion zsh  > "${fpath[1]}/_muxcorectl"
muxcorectl completion fish > ~/.config/fish/completions/muxcorectl.fish
```

## admin-ui parity

**216 HTTP routes** in `admin-ui/handler/handler.go` are mapped in `parity_routes_test.go` (222 including music/tagging stubs). CI verifies every route has a CLI equivalent or is documented as browser-only.

### Intentionally browser-only (no CLI equivalent)

| admin-ui | Reason |
|----------|--------|
| `/login`, `/logout` | Browser cookie session |
| Passkey **registration** | WebAuthn ceremony |
| `/auth/callback`, `/auth/status` | SSO redirect flow |
| `/branding.css` | Static asset |
| `POST /devices/{token}/revoke` | Revokes in-memory admin-ui session only |

For `/devices`, run `muxcorectl devices` for guidance; use `users tokens`, `keys list`, or `audit query` for related admin tasks.

### Command groups (admin-ui parity)

### Overview
| Command | admin-ui | Description |
|---------|----------|-------------|
| `health status` | Dashboard | Cluster leader, nodes, library count |
| `modules list` / `status` | Modules | Registered modules |
| `cluster status` | Cluster | Cluster membership |
| `lifecycle list` / `stop` / `restart` / `spawn` | Modules | Module lifecycle control |
| `marketplace spools` / `tags` / `deploy` | Marketplace | Spool browse and tag deploy |
| `spool resolve <tag>` | Marketplace | Inspect tag without deploying |

### Monitoring
| Command | admin-ui | Description |
|---------|----------|-------------|
| `events tail` / `stats` | Events | Stream or sample event bus |
| `health monitor` | Dashboard monitor | Health grid summary |
| `audit query` / `export` | Audit | Query or export audit log |
| `activity` | Activity | Cross-library grab/import activity |
| `logs list` / `tail` | Logs | Module log files |

### Library
| Command | admin-ui | Description |
|---------|----------|-------------|
| `metadata` | Metadata manager | List media library modules |
| `media libraries` / `items` / `get` / `missing` / `metadata` / `collections` / `tags` / `monitor` / `dispatch` / `refresh` / `delete` / `artwork` / `titles` | Media | Full library admin |
| `formats list` / `get` / `create` / `update` / `profiles` / `release-profiles` | Formats | Custom formats and profiles |
| `roots list` / `create` / `update` / `browse` / `delete` | Root Folders | Library roots |
| `rename templates` / `organize` | Naming | Naming templates and organize |

### Automation
| Command | admin-ui | Description |
|---------|----------|-------------|
| `request list` / `search` / `add` / `approve` / `deny` | Request | TMDB search and requests |
| `queue list` / `history` / `remove` / `retry-import` / `blocklist` | Queue | Wanted queue and import history |
| `automation search` / `dispatch` | Automation | Indexer search and grab dispatch |
| `calendar list` | Calendar | TV air-date calendar |
| `subtitles wanted` / `sync` / `profiles` / `media` / `mass-edit` | Subtitles | Subtitle management |
| `maintainer rules` / `candidates` / `scan` / `act` / `exclusions` | Maintainer | Library cleanup rules |
| `list-sync sources` / `sync` / `history` / `items` | List Sync | Trakt/list sync |
| `import candidates` / `path` | Import | Manual disk import |
| `migrate` | Migrate | Radarr/Sonarr library import |
| `schedules list` / `add` / … | Tasks | scheduler-cron HTTP API |
| `tasks list` / `cancel` | Tasks | Same as schedules (admin-ui naming) |

### Playback
| Command | admin-ui | Description |
|---------|----------|-------------|
| `jellyfin status` / `sync` / `refresh` | Jellyfin | Jellyfin bridge admin |
| `streams active` / `history` / `stats` / `users` / `libraries` / `servers` / `map` / `events` | Streams | Playback sessions and analytics |
| `streams guard` / `notifications` | Streams guard / notifications | playback-guard gRPC + monitor HTTP |
| `transcode profiles` / `setups` / `runs` / `approve` / `reject` / `apply-template` | Transcode | Transcode pipeline admin |
| `playback show` / `set` | Playback | Local playback policy JSON |
| `livetv show` / `set` | Live TV | Live TV guide settings |

### Access
| Command | admin-ui | Description |
|---------|----------|-------------|
| `users list` / `create` / `password` / `roles` / `totp` / `tokens` / `passkeys` / `parental` | Users | Local auth users |
| `devices` | Devices | Explains browser-session limitation |
| `keys list` / `revoke` | API Keys | Cross-user token catalog |
| `invites list` / `create` / `revoke` | Invites | Signup invite links |
| `auth` | Auth / SSO | Auth module overview |

### System
| Command | admin-ui | Description |
|---------|----------|-------------|
| `settings list` / `get` / `set` | Settings | Module settings via mesh |
| `storage ls` | Storage | Storage key listing |
| `backups list` / `create` / `delete` / `restore` | Backups | Backup archives |
| `config` | Config | MUXCORE_/ADMIN_UI_ environment |
| `plugins` | Plugins | Registered modules by capability |
| `branding show` / `set` | Branding | Admin UI branding file |
| `networking show` / `set` | Networking | Published URLs and proxies |

## Examples

**Settings (beginner-friendly tables):**
```bash
muxcorectl settings list
muxcorectl settings get auth-local session_timeout
muxcorectl settings set metadata-tmdb api_key "your-key"
```

**Media library:**
```bash
muxcorectl media libraries
muxcorectl media items media-movies --search "inception"
muxcorectl media missing media-tvshows
muxcorectl media refresh media-movies 42
```

**Scripting with JSON:**
```bash
muxcorectl --json users list | jq '.[].username'
muxcorectl --json --yes marketplace deploy media
```

**Destructive actions (prompts unless `--yes`):**
```bash
muxcorectl media delete media-movies 42 --delete-files   # prompts
muxcorectl --yes lifecycle stop media-scanner            # no prompt
```

## Develop

```bash
export PATH="$HOME/.local/go/bin:$PATH"
export GOPRIVATE=github.com/Muxcore-Media/*

go test ./...
# Verify all 216 admin-ui routes are mapped:
go test ./internal/cli -run TestAdminUIRoutesMatchHandler -v
# Live smoke (muxcored running):
MUXCORE_LIVE_TEST=1 MUXCORE_TOKEN=... go test ./internal/cli -run TestLiveSmoke -v
go build -o bin/muxcorectl ./cmd/muxcorectl
```

Integration dial (optional, needs a running muxcored):

```bash
MUXCORE_INTEGRATION=1 MUXCORE_INSECURE_DISABLE_TLS=true go test ./internal/connect -count=1 -v
```

## License

GPL-3.0 (same as MuxCore core).
