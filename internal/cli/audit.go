package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
)

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit log commands",
	}

	var (
		actor, action, resource, traceID string
		fromTime, toTime                 string
		maxResults                       int32
	)
	query := &cobra.Command{
		Use:   "query",
		Short: "Query audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx, cancel := connect.Context(opts)
			defer cancel()

			entries, err := c.Audit.Query(ctx, actor, action, resource, traceID, fromTime, toTime, maxResults)
			if err != nil {
				return fmt.Errorf("audit query: %w", err)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTIME\tACTOR\tACTION\tRESOURCE\tRESOURCE_ID")
			for _, e := range entries {
				ts := time.Unix(e.GetTimestamp(), 0).UTC().Format(time.RFC3339)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					e.GetId(), ts, e.GetActor(), e.GetAction(), e.GetResource(), e.GetResourceId())
			}
			return w.Flush()
		},
	}
	query.Flags().StringVar(&actor, "actor", "", "filter by actor")
	query.Flags().StringVar(&action, "action", "", "filter by action")
	query.Flags().StringVar(&resource, "resource", "", "filter by resource")
	query.Flags().StringVar(&traceID, "trace-id", "", "filter by trace id")
	query.Flags().StringVar(&fromTime, "from", "", "RFC3339 lower bound")
	query.Flags().StringVar(&toTime, "to", "", "RFC3339 upper bound")
	query.Flags().Int32Var(&maxResults, "max", 50, "max results (0 = server default)")
	cmd.AddCommand(query)
	return cmd
}
