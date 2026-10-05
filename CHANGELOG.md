# Changelog

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
