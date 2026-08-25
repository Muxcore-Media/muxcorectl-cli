package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Muxcore-Media/core/sdk/go/client"
	formatsv1 "github.com/Muxcore-Media/media-custom-formats/proto/formatsv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const capMediaFormats = "media.formats"

func newFormatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "formats",
		Short:   "Custom formats and quality profiles (admin-ui /formats parity)",
		GroupID: groupLibrary,
	}
	cmd.AddCommand(newFormatsListCmd())
	cmd.AddCommand(newFormatsDeleteCmd())
	cmd.AddCommand(newFormatsSyncTrashCmd())
	cmd.AddCommand(newFormatsCreateCmd())
	cmd.AddCommand(newFormatsGetCmd())
	cmd.AddCommand(newFormatsUpdateCmd())
	cmd.AddCommand(newFormatsProfilesCmd())
	cmd.AddCommand(newFormatsReleaseProfilesCmd())
	return cmd
}

func newFormatsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List custom formats",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.ListFormats(ctx, &formatsv1.ListFormatsRequest{})
				if err != nil {
					return fmt.Errorf("formats list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetFormats())
				}
				rows := make([][]string, 0, len(resp.GetFormats()))
				for _, f := range resp.GetFormats() {
					rows = append(rows, []string{
						f.GetId(),
						f.GetName(),
						fmt.Sprintf("%d", f.GetDefaultScore()),
					})
				}
				return printTable([]string{"ID", "NAME", "DEFAULT_SCORE"}, rows)
			})
		},
	}
}

func newFormatsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <format-id>",
		Short: "Delete a custom format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete format " + args[0] + "?"); err != nil {
				return err
			}
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				_, err := cli.DeleteFormat(ctx, &formatsv1.DeleteFormatRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("formats delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newFormatsSyncTrashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync-trash",
		Short: "Sync custom formats from TRaSH guides",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.SyncTrashGuides(ctx, &formatsv1.SyncTrashGuidesRequest{})
				if err != nil {
					return fmt.Errorf("formats sync-trash: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("formats_upserted=%d profiles_upserted=%d\n",
						resp.GetFormatsUpserted(), resp.GetProfilesUpserted())
				}
				return nil
			})
		},
	}
}

func newFormatsCreateCmd() *cobra.Command {
	var score int32
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a custom format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.CreateFormat(ctx, &formatsv1.CreateFormatRequest{
					Name: args[0], DefaultScore: score,
				})
				if err != nil {
					return fmt.Errorf("formats create: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetFormat())
				}
				if !flagQuiet {
					fmt.Printf("created %s\n", resp.GetFormat().GetId())
				}
				return nil
			})
		},
	}
	cmd.Flags().Int32Var(&score, "score", 0, "default score")
	return cmd
}

func newFormatsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <format-id>",
		Short: "Show a custom format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.ListFormats(ctx, &formatsv1.ListFormatsRequest{})
				if err != nil {
					return fmt.Errorf("formats get: %w", err)
				}
				for _, f := range resp.GetFormats() {
					if f.GetId() == args[0] {
						return printJSON(f)
					}
				}
				return fmt.Errorf("format %q not found", args[0])
			})
		},
	}
}

func parseFormatRules(raw string) []*formatsv1.FormatRule {
	var out []*formatsv1.FormatRule
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		out = append(out, &formatsv1.FormatRule{
			Field: strings.TrimSpace(parts[0]),
			Op:    strings.TrimSpace(parts[1]),
			Value: strings.TrimSpace(parts[2]),
		})
	}
	return out
}

