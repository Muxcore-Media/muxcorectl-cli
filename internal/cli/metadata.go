package cli

import (
	"context"
	"fmt"

	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newMetadataCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "metadata",
		Short:   "List media libraries for metadata management (admin-ui /metadata parity)",
		GroupID: groupLibrary,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCore(func(ctx context.Context, c *client.Client) error {
				mods, err := listMediaLibraries(ctx, c)
				if err != nil {
					return err
				}
				if flagJSON {
					type lib struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					}
					out := make([]lib, 0, len(mods))
					for _, m := range mods {
						out = append(out, lib{ID: m.GetId(), Name: m.GetName()})
					}
					return printJSON(out)
				}
				rows := make([][]string, 0, len(mods))
				for _, m := range mods {
					rows = append(rows, []string{m.GetId(), m.GetName(), "media items --module " + m.GetId()})
				}
				if len(rows) == 0 {
					fmt.Println("No media library modules registered.")
					return nil
				}
				fmt.Println("Use media commands to manage items in each library:")
				return printTable([]string{"MODULE", "NAME", "HINT"}, rows)
			})
		},
	}
}
