# Changelog

## [Unreleased]

## [0.1.5] - 2026-10-10

### Added
- `users erasures [erasure-id]` shows user-erasure completion from the identity provider's `GetUserErasureStatus` (ADR-0035 E9): one row per erasure and module with `ok` / `failed` / `unsupported` / `pending` outcomes, `--all` for completed erasures too, `--json` for scripts. It sends the administrator token in `x-auth-token` and never calls `ListUserErasures` / `AckUserErasure`, which admit only an allowlisted module certificate.

### Changed
- Built on core v0.6.17 (auth `GetUserErasureStatus`, `DeleteUserResponse.erasure_id`).
- `users delete` also sends the operator token in `x-auth-token`, which ADR-0035 §1 requires for `DeleteUser`, and prints the returned `erasure_id`. Delete semantics are unchanged.
- `users parental show` is read-only and labelled as the LEGACY, NON-AUTHORITATIVE `parental.json` (not what is enforced). It now shows `kids_mode` and whether a `pin_hash` is present (the hash is never printed), and a missing, unreadable or corrupt file is an explicit error instead of an empty result (ADR-0031).

### Removed
- `users parental set` no longer writes `parental.json`: it fails with a non-zero exit and points to admin-ui (`/users/{id}/parental`, one-time import at `/users/parental/migrate`). Parental restrictions are authoritative in userdata-local (ADR-0030) and enforced by the BFF (ADR-0031); the old write reported success for a file nothing enforces and dropped `kids_mode` and `pin_hash` from it.

## [0.1.4] - 2026-10-05

### Changed
- Built on core v0.6.14 / sdk/go/module v0.6.4: unregisters on shutdown and re-registers after core restarts (ADR-0022).

## [0.1.3] - 2026-10-05


### Changed
- Bump core v0.6.12, sdk/go/client v0.6.1 and all sibling module requires to their latest tags (T-M3-03f).

## [0.1.2] - 2026-10-05


### Added
- `schedules --scheduler-token` and `health monitor --health-monitor-token` (env `MUXCORE_SCHEDULER_TOKEN`/`SCHEDULER_HTTP_TOKEN`, `MUXCORE_HEALTH_MONITOR_TOKEN`/`HEALTH_MONITOR_HTTP_TOKEN`) send `Authorization: Bearer` to scheduler-cron v0.1.7+ and health-monitor when bound off-loopback; HTTP 401 now reports how to supply the token.

### Changed
- scheduler-cron and health-monitor HTTP calls use a client with a timeout instead of `http.DefaultClient`.

## [0.1.1] - 2026-10-05

### Changed
- Builds against published module tags (core v0.6.2, admin-ui v0.1.13); CI and release from the umbrella templates.
- `list-sync update` only changes the fields you pass.

### Removed
- `list-sync update` flags `--clean-level`, `--tags`, `--monitor-mode`, `--min-availability`, `--search-on-add`, `--set-search-on-add` and the `removed=` sync output (not in media-list-sync v0.1.10).
- `subtitles profiles delete` (RPC removed from media-subtitles).
