package cli

import (
	"context"
	"fmt"

	mediaadminv1 "github.com/Muxcore-Media/contracts-media-admin/gen/muxcore/media/admin/v1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "media",
		Short:   "Manage media libraries (admin-ui /media parity)",
		GroupID: groupLibrary,
	}
	cmd.AddCommand(newMediaLibrariesCmd())
	cmd.AddCommand(newMediaItemsCmd())
	cmd.AddCommand(newMediaGetCmd())
	cmd.AddCommand(newMediaMissingCmd())
	cmd.AddCommand(newMediaRefreshCmd())
	cmd.AddCommand(newMediaDeleteCmd())
	cmd.AddCommand(newMediaTagsCmd())
	cmd.AddCommand(newMediaMonitorCmd())
	cmd.AddCommand(newMediaMetadataCmd())
	cmd.AddCommand(newMediaCollectionsCmd())
	cmd.AddCommand(newMediaDispatchCmd())
	cmd.AddCommand(newMediaFilesCmd())
	cmd.AddCommand(newMediaTitlesCmd())
	cmd.AddCommand(newMediaArtworkCmd())
	return cmd
}

func newMediaLibrariesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "libraries",
		Short: "List media library modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCore(func(ctx context.Context, c *client.Client) error {
				mods, err := listMediaLibraries(ctx, c)
				if err != nil {
					return fmt.Errorf("media libraries: %w", err)
				}
				if flagJSON {
					out := make([]map[string]string, 0, len(mods))
					for _, m := range mods {
						out = append(out, map[string]string{
							"id":   m.GetId(),
							"name": m.GetName(),
						})
					}
					return printJSON(out)
				}
				rows := make([][]string, 0, len(mods))
				for _, m := range mods {
					rows = append(rows, []string{m.GetId(), m.GetName(), m.GetVersion()})
				}
				return printTable([]string{"ID", "NAME", "VERSION"}, rows)
			})
		},
	}
}

func newMediaItemsCmd() *cobra.Command {
	var page, pageSize int32
	var search, tagID string
	cmd := &cobra.Command{
		Use:   "items <module-id>",
		Short: "List items in a media library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleID := args[0]
			if page < 1 {
				page = 1
			}
			if pageSize < 1 {
				pageSize = 50
			}
			return withMediaClient(moduleID, func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.ListItems(ctx, &mediaadminv1.ListItemsRequest{
					Page: page, PageSize: pageSize, Search: search, TagId: tagID,
				})
				if err != nil {
					return fmt.Errorf("media items: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				rows := make([][]string, 0, len(resp.GetItems()))
				for _, item := range resp.GetItems() {
					rows = append(rows, []string{
						item.GetId(),
						item.GetTitle(),
						fmt.Sprintf("%d", item.GetYear()),
						item.GetMetadata()["monitored"],
					})
				}
				pageCount := int32(1)
				if resp.GetPageSize() > 0 && resp.GetTotal() > 0 {
					pageCount = (resp.GetTotal() + resp.GetPageSize() - 1) / resp.GetPageSize()
				}
				fmt.Printf("page %d/%d total=%d\n", resp.GetPage(), pageCount, resp.GetTotal())
				return printTable([]string{"ID", "TITLE", "YEAR", "MONITORED"}, rows)
			})
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&pageSize, "page-size", 50, "items per page")
	cmd.Flags().StringVar(&search, "search", "", "search filter")
	cmd.Flags().StringVar(&tagID, "tag", "", "filter by tag id")
	return cmd
}

func newMediaGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <module-id> <item-id>",
		Short: "Show one library item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleID := args[0]
			itemID := args[1]
			return withMediaClient(moduleID, func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.GetItem(ctx, &mediaadminv1.GetItemRequest{Id: itemID})
				if err != nil {
					return fmt.Errorf("media get: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetItem())
				}
				item := resp.GetItem()
				fmt.Printf("id:        %s\n", item.GetId())
				fmt.Printf("title:     %s\n", item.GetTitle())
				fmt.Printf("year:      %d\n", item.GetYear())
				if item.GetDescription() != "" {
					fmt.Printf("overview:  %s\n", truncate(item.GetDescription(), 200))
				}
				if len(item.GetMetadata()) > 0 {
					fmt.Println("metadata:")
					for k, v := range item.GetMetadata() {
						fmt.Printf("  %s: %s\n", k, v)
					}
				}
				return nil
			})
		},
	}
}

func newMediaMissingCmd() *cobra.Command {
	var page, pageSize int32
	cmd := &cobra.Command{
		Use:   "missing <module-id>",
		Short: "List missing monitored items",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if page < 1 {
				page = 1
			}
			if pageSize < 1 {
				pageSize = 50
			}
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.ListMissing(ctx, &mediaadminv1.ListMissingRequest{Page: page, PageSize: pageSize})
				if err != nil {
					return fmt.Errorf("media missing: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				rows := make([][]string, 0, len(resp.GetItems()))
				for _, item := range resp.GetItems() {
					rows = append(rows, []string{
						item.GetId(),
						item.GetTitle(),
						item.GetParentId(),
						fmt.Sprintf("%d", item.GetYear()),
					})
				}
				return printTable([]string{"ID", "TITLE", "PARENT", "YEAR"}, rows)
			})
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&pageSize, "page-size", 50, "items per page")
	return cmd
}

func newMediaRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh <module-id> <item-id>",
		Short: "Refresh metadata for an item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.RefreshItem(ctx, &mediaadminv1.RefreshItemRequest{Id: args[1]})
				if err != nil {
					return fmt.Errorf("media refresh: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Println("refresh queued")
				}
				return nil
			})
		},
	}
}

func newMediaDeleteCmd() *cobra.Command {
	var deleteFiles bool
	cmd := &cobra.Command{
		Use:   "delete <module-id> <item-id>",
		Short: "Delete a library item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := fmt.Sprintf("Delete item %s from %s", args[1], args[0])
			if deleteFiles {
				msg += " including files on disk"
			}
			if err := confirmAction(msg + "?"); err != nil {
				return err
			}
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.DeleteItem(ctx, &mediaadminv1.DeleteItemRequest{
					Id: args[1], DeleteFiles: deleteFiles,
				})
				if err != nil {
					return fmt.Errorf("media delete: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&deleteFiles, "delete-files", false, "also delete files on disk")
	return cmd
}

func newMediaTagsCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list <module-id>",
		Short: "List tags for a media library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.ListTags(ctx, &mediaadminv1.ListTagsRequest{})
				if err != nil {
					return fmt.Errorf("media tags list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetTags())
				}
				rows := make([][]string, 0, len(resp.GetTags()))
				for _, t := range resp.GetTags() {
					rows = append(rows, []string{t.GetId(), t.GetLabel()})
				}
				return printTable([]string{"ID", "LABEL"}, rows)
			})
		},
	}
	cmd := &cobra.Command{Use: "tags", Short: "Library tag commands"}
	cmd.AddCommand(list)
	cmd.AddCommand(newMediaTagsCreateCmd())
	cmd.AddCommand(newMediaTagsDeleteCmd())
	cmd.AddCommand(newMediaTagsSetCmd())
	return cmd
}

func withMediaClient(moduleID string, fn func(context.Context, mediaadminv1.MediaAdminServiceClient) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		mod, err := findModuleByID(ctx, c, moduleID)
		if err != nil {
			return err
		}
		conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		return fn(ctx, mediaadminv1.NewMediaAdminServiceClient(conn))
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func newMediaMetadataCmd() *cobra.Command {
	var title, description string
	cmd := &cobra.Command{
		Use:   "metadata <module-id> <item-id>",
		Short: "Update item metadata fields",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" && description == "" {
				return fmt.Errorf("pass at least one of --title or --description")
			}
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.UpdateMetadata(ctx, &mediaadminv1.UpdateMetadataRequest{
					Id: args[1], Title: title, Description: description,
				})
				if err != nil {
					return fmt.Errorf("media metadata: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetItem())
				}
				if !flagQuiet {
					fmt.Println("updated")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&description, "description", "", "new overview/description")
	return cmd
}

func newMediaCollectionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collections",
		Short: "Collection commands for a media library",
	}
	cmd.AddCommand(newMediaCollectionsListCmd())
	cmd.AddCommand(newMediaCollectionsItemsCmd())
	cmd.AddCommand(newMediaCollectionsSyncCmd())
	cmd.AddCommand(newMediaCollectionsMonitorCmd())
	return cmd
}

func newMediaCollectionsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <module-id>",
		Short: "List collections in a media library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.ListCollections(ctx, &mediaadminv1.ListCollectionsRequest{})
				if err != nil {
					return fmt.Errorf("media collections: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetCollections())
				}
				rows := make([][]string, 0, len(resp.GetCollections()))
				for _, c := range resp.GetCollections() {
					rows = append(rows, []string{c.GetId(), c.GetName(), fmt.Sprintf("%d", c.GetItemCount())})
				}
				return printTable([]string{"ID", "NAME", "ITEMS"}, rows)
			})
		},
	}
}

func newMediaCollectionsItemsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "items <module-id> <collection-id>",
		Short: "List items in a collection",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.GetCollectionItems(ctx, &mediaadminv1.GetCollectionItemsRequest{
					CollectionId: args[1],
				})
				if err != nil {
					return fmt.Errorf("media collection items: %w", err)
				}
				if flagJSON {
					return printJSON(map[string]any{
						"id":    resp.GetCollectionId(),
						"name":  resp.GetName(),
						"items": resp.GetItems(),
					})
				}
				fmt.Printf("%s (%s)\n", resp.GetName(), resp.GetCollectionId())
				rows := make([][]string, 0, len(resp.GetItems()))
				for _, item := range resp.GetItems() {
					rows = append(rows, []string{item.GetId(), item.GetTitle(), fmt.Sprintf("%d", item.GetYear())})
				}
				return printTable([]string{"ID", "TITLE", "YEAR"}, rows)
			})
		},
	}
}
