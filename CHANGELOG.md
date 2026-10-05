# Changelog

## [0.1.1] - 2026-10-05

### Changed
- Builds against published module tags (core v0.6.2, admin-ui v0.1.13); CI and release from the umbrella templates.
- `list-sync update` only changes the fields you pass.

### Removed
- `list-sync update` flags `--clean-level`, `--tags`, `--monitor-mode`, `--min-availability`, `--search-on-add`, `--set-search-on-add` and the `removed=` sync output (not in media-list-sync v0.1.10).
- `subtitles profiles delete` (RPC removed from media-subtitles).
