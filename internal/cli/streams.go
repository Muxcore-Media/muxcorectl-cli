package cli

import (
	"context"
	"fmt"
	"time"

	monitorv1 "github.com/Muxcore-Media/playback-monitor/proto/monitorv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capPlaybackMonitor = "playback.monitor"

func withPlaybackMonitorClient(fn func(context.Context, monitorv1.PlaybackMonitorServiceClient) error) error {
	return withModuleConn(capPlaybackMonitor, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, monitorv1.NewPlaybackMonitorServiceClient(conn))
	})
}

func newStreamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "streams",
		Short:   "Active playback sessions and history (admin-ui /streams parity)",
		GroupID: groupPlayback,
	}
	cmd.AddCommand(newStreamsActiveCmd())
	cmd.AddCommand(newStreamsHistoryCmd())
	cmd.AddCommand(newStreamsStatsCmd())
	cmd.AddCommand(newStreamsUsersCmd())
	cmd.AddCommand(newStreamsLibrariesCmd())
	cmd.AddCommand(newStreamsServersCmd())
	cmd.AddCommand(newStreamsGuardCmd())
	cmd.AddCommand(newStreamsNotificationsCmd())
	cmd.AddCommand(newStreamsMapCmd())
	cmd.AddCommand(newStreamsEventsCmd())
	return cmd
}

func sessionUser(s *monitorv1.SessionRecord) string {
	if s.GetUserName() != "" {
		return s.GetUserName()
	}
	return s.GetUserId()
}

func sessionStarted(s *monitorv1.SessionRecord) string {
	if s.GetStartedAtUnix() <= 0 {
		return ""
	}
	return time.Unix(s.GetStartedAtUnix(), 0).UTC().Format(time.RFC3339)
}

func newStreamsActiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "active",
		Short: "List active playback sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlaybackMonitorClient(func(ctx context.Context, cli monitorv1.PlaybackMonitorServiceClient) error {
				resp, err := cli.ListActiveSessions(ctx, &monitorv1.ListActiveSessionsRequest{Limit: 50})
				if err != nil {
					return fmt.Errorf("streams active: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetSessions())
				}
				rows := make([][]string, 0, len(resp.GetSessions()))
				for _, s := range resp.GetSessions() {
					rows = append(rows, []string{s.GetId(), sessionUser(s), s.GetTitle(), s.GetState().String(), sessionStarted(s)})
				}
				return printTable([]string{"ID", "USER", "TITLE", "STATE", "STARTED"}, rows)
			})
		},
	}
}

func newStreamsHistoryCmd() *cobra.Command {
	var query string
	var limit int32
	cmd := &cobra.Command{
		Use:   "history",
		Short: "List playback history",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 {
				limit = 50
			}
			return withPlaybackMonitorClient(func(ctx context.Context, cli monitorv1.PlaybackMonitorServiceClient) error {
				resp, err := cli.ListHistory(ctx, &monitorv1.ListHistoryRequest{Query: query, Limit: limit})
				if err != nil {
					return fmt.Errorf("streams history: %w", err)
				}
				if flagJSON {
					return printJSON(map[string]any{"total": resp.GetTotal(), "sessions": resp.GetSessions()})
				}
				rows := make([][]string, 0, len(resp.GetSessions()))
				for _, s := range resp.GetSessions() {
					rows = append(rows, []string{s.GetId(), sessionUser(s), s.GetTitle(), sessionStarted(s)})
				}
				fmt.Printf("total=%d\n", resp.GetTotal())
				return printTable([]string{"ID", "USER", "TITLE", "STARTED"}, rows)
			})
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "search filter")
	cmd.Flags().Int32Var(&limit, "limit", 50, "max rows")
	return cmd
}

