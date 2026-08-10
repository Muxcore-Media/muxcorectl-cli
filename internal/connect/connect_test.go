package connect_test

import (
	"os"
	"testing"

	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
)

func TestFromEnvDefaults(t *testing.T) {
	t.Setenv("MUXCORE_GRPC_ADDR", "")
	t.Setenv("MUXCORE_INSECURE_DISABLE_TLS", "")
	t.Setenv("MUXCORE_TOKEN", "")
	t.Setenv("MUXCORE_ADMIN_TOKEN", "")
	// Clear may not empty if unset; use Unsetenv
	_ = os.Unsetenv("MUXCORE_GRPC_ADDR")
	_ = os.Unsetenv("MUXCORE_INSECURE_DISABLE_TLS")
	_ = os.Unsetenv("MUXCORE_TOKEN")
	_ = os.Unsetenv("MUXCORE_ADMIN_TOKEN")

	opts := connect.FromEnv()
	if opts.Addr != "127.0.0.1:9090" {
		t.Fatalf("addr=%q", opts.Addr)
	}
	if opts.Insecure {
		t.Fatal("expected insecure=false by default")
	}
}

func TestFromEnvInsecure(t *testing.T) {
	t.Setenv("MUXCORE_INSECURE_DISABLE_TLS", "true")
	t.Setenv("MUXCORE_GRPC_ADDR", "127.0.0.1:19090")
	opts := connect.FromEnv()
	if !opts.Insecure {
		t.Fatal("expected insecure")
	}
	if opts.Addr != "127.0.0.1:19090" {
		t.Fatalf("addr=%q", opts.Addr)
	}
}

func TestDialModulesListIntegration(t *testing.T) {
	if os.Getenv("MUXCORE_INTEGRATION") != "1" {
		t.Skip("set MUXCORE_INTEGRATION=1 against a running muxcored")
	}
	opts := connect.FromEnv()
	if !opts.Insecure {
		opts.Insecure = true
	}
	c, err := connect.Dial(opts)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	ctx, cancel := connect.Context(opts)
	defer cancel()
	members, leader, err := c.Discovery.Members(ctx)
	if err != nil {
		t.Fatalf("members: %v", err)
	}
	t.Logf("leader=%s members=%d", leader, len(members))
}
