package cli

import (
	"context"
	"fmt"
	"os"

	spoolv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/spool/v1"
	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
)

func newMarketplaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "marketplace",
		Short:   "Browse spools and deploy tags (admin-ui /marketplace parity)",
		GroupID: groupOverview,
	}
	cmd.AddCommand(newMarketplaceSpoolsCmd())
	cmd.AddCommand(newMarketplaceTagsCmd())
	cmd.AddCommand(newMarketplaceDeployCmd())
	cmd.AddCommand(newMarketplaceTrustCmd())
	return cmd
}

func newMarketplaceSpoolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "spools",
		Short: "List configured spool URLs",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, ctx, cancel, err := spoolClient()
			if err != nil {
				return err
			}
			defer cancel()
			resp, err := client.ListSpools(ctx, &spoolv1.ListSpoolsRequest{})
			if err != nil {
				return fmt.Errorf("marketplace spools: %w", err)
			}
			if flagJSON {
				return printJSON(resp.GetSpools())
			}
			for _, s := range resp.GetSpools() {
				active := ""
				if s.GetActive() {
					active = " (active)"
				}
				fmt.Printf("%s%s\n", s.GetUrl(), active)
			}
			return nil
		},
	}
}

func newMarketplaceTagsCmd() *cobra.Command {
	var spoolURL string
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List tags on a spool",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, ctx, cancel, err := spoolClient()
			if err != nil {
				return err
			}
			defer cancel()
			resp, err := client.ListTags(ctx, &spoolv1.ListTagsRequest{SpoolUrl: spoolURL})
			if err != nil {
				return fmt.Errorf("marketplace tags: %w", err)
			}
			if flagJSON {
				return printJSON(resp.GetTags())
			}
			rows := make([][]string, 0, len(resp.GetTags()))
			for _, t := range resp.GetTags() {
				rows = append(rows, []string{t.GetName(), t.GetVersion(), t.GetDescription()})
			}
			return printTable([]string{"NAME", "VERSION", "DESCRIPTION"}, rows)
		},
	}
	cmd.Flags().StringVar(&spoolURL, "spool-url", "", "override spool URL (empty = default)")
	return cmd
}

func newMarketplaceDeployCmd() *cobra.Command {
	var spoolURL string
	cmd := &cobra.Command{
		Use:   "deploy <tag>",
		Short: "Deploy a spool tag (spawn its modules)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Deploy tag " + args[0] + "?"); err != nil {
				return err
			}
			client, ctx, cancel, err := spoolClient()
			if err != nil {
				return err
			}
			defer cancel()
			resp, err := client.DeployTag(ctx, &spoolv1.DeployTagRequest{
				SpoolUrl: spoolURL,
				TagName:  args[0],
			})
			if err != nil {
				return fmt.Errorf("marketplace deploy: %w", err)
			}
			if flagJSON {
				return printJSON(resp)
			}
			fmt.Printf("tag=%s spawned=%d skipped=%d failed=%d\n",
				resp.GetTagName(), resp.GetSpawned(), resp.GetSkipped(), resp.GetFailed())
			for _, r := range resp.GetResults() {
				fmt.Fprintf(os.Stdout, "  %s spawned=%v already_running=%v error=%q\n",
					r.GetModuleId(), r.GetSpawned(), r.GetAlreadyRunning(), r.GetError())
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&spoolURL, "spool-url", "", "override spool URL (empty = default)")
	return cmd
}

func spoolClient() (spoolv1.SpoolServiceClient, context.Context, context.CancelFunc, error) {
	opts := dialOpts()
	conn, err := dialRaw(opts)
	if err != nil {
		return nil, nil, nil, err
	}
	ctx, cancel := connect.Context(opts)
	if opts.Token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+opts.Token)
	}
	// conn is owned by caller lifetime via cancel - we'll leak if we don't close.
	// Wrap cancel to close conn.
	origCancel := cancel
	cancel = func() {
		origCancel()
		_ = conn.Close()
	}
	return spoolv1.NewSpoolServiceClient(conn), ctx, cancel, nil
}
