package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	mediaadminv1 "github.com/Muxcore-Media/contracts-media-admin/gen/muxcore/media/admin/v1"
	automationv1 "github.com/Muxcore-Media/contracts-automation/muxcore/automation/v1"
	mgmntv1 "github.com/Muxcore-Media/media-movies/proto/mgmntv1"
	tvmgmtv1 "github.com/Muxcore-Media/media-tvshows/proto/tvmgmtv1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func newMediaTagsCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <module-id> <label>",
		Short: "Create a library tag",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.CreateTag(ctx, &mediaadminv1.CreateTagRequest{Label: args[1]})
				if err != nil {
					return fmt.Errorf("media tag create: %w", err)
				}
				if flagJSON {
					return printJSON(map[string]string{"tag_id": resp.GetTagId()})
				}
				if !flagQuiet {
					fmt.Printf("created tag %s\n", resp.GetTagId())
				}
				return nil
			})
		},
	}
}

func newMediaTagsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <module-id> <tag-id>",
		Short: "Delete a library tag",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete tag " + args[1] + "?"); err != nil {
				return err
			}
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				_, err := cli.DeleteTag(ctx, &mediaadminv1.DeleteTagRequest{TagId: args[1]})
				if err != nil {
					return fmt.Errorf("media tag delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newMediaTagsSetCmd() *cobra.Command {
	var tagIDs []string
	cmd := &cobra.Command{
		Use:   "set <module-id> <item-id>",
		Short: "Set tags on a library item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				_, err := cli.SetItemTags(ctx, &mediaadminv1.SetItemTagsRequest{
					ItemId: args[1], TagIds: tagIDs,
				})
				if err != nil {
					return fmt.Errorf("media tags set: %w", err)
				}
				if !flagQuiet {
					fmt.Println("updated")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringSliceVar(&tagIDs, "tag", nil, "tag ids (repeatable)")
	return cmd
}

func newMediaMonitorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Toggle monitored state on movies, seasons, or episodes",
	}
	cmd.AddCommand(newMediaMonitorMovieCmd())
	cmd.AddCommand(newMediaMonitorSeasonCmd())
	cmd.AddCommand(newMediaMonitorEpisodeCmd())
	return cmd
}

