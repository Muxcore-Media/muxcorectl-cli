package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	mediaadminv1 "github.com/Muxcore-Media/contracts-media-admin/gen/muxcore/media/admin/v1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newCalendarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "calendar",
		Short:   "Air-date calendar across TV libraries (admin-ui /calendar parity)",
		GroupID: groupAutomation,
	}
	cmd.AddCommand(newCalendarListCmd())
	return cmd
}

func newCalendarListCmd() *cobra.Command {
	var start, end string
	var includeUnmonitored bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List calendar events from all TV library modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			now := time.Now().UTC()
			if start == "" {
				start = now.AddDate(0, 0, -7).Format("2006-01-02")
			}
			if end == "" {
				end = now.AddDate(0, 0, 21).Format("2006-01-02")
			}
			return withCore(func(ctx context.Context, c *client.Client) error {
				mods, err := listMediaLibraries(ctx, c)
				if err != nil {
					return fmt.Errorf("calendar: %w", err)
				}
				type row struct {
					Module    string `json:"module"`
					Date      string `json:"date"`
					Title     string `json:"title"`
					Subtitle  string `json:"subtitle"`
					Season    string `json:"season"`
					Episode   string `json:"episode"`
					HasFile   bool   `json:"has_file"`
					Monitored bool   `json:"monitored"`
				}
				var out []row
				var skips []string
				for _, mod := range mods {
					conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
					if err != nil {
						skips = append(skips, mod.GetId()+": "+err.Error())
						continue
					}
					cli := mediaadminv1.NewMediaAdminServiceClient(conn)
					resp, err := cli.GetCalendar(ctx, &mediaadminv1.GetCalendarRequest{
						StartDate: start, EndDate: end, IncludeUnmonitored: includeUnmonitored,
					})
					_ = conn.Close()
					if err != nil {
						if strings.Contains(strings.ToLower(err.Error()), "unimplemented") {
							continue
						}
						skips = append(skips, mod.GetId()+": "+err.Error())
						continue
					}
					for _, it := range resp.GetItems() {
						out = append(out, row{
							Module: mod.GetId(), Date: it.GetDate(), Title: it.GetTitle(),
							Subtitle: it.GetSubtitle(),
							Season: it.GetMetadata()["season_number"],
							Episode: it.GetMetadata()["episode_number"],
							HasFile: it.GetHasFile(), Monitored: it.GetMonitored(),
						})
					}
				}
				sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
				if flagJSON {
					return printJSON(map[string]any{"start": start, "end": end, "items": out, "skipped": skips})
				}
				if len(skips) > 0 && !flagQuiet {
					fmt.Fprintf(os.Stderr, "note: skipped modules: %s\n", strings.Join(skips, "; "))
				}
				rows := make([][]string, 0, len(out))
				for _, r := range out {
					ep := r.Subtitle
					if r.Season != "" || r.Episode != "" {
						ep = fmt.Sprintf("S%sE%s %s", r.Season, r.Episode, r.Subtitle)
					}
					rows = append(rows, []string{r.Date, r.Module, r.Title, ep, fmt.Sprintf("%t", r.HasFile)})
				}
				fmt.Printf("range %s .. %s (%d events)\n", start, end, len(out))
				return printTable([]string{"DATE", "MODULE", "SERIES", "EPISODE", "HAS_FILE"}, rows)
			})
		},
	}
	cmd.Flags().StringVar(&start, "start", "", "start date YYYY-MM-DD (default: 7 days ago)")
	cmd.Flags().StringVar(&end, "end", "", "end date YYYY-MM-DD (default: 21 days ahead)")
	cmd.Flags().BoolVar(&includeUnmonitored, "unmonitored", false, "include unmonitored episodes")
	return cmd
}
