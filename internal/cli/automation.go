package cli

import (
	"context"
	"fmt"

	automationv1 "github.com/Muxcore-Media/contracts-automation/muxcore/automation/v1"
	"github.com/spf13/cobra"
)

func newAutomationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "automation",
		Short:   "Search and dispatch downloads (admin-ui /automation parity)",
		GroupID: groupAutomation,
	}
	cmd.AddCommand(newAutomationDispatchCmd())
	cmd.AddCommand(newAutomationSearchCmd())
	cmd.AddCommand(newAutomationDelayCmd())
	cmd.AddCommand(newAutomationBlocklistCmd())
	return cmd
}

func newAutomationDispatchCmd() *cobra.Command {
	var (
		itemID   string
		itemType string
		title    string
		tmdbID   int32
		year     int32
		guid     string
		download string
	)
	cmd := &cobra.Command{
		Use:   "dispatch",
		Short: "Dispatch a grab for a library item",
		Long: `Dispatch sends a release to the download client.

For a quick fixture/demo dispatch, pass --item-id, --type, and --title.
For a specific release, also pass --guid and --download-url.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if itemID == "" {
				return fmt.Errorf("--item-id is required")
			}
			if itemType == "" {
				itemType = "movie"
			}
			req := &automationv1.DispatchRequest{
				ItemId: itemID, ItemType: itemType, Title: title, TmdbId: tmdbID,
				Guid: guid, DownloadUrl: download,
			}
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.Dispatch(ctx, req)
				if err != nil {
					return fmt.Errorf("automation dispatch: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("status=%s download_id=%s\n", resp.GetStatus(), resp.GetDownloadId())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&itemID, "item-id", "", "library item id")
	cmd.Flags().StringVar(&itemType, "type", "movie", "movie or tv")
	cmd.Flags().StringVar(&title, "title", "", "title hint")
	cmd.Flags().Int32Var(&tmdbID, "tmdb-id", 0, "TMDB id")
	cmd.Flags().Int32Var(&year, "year", 0, "release year (search hint)")
	cmd.Flags().StringVar(&guid, "guid", "", "release guid from search")
	cmd.Flags().StringVar(&download, "download-url", "", "release download URL")
	_ = cmd.MarkFlagRequired("item-id")
	return cmd
}

func newAutomationSearchCmd() *cobra.Command {
	var (
		itemType string
		tmdbID   int32
		year     int32
		limit    int32
	)
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search indexers for releases matching a query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if itemType == "" {
				itemType = "movie"
			}
			if limit < 1 {
				limit = 10
			}
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.SearchItem(ctx, &automationv1.SearchItemRequest{
					ItemType: itemType, Query: args[0], TmdbId: tmdbID, Year: year, Limit: limit,
				})
				if err != nil {
					return fmt.Errorf("automation search: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetMatches())
				}
				rows := make([][]string, 0, len(resp.GetMatches()))
				for _, m := range resp.GetMatches() {
					rows = append(rows, []string{
						m.GetGuid(), m.GetTitle(), m.GetIndexerName(),
						fmt.Sprintf("%d", m.GetScore()), fmt.Sprintf("%d", m.GetSize()),
					})
				}
				return printTable([]string{"GUID", "TITLE", "INDEXER", "SCORE", "SIZE"}, rows)
			})
		},
	}
	cmd.Flags().StringVar(&itemType, "type", "movie", "movie or tv")
	cmd.Flags().Int32Var(&tmdbID, "tmdb-id", 0, "TMDB id filter")
	cmd.Flags().Int32Var(&year, "year", 0, "year filter")
	cmd.Flags().Int32Var(&limit, "limit", 10, "max results")
	return cmd
}

func newAutomationDelayCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "delay", Short: "Download delay profile commands"}
	list := &cobra.Command{
		Use:   "list",
		Short: "List delay profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.ListDelayProfiles(ctx, &automationv1.ListDelayProfilesRequest{})
				if err != nil {
					return fmt.Errorf("automation delay list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfiles())
				}
				rows := make([][]string, 0, len(resp.GetProfiles()))
				for _, p := range resp.GetProfiles() {
					rows = append(rows, []string{p.GetProtocol(), fmt.Sprintf("%d", p.GetWaitMinutes())})
				}
				return printTable([]string{"PROTOCOL", "WAIT_MIN"}, rows)
			})
		},
	}
	set := &cobra.Command{
		Use:   "set <protocol>",
		Short: "Set delay wait minutes for a protocol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wait, _ := cmd.Flags().GetInt32("wait-minutes")
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.UpsertDelayProfile(ctx, &automationv1.UpsertDelayProfileRequest{
					Protocol: args[0], WaitMinutes: wait,
				})
				if err != nil {
					return fmt.Errorf("automation delay set: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfile())
				}
				if !flagQuiet {
					fmt.Println("updated")
				}
				return nil
			})
		},
	}
	set.Flags().Int32("wait-minutes", 0, "wait minutes")
	cmd.AddCommand(list, set)
	return cmd
}

func newAutomationBlocklistCmd() *cobra.Command {
	var wantedID, guid string
	var clearAll bool
	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear automation blocklist entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Clear automation blocklist?"); err != nil {
				return err
			}
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.ClearBlocklist(ctx, &automationv1.ClearBlocklistRequest{
					WantedItemId: wantedID, Guid: guid, ClearAll: clearAll,
				})
				if err != nil {
					return fmt.Errorf("automation blocklist clear: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("removed=%d\n", resp.GetRemoved())
				}
				return nil
			})
		},
	}
	clearCmd.Flags().StringVar(&wantedID, "wanted-id", "", "clear entries for wanted item")
	clearCmd.Flags().StringVar(&guid, "guid", "", "clear entries for release guid")
	clearCmd.Flags().BoolVar(&clearAll, "all", false, "clear entire blocklist")
	cmd := &cobra.Command{Use: "blocklist", Short: "Automation blocklist commands"}
	cmd.AddCommand(clearCmd)
	return cmd
}
