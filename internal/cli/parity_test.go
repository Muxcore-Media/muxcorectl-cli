package cli

import (
	"testing"
)

func TestRootRegistersAdminCommands(t *testing.T) {
	root := NewRoot()
	want := []string{
		"health", "modules", "cluster", "lifecycle", "marketplace", "spool",
		"events", "logs", "audit", "activity", "media", "metadata", "formats", "roots", "rename",
		"request", "queue", "automation", "calendar", "subtitles", "migrate",
		"schedules", "tasks", "settings", "users", "devices", "invites", "keys", "auth",
		"storage", "backups", "config", "plugins", "branding", "networking", "jellyfin",
		"maintainer", "list-sync", "import", "streams", "transcode",
		"playback", "livetv",
	}
	seen := map[string]bool{}
	for _, c := range root.Commands() {
		seen[c.Name()] = true
	}
	for _, name := range want {
		if !seen[name] {
			t.Errorf("missing top-level command %q", name)
		}
	}
}

func TestRootHelpGroups(t *testing.T) {
	root := NewRoot()
	var groups []string
	for _, g := range root.Groups() {
		groups = append(groups, g.ID)
	}
	for _, id := range []string{
		groupOverview, groupMonitoring, groupLibrary, groupAutomation,
		groupPlayback, groupAccess, groupSystem,
	} {
		found := false
		for _, g := range groups {
			if g == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing command group %q", id)
		}
	}
}
