package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLiveSmoke runs read-only muxcorectl commands against a running muxcored.
// Enable with MUXCORE_LIVE_TEST=1 and the usual MUXCORE_* env vars.
func TestLiveSmoke(t *testing.T) {
	if os.Getenv("MUXCORE_LIVE_TEST") == "" {
		t.Skip("set MUXCORE_LIVE_TEST=1 to run live smoke tests")
	}

	bin := os.Getenv("MUXCORECTL_BIN")
	if bin == "" {
		bin = filepath.Join("..", "..", "bin", "muxcorectl")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("muxcorectl binary %q: %v", bin, err)
	}

	env := append(os.Environ(),
		"MUXCORE_INSECURE_DISABLE_TLS=true",
		"MUXCORE_MESH_DIAL_LOCAL=true",
	)
	if v := os.Getenv("MUXCORE_GRPC_ADDR"); v == "" {
		env = append(env, "MUXCORE_GRPC_ADDR=127.0.0.1:9090")
	}
	if v := os.Getenv("MUXCORE_TOKEN"); v == "" {
		for _, p := range []string{
			filepath.Join("..", "..", "..", "_mvp", "run", "admin.token"),
			filepath.Join("..", "..", "_mvp", "run", "admin.token"),
		} {
			if tok, err := os.ReadFile(p); err == nil {
				env = append(env, "MUXCORE_TOKEN="+strings.TrimSpace(string(tok)))
				break
			}
		}
	}

	commands := [][]string{
		{"health", "status"},
		{"modules", "list"},
		{"cluster", "status"},
		{"settings", "list"},
		{"users", "list"},
		{"media", "libraries"},
		{"metadata"},
		{"formats", "list"},
		{"roots", "list"},
		{"queue", "list"},
		{"calendar", "list"},
		{"plugins"},
		{"config"},
		{"devices"},
	}

	for _, args := range commands {
		args := args
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := exec.Command(bin, args...)
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s: %v\n%s", strings.Join(args, " "), err, out)
			}
		})
	}
}