func newStreamsStatsCmd() *cobra.Command {
	var days int32
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Home dashboard playback stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			if days < 1 {
				days = 30
			}
			return withPlaybackMonitorClient(func(ctx context.Context, cli monitorv1.PlaybackMonitorServiceClient) error {
				resp, err := cli.GetHomeStats(ctx, &monitorv1.GetHomeStatsRequest{Days: days})
				if err != nil {
					return fmt.Errorf("streams stats: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetStats())
				}
				rows := make([][]string, 0, len(resp.GetStats()))
				for _, s := range resp.GetStats() {
					rows = append(rows, []string{s.GetKey(), s.GetLabel(), fmt.Sprintf("%.0f", s.GetValue())})
				}
				return printTable([]string{"KEY", "LABEL", "VALUE"}, rows)
			})
		},
	}
	cmd.Flags().Int32Var(&days, "days", 30, "lookback window in days")
	return cmd
}

func newStreamsUsersCmd() *cobra.Command {
	var days, limit int32
	cmd := &cobra.Command{
		Use:   "users",
		Short: "User watch statistics (admin-ui /streams/users parity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if days < 1 {
				days = 30
			}
			if limit < 1 {
				limit = 100
			}
			return withPlaybackMonitorClient(func(ctx context.Context, cli monitorv1.PlaybackMonitorServiceClient) error {
				resp, err := cli.ListUserWatchStats(ctx, &monitorv1.ListUserWatchStatsRequest{Days: days, Limit: limit})
				if err != nil {
					return fmt.Errorf("streams users: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetUsers())
				}
				rows := make([][]string, 0, len(resp.GetUsers()))
				for _, u := range resp.GetUsers() {
					rows = append(rows, []string{u.GetUsername(), fmt.Sprintf("%d", u.GetPlayCount()), fmt.Sprintf("%.0f", u.GetWatchMinutes())})
				}
				return printTable([]string{"USER", "PLAYS", "WATCH_MIN"}, rows)
			})
		},
	}
	cmd.Flags().Int32Var(&days, "days", 30, "lookback window in days")
	cmd.Flags().Int32Var(&limit, "limit", 100, "max users")
	return cmd
}

func newStreamsLibrariesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "libraries",
		Short: "Library storage summary (admin-ui /streams/libraries parity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlaybackMonitorClient(func(ctx context.Context, cli monitorv1.PlaybackMonitorServiceClient) error {
				resp, err := cli.GetLibraryStorageSummary(ctx, &monitorv1.GetLibraryStorageSummaryRequest{})
				if err != nil {
					return fmt.Errorf("streams libraries: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				fmt.Printf("total_bytes=%d items=%d duplicate_waste=%d\n",
					resp.GetTotalBytes(), resp.GetTotalItems(), resp.GetDuplicateWasteBytes())
				rows := make([][]string, 0, len(resp.GetLibraries()))
				for _, lib := range resp.GetLibraries() {
					rows = append(rows, []string{lib.GetLibraryName(), lib.GetServerId(), fmt.Sprintf("%d", lib.GetItemCount()), fmt.Sprintf("%d", lib.GetTotalBytes())})
				}
				return printTable([]string{"LIBRARY", "SERVER", "ITEMS", "BYTES"}, rows)
			})
		},
	}
	return cmd
}

func newStreamsServersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "servers",
		Short: "Registered playback servers (admin-ui /streams/servers parity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlaybackMonitorClient(func(ctx context.Context, cli monitorv1.PlaybackMonitorServiceClient) error {
				resp, err := cli.ListServers(ctx, &monitorv1.ListServersRequest{})
				if err != nil {
					return fmt.Errorf("streams servers: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetServers())
				}
				rows := make([][]string, 0, len(resp.GetServers()))
				for _, s := range resp.GetServers() {
					rows = append(rows, []string{s.GetId(), s.GetName(), s.GetType(), s.GetSourceModule()})
				}
				return printTable([]string{"ID", "NAME", "TYPE", "SOURCE"}, rows)
			})
		},
	}
	return cmd
}
