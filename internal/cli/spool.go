package cli

import (
	"fmt"
	"os"

	spoolv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/spool/v1"
	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func newSpoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "spool",
		Short:   "Spool / tag commands",
		GroupID: groupOverview,
	}

	var spoolURL string
	resolve := &cobra.Command{
		Use:   "resolve <tag>",
		Short: "Fetch a spool tag definition without deploying it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dialOpts()
			conn, err := dialRaw(opts)
			if err != nil {
				return err
			}
			defer func() { _ = conn.Close() }()

			ctx, cancel := connect.Context(opts)
			defer cancel()
			if opts.Token != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+opts.Token)
			}

			client := spoolv1.NewSpoolServiceClient(conn)
			resp, err := client.FetchTag(ctx, &spoolv1.FetchTagRequest{
				SpoolUrl: spoolURL,
				TagName:  args[0],
			})
			if err != nil {
				return fmt.Errorf("spool resolve: %w", err)
			}

			fmt.Printf("name:        %s\n", resp.GetName())
			fmt.Printf("version:     %s\n", resp.GetVersion())
			fmt.Printf("description: %s\n", resp.GetDescription())
			fmt.Printf("spool_url:   %s\n", resp.GetSpoolUrl())
			fmt.Printf("modules:     %d\n", len(resp.GetModules()))
			for _, m := range resp.GetModules() {
				_, _ = fmt.Fprintf(os.Stdout, "  - %s@%s required=%v instance=%s\n",
					m.GetRepo(), m.GetVersion(), m.GetRequired(), m.GetInstanceId())
			}
			return nil
		},
	}
	resolve.Flags().StringVar(&spoolURL, "spool-url", "", "override spool URL (empty = muxcored default)")
	cmd.AddCommand(resolve)
	return cmd
}

func dialRaw(opts connect.Options) (*grpc.ClientConn, error) {
	dialOpts, err := meshGRPCDialOptions()
	if err != nil {
		return nil, err
	}
	return grpc.NewClient(opts.Addr, dialOpts...)
}
