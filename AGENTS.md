# AGENTS.md — muxcorectl CLI

Operator CLI for the MuxCore mesh. Uses gRPC client SDK against core discovery.

Workspace deploy: [`../AGENTS.md`](../AGENTS.md).

## Build

```bash
cd muxcorectl-cli
go build -o muxcorectl ./cmd/muxcorectl
```

Vault smoke (SSH wrapper — mesh is local on vault):

```bash
../_mvp/scripts/muxcorectl-vault.sh health status
../_mvp/scripts/smoke-vault-all.sh
```

Or on vault directly after `deploy-module-to-vault.sh muxcorectl --verify-all`:

```bash
export MUXCORE_INSECURE_DISABLE_TLS=true MUXCORE_MESH_DIAL_LOCAL=true
export MUXCORE_GRPC_ADDR=127.0.0.1:9090
export MUXCORE_TOKEN="$(cat /mnt/fast-storage/appdata/muxcore/mvp/run/admin.token)"
muxcorectl health status
```

## Agent rules

- Parity tests in `internal/cli/parity_*` guard admin API route coverage — extend when adding CLI commands.
- Do not confuse with quarantined `muxcorectl/` workspace dump; this repo is canonical.
- Match existing cobra command structure under `internal/cli/`.
- Roadmaps, task lists, and remaining-work checklists live in workspace [`MASTER-ROADMAP.md`](../MASTER-ROADMAP.md) and umbrella GitHub Issues. Do not add `ROADMAP.md` / `TASKS.md` in this repo.
