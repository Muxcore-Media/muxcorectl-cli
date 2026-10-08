package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// adminUIRouteCoverage maps admin-ui HTTP routes to muxcorectl command paths.
// Empty string means intentionally unsupported (browser-only or no remote API).
var adminUIRouteCoverage = map[string]string{
	// Overview
	"GET /":                                    "health status",
	"GET /dashboard/health":                    "health status",
	"GET /dashboard/monitor":                   "health monitor",
	"GET /modules":                             "modules list",
	"GET /modules/{id}":                        "modules status",
	"GET /marketplace":                         "marketplace spools",
	"POST /marketplace/deploy":                 "marketplace deploy",
	"GET /marketplace/trust":                   "marketplace trust show",
	"POST /marketplace/trust/settings":         "marketplace trust settings",
	"POST /marketplace/trust/keys":             "marketplace trust add",
	"POST /marketplace/trust/keys/{id}/revoke": "marketplace trust revoke",
	"GET /cluster":                             "cluster status",
	"GET /cluster/nodes":                       "cluster status",
	"GET /cluster/sse":                         "cluster status",
	"GET /storage":                             "storage",

	// Monitoring
	"GET /events":        "events tail",
	"GET /events/stream": "events tail",
	"GET /events/stats":  "events stats",
	"GET /audit":         "audit query",
	"GET /activity":      "activity",
	"GET /logs":          "logs list",
	"GET /logs/partial":  "logs list",

	// Settings / config
	"GET /settings":                   "settings list",
	"POST /settings/{moduleID}/{key}": "settings set",
	"GET /config":                     "config",

	// Automation
	"GET /calendar":                            "calendar list",
	"GET /queue":                               "queue list",
	"POST /queue/remove":                       "queue remove",
	"POST /queue/retry-import":                 "queue retry-import",
	"POST /queue/blocklist":                    "queue blocklist add",
	"GET /automation":                          "queue list",
	"POST /automation/dispatch":                "automation dispatch",
	"POST /automation/queue/remove":            "queue remove",
	"POST /automation/blocklist/clear":         "automation blocklist clear",
	"POST /automation/delay":                   "automation delay set",
	"GET /request":                             "request list",
	"POST /request":                            "request add",
	"POST /request/{id}/approve":               "request approve",
	"POST /request/{id}/deny":                  "request deny",
	"GET /import":                              "import candidates",
	"POST /import":                             "import path",
	"GET /migrate":                             "migrate",
	"POST /migrate":                            "migrate",
	"GET /list-sync":                           "list-sync sources",
	"GET /list-sync/history":                   "list-sync history",
	"GET /list-sync/items":                     "list-sync items",
	"POST /list-sync/sync":                     "list-sync sync",
	"POST /list-sync/sources":                  "list-sync add",
	"POST /list-sync/sources/{id}":             "list-sync update",
	"POST /list-sync/sources/{id}/sync":        "list-sync sync",
	"POST /list-sync/sources/{id}/toggle":      "list-sync toggle",
	"POST /list-sync/sources/{id}/test":        "list-sync test",
	"POST /list-sync/sources/{id}/delete":      "list-sync delete",
	"GET /maintainer":                          "maintainer rules",
	"POST /maintainer/scan":                    "maintainer scan",
	"POST /maintainer/act":                     "maintainer act",
	"POST /maintainer/act/free-up":             "maintainer free-up",
	"POST /maintainer/exclusions":              "maintainer exclusions add",
	"POST /maintainer/exclusions/sync":         "maintainer exclusions sync",
	"POST /maintainer/exclusions/{id}/delete":  "maintainer exclusions delete",
	"POST /maintainer/rules":                   "maintainer rules-add",
	"GET /maintainer/rules/export":             "maintainer rules-export",
	"POST /maintainer/rules/import":            "maintainer rules-import",
	"POST /maintainer/rules/{id}/delete":       "maintainer rules-delete",
	"POST /maintainer/candidates/{id}/approve": "maintainer approve",
	"POST /maintainer/candidates/{id}/cancel":  "maintainer cancel",
	"GET /tasks":                               "tasks list",
	"POST /tasks/{id}/cancel":                  "tasks cancel",

	// Subtitles
	"GET /subtitles":                           "subtitles wanted",
	"POST /subtitles/sync":                     "subtitles sync",
	"POST /subtitles/search-wanted":            "subtitles search-wanted",
	"POST /subtitles/upgrade":                  "subtitles upgrade",
	"POST /subtitles/providers/{id}":           "subtitles providers toggle",
	"POST /subtitles/blacklist/{id}/delete":    "subtitles blacklist remove",
	"POST /subtitles/mass-edit":                "subtitles mass-edit",
	"POST /subtitles/history/clear":            "subtitles history clear",
	"POST /subtitles/download":                 "subtitles download",
	"POST /subtitles/profiles":                 "subtitles profiles create",
	"GET /subtitles/media/{id}":                "subtitles media get",
	"GET /subtitles/series/{id}":               "subtitles media get",
	"POST /subtitles/media/{id}/search-wanted": "subtitles media search",
	"POST /subtitles/media/{id}/profile":       "subtitles media set-profile",
	"POST /subtitles/files/{id}/delete":        "subtitles files delete",
	"POST /subtitles/test-arr":                 "subtitles test-arr",

	// Playback
	"GET /jellyfin":                       "jellyfin status",
	"POST /jellyfin/sync":                 "jellyfin sync",
	"POST /jellyfin/refresh":              "jellyfin refresh",
	"GET /playback":                       "playback show",
	"POST /playback":                      "playback set",
	"GET /streams":                        "streams active",
	"GET /streams/active.json":            "streams active",
	"GET /streams/history":                "streams history",
	"GET /streams/stats":                  "streams stats",
	"GET /streams/libraries":              "streams libraries",
	"GET /streams/users":                  "streams users",
	"GET /streams/servers":                "streams servers",
	"GET /transcode":                      "transcode profiles",
	"POST /transcode/save":                "transcode profiles",
	"POST /transcode/delete/{id}":         "transcode profiles",
	"POST /transcode/scan":                "transcode runs",
	"POST /transcode/review/{id}/approve": "transcode approve",
	"POST /transcode/review/{id}/reject":  "transcode reject",
	"GET /livetv":                         "livetv show",
	"POST /livetv":                        "livetv set",

	// Library
	"GET /metadata":                                                     "metadata",
	"GET /media/{moduleID}":                                             "media items",
	"GET /media/{moduleID}/missing":                                     "media missing",
	"GET /media/{moduleID}/tags":                                        "media tags list",
	"POST /media/{moduleID}/tags":                                       "media tags create",
	"GET /media/{moduleID}/collections":                                 "media collections list",
	"GET /media/{moduleID}/collections/{collectionID}":                  "media collections items",
	"POST /media/{moduleID}/collections/{collectionID}/monitor":         "media collections monitor",
	"POST /media/{moduleID}/collections/{collectionID}/sync":            "media collections sync",
	"GET /media/{moduleID}/calendar":                                    "calendar list",
	"GET /media/{moduleID}/item/{id}":                                   "media get",
	"POST /media/{moduleID}/item/{id}/dispatch":                         "media dispatch",
	"POST /media/{moduleID}/item/{id}/metadata":                         "media metadata",
	"POST /media/{moduleID}/item/{id}/refresh":                          "media refresh",
	"POST /media/{moduleID}/item/{id}/delete":                           "media delete",
	"POST /media/{moduleID}/item/{id}/files/{fileID}/delete":            "media files delete",
	"POST /media/{moduleID}/item/{id}/season/{seasonID}/monitor":        "media monitor season",
	"POST /media/{moduleID}/item/{id}/episode/{episodeID}/monitor":      "media monitor episode",
	"POST /media/{moduleID}/item/{id}/episode/{episodeID}/files/delete": "media files episode-delete",
	"POST /media/{moduleID}/item/{id}/titles":                           "media titles add",
	"POST /media/{moduleID}/item/{id}/titles/{titleID}/delete":          "media titles delete",
	"GET /media/{moduleID}/item/{id}/artwork":                           "media artwork",
	"GET /formats":                               "formats list",
	"POST /formats/sync-trash":                   "formats sync-trash",
	"POST /formats":                              "formats create",
	"GET /formats/profiles":                      "formats profiles list",
	"POST /formats/profiles":                     "formats profiles create",
	"POST /formats/profiles/{id}":                "formats profiles update",
	"POST /formats/profiles/{id}/delete":         "formats profiles delete",
	"GET /formats/release-profiles":              "formats release-profiles list",
	"POST /formats/release-profiles":             "formats release-profiles create",
	"POST /formats/release-profiles/{id}/delete": "formats release-profiles delete",
	"GET /formats/item/{id}":                     "formats get",
	"POST /formats/item/{id}":                    "formats update",
	"POST /formats/item/{id}/delete":             "formats delete",
	"GET /roots":                                 "roots list",
	"POST /roots":                                "roots create",
	"GET /roots/browse":                          "roots browse",
	"POST /roots/{id}":                           "roots update",
	"POST /roots/{id}/delete":                    "roots delete",
	"GET /rename/templates":                      "rename templates list",
	"POST /rename/templates":                     "rename templates create",
	"POST /rename/templates/{id}":                "rename templates update",
	"POST /rename/templates/{id}/delete":         "rename templates delete",
	"GET /rename/organize":                       "rename organize",
	"POST /rename/organize":                      "rename organize",

	// Access
	"GET /users":                           "users list",
	"POST /users":                          "users create",
	"DELETE /users/{id}":                   "users delete",
	"POST /users/{id}/password":            "users password",
	"POST /users/{id}/roles":               "users roles set",
	"GET /users/{id}/totp":                 "users totp status",
	"POST /users/{id}/totp":                "users totp enable",
	"GET /users/{id}/tokens":               "users tokens list",
	"POST /users/{id}/tokens":              "users tokens create",
	"DELETE /users/{id}/tokens/{tokenId}":  "users tokens delete",
	"GET /users/{id}/passkeys":             "users passkeys list",
	"DELETE /users/{id}/passkeys/{credId}": "users passkeys delete",
	"GET /invites":                         "invites list",
	"POST /invites":                        "invites create",
	"POST /invites/{id}/revoke":            "invites revoke",
	"GET /keys":                            "keys list",
	"POST /keys/{id}/revoke":               "keys revoke",
	"GET /auth":                            "auth",

	// System
	"GET /backups":               "backups list",
	"POST /backups/create":       "backups create",
	"POST /backups/{id}/delete":  "backups delete",
	"POST /backups/{id}/restore": "backups restore",
	"GET /plugins":               "plugins",
	"GET /branding":              "branding show",
	"POST /branding":             "branding set",
	"GET /networking":            "networking show",
	"POST /networking":           "networking set",

	// admin-ui v0.1.13 routes with no CLI command yet (parity pending; no CLI RPC wiring)
	"GET /acquisition-health":                   "",
	"GET /approvals":                            "",
	"GET /dashboard/now-playing":                "",
	"GET /failed-downloads":                     "",
	"GET /invite/redeem":                        "",
	"GET /library-scan":                         "",
	"GET /music/wanted":                         "",
	"GET /now-playing":                          "",
	"GET /password-resets":                      "",
	"GET /release-search":                       "",
	"GET /wanted":                               "",
	"GET /watchstats":                           "",
	"POST /activity/dismiss":                    "",
	"POST /activity/retry":                      "",
	"POST /approvals/{id}/approve":              "",
	"POST /approvals/{id}/deny":                 "",
	"POST /backups/schedule":                    "",
	"POST /devices/{token}/rename":              "",
	"POST /failed-downloads/dismiss":            "",
	"POST /failed-downloads/retry":              "",
	"POST /invite/redeem":                       "",
	"POST /keys/create":                         "",
	"POST /keys/{id}/rotate":                    "",
	"POST /library-scan/scan":                   "",
	"POST /maintainer/candidates/{id}/postpone": "",
	"POST /maintainer/collections":              "",
	"POST /maintainer/collections/{id}/delete":  "",
	"POST /maintainer/protections":              "",
	"POST /maintainer/protections/{id}/delete":  "",
	"POST /maintainer/rules/{id}/toggle":        "",
	"POST /password-resets/{id}/dismiss":        "",
	"POST /password-resets/{id}/password":       "",
	"POST /release-search/grab":                 "",

	// Music / tagging (admin-ui pages; CLI parity pending)
	"GET /music":                      "",
	"GET /music/{id}":                 "",
	"GET /ai":                         "ai status",
	"GET /tagging":                    "",
	"POST /tagging/tags":              "",
	"POST /tagging/rules":             "",
	"POST /tagging/rules/{id}/delete": "",

	// Intentionally unsupported (browser-only or in-memory admin-ui state)
	"GET /login":                   "",
	"POST /login":                  "",
	"GET /logout":                  "",
	"GET /devices":                 "devices",
	"POST /devices/{token}/revoke": "",
	"GET /api/auth/passkey/register/{id}/begin":     "",
	"POST /api/auth/passkey/register/{id}/complete": "",
	"GET /auth/callback":                            "",
	"GET /auth/status":                              "",
	"GET /branding.css":                             "",
	"GET /libraries":                                "media libraries",
	"GET /formats/new":                              "formats create",
	"GET /roots/new":                                "roots create",
	"GET /rename/templates/new":                     "rename templates create",
	"GET /users/create-form":                        "users create",
	"GET /list-sync/sources/{id}/edit":              "list-sync update",
	"GET /transcode/edit":                           "transcode profiles",
	"GET /users/{id}/detail":                        "users list",
	"GET /formats/profiles/new":                     "formats profiles create",
	"GET /formats/profiles/{id}":                    "formats profiles list",
	"GET /roots/{id}":                               "roots list",
	"GET /rename/templates/{id}":                    "rename templates list",

	"GET /streams/guard":                              "streams guard show",
	"POST /streams/guard/acknowledge":                 "streams guard acknowledge",
	"POST /streams/guard/trust/reset":                 "streams guard trust reset",
	"POST /streams/guard/merge":                       "streams guard merge",
	"POST /streams/guard/terminate":                   "streams guard terminate",
	"GET /streams/notifications":                      "streams notifications rules list",
	"POST /streams/notifications/create":              "streams notifications rules create",
	"POST /streams/notifications/delete":              "streams notifications rules delete",
	"POST /streams/notifications/destinations/create": "streams notifications destinations create",
	"POST /streams/notifications/destinations/delete": "streams notifications destinations delete",
	"POST /streams/notifications/destinations/test":   "streams notifications destinations test",
	"GET /streams/map":                                "streams map",
	"GET /streams/map/data":                           "streams map",
	"GET /users/{id}/parental":                        "users parental show",

	// ADR-0031: restrictions live in userdata-local and are edited in admin-ui;
	// `users parental set` was removed (it only fails with guidance).
	"POST /users/{id}/parental": "",

	// SSE live feed — use active session poll instead of browser SSE proxy
	"GET /streams/events":                 "streams events",
	"POST /transcode/apply-template/{id}": "transcode apply-template",
}

func TestAdminUIRouteCoverage(t *testing.T) {
	root := NewRoot()
	var missing []string
	for route, want := range adminUIRouteCoverage {
		if want == "" {
			continue
		}
		if !commandPathExists(root, want) {
			missing = append(missing, route+" -> "+want)
		}
	}
	if len(missing) > 0 {
		t.Errorf("missing CLI coverage for %d admin-ui routes:\n%s", len(missing), strings.Join(missing, "\n"))
	}
}

func TestAdminUIRouteCoverageCompleteness(t *testing.T) {
	handler := handlerRoutes(t)
	if len(handler) != len(adminUIRouteCoverage) {
		t.Fatalf("handler routes=%d coverage entries=%d (run TestAdminUIRoutesMatchHandler for details)", len(handler), len(adminUIRouteCoverage))
	}
}

func commandPathExists(root *cobra.Command, path string) bool {
	parts := strings.Fields(path)
	cur := root
	for _, part := range parts {
		found := false
		for _, c := range cur.Commands() {
			if c.Name() == part {
				cur = c
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
