package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func healthMonitorURL() string {
	if v := strings.TrimRight(envOr("ADMIN_UI_HEALTH_MONITOR_URL", "http://127.0.0.1:9203"), "/"); v != "" {
		return v
	}
	return "http://127.0.0.1:9203"
}

var flagHealthMonitorToken string

var healthMonitorHTTPClient = &http.Client{Timeout: 10 * time.Second}

// resolveHealthMonitorToken returns --health-monitor-token, else
// MUXCORE_HEALTH_MONITOR_TOKEN, else HEALTH_MONITOR_HTTP_TOKEN.
func resolveHealthMonitorToken() string {
	return firstToken(flagHealthMonitorToken, "MUXCORE_HEALTH_MONITOR_TOKEN", "HEALTH_MONITOR_HTTP_TOKEN")
}

func newHealthMonitorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Fetch health monitor summary (admin-ui /dashboard/monitor parity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := healthMonitorURL()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/status", http.NoBody)
			if err != nil {
				return err
			}
			if tok := resolveHealthMonitorToken(); tok != "" {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
			resp, err := healthMonitorHTTPClient.Do(req)
			if err != nil {
				return fmt.Errorf("health monitor: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if err != nil {
				return err
			}
			if resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf("health monitor HTTP 401 unauthorized: a bearer token is required; pass --health-monitor-token or set MUXCORE_HEALTH_MONITOR_TOKEN (or HEALTH_MONITOR_HTTP_TOKEN)")
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("health monitor HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
			}
			var st map[string]any
			if err := json.Unmarshal(body, &st); err != nil {
				return fmt.Errorf("health monitor JSON: %w", err)
			}
			if flagJSON {
				return printJSON(st)
			}
			fmt.Printf("status:       %v\n", st["status"])
			fmt.Printf("modules:      %v\n", st["module_count"])
			fmt.Printf("stale:        %v\n", st["stale_count"])
			fmt.Printf("degraded:     %v\n", st["degraded_transitions"])
			fmt.Printf("events_pub:   %v\n", st["events_published"])
			return nil
		},
	}
	cmd.Flags().StringVar(&flagHealthMonitorToken, "health-monitor-token", "", "health-monitor bearer token (prefer MUXCORE_HEALTH_MONITOR_TOKEN or HEALTH_MONITOR_HTTP_TOKEN env)")
	return cmd
}
