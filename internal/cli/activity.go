package cli

import (
	"context"
	"fmt"
	"sort"

	mediaadminv1 "github.com/Muxcore-Media/contracts-media-admin/gen/muxcore/media/admin/v1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newActivityCmd() *cobra.Command {
	var eventType string
	var limit int
	cmd := &cobra.Command{
		Use:     "activity",
		Short:   "Grab/import activity across libraries (admin-ui /activity parity)",
		GroupID: groupMonitoring,
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 {
				limit = 50
			}
			type row struct {
				Module    string `json:"module"`
				EventType string `json:"event_type"`
				Title     string `json:"title"`
				Source    string `json:"source_title"`
				Indexer   string `json:"indexer"`
				At        string `json:"created_at"`
			}
			var out []row
			err := withCore(func(ctx context.Context, c *client.Client) error {
				mods, err := listMediaLibraries(ctx, c)
				if err != nil {
					return err
				}
				for _, mod := range mods {
					conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
					if err != nil {
						continue
					}
					cli := mediaadminv1.NewMediaAdminServiceClient(conn)
					resp, err := cli.ListHistory(ctx, &mediaadminv1.ListHistoryRequest{
						Page: 1, PageSize: 200, EventType: eventType,
					})
					_ = conn.Close()
					if err != nil {
						continue
					}
					for _, rec := range resp.GetRecords() {
						out = append(out, row{
							Module: mod.GetId(), EventType: rec.GetEventType(), Title: rec.GetTitle(),
							Source: rec.GetSourceTitle(), Indexer: rec.GetIndexer(), At: rec.GetCreatedAt(),
						})
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("activity: %w", err)
			}
			sort.Slice(out, func(i, j int) bool { return out[i].At > out[j].At })
			if len(out) > limit {
				out = out[:limit]
			}
			if flagJSON {
				return printJSON(out)
			}
			rows := make([][]string, 0, len(out))
			for _, r := range out {
				rows = append(rows, []string{r.At, r.Module, r.EventType, r.Title, r.Indexer})
			}
			return printTable([]string{"AT", "MODULE", "EVENT", "TITLE", "INDEXER"}, rows)
		},
	}
	cmd.Flags().StringVar(&eventType, "event-type", "", "filter by event type")
	cmd.Flags().IntVar(&limit, "limit", 50, "max rows")
	return cmd
}
