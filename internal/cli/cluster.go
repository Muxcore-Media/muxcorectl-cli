package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
)

func newClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "Cluster membership commands",
		GroupID: groupOverview,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show cluster members and leader",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer func() { _ = c.Close() }()

			ctx, cancel := connect.Context(opts)
			defer cancel()

			members, leader, err := c.Discovery.Members(ctx)
			if err != nil {
				return fmt.Errorf("cluster status: %w", err)
			}

			fmt.Printf("leader: %s\n", leader)
			fmt.Printf("members: %d\n", len(members))
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tGRPC\tHTTP\tMODULES")
			for _, m := range members {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					m.GetId(), m.GetGrpcAddr(), m.GetHttpAddr(), strings.Join(m.GetModules(), ","))
			}
			return w.Flush()
		},
	})
	return cmd
}
