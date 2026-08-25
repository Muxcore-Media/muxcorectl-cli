package cli

import (
	"context"
	"fmt"

	"github.com/Muxcore-Media/core/sdk/go/client"
	rootsv1 "github.com/Muxcore-Media/media-root-folders/proto/rootsv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const capMediaRoots = "media.roots"

func newRootsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "roots",
		Short:   "Library root folders (admin-ui /roots parity)",
		GroupID: groupLibrary,
	}
	cmd.AddCommand(newRootsListCmd())
	cmd.AddCommand(newRootsCreateCmd())
	cmd.AddCommand(newRootsUpdateCmd())
	cmd.AddCommand(newRootsBrowseCmd())
	cmd.AddCommand(newRootsDeleteCmd())
	return cmd
}

func newRootsListCmd() *cobra.Command {
	var mediaKind string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List root folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRootsClient(func(ctx context.Context, cli rootsv1.RootFolderServiceClient) error {
				resp, err := cli.ListRoots(ctx, &rootsv1.ListRootsRequest{MediaKind: mediaKind})
				if err != nil {
					return fmt.Errorf("roots list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetRoots())
				}
				rows := make([][]string, 0, len(resp.GetRoots()))
				for _, r := range resp.GetRoots() {
					rows = append(rows, []string{
						r.GetId(),
						r.GetName(),
						r.GetPath(),
						fmt.Sprintf("%t", r.GetAccessible()),
						fmt.Sprintf("%d", r.GetFreeBytes()),
					})
				}
				return printTable([]string{"ID", "NAME", "PATH", "ACCESSIBLE", "FREE_BYTES"}, rows)
			})
		},
	}
	cmd.Flags().StringVar(&mediaKind, "kind", "", "filter by media kind (movies, tv)")
	return cmd
}

func newRootsCreateCmd() *cobra.Command {
	var name, mediaKind string
	cmd := &cobra.Command{
		Use:   "create <path>",
		Short: "Create a root folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRootsClient(func(ctx context.Context, cli rootsv1.RootFolderServiceClient) error {
				resp, err := cli.CreateRoot(ctx, &rootsv1.CreateRootRequest{
					Path: args[0], Name: name, MediaKind: mediaKind,
				})
				if err != nil {
					return fmt.Errorf("roots create: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetRoot())
				}
				if !flagQuiet {
					fmt.Printf("created root %s at %s\n", resp.GetRoot().GetId(), resp.GetRoot().GetPath())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "display name (defaults to path)")
	cmd.Flags().StringVar(&mediaKind, "kind", "movies", "media kind (movies or tv)")
	return cmd
}

func newRootsUpdateCmd() *cobra.Command {
	var name, mediaKind, templateID string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a root folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRootsClient(func(ctx context.Context, cli rootsv1.RootFolderServiceClient) error {
				req := &rootsv1.UpdateRootRequest{Id: args[0]}
				if cmd.Flags().Changed("path") {
					p, _ := cmd.Flags().GetString("path")
					req.Path = p
				}
				if name != "" {
					req.Name = name
				}
				if mediaKind != "" {
					req.MediaKind = mediaKind
				}
				if templateID != "" {
					req.NamingTemplateId = templateID
				}
				resp, err := cli.UpdateRoot(ctx, req)
				if err != nil {
					return fmt.Errorf("roots update: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetRoot())
				}
				if !flagQuiet {
					fmt.Println("updated")
				}
				return nil
			})
		},
	}
	cmd.Flags().String("path", "", "filesystem path")
	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().StringVar(&mediaKind, "kind", "", "media kind")
	cmd.Flags().StringVar(&templateID, "template", "", "naming template id")
	return cmd
}

func newRootsBrowseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "browse [path]",
		Short: "Browse directories for root folder selection",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/"
			if len(args) > 0 {
				path = args[0]
			}
			return withRootsClient(func(ctx context.Context, cli rootsv1.RootFolderServiceClient) error {
				resp, err := cli.BrowsePath(ctx, &rootsv1.BrowsePathRequest{Path: path})
				if err != nil {
					return fmt.Errorf("roots browse: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				fmt.Printf("path: %s parent: %s\n", resp.GetPath(), resp.GetParent())
				rows := make([][]string, 0, len(resp.GetEntries()))
				for _, e := range resp.GetEntries() {
					rows = append(rows, []string{e.GetName(), fmt.Sprintf("%t", e.GetIsDir())})
				}
				return printTable([]string{"NAME", "DIR"}, rows)
			})
		},
	}
}

func newRootsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a root folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := confirmAction("Delete root folder " + id + "?"); err != nil {
				return err
			}
			return withRootsClient(func(ctx context.Context, cli rootsv1.RootFolderServiceClient) error {
				_, err := cli.DeleteRoot(ctx, &rootsv1.DeleteRootRequest{Id: id})
				if err != nil {
					return fmt.Errorf("roots delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func withRootsClient(fn func(context.Context, rootsv1.RootFolderServiceClient) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		mod, err := findModuleByCapability(ctx, c, capMediaRoots)
		if err != nil {
			return err
		}
		conn, err := grpc.NewClient(
			normalizeDialAddr(mod.GetId(), mod.GetHttpAddr()),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return fmt.Errorf("dial roots: %w", err)
		}
		defer func() { _ = conn.Close() }()
		return fn(ctx, rootsv1.NewRootFolderServiceClient(conn))
	})
}
