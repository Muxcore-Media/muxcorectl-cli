package cli

import (
	"context"
	"fmt"

	automationv1 "github.com/Muxcore-Media/contracts-automation/muxcore/automation/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capMediaAutomation = "media.automation"

func withAutomationClient(fn func(context.Context, automationv1.AutomationServiceClient) error) error {
	return withModuleConn(capMediaAutomation, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, automationv1.NewAutomationServiceClient(conn))
	})
}

func newQueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "queue",
		Short:   "Wanted queue and import history (admin-ui /queue parity)",
		GroupID: groupAutomation,
	}
	cmd.AddCommand(newQueueListCmd())
	cmd.AddCommand(newQueueHistoryCmd())
	cmd.AddCommand(newQueueRemoveCmd())
	cmd.AddCommand(newQueueRetryImportCmd())
	cmd.AddCommand(newQueueBlocklistCmd())
	return cmd
}

func newQueueListCmd() *cobra.Command {
	var page, pageSize int32
	var filter string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List wanted queue items",
		RunE: func(cmd *cobra.Command, args []string) error {
			if page < 1 {
				page = 1
			}
			if pageSize < 1 {
				pageSize = 50
			}
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.GetQueue(ctx, &automationv1.GetQueueRequest{
					Page: page, PageSize: pageSize, Filter: filter,
				})
				if err != nil {
					return fmt.Errorf("queue list: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				rows := make([][]string, 0, len(resp.GetItems()))
				for _, it := range resp.GetItems() {
					rows = append(rows, []string{
						it.GetId(), it.GetItemId(), it.GetItemType(), it.GetTitle(),
						fmt.Sprintf("%d", it.GetYear()), fmt.Sprintf("%t", it.GetMissing()),
					})
				}
				fmt.Printf("total=%d page=%d\n", resp.GetTotal(), resp.GetPage())
				return printTable([]string{"QUEUE_ID", "ITEM_ID", "TYPE", "TITLE", "YEAR", "MISSING"}, rows)
			})
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&pageSize, "page-size", 50, "items per page")
	cmd.Flags().StringVar(&filter, "filter", "", "queue filter")
	return cmd
}

func newQueueHistoryCmd() *cobra.Command {
	var page, pageSize int32
	cmd := &cobra.Command{
		Use:   "history",
		Short: "List import/grab history",
		RunE: func(cmd *cobra.Command, args []string) error {
			if page < 1 {
				page = 1
			}
			if pageSize < 1 {
				pageSize = 50
			}
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.GetHistory(ctx, &automationv1.GetHistoryRequest{Page: page, PageSize: pageSize})
				if err != nil {
					return fmt.Errorf("queue history: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetRecords())
				}
				rows := make([][]string, 0, len(resp.GetRecords()))
				for _, rec := range resp.GetRecords() {
					rows = append(rows, []string{
						rec.GetId(), rec.GetTitle(), rec.GetStatus(), rec.GetIndexer(), rec.GetCreatedAt(),
					})
				}
				return printTable([]string{"ID", "TITLE", "STATUS", "SOURCE", "AT"}, rows)
			})
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&pageSize, "page-size", 50, "items per page")
	return cmd
}

func newQueueRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <queue-id>",
		Short: "Remove an item from the wanted queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				_, err := cli.RemoveFromQueue(ctx, &automationv1.RemoveFromQueueRequest{QueueId: args[0]})
				if err != nil {
					return fmt.Errorf("queue remove: %w", err)
				}
				if !flagQuiet {
					fmt.Println("removed")
				}
				return nil
			})
		},
	}
}

func newQueueRetryImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "retry-import <history-id>",
		Short: "Retry a failed import from history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				_, err := cli.RetryImport(ctx, &automationv1.RetryImportRequest{HistoryId: args[0]})
				if err != nil {
					return fmt.Errorf("queue retry-import: %w", err)
				}
				if !flagQuiet {
					fmt.Println("retry queued")
				}
				return nil
			})
		},
	}
}

func newQueueBlocklistCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List blocklisted releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.ListBlocklist(ctx, &automationv1.ListBlocklistRequest{Page: 1, PageSize: 100})
				if err != nil {
					return fmt.Errorf("queue blocklist: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetEntries())
				}
				rows := make([][]string, 0, len(resp.GetEntries()))
				for _, e := range resp.GetEntries() {
					rows = append(rows, []string{e.GetGuid(), e.GetTitle(), e.GetReason(), e.GetCreatedAt()})
				}
				return printTable([]string{"GUID", "TITLE", "REASON", "AT"}, rows)
			})
		},
	}
	clear := &cobra.Command{
		Use:   "clear",
		Short: "Clear the blocklist",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Clear entire blocklist?"); err != nil {
				return err
			}
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				_, err := cli.ClearBlocklist(ctx, &automationv1.ClearBlocklistRequest{})
				if err != nil {
					return fmt.Errorf("queue blocklist clear: %w", err)
				}
				if !flagQuiet {
					fmt.Println("cleared")
				}
				return nil
			})
		},
	}
	add := &cobra.Command{
		Use:   "add",
		Short: "Blocklist a release",
		RunE: func(cmd *cobra.Command, args []string) error {
			guid, _ := cmd.Flags().GetString("guid")
			wantedID, _ := cmd.Flags().GetString("wanted-id")
			reason, _ := cmd.Flags().GetString("reason")
			if guid == "" || wantedID == "" {
				return fmt.Errorf("--guid and --wanted-id are required")
			}
			return withAutomationClient(func(ctx context.Context, cli automationv1.AutomationServiceClient) error {
				resp, err := cli.BlocklistRelease(ctx, &automationv1.BlocklistReleaseRequest{
					WantedItemId: wantedID, Guid: guid, Reason: reason,
				})
				if err != nil {
					return fmt.Errorf("queue blocklist add: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Println("blocklisted")
				}
				return nil
			})
		},
	}
	add.Flags().String("guid", "", "release guid")
	add.Flags().String("wanted-id", "", "wanted queue item id")
	add.Flags().String("reason", "operator", "blocklist reason")
	cmd := &cobra.Command{Use: "blocklist", Short: "Blocklist commands"}
	cmd.AddCommand(list, add, clear)
	return cmd
}
