package cli

import (
	"context"
	"fmt"

	renamev1 "github.com/Muxcore-Media/media-rename/proto/renamev1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const capMediaRenamer = "media.renamer"

func newRenameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rename",
		Short:   "Naming templates (admin-ui /rename parity)",
		GroupID: groupLibrary,
	}
	cmd.AddCommand(newRenameTemplatesCmd())
	cmd.AddCommand(newRenameOrganizeCmd())
	return cmd
}

func newRenameTemplatesCmd() *cobra.Command {
	var mediaType string
	list := &cobra.Command{
		Use:   "list",
		Short: "List naming templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRenameClient(func(ctx context.Context, cli renamev1.RenameServiceClient) error {
				resp, err := cli.ListTemplates(ctx, &renamev1.ListTemplatesRequest{MediaType: mediaType})
				if err != nil {
					return fmt.Errorf("rename templates list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetTemplates())
				}
				rows := make([][]string, 0, len(resp.GetTemplates()))
				for _, t := range resp.GetTemplates() {
					rows = append(rows, []string{t.GetId(), t.GetName(), t.GetMediaType(), t.GetPattern()})
				}
				return printTable([]string{"ID", "NAME", "MEDIA_TYPE", "PATTERN"}, rows)
			})
		},
	}
	cmd := &cobra.Command{Use: "templates", Short: "Naming template commands"}
	cmd.AddCommand(list)
	cmd.AddCommand(newRenameTemplateCreateCmd())
	cmd.AddCommand(newRenameTemplateUpdateCmd())
	cmd.AddCommand(newRenameTemplateDeleteCmd())
	cmd.PersistentFlags().StringVar(&mediaType, "type", "", "filter by media type (movie, tv)")
	return cmd
}

func newRenameTemplateCreateCmd() *cobra.Command {
	var pattern string
	var isDefault bool
	cmd := &cobra.Command{
		Use:   "create <name> <media-type>",
		Short: "Create a naming template",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pattern == "" {
				return fmt.Errorf("--pattern is required")
			}
			return withRenameClient(func(ctx context.Context, cli renamev1.RenameServiceClient) error {
				resp, err := cli.CreateTemplate(ctx, &renamev1.CreateTemplateRequest{
					Name: args[0], MediaType: args[1], Pattern: pattern, IsDefault: isDefault,
				})
				if err != nil {
					return fmt.Errorf("rename template create: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetTemplate())
				}
				if !flagQuiet {
					fmt.Printf("created %s\n", resp.GetTemplate().GetId())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&pattern, "pattern", "", "naming pattern")
	cmd.Flags().BoolVar(&isDefault, "default", false, "set as default template")
	_ = cmd.MarkFlagRequired("pattern")
	return cmd
}

func newRenameTemplateUpdateCmd() *cobra.Command {
	var pattern string
	var isDefault bool
	cmd := &cobra.Command{
		Use:   "update <template-id>",
		Short: "Update a naming template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			return withRenameClient(func(ctx context.Context, cli renamev1.RenameServiceClient) error {
				req := &renamev1.UpdateTemplateRequest{Id: args[0], IsDefault: isDefault}
				if name != "" {
					req.Name = name
				}
				if pattern != "" {
					req.Pattern = pattern
				}
				resp, err := cli.UpdateTemplate(ctx, req)
				if err != nil {
					return fmt.Errorf("rename template update: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetTemplate())
				}
				if !flagQuiet {
					fmt.Println("updated")
				}
				return nil
			})
		},
	}
	cmd.Flags().String("name", "", "template name")
	cmd.Flags().StringVar(&pattern, "pattern", "", "naming pattern")
	cmd.Flags().BoolVar(&isDefault, "default", false, "set as default template")
	return cmd
}

func newRenameTemplateDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <template-id>",
		Short: "Delete a naming template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete template " + args[0] + "?"); err != nil {
				return err
			}
			return withRenameClient(func(ctx context.Context, cli renamev1.RenameServiceClient) error {
				_, err := cli.DeleteTemplate(ctx, &renamev1.DeleteTemplateRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("rename template delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newRenameOrganizeCmd() *cobra.Command {
	var directory, mediaType, importMode string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "organize",
		Short: "Preview or execute batch rename in a directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if directory == "" {
				return fmt.Errorf("--directory is required")
			}
			return withRenameClient(func(ctx context.Context, cli renamev1.RenameServiceClient) error {
				resp, err := cli.BatchRename(ctx, &renamev1.BatchRenameRequest{
					Directory: directory, MediaType: mediaType, DryRun: dryRun, ImportMode: importMode,
				})
				if err != nil {
					return fmt.Errorf("rename organize: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("total=%d renamed=%d errors=%d dry_run=%t\n",
						resp.GetTotal(), resp.GetRenamed(), resp.GetErrors(), dryRun)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&directory, "directory", "", "directory to scan")
	cmd.Flags().StringVar(&mediaType, "type", "movie", "media type (movie or tv)")
	cmd.Flags().StringVar(&importMode, "import-mode", "", "optional import mode")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "preview without renaming files")
	_ = cmd.MarkFlagRequired("directory")
	return cmd
}

func withRenameClient(fn func(context.Context, renamev1.RenameServiceClient) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		mod, err := findModuleByCapability(ctx, c, capMediaRenamer)
		if err != nil {
			return err
		}
		conn, err := grpc.NewClient(
			normalizeDialAddr(mod.GetId(), mod.GetHttpAddr()),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return fmt.Errorf("dial rename: %w", err)
		}
		defer conn.Close()
		return fn(ctx, renamev1.NewRenameServiceClient(conn))
	})
}
