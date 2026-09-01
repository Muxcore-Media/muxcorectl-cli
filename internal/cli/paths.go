package cli

import (
	"os"
	"path/filepath"
)

// adminDataDir mirrors admin-ui/handler/paths.go AdminDataDir defaults so CLI
// edits land in the same JSON files as the web UI and run-host.sh.
func adminDataDir() string {
	if v := os.Getenv("ADMIN_UI_DATA_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("MEDIA_UI_USERDATA_DIR"); v != "" {
		return v
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".muxcore", "admin-ui")
	}
	return filepath.Join(os.TempDir(), "muxcore-admin-ui")
}

func adminDataFile(envKey, basename string) string {
	return envOr(envKey, filepath.Join(adminDataDir(), basename))
}
