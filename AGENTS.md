# AGENTS.md — muxcorectl CLI

Operator CLI for the MuxCore mesh. Uses gRPC client SDK against core discovery.

Workspace deploy: [`../AGENTS.md`](../AGENTS.md).

## Build

```bash
cd muxcorectl-cli
go build -o muxcorectl ./cmd/muxcorectl
```

Vault smoke:

```bash
export MUXCORE_INSECURE_DISABLE_TLS=true MUXCORE_MESH_DIAL_LOCAL=true
export MUXCORE_GRPC_ADDR=127.0.0.1:9090
export MUXCORE_TOKEN="$(cat ../_mvp/run/admin.token)"
./muxcorectl health status
```

## Agent rules

- Parity tests in `internal/cli/parity_*` guard admin API route coverage — extend when adding CLI commands.
- Do not confuse with quarantined `muxcorectl/` workspace dump; this repo is canonical.
- Match existing cobra command structure under `internal/cli/`.
