# muxcorectl

Operator CLI for a running [MuxCore](https://github.com/Muxcore-Media/core) (`muxcored`) node.

Clean replacement for the archived polluted `muxcorectl` dump — this repo is `muxcorectl-cli` on disk/GitHub so it never collides with that tree.

Talks to `muxcored` over gRPC via `github.com/Muxcore-Media/core/sdk/go/client` (**pinned to core `v0.5.0`**).

## Laptop usage (with `_mvp`)

With the host stack up (`_mvp/run-host.sh`), muxcored listens on `127.0.0.1:9090` with TLS disabled:

```bash
export MUXCORE_INSECURE_DISABLE_TLS=true
export MUXCORE_GRPC_ADDR=127.0.0.1:9090   # optional; this is the default

go install github.com/Muxcore-Media/muxcorectl-cli/cmd/muxcorectl@latest
# or from a checkout:
go build -o muxcorectl ./cmd/muxcorectl

muxcorectl version
muxcorectl modules list
muxcorectl modules status auth-local
muxcorectl cluster status
```

Equivalent flags (no env):

```bash
muxcorectl --insecure --addr 127.0.0.1:9090 modules list
```

RPCs that are not on the public discovery allowlist (storage / audit / spool) need a bearer session token from `auth-local`:

```bash
export MUXCORE_TOKEN="$(cat ../_mvp/run/admin.token)"   # path may vary
muxcorectl --insecure storage ls
muxcorectl --insecure audit query --max 20
muxcorectl --insecure spool resolve default
```

## Commands

| Command | Notes |
|---------|--------|
| `muxcorectl version` | CLI version |
| `muxcorectl modules list` | `Discovery.ListAll` |
| `muxcorectl modules status [id]` | Resolve one or summarize all |
| `muxcorectl cluster status` | Members + leader |
| `muxcorectl events tail [--type=*]` | Subscribe until Ctrl-C |
| `muxcorectl storage ls [prefix]` | Requires auth when authorizer is wired |
| `muxcorectl audit query` | Requires auth when authorizer is wired |
| `muxcorectl spool resolve <tag>` | `SpoolService.FetchTag` |

## Build / test

```bash
export GOPRIVATE=github.com/Muxcore-Media/*
go test ./...
go build -o muxcorectl ./cmd/muxcorectl

# optional live check against local muxcored
MUXCORE_INTEGRATION=1 MUXCORE_INSECURE_DISABLE_TLS=true go test ./internal/connect -run Integration -v
```

## License

GPL-3.0 (same as MuxCore core).
