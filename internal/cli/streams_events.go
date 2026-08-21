package cli

import (
	"context"
	"encoding/json"
	"fmt"

	monitorv1 "github.com/Muxcore-Media/playback-monitor/proto/monitorv1"
	"github.com/spf13/cobra"
)

func newStreamsEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "events",
		Short: "Active sessions JSON (admin-ui /streams/events and /streams/active.json parity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlaybackMonitorClient(func(ctx context.Context, cli monitorv1.PlaybackMonitorServiceClient) error {
				resp, err := cli.ListActiveSessions(ctx, &monitorv1.ListActiveSessionsRequest{Limit: 100})
				if err != nil {
					return fmt.Errorf("streams events: %w", err)
				}
				type row struct {
					User     string `json:"user"`
					Title    string `json:"title"`
					State    string `json:"state"`
					Platform string `json:"platform"`
					Player   string `json:"player"`
					IP       string `json:"ip"`
				}
				rows := make([]row, 0, len(resp.GetSessions()))
				for _, s := range resp.GetSessions() {
					rows = append(rows, row{
						User:     sessionUser(s),
						Title:    s.GetTitle(),
						State:    s.GetState().String(),
						Platform: s.GetPlatform(),
						Player:   s.GetPlayer(),
						IP:       s.GetIpAddress(),
					})
				}
				out := map[string]any{"sessions": rows, "count": len(rows)}
				if flagJSON {
					return printJSON(out)
				}
				raw, _ := json.MarshalIndent(out, "", "  ")
				fmt.Println(string(raw))
				return nil
			})
		},
	}
}
