package cli

import (
	"context"
	"fmt"

	jellyfinv1 "github.com/Muxcore-Media/jellyfin/proto/jellyfinv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capPlaybackJellyfin = "playback.jellyfin"

func withJellyfinClient(fn func(context.Context, jellyfinv1.JellyfinBridgeClient) error) error {
	return withModuleConn(capPlaybackJellyfin, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, jellyfinv1.NewJellyfinBridgeClient(conn))
	})
}

func newJellyfinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "jellyfin",
		Short:   "Jellyfin bridge status and sync (admin-ui /jellyfin parity)",
		GroupID: groupPlayback,
	}
	cmd.AddCommand(newJellyfinStatusCmd())
	cmd.AddCommand(newJellyfinSyncCmd())
	cmd.AddCommand(newJellyfinRefreshCmd())
	return cmd
}

func newJellyfinStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Jellyfin bridge configuration and link count",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withJellyfinClient(func(ctx context.Context, cli jellyfinv1.JellyfinBridgeClient) error {
				resp, err := cli.Status(ctx, &jellyfinv1.StatusRequest{})
				if err != nil {
					return fmt.Errorf("jellyfin status: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				fmt.Printf("configured:   %t\n", resp.GetConfigured())
				fmt.Printf("base_url:     %s\n", resp.GetBaseUrl())
				fmt.Printf("item_links:   %d\n", resp.GetItemLinks())
				fmt.Printf("conflict:     %s\n", resp.GetConflictMode())
				fmt.Printf("sessions_poll: %t\n", resp.GetSessionsPollEnabled())
				return nil
			})
		},
	}
}

func newJellyfinSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Rebuild MuxCore item links from Jellyfin library",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withJellyfinClient(func(ctx context.Context, cli jellyfinv1.JellyfinBridgeClient) error {
				resp, err := cli.SyncLibrary(ctx, &jellyfinv1.SyncLibraryRequest{})
				if err != nil {
					return fmt.Errorf("jellyfin sync: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("synced matched=%d upserted=%d scanned=%d\n",
						resp.GetMatched(), resp.GetUpserted(), resp.GetScanned())
				}
				return nil
			})
		},
	}
}

func newJellyfinRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Ask Jellyfin to rescan its library",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withJellyfinClient(func(ctx context.Context, cli jellyfinv1.JellyfinBridgeClient) error {
				resp, err := cli.RefreshLibrary(ctx, &jellyfinv1.RefreshLibraryRequest{})
				if err != nil {
					return fmt.Errorf("jellyfin refresh: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Println("refresh requested")
				}
				return nil
			})
		},
	}
}
