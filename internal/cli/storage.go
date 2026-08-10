package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
)

func newStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Storage commands",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "ls [prefix]",
		Short: "List objects under a storage key prefix",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prefix := ""
			if len(args) == 1 {
				prefix = args[0]
			}
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx, cancel := connect.Context(opts)
			defer cancel()

			objs, err := c.Storage.List(ctx, prefix)
			if err != nil {
				return fmt.Errorf("storage ls: %w", err)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "KEY\tSIZE\tCONTENT_TYPE\tMODIFIED")
			for _, o := range objs {
				fmt.Fprintf(w, "%s\t%d\t%s\t%d\n", o.GetKey(), o.GetSize(), o.GetContentType(), o.GetLastModified())
			}
			return w.Flush()
		},
	})
	return cmd
}