func newMediaMonitorMovieCmd() *cobra.Command {
	var monitored bool
	cmd := &cobra.Command{
		Use:   "movie <module-id> <movie-id>",
		Short: "Set movie monitored flag",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMovieClient(args[0], func(ctx context.Context, cli mgmntv1.MovieManagementServiceClient) error {
				_, err := cli.UpdateMovie(ctx, &mgmntv1.UpdateMovieRequest{
					MovieId: args[1], Monitored: proto.Bool(monitored),
				})
				if err != nil {
					return fmt.Errorf("media monitor movie: %w", err)
				}
				if !flagQuiet {
					fmt.Printf("monitored=%t\n", monitored)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&monitored, "monitored", true, "monitored state")
	return cmd
}

func newMediaMonitorSeasonCmd() *cobra.Command {
	var monitored bool
	cmd := &cobra.Command{
		Use:   "season <module-id> <season-id>",
		Short: "Set TV season monitored flag",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTVClient(args[0], func(ctx context.Context, cli tvmgmtv1.TvManagementServiceClient) error {
				_, err := cli.UpdateSeasonMonitored(ctx, &tvmgmtv1.UpdateSeasonMonitoredRequest{
					SeasonId: args[1], Monitored: monitored,
				})
				if err != nil {
					return fmt.Errorf("media monitor season: %w", err)
				}
				if !flagQuiet {
					fmt.Printf("monitored=%t\n", monitored)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&monitored, "monitored", true, "monitored state")
	return cmd
}

func newMediaMonitorEpisodeCmd() *cobra.Command {
	var monitored bool
	cmd := &cobra.Command{
		Use:   "episode <module-id> <episode-id>",
		Short: "Set TV episode monitored flag",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTVClient(args[0], func(ctx context.Context, cli tvmgmtv1.TvManagementServiceClient) error {
				_, err := cli.UpdateEpisodeMonitored(ctx, &tvmgmtv1.UpdateEpisodeMonitoredRequest{
					EpisodeId: args[1], Monitored: monitored,
				})
				if err != nil {
					return fmt.Errorf("media monitor episode: %w", err)
				}
				if !flagQuiet {
					fmt.Printf("monitored=%t\n", monitored)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&monitored, "monitored", true, "monitored state")
	return cmd
}

func withMovieClient(moduleID string, fn func(context.Context, mgmntv1.MovieManagementServiceClient) error) error {
	return dialModuleByID(moduleID, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, mgmntv1.NewMovieManagementServiceClient(conn))
	})
}

func withTVClient(moduleID string, fn func(context.Context, tvmgmtv1.TvManagementServiceClient) error) error {
	return dialModuleByID(moduleID, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, tvmgmtv1.NewTvManagementServiceClient(conn))
	})
}

func dialModuleByID(moduleID string, fn func(context.Context, *grpc.ClientConn) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		mod, err := findModuleByID(ctx, c, moduleID)
		if err != nil {
			return err
		}
		conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
		if err != nil {
			return err
		}
		defer conn.Close()
		return fn(ctx, conn)
	})
}

func automationItemType(moduleID, displayName string) string {
	s := strings.ToLower(moduleID + " " + displayName)
	switch {
	case strings.Contains(s, "movie"):
		return "movie"
	case strings.Contains(s, "tv") || strings.Contains(s, "show"):
		return "tv"
	default:
		return "movie"
	}
}

func itemTMDBID(item *mediaadminv1.MediaItem) int32 {
	if item == nil {
		return 0
	}
	if v := item.GetMetadata()["tmdb_id"]; v != "" {
		id, _ := strconv.Atoi(v)
		return int32(id)
	}
	return 0
}

func newMediaDispatchCmd() *cobra.Command {
	var searchBest bool
	cmd := &cobra.Command{
		Use:   "dispatch <module-id> <item-id>",
		Short: "Dispatch automation grab for a library item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleID, itemID := args[0], args[1]
			return withMediaClient(moduleID, func(ctx context.Context, mediaCli mediaadminv1.MediaAdminServiceClient) error {
				info, _ := mediaCli.GetMediaTypeInfo(ctx, &mediaadminv1.GetMediaTypeInfoRequest{})
				displayName := moduleID
				if info != nil {
					displayName = info.GetDisplayName()
				}
				itemType := automationItemType(moduleID, displayName)
				got, err := mediaCli.GetItem(ctx, &mediaadminv1.GetItemRequest{Id: itemID})
				if err != nil {
					return fmt.Errorf("media dispatch: %w", err)
				}
				it := got.GetItem()
				title := it.GetTitle()
				tmdbID := itemTMDBID(it)
				year := int32(it.GetYear())
				return withAutomationClient(func(ctx context.Context, autoCli automationv1.AutomationServiceClient) error {
					req := &automationv1.DispatchRequest{
						ItemId: itemID, ItemType: itemType, Title: title, TmdbId: tmdbID,
					}
					if searchBest {
						search, err := autoCli.SearchItem(ctx, &automationv1.SearchItemRequest{
							ItemType: itemType, Query: title, TmdbId: tmdbID, Year: year, Limit: 1,
						})
						if err != nil {
							return fmt.Errorf("media dispatch search: %w", err)
						}
						if len(search.GetMatches()) > 0 {
							rel := search.GetMatches()[0]
							req.Guid = rel.GetGuid()
							req.DownloadUrl = rel.GetDownloadUrl()
						}
					}
					resp, err := autoCli.Dispatch(ctx, req)
					if err != nil {
						return fmt.Errorf("media dispatch: %w", err)
					}
					if flagJSON {
						return printJSON(resp)
					}
					if !flagQuiet {
						fmt.Printf("status=%s download_id=%s\n", resp.GetStatus(), resp.GetDownloadId())
					}
					return nil
				})
			})
		},
	}
	cmd.Flags().BoolVar(&searchBest, "search-best", false, "search indexers and dispatch best release")
	return cmd
}

func newMediaCollectionsSyncCmd() *cobra.Command {
	var addMissing bool
	cmd := &cobra.Command{
		Use:   "sync <module-id> <collection-id>",
		Short: "Sync a movie collection from TMDB",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[1])
			if err != nil || id == 0 {
				return fmt.Errorf("invalid collection id")
			}
			return withMovieClient(args[0], func(ctx context.Context, cli mgmntv1.MovieManagementServiceClient) error {
				resp, err := cli.SyncCollection(ctx, &mgmntv1.SyncCollectionRequest{
					CollectionId: int32(id), AddMissing: addMissing,
				})
				if err != nil {
					return fmt.Errorf("collection sync: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("added=%d already_present=%d\n", resp.GetAdded(), resp.GetAlreadyPresent())
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&addMissing, "add-missing", true, "add missing movies from collection")
	return cmd
}

func newMediaCollectionsMonitorCmd() *cobra.Command {
	var monitored bool
	cmd := &cobra.Command{
		Use:   "monitor <module-id> <collection-id>",
		Short: "Set collection monitored state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[1])
			if err != nil || id == 0 {
				return fmt.Errorf("invalid collection id")
			}
			return withMovieClient(args[0], func(ctx context.Context, cli mgmntv1.MovieManagementServiceClient) error {
				_, err := cli.SetCollectionMonitored(ctx, &mgmntv1.SetCollectionMonitoredRequest{
					CollectionId: int32(id), Monitored: monitored, SearchOnAdd: true,
				})
				if err != nil {
					return fmt.Errorf("collection monitor: %w", err)
				}
				if !flagQuiet {
					fmt.Printf("monitored=%t\n", monitored)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&monitored, "monitored", true, "monitored state")
	return cmd
}

func newMediaFilesCmd() *cobra.Command {
	var deleteFiles bool
	deleteCmd := &cobra.Command{
		Use:   "delete <module-id> <file-id>",
		Short: "Remove a media file from the library",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete media file " + args[1] + "?"); err != nil {
				return err
			}
			return withMovieClient(args[0], func(ctx context.Context, cli mgmntv1.MovieManagementServiceClient) error {
				_, err := cli.RemoveFile(ctx, &mgmntv1.RemoveFileRequest{FileId: args[1], DeleteFiles: deleteFiles})
				if err != nil {
					return fmt.Errorf("media file delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("removed")
				}
				return nil
			})
		},
	}
	deleteCmd.Flags().BoolVar(&deleteFiles, "delete-disk", false, "also delete file from disk")
	episodeDelete := &cobra.Command{
		Use:   "episode-delete <module-id> <episode-id>",
		Short: "Remove episode file from a TV library",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete episode file " + args[1] + "?"); err != nil {
				return err
			}
			return withTVClient(args[0], func(ctx context.Context, cli tvmgmtv1.TvManagementServiceClient) error {
				_, err := cli.RemoveEpisodeFile(ctx, &tvmgmtv1.RemoveEpisodeFileRequest{
					EpisodeId: args[1], DeleteFiles: deleteFiles,
				})
				if err != nil {
					return fmt.Errorf("episode file delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("removed")
				}
				return nil
			})
		},
	}
	episodeDelete.Flags().BoolVar(&deleteFiles, "delete-disk", false, "also delete file from disk")
	cmd := &cobra.Command{Use: "files", Short: "Media file commands"}
	cmd.AddCommand(deleteCmd, episodeDelete)
	return cmd
}

func newMediaArtworkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "artwork <module-id> <item-id>",
		Short: "List artwork for a media item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMediaClient(args[0], func(ctx context.Context, cli mediaadminv1.MediaAdminServiceClient) error {
				resp, err := cli.ListArtwork(ctx, &mediaadminv1.ListArtworkRequest{Id: args[1]})
				if err != nil {
					return fmt.Errorf("media artwork: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetArtwork())
				}
				rows := make([][]string, 0, len(resp.GetArtwork()))
				for _, a := range resp.GetArtwork() {
					rows = append(rows, []string{a.GetType(), a.GetUrl()})
				}
				return printTable([]string{"TYPE", "URL"}, rows)
			})
		},
	}
}

func newMediaTitlesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "titles", Short: "Alternate title commands"}
	cmd.AddCommand(newMediaTitlesAddCmd())
	cmd.AddCommand(newMediaTitlesDeleteCmd())
	return cmd
}

func newMediaTitlesAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <module-id> <item-id> <title>",
		Short: "Add an alternate title",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleID, itemID, title := args[0], args[1], args[2]
			return withMediaClient(moduleID, func(ctx context.Context, mediaCli mediaadminv1.MediaAdminServiceClient) error {
				info, _ := mediaCli.GetMediaTypeInfo(ctx, &mediaadminv1.GetMediaTypeInfoRequest{})
				displayName := moduleID
				if info != nil {
					displayName = info.GetDisplayName()
				}
				kind := automationItemType(moduleID, displayName)
				switch kind {
				case "movie":
					return withMovieClient(moduleID, func(ctx context.Context, cli mgmntv1.MovieManagementServiceClient) error {
						resp, err := cli.AddAlternateTitle(ctx, &mgmntv1.AddAlternateTitleRequest{MovieId: itemID, Title: title})
						if err != nil {
							return fmt.Errorf("media title add: %w", err)
						}
						if flagJSON {
							return printJSON(resp.GetTitle())
						}
						if !flagQuiet {
							fmt.Println("added")
						}
						return nil
					})
				case "tv":
					return withTVClient(moduleID, func(ctx context.Context, cli tvmgmtv1.TvManagementServiceClient) error {
						resp, err := cli.AddAlternateTitle(ctx, &tvmgmtv1.AddAlternateTitleRequest{SeriesId: itemID, Title: title})
						if err != nil {
							return fmt.Errorf("media title add: %w", err)
						}
						if flagJSON {
							return printJSON(resp.GetTitle())
						}
						if !flagQuiet {
							fmt.Println("added")
						}
						return nil
					})
				default:
					return fmt.Errorf("unsupported library type")
				}
			})
		},
	}
}

func newMediaTitlesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <module-id> <item-id> <title-id>",
		Short: "Remove an alternate title",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleID, itemID, titleID := args[0], args[1], args[2]
			return withMediaClient(moduleID, func(ctx context.Context, mediaCli mediaadminv1.MediaAdminServiceClient) error {
				info, _ := mediaCli.GetMediaTypeInfo(ctx, &mediaadminv1.GetMediaTypeInfoRequest{})
				displayName := moduleID
				if info != nil {
					displayName = info.GetDisplayName()
				}
				kind := automationItemType(moduleID, displayName)
				switch kind {
				case "movie":
					return withMovieClient(moduleID, func(ctx context.Context, cli mgmntv1.MovieManagementServiceClient) error {
						_, err := cli.RemoveAlternateTitle(ctx, &mgmntv1.RemoveAlternateTitleRequest{MovieId: itemID, TitleId: titleID})
						if err != nil {
							return fmt.Errorf("media title delete: %w", err)
						}
						if !flagQuiet {
							fmt.Println("deleted")
						}
						return nil
					})
				case "tv":
					return withTVClient(moduleID, func(ctx context.Context, cli tvmgmtv1.TvManagementServiceClient) error {
						_, err := cli.RemoveAlternateTitle(ctx, &tvmgmtv1.RemoveAlternateTitleRequest{SeriesId: itemID, TitleId: titleID})
						if err != nil {
							return fmt.Errorf("media title delete: %w", err)
						}
						if !flagQuiet {
							fmt.Println("deleted")
						}
						return nil
					})
				default:
					return fmt.Errorf("unsupported library type")
				}
			})
		},
	}
}
