package cli

import (
	"context"
	"fmt"

	listsyncv1 "github.com/Muxcore-Media/media-list-sync/proto/listsyncv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capMediaListSync = "media.listsync"

func withListSyncClient(fn func(context.Context, listsyncv1.ListSyncServiceClient) error) error {
	return withModuleConn(capMediaListSync, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, listsyncv1.NewListSyncServiceClient(conn))
	})
}

func newListSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-sync",
		Short:   "Trakt/list sync sources (admin-ui /list-sync parity)",
		GroupID: groupAutomation,
	}
	cmd.AddCommand(newListSyncSourcesCmd())
	cmd.AddCommand(newListSyncSyncCmd())
	cmd.AddCommand(newListSyncHistoryCmd())
	cmd.AddCommand(newListSyncDeleteCmd())
	cmd.AddCommand(newListSyncAddCmd())
	cmd.AddCommand(newListSyncUpdateCmd())
	cmd.AddCommand(newListSyncTestCmd())
	cmd.AddCommand(newListSyncToggleCmd())
	cmd.AddCommand(newListSyncItemsCmd())
	return cmd
}

func newListSyncSourcesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sources",
		Short: "List configured sync sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				resp, err := cli.ListSources(ctx, &listsyncv1.ListSourcesRequest{})
				if err != nil {
					return fmt.Errorf("list-sync sources: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetSources())
				}
				rows := make([][]string, 0, len(resp.GetSources()))
				for _, s := range resp.GetSources() {
					rows = append(rows, []string{
						s.GetId(), s.GetName(), s.GetType(), fmt.Sprintf("%t", s.GetEnabled()), s.GetLastSynced(),
					})
				}
				return printTable([]string{"ID", "NAME", "TYPE", "ENABLED", "LAST_SYNCED"}, rows)
			})
		},
	}
}

func newListSyncSyncCmd() *cobra.Command {
	var sourceID string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Run list sync now (all sources or one source)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				resp, err := cli.SyncNow(ctx, &listsyncv1.SyncNowRequest{SourceId: sourceID})
				if err != nil {
					return fmt.Errorf("list-sync sync: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("found=%d new=%d removed=%d\n", resp.GetItemsFound(), resp.GetItemsNew(), resp.GetItemsRemoved())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "sync only this source id (default: all enabled)")
	return cmd
}

func newListSyncHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "List sync history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				resp, err := cli.GetHistory(ctx, &listsyncv1.GetHistoryRequest{Page: 1, PageSize: 50})
				if err != nil {
					return fmt.Errorf("list-sync history: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetEntries())
				}
				rows := make([][]string, 0, len(resp.GetEntries()))
				for _, e := range resp.GetEntries() {
					rows = append(rows, []string{
						e.GetId(), e.GetSourceName(), e.GetStatus(), fmt.Sprintf("%d", e.GetItemsNew()), e.GetStartedAt(),
					})
				}
				return printTable([]string{"ID", "SOURCE", "STATUS", "NEW", "STARTED"}, rows)
			})
		},
	}
}

func newListSyncDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <source-id>",
		Short: "Remove a sync source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete list-sync source " + args[0] + "?"); err != nil {
				return err
			}
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				_, err := cli.RemoveSource(ctx, &listsyncv1.RemoveSourceRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("list-sync delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newListSyncAddCmd() *cobra.Command {
	var (
		srcType, listURL, username, clientID string
		interval                             int32
	)
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a list sync source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if srcType == "" {
				srcType = "trakt"
			}
			if interval < 1 {
				interval = 60
			}
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				resp, err := cli.AddSource(ctx, &listsyncv1.AddSourceRequest{
					Name: args[0], Type: srcType, ListUrl: listURL, Username: username,
					ClientId: clientID, SyncIntervalMinutes: interval,
				})
				if err != nil {
					return fmt.Errorf("list-sync add: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetSource())
				}
				if !flagQuiet {
					fmt.Printf("created %s\n", resp.GetSource().GetId())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&srcType, "type", "trakt", "source type")
	cmd.Flags().StringVar(&listURL, "list-url", "", "list URL")
	cmd.Flags().StringVar(&username, "username", "", "username")
	cmd.Flags().StringVar(&clientID, "client-id", "", "client id")
	cmd.Flags().Int32Var(&interval, "interval", 60, "sync interval minutes")
	return cmd
}

func newListSyncUpdateCmd() *cobra.Command {
	var (
		name, listURL, username, clientID, baseURL, apiKey string
		qualityProfile, rootFolder, cleanLevel, tagIDs     string
		monitorMode, minAvailability                       string
		interval                                           int32
		enabled                                            bool
		searchOnAdd                                        bool
		hasEnabled, hasSearchOnAdd, hasInterval            bool
	)
	cmd := &cobra.Command{
		Use:   "update <source-id>",
		Short: "Update a list sync source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &listsyncv1.UpdateSourceRequest{Id: args[0]}
			if name != "" {
				req.Name = name
			}
			if listURL != "" {
				req.ListUrl = listURL
			}
			if username != "" {
				req.Username = username
			}
			if clientID != "" {
				req.ClientId = clientID
			}
			if baseURL != "" {
				req.BaseUrl = baseURL
			}
			if apiKey != "" {
				req.ApiKey = apiKey
			}
			if qualityProfile != "" {
				req.QualityProfileId = qualityProfile
			}
			if rootFolder != "" {
				req.RootFolderPath = rootFolder
			}
			if cleanLevel != "" {
				req.CleanLibraryLevel = cleanLevel
			}
			if tagIDs != "" {
				req.TagIds = tagIDs
			}
			if monitorMode != "" {
				req.MonitorMode = monitorMode
			}
			if minAvailability != "" {
				req.MinimumAvailability = minAvailability
			}
			if hasInterval {
				req.SyncIntervalMinutes = interval
			}
			if hasEnabled {
				req.Enabled = &enabled
			}
			if hasSearchOnAdd {
				req.SearchOnAdd = &searchOnAdd
			}
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				resp, err := cli.UpdateSource(ctx, req)
				if err != nil {
					return fmt.Errorf("list-sync update: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetSource())
				}
				if !flagQuiet {
					fmt.Println("updated")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "source name")
	cmd.Flags().StringVar(&listURL, "list-url", "", "list URL")
	cmd.Flags().StringVar(&username, "username", "", "username")
	cmd.Flags().StringVar(&clientID, "client-id", "", "client id")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "arr base URL")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "api key (empty keeps existing)")
	cmd.Flags().StringVar(&qualityProfile, "quality-profile", "", "quality profile id")
	cmd.Flags().StringVar(&rootFolder, "root-folder", "", "root folder path")
	cmd.Flags().StringVar(&cleanLevel, "clean-level", "", "clean library level")
	cmd.Flags().StringVar(&tagIDs, "tags", "", "tag ids")
	cmd.Flags().StringVar(&monitorMode, "monitor-mode", "", "monitor mode")
	cmd.Flags().StringVar(&minAvailability, "min-availability", "", "minimum availability")
	cmd.Flags().Int32Var(&interval, "interval", 60, "sync interval minutes")
	cmd.Flags().BoolVar(&hasInterval, "set-interval", false, "apply --interval")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "source enabled")
	cmd.Flags().BoolVar(&hasEnabled, "set-enabled", false, "apply --enabled")
	cmd.Flags().BoolVar(&searchOnAdd, "search-on-add", true, "search on add")
	cmd.Flags().BoolVar(&hasSearchOnAdd, "set-search-on-add", false, "apply --search-on-add")
	return cmd
}

func newListSyncTestCmd() *cobra.Command {
	var sourceID string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test a list sync source connection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				resp, err := cli.TestSource(ctx, &listsyncv1.TestSourceRequest{SourceId: sourceID})
				if err != nil {
					return fmt.Errorf("list-sync test: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("ok=%t message=%s\n", resp.GetOk(), resp.GetMessage())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "saved source id to test")
	return cmd
}

func newListSyncToggleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "toggle <source-id>",
		Short: "Toggle a source enabled/disabled",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				srcs, err := cli.ListSources(ctx, &listsyncv1.ListSourcesRequest{})
				if err != nil {
					return fmt.Errorf("list-sync toggle: %w", err)
				}
				var enabled bool
				for _, s := range srcs.GetSources() {
					if s.GetId() == args[0] {
						enabled = !s.GetEnabled()
						break
					}
				}
				en := enabled
				resp, err := cli.UpdateSource(ctx, &listsyncv1.UpdateSourceRequest{Id: args[0], Enabled: &en})
				if err != nil {
					return fmt.Errorf("list-sync toggle: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetSource())
				}
				if !flagQuiet {
					fmt.Printf("enabled=%t\n", resp.GetSource().GetEnabled())
				}
				return nil
			})
		},
	}
}

func newListSyncItemsCmd() *cobra.Command {
	var sourceID, mediaType string
	cmd := &cobra.Command{
		Use:   "items",
		Short: "List synced list items",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withListSyncClient(func(ctx context.Context, cli listsyncv1.ListSyncServiceClient) error {
				resp, err := cli.GetItems(ctx, &listsyncv1.GetItemsRequest{
					Page: 1, PageSize: 50, SourceId: sourceID, MediaType: mediaType,
				})
				if err != nil {
					return fmt.Errorf("list-sync items: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetItems())
				}
				rows := make([][]string, 0, len(resp.GetItems()))
				for _, it := range resp.GetItems() {
					matched := it.GetMatchedItemId() != ""
					rows = append(rows, []string{it.GetTitle(), it.GetMediaType(), fmt.Sprintf("%d", it.GetTmdbId()), fmt.Sprintf("%t", matched)})
				}
				return printTable([]string{"TITLE", "TYPE", "TMDB", "MATCHED"}, rows)
			})
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "filter by source id")
	cmd.Flags().StringVar(&mediaType, "type", "", "filter by media type")
	return cmd
}
