package cli

import (
	"context"
	"fmt"
	"strings"

	subtv1 "github.com/Muxcore-Media/media-subtitles/proto/subtv1"
	"github.com/spf13/cobra"
)

func newSubtitlesProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "profiles", Short: "Language profile commands"}
	cmd.AddCommand(newSubtitlesProfilesListCmd())
	cmd.AddCommand(newSubtitlesProfilesCreateCmd())
	cmd.AddCommand(newSubtitlesProfilesDeleteCmd())
	return cmd
}

func newSubtitlesProfilesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List subtitle language profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.ListLanguageProfiles(ctx, &subtv1.ListLanguageProfilesRequest{})
				if err != nil {
					return fmt.Errorf("subtitles profiles list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfiles())
				}
				rows := make([][]string, 0, len(resp.GetProfiles()))
				for _, p := range resp.GetProfiles() {
					rows = append(rows, []string{p.GetId(), p.GetName(), fmt.Sprintf("%t", p.GetIsDefault())})
				}
				return printTable([]string{"ID", "NAME", "DEFAULT"}, rows)
			})
		},
	}
}

func parseLanguageRequirements(raw string) ([]*subtv1.LanguageRequirement, error) {
	var reqs []*subtv1.LanguageRequirement
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		lr := &subtv1.LanguageRequirement{}
		bits := strings.Split(part, "+")
		lr.Language = bits[0]
		for _, b := range bits[1:] {
			switch b {
			case "hi", "hearing_impaired", "sdh":
				lr.HearingImpaired = true
			case "forced", "force":
				lr.Forced = true
			}
		}
		reqs = append(reqs, lr)
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("no valid languages (example: en,es+forced)")
	}
	return reqs, nil
}

func newSubtitlesProfilesCreateCmd() *cobra.Command {
	var languages string
	var isDefault bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create or update a language profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqs, err := parseLanguageRequirements(languages)
			if err != nil {
				return err
			}
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.UpsertLanguageProfile(ctx, &subtv1.UpsertLanguageProfileRequest{
					Profile: &subtv1.LanguageProfile{
						Name: args[0], Languages: reqs, IsDefault: isDefault,
					},
				})
				if err != nil {
					return fmt.Errorf("subtitles profiles create: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfile())
				}
				if !flagQuiet {
					fmt.Printf("profile %s\n", resp.GetProfile().GetId())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&languages, "languages", "", "comma-separated langs (en,es+forced,de+hi)")
	_ = cmd.MarkFlagRequired("languages")
	cmd.Flags().BoolVar(&isDefault, "default", false, "set as default profile")
	return cmd
}

func newSubtitlesProfilesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <profile-id>",
		Short: "Delete a language profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete profile " + args[0] + "?"); err != nil {
				return err
			}
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				_, err := cli.DeleteLanguageProfile(ctx, &subtv1.DeleteLanguageProfileRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("subtitles profiles delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newSubtitlesMediaCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "media", Short: "Subtitle media item commands"}
	cmd.AddCommand(newSubtitlesMediaGetCmd())
	cmd.AddCommand(newSubtitlesMediaSearchCmd())
	cmd.AddCommand(newSubtitlesMediaSetProfileCmd())
	return cmd
}

func newSubtitlesMediaGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <media-id>",
		Short: "Show synced subtitle media item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.GetMedia(ctx, &subtv1.GetMediaRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("subtitles media get: %w", err)
				}
				return printJSON(resp.GetItem())
			})
		},
	}
}

func newSubtitlesMediaSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <media-id>",
		Short: "Search providers for wanted subtitles on one media item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.SearchWanted(ctx, &subtv1.SearchWantedRequest{MediaIds: []string{args[0]}, Limit: 8})
				if err != nil {
					return fmt.Errorf("subtitles media search: %w", err)
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

func newSubtitlesMediaSetProfileCmd() *cobra.Command {
	var profileID string
	cmd := &cobra.Command{
		Use:   "set-profile <media-id>",
		Short: "Assign a language profile to a media item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID == "" {
				return fmt.Errorf("--profile is required")
			}
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				_, err := cli.SetMediaLanguageProfile(ctx, &subtv1.SetMediaLanguageProfileRequest{
					MediaId: args[0], LanguageProfileId: profileID,
				})
				if err != nil {
					return fmt.Errorf("subtitles media set-profile: %w", err)
				}
				if !flagQuiet {
					fmt.Println("updated")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&profileID, "profile", "", "language profile id")
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func newSubtitlesMassEditCmd() *cobra.Command {
	var (
		action    string
		profileID string
	)
	cmd := &cobra.Command{
		Use:   "mass-edit",
		Short: "Bulk update subtitle media items",
		Long: `Actions: monitor, unmonitor, profile, search

Pass media ids with repeated --id flags.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, _ := cmd.Flags().GetStringArray("id")
			if len(ids) == 0 {
				return fmt.Errorf("at least one --id is required")
			}
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				switch action {
				case "monitor", "unmonitor":
					resp, err := cli.MassEditMedia(ctx, &subtv1.MassEditMediaRequest{
						MediaIds: ids, SetMonitored: true, Monitored: action == "monitor",
					})
					if err != nil {
						return fmt.Errorf("subtitles mass-edit: %w", err)
					}
					if flagJSON {
						return printJSON(resp)
					}
					if !flagQuiet {
						fmt.Printf("updated=%d\n", resp.GetUpdated())
					}
				case "profile":
					if profileID == "" {
						return fmt.Errorf("--profile is required for profile action")
					}
					resp, err := cli.MassEditMedia(ctx, &subtv1.MassEditMediaRequest{
						MediaIds: ids, LanguageProfileId: profileID,
					})
					if err != nil {
						return fmt.Errorf("subtitles mass-edit: %w", err)
					}
					if flagJSON {
						return printJSON(resp)
					}
					if !flagQuiet {
						fmt.Printf("updated=%d\n", resp.GetUpdated())
					}
				case "search":
					resp, err := cli.SearchWanted(ctx, &subtv1.SearchWantedRequest{
						MediaIds: ids, Limit: int32(len(ids) * 4),
					})
					if err != nil {
						return fmt.Errorf("subtitles mass-edit search: %w", err)
					}
					if flagJSON {
						return printJSON(resp)
					}
					if !flagQuiet {
						fmt.Printf("searched=%d downloaded=%d\n", resp.GetSearched(), resp.GetDownloaded())
					}
				default:
					return fmt.Errorf("unknown action %q (use monitor, unmonitor, profile, search)", action)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringArray("id", nil, "media id (repeatable)")
	cmd.Flags().StringVar(&action, "action", "", "monitor|unmonitor|profile|search")
	cmd.Flags().StringVar(&profileID, "profile", "", "profile id for profile action")
	_ = cmd.MarkFlagRequired("action")
	return cmd
}

func newSubtitlesUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade existing subtitles to better matches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.UpgradeSubtitles(ctx, &subtv1.UpgradeSubtitlesRequest{})
				if err != nil {
					return fmt.Errorf("subtitles upgrade: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("checked=%d upgraded=%d\n", resp.GetChecked(), resp.GetUpgraded())
				}
				return nil
			})
		},
	}
}

func newSubtitlesFilesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <file-id>",
		Short: "Delete a subtitle file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete subtitle file " + args[0] + "?"); err != nil {
				return err
			}
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				_, err := cli.Delete(ctx, &subtv1.DeleteRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("subtitles file delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newSubtitlesFilesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "files", Short: "Subtitle file commands"}
	cmd.AddCommand(newSubtitlesFilesDeleteCmd())
	return cmd
}

func newSubtitlesDownloadCmd() *cobra.Command {
	var (
		provider, fileID, language, release, mediaFileID string
		score                                            int32
	)
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download a subtitle from a provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.DownloadSubtitle(ctx, &subtv1.DownloadSubtitleRequest{
					Provider: provider, FileId: fileID, Language: language,
					ReleaseName: release, Score: score, MediaFileId: mediaFileID,
				})
				if err != nil {
					return fmt.Errorf("subtitles download: %w", err)
				}
				if flagJSON {
					if resp.GetSubtitle() != nil {
						return printJSON(resp.GetSubtitle())
					}
					return printJSON(resp)
				}
				if !flagQuiet {
					if resp.GetSubtitle() != nil {
						fmt.Printf("downloaded %s\n", resp.GetSubtitle().GetId())
					} else {
						fmt.Println("downloaded")
					}
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "provider id")
	cmd.Flags().StringVar(&fileID, "file-id", "", "provider file id")
	cmd.Flags().StringVar(&language, "language", "", "language code")
	cmd.Flags().StringVar(&release, "release", "", "release name")
	cmd.Flags().StringVar(&mediaFileID, "media-file-id", "", "media file id")
	cmd.Flags().Int32Var(&score, "score", 0, "match score")
	_ = cmd.MarkFlagRequired("provider")
	_ = cmd.MarkFlagRequired("file-id")
	return cmd
}

func newSubtitlesTestArrCmd() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "test-arr",
		Short: "Test Radarr/Sonarr connection for subtitles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSubtitlesClient(func(ctx context.Context, cli subtv1.SubtitleServiceClient) error {
				resp, err := cli.TestArrConnection(ctx, &subtv1.TestArrConnectionRequest{Target: target})
				if err != nil {
					return fmt.Errorf("subtitles test-arr: %w", err)
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
	cmd.Flags().StringVar(&target, "target", "radarr", "radarr or sonarr")
	return cmd
}
