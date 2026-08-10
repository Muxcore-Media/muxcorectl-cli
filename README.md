# muxcorectl

Operator CLI for a running MuxCore (`muxcored`) node. Clean rewrite — do **not** confuse with the archived `muxcorectl` dump.

Talks to muxcored over gRPC (core SDK client). For laptop / fixture stacks, dial with insecure-dev TLS disabled.

## Install

```bash
go install github.com/Muxcore-Media/muxcorectl-cli/cmd/muxcorectl@latest
```

Or build from this repo:

```bash
go build -o bin/muxcorectl ./cmd/muxcorectl
```

Requires Go 1.26+ and access to `github.com/Muxcore-Media/core` (`GOPRIVATE=github.com/Muxcore-Media/*` for private modules).

## Laptop usage

Against a local host stack (`_mvp` / `muxcore-installer`) listening on `127.0.0.1:9090`:

```bash
export MUXCORE_INSECURE_DISABLE_TLS=true
# optional: export MUXCORE_GRPC_ADDR=127.0.0.1:9090
# optional: export MUXCORE_TOKEN=…   # or MUXCORE_ADMIN_TOKEN

muxcorectl version
muxcorectl modules list
muxcorectl modules status
muxcorectl cluster status
```

Equivalent flags (override env):

```bash
muxcorectl --insecure --addr 127.0.0.1:9090 modules list
```

RPCs outside the public discovery allowlist (`storage`, `audit`, `spool`) need a bearer token from `auth-local`:

```bash
export MUXCORE_TOKEN="$(cat ../_mvp/run/admin.token)"   # path may vary
muxcorectl --insecure storage ls
muxcorectl --insecure audit query --max 20
muxcorectl --insecure spool resolve media
muxcorectl --insecure schedules list
muxcorectl --insecure schedules add --name hourly-ping --cron '@hourly' --webhook http://127.0.0.1:9999/hook
```

Override discovery when needed:

```bash
muxcorectl schedules list --scheduler-url http://127.0.0.1:9200
# or: export SCHEDULER_URL=http://127.0.0.1:9200
```

## Commands

| Command | Description |
|---------|-------------|
| `version` | Print CLI version |
| `modules list` | List modules (`Discovery.ListAll`) |
| `modules status [id]` | One module or all |
| `cluster status` | Members + leader |
| `events tail` | Subscribe to events (`--type`, `--max`) |
| `storage ls [prefix]` | List storage keys |
| `audit query` | Query audit log |
| `spool resolve <tag>` | Fetch spool tag without deploying |
| `schedules list` / `status` / `add` / `cancel` | Manage `scheduler-cron` via its HTTP API (`--scheduler-url` / `SCHEDULER_URL`, or discover `HttpAddr`) |

## Flags / env

| Flag | Env | Default |
|------|-----|---------|
| `--addr` | `MUXCORE_GRPC_ADDR` | `127.0.0.1:9090` |
| `--insecure` | `MUXCORE_INSECURE_DISABLE_TLS` | off |
| `--token` | `MUXCORE_TOKEN` / `MUXCORE_ADMIN_TOKEN` | empty |
| `--timeout` | — | `15s` |

## Develop

```bash
export PATH="$HOME/.local/go/bin:$PATH"
export GOPRIVATE=github.com/Muxcore-Media/*

go test ./...
go build -o bin/muxcorectl ./cmd/muxcorectl
```

Integration dial (optional, needs a running muxcored):

```bash
MUXCORE_INTEGRATION=1 MUXCORE_INSECURE_DISABLE_TLS=true go test ./internal/connect -count=1 -v
```

## License

GPL-3.0 (same as MuxCore core).