func newFormatsUpdateCmd() *cobra.Command {
	var name, rulesFile, rulesText string
	var score int32
	cmd := &cobra.Command{
		Use:   "update <format-id>",
		Short: "Update a custom format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var rules []*formatsv1.FormatRule
			if rulesFile != "" {
				raw, err := os.ReadFile(rulesFile) //nolint:gosec // operator-selected rules JSON path
				if err != nil {
					return err
				}
				if err := json.Unmarshal(raw, &rules); err != nil {
					return fmt.Errorf("parse rules file: %w", err)
				}
			} else if rulesText != "" {
				rules = parseFormatRules(rulesText)
			}
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				req := &formatsv1.UpdateFormatRequest{Id: args[0], DefaultScore: score, Rules: rules}
				if name != "" {
					req.Name = name
				}
				resp, err := cli.UpdateFormat(ctx, req)
				if err != nil {
					return fmt.Errorf("formats update: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetFormat())
				}
				if !flagQuiet {
					fmt.Println("updated")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "format name")
	cmd.Flags().Int32Var(&score, "score", 0, "default score")
	cmd.Flags().StringVar(&rulesFile, "rules-file", "", "JSON array of format rules")
	cmd.Flags().StringVar(&rulesText, "rules", "", "line rules as field|op|value")
	return cmd
}

func newFormatsProfilesCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List quality profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.ListProfiles(ctx, &formatsv1.ListProfilesRequest{})
				if err != nil {
					return fmt.Errorf("profiles list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfiles())
				}
				rows := make([][]string, 0, len(resp.GetProfiles()))
				for _, p := range resp.GetProfiles() {
					rows = append(rows, []string{
						p.GetId(),
						p.GetName(),
						fmt.Sprintf("%t", p.GetUpgradeAllowed()),
					})
				}
				return printTable([]string{"ID", "NAME", "UPGRADE"}, rows)
			})
		},
	}
	cmd := &cobra.Command{Use: "profiles", Short: "Quality profile commands"}
	cmd.AddCommand(list)
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a quality profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			minScore, _ := cmd.Flags().GetInt32("min-score")
			cutoff, _ := cmd.Flags().GetInt32("cutoff")
			upgrade, _ := cmd.Flags().GetBool("upgrade")
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.CreateProfile(ctx, &formatsv1.CreateProfileRequest{
					Name: args[0], MinScore: minScore, CutoffScore: cutoff, UpgradeAllowed: upgrade,
				})
				if err != nil {
					return fmt.Errorf("profiles create: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfile())
				}
				if !flagQuiet {
					fmt.Printf("created %s\n", resp.GetProfile().GetId())
				}
				return nil
			})
		},
	}
	create.Flags().Int32("min-score", 0, "minimum format score")
	create.Flags().Int32("cutoff", 0, "cutoff format score")
	create.Flags().Bool("upgrade", false, "allow quality upgrades")
	cmd.AddCommand(create)
	update := &cobra.Command{
		Use:   "update <profile-id>",
		Short: "Update a quality profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			minScore, _ := cmd.Flags().GetInt32("min-score")
			cutoff, _ := cmd.Flags().GetInt32("cutoff")
			upgrade, _ := cmd.Flags().GetBool("upgrade")
			name, _ := cmd.Flags().GetString("name")
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.UpdateProfile(ctx, &formatsv1.UpdateProfileRequest{
					Id: args[0], Name: name, MinScore: minScore, CutoffScore: cutoff, UpgradeAllowed: upgrade,
				})
				if err != nil {
					return fmt.Errorf("profiles update: %w", err)
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
	update.Flags().String("name", "", "profile name")
	update.Flags().Int32("min-score", 0, "minimum format score")
	update.Flags().Int32("cutoff", 0, "cutoff format score")
	update.Flags().Bool("upgrade", false, "allow quality upgrades")
	cmd.AddCommand(update)
	del := &cobra.Command{
		Use:   "delete <profile-id>",
		Short: "Delete a quality profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete profile " + args[0] + "?"); err != nil {
				return err
			}
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				_, err := cli.DeleteProfile(ctx, &formatsv1.DeleteProfileRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("profiles delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
	cmd.AddCommand(del)
	return cmd
}

func newFormatsReleaseProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "release-profiles", Short: "Release profile commands"}
	list := &cobra.Command{
		Use:   "list",
		Short: "List release profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.ListReleaseProfiles(ctx, &formatsv1.ListReleaseProfilesRequest{})
				if err != nil {
					return fmt.Errorf("release-profiles list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfiles())
				}
				rows := make([][]string, 0, len(resp.GetProfiles()))
				for _, p := range resp.GetProfiles() {
					rows = append(rows, []string{p.GetId(), p.GetName(), fmt.Sprintf("%t", p.GetEnabled())})
				}
				return printTable([]string{"ID", "NAME", "ENABLED"}, rows)
			})
		},
	}
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a release profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			score, _ := cmd.Flags().GetInt32("preferred-score")
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				resp, err := cli.UpsertReleaseProfile(ctx, &formatsv1.UpsertReleaseProfileRequest{
					Name: args[0], Enabled: enabled, PreferredScore: score,
				})
				if err != nil {
					return fmt.Errorf("release-profiles create: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfile())
				}
				if !flagQuiet {
					fmt.Printf("created %s\n", resp.GetProfile().GetId())
				}
				return nil
			})
		},
	}
	create.Flags().Bool("enabled", true, "profile enabled")
	create.Flags().Int32("preferred-score", 0, "preferred score")
	deleteCmd := &cobra.Command{
		Use:   "delete <profile-id>",
		Short: "Delete a release profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete release profile " + args[0] + "?"); err != nil {
				return err
			}
			return withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
				_, err := cli.DeleteReleaseProfile(ctx, &formatsv1.DeleteReleaseProfileRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("release-profiles delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
	cmd.AddCommand(list, create, deleteCmd)
	return cmd
}

func withFormatsClient(fn func(context.Context, formatsv1.FormatServiceClient) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		mod, err := findModuleByCapability(ctx, c, capMediaFormats)
		if err != nil {
			return err
		}
		conn, err := grpc.NewClient(
			normalizeDialAddr(mod.GetId(), mod.GetHttpAddr()),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return fmt.Errorf("dial formats: %w", err)
		}
		defer func() { _ = conn.Close() }()
		return fn(ctx, formatsv1.NewFormatServiceClient(conn))
	})
}
