package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Muxcore-Media/admin-ui/arrmigrate"
	formatsv1 "github.com/Muxcore-Media/media-custom-formats/proto/formatsv1"
	mgmntv1 "github.com/Muxcore-Media/media-movies/proto/mgmntv1"
	tvmgmtv1 "github.com/Muxcore-Media/media-tvshows/proto/tvmgmtv1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func newMigrateCmd() *cobra.Command {
	var (
		service string
		baseURL string
		apiKey  string
		dryRun  bool
	)
	cmd := &cobra.Command{
		Use:     "migrate",
		Short:   "Import library from Radarr or Sonarr (admin-ui /migrate parity)",
		GroupID: groupAutomation,
		RunE: func(cmd *cobra.Command, args []string) error {
			service = strings.ToLower(strings.TrimSpace(service))
			if service == "" {
				service = "radarr"
			}
			if baseURL == "" || apiKey == "" {
				return fmt.Errorf("--url and --api-key are required")
			}
			ctx := cmd.Context()
			cli := &arrmigrate.Client{}
			var items []arrmigrate.Item
			var err error
			switch service {
			case "radarr":
				items, err = cli.FetchRadarr(ctx, baseURL, apiKey)
			case "sonarr":
				items, err = cli.FetchSonarr(ctx, baseURL, apiKey)
			case "lidarr":
				items, err = cli.FetchLidarr(ctx, baseURL, apiKey)
			default:
				return fmt.Errorf("service must be radarr, sonarr, or lidarr")
			}
			if err != nil {
				return fmt.Errorf("migrate fetch: %w", err)
			}
			movies, tv, closer, resolveErr := resolveMigrateImporters()
			if !dryRun && resolveErr != nil {
				return resolveErr
			}
			if closer != nil {
				defer closer()
			}
			res := arrmigrate.Run(ctx, items, dryRun, movies, tv, nil, resolveProfileByName)
			if flagJSON {
				return printJSON(res)
			}
			fmt.Printf("dry_run=%t fetched=%d imported=%d skipped=%d errors=%d\n",
				res.DryRun, res.Fetched, res.Imported, res.Skipped, len(res.Errors))
			for i, e := range res.Errors {
				if i >= 20 {
					fmt.Printf("  ... and %d more errors\n", len(res.Errors)-20)
					break
				}
				fmt.Printf("  error: %s\n", e)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&service, "service", "radarr", "radarr, sonarr, or lidarr")
	cmd.Flags().StringVar(&baseURL, "url", "", "Arr base URL")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Arr API key")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "preview without importing")
	return cmd
}

type movieImporter struct {
	client mgmntv1.MovieManagementServiceClient
}

func (a movieImporter) ImportMovie(ctx context.Context, title string, year, tmdbID int, qualityProfileID, rootFolder string, monitored bool) (string, error) {
	resp, err := a.client.AddMovie(ctx, &mgmntv1.AddMovieRequest{
		TmdbId: int32(tmdbID), Title: title, Year: int32(year),
		QualityProfileId: qualityProfileID, RootFolderPath: rootFolder,
	})
	if err != nil {
		return "", err
	}
	id := resp.GetMovieId()
	if id != "" {
		_, _ = a.client.UpdateMovie(ctx, &mgmntv1.UpdateMovieRequest{
			MovieId: id, Monitored: proto.Bool(monitored),
		})
	}
	return id, nil
}

type tvImporter struct {
	client tvmgmtv1.TvManagementServiceClient
}

func (a tvImporter) ImportSeries(ctx context.Context, title string, year, tmdbID int, qualityProfileID, rootFolder string, monitored bool) (string, error) {
	resp, err := a.client.AddTVShow(ctx, &tvmgmtv1.AddTVShowRequest{
		TmdbId: int32(tmdbID), Name: title, Year: int32(year),
		QualityProfileId: qualityProfileID, RootFolderPath: rootFolder,
	})
	if err != nil {
		return "", err
	}
	id := resp.GetSeriesId()
	if id != "" {
		_, _ = a.client.UpdateTVShow(ctx, &tvmgmtv1.UpdateTVShowRequest{
			SeriesId: id, Monitored: proto.Bool(monitored),
		})
	}
	return id, nil
}

func resolveMigrateImporters() (arrmigrate.MovieImporter, arrmigrate.TVImporter, func(), error) {
	var movies arrmigrate.MovieImporter
	var tv arrmigrate.TVImporter
	var closers []func()
	err := withCore(func(ctx context.Context, c *client.Client) error {
		if mod, err := findModuleByCapability(ctx, c, "media.library.movies"); err == nil {
			conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
			if err == nil {
				closers = append(closers, func() { _ = conn.Close() })
				movies = movieImporter{client: mgmntv1.NewMovieManagementServiceClient(conn)}
			}
		}
		if mod, err := findModuleByCapability(ctx, c, "media.library.tv"); err == nil {
			conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
			if err == nil {
				closers = append(closers, func() { _ = conn.Close() })
				tv = tvImporter{client: tvmgmtv1.NewTvManagementServiceClient(conn)}
			}
		}
		return nil
	})
	closeAll := func() {
		for _, cl := range closers {
			cl()
		}
	}
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	if movies == nil && tv == nil {
		closeAll()
		return nil, nil, nil, fmt.Errorf("no media.library.movies or media.library.tv module")
	}
	return movies, tv, closeAll, nil
}

func resolveProfileByName(ctx context.Context, name string) string {
	if name == "" {
		return ""
	}
	var id string
	_ = withFormatsClient(func(ctx context.Context, cli formatsv1.FormatServiceClient) error {
		resp, err := cli.ListProfiles(ctx, &formatsv1.ListProfilesRequest{})
		if err != nil {
			return err
		}
		for _, p := range resp.GetProfiles() {
			if strings.EqualFold(p.GetName(), name) {
				id = p.GetId()
				break
			}
		}
		return nil
	})
	return id
}
