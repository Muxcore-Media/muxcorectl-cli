package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func runHealthMonitor(t *testing.T, url string, args ...string) error {
	t.Helper()
	t.Setenv("ADMIN_UI_HEALTH_MONITOR_URL", url)
	cmd := newHealthMonitorCmd()
	if len(args) > 0 {
		if err := cmd.Flags().Set("health-monitor-token", args[0]); err != nil {
			t.Fatal(err)
		}
	}
	return cmd.RunE(cmd, nil)
}

func TestHealthMonitorTokenHeader(t *testing.T) {
	var got string
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MUXCORE_HEALTH_MONITOR_TOKEN", "")
	t.Setenv("HEALTH_MONITOR_HTTP_TOKEN", "")
	flagHealthMonitorToken = ""
	t.Cleanup(func() { flagHealthMonitorToken = "" })

	if err := runHealthMonitor(t, srv.URL); err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected no Authorization, got %q", got)
	}
	t.Setenv("HEALTH_MONITOR_HTTP_TOKEN", "fallback")
	_ = runHealthMonitor(t, srv.URL)
	if got != "Bearer fallback" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("MUXCORE_HEALTH_MONITOR_TOKEN", "primary")
	_ = runHealthMonitor(t, srv.URL)
	if got != "Bearer primary" {
		t.Fatalf("got %q", got)
	}
	_ = runHealthMonitor(t, srv.URL, "flagtok")
	if got != "Bearer flagtok" {
		t.Fatalf("got %q", got)
	}

	status = http.StatusUnauthorized
	err := runHealthMonitor(t, srv.URL)
	if err == nil || !strings.Contains(err.Error(), "--health-monitor-token") || !strings.Contains(err.Error(), "MUXCORE_HEALTH_MONITOR_TOKEN") {
		t.Fatalf("err=%v", err)
	}
}
