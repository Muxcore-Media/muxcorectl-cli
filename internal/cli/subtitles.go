package cli

import (
	"context"
	"fmt"

	subtv1 "github.com/Muxcore-Media/media-subtitles/proto/subtv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capMediaSubtitles = "media.subtitles"

func withSubtitlesClient(fn func(context.Context, subtv1.SubtitleServiceClient) error) error {
	return withModuleConn(capMediaSubtitles, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, subtv1.NewSubtitleServiceClient(conn))
	})
}

func newSubtitlesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subtitles",
		Short:   "Subtitle wanted list and sync (admin-ui /subtitles parity)",
		GroupID: groupAutomation,
	}
	cmd.AddCommand(newSubtitlesWantedCmd())
	cmd.AddCommand(newSubtitlesSyncCmd())
	cmd.AddCommand(newSubtitlesSearchWantedCmd())
	cmd.AddCommand(newSubtitlesProvidersCmd())
	cmd.AddCommand(newSubtitlesBlacklistCmd())
	cmd.AddCommand(newSubtitlesHistoryCmd())
	cmd.AddCommand(newSubtitlesProfilesCmd())
	cmd.AddCommand(newSubtitlesMediaCmd())
	cmd.AddCommand(newSubtitlesMassEditCmd())
	cmd.AddCommand(newSubtitlesUpgradeCmd())
	cmd.AddCommand(newSubtitlesFilesCmd())
	cmd.AddCommand(newSubtitlesDownloadCmd())
	cmd.AddCommand(newSubtitlesTestArrCmd())
	return cmd
}

func newSubtitlesWantedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wanted",
		Short: "List wanted subtitles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.ListWanted(ctx, &subtv1.ListWantedRequest{Page: 1, PageSize: 50})
				if err != nil {
					return fmt.Errorf("subtitles wanted: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				rows := make([][]string, 0, len(resp.GetItems()))
				for _, it := range resp.GetItems() {
					rows = append(rows, []string{
						it.GetId(), it.GetTitle(), it.GetLanguage(), it.GetMediaType(),
						fmt.Sprintf("S%dE%d", it.GetSeason(), it.GetEpisode()),
					})
				}
				fmt.Printf("total=%d\n", resp.GetTotal())
				return printTable([]string{"ID", "TITLE", "LANG", "TYPE", "EP"}, rows)
			})
		},
	}
}

func newSubtitlesSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync subtitle library from media modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.SyncLibrary(ctx, &subtv1.SyncLibraryRequest{})
				if err != nil {
					return fmt.Errorf("subtitles sync: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("synced: movies=%d episodes=%d wanted=%d\n",
						resp.GetMovies(), resp.GetEpisodes(), resp.GetWanted())
				}
				return nil
			})
		},
	}
}

func newSubtitlesSearchWantedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search-wanted",
		Short: "Search providers for all wanted subtitles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.SearchWanted(ctx, &subtv1.SearchWantedRequest{})
				if err != nil {
					return fmt.Errorf("subtitles search-wanted: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("searched=%d downloaded=%d\n", resp.GetSearched(), resp.GetDownloaded())
				}
				return nil
			})
		},
	}
}

func newSubtitlesProvidersCmd() *cobra.Command {
	toggle := &cobra.Command{
		Use:   "toggle <provider-id>",
		Short: "Enable or disable a subtitle provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.SetProviderEnabled(ctx, &subtv1.SetProviderEnabledRequest{Id: args[0], Enabled: enabled})
				if err != nil {
					return fmt.Errorf("subtitles provider toggle: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProvider())
				}
				if !flagQuiet {
					fmt.Printf("enabled=%t\n", resp.GetProvider().GetEnabled())
				}
				return nil
			})
		},
	}
	toggle.Flags().Bool("enabled", true, "provider enabled state")
	list := &cobra.Command{
		Use:   "list",
		Short: "List subtitle providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.ListProviders(ctx, &subtv1.ListProvidersRequest{})
				if err != nil {
					return fmt.Errorf("subtitles providers: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProviders())
				}
				rows := make([][]string, 0, len(resp.GetProviders()))
				for _, p := range resp.GetProviders() {
					rows = append(rows, []string{p.GetId(), p.GetName(), fmt.Sprintf("%t", p.GetEnabled())})
				}
				return printTable([]string{"ID", "NAME", "ENABLED"}, rows)
			})
		},
	}
	cmd := &cobra.Command{Use: "providers", Short: "Subtitle provider commands"}
	cmd.AddCommand(list, toggle)
	return cmd
}

func newSubtitlesBlacklistCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List blacklisted subtitles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.ListBlacklist(ctx, &subtv1.ListBlacklistRequest{Page: 1, PageSize: 50})
				if err != nil {
					return fmt.Errorf("subtitles blacklist: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetEntries())
				}
				rows := make([][]string, 0, len(resp.GetEntries()))
				for _, e := range resp.GetEntries() {
					rows = append(rows, []string{e.GetId(), e.GetTitle(), e.GetProvider()})
				}
				return printTable([]string{"ID", "TITLE", "PROVIDER"}, rows)
			})
		},
	}
	remove := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a blacklist entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				_, err := cli.RemoveBlacklist(ctx, &subtv1.RemoveBlacklistRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("subtitles blacklist remove: %w", err)
				}
				if !flagQuiet {
					fmt.Println("removed")
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{Use: "blacklist", Short: "Subtitle blacklist commands"}
	cmd.AddCommand(list, remove)
	return cmd
}

func newSubtitlesHistoryCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List subtitle download history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.ListHistory(ctx, &subtv1.ListHistoryRequest{Page: 1, PageSize: 50})
				if err != nil {
					return fmt.Errorf("subtitles history: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetEntries())
				}
				rows := make([][]string, 0, len(resp.GetEntries()))
				for _, e := range resp.GetEntries() {
					rows = append(rows, []string{e.GetTitle(), e.GetLanguage(), e.GetProvider(), e.GetCreatedAt()})
				}
				return printTable([]string{"TITLE", "LANG", "PROVIDER", "AT"}, rows)
			})
		},
	}
	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear subtitle history",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Clear subtitle history?"); err != nil {
				return err
			}
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				_, err := cli.ClearHistory(ctx, &subtv1.ClearHistoryRequest{})
				if err != nil {
					return fmt.Errorf("subtitles history clear: %w", err)
				}
				if !flagQuiet {
					fmt.Println("cleared")
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{Use: "history", Short: "Subtitle history commands"}
	cmd.AddCommand(list, clearCmd)
	return cmd
}
