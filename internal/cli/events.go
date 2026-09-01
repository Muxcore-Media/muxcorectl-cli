package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
)

func newEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "events",
		Short:   "Event bus commands",
		GroupID: groupMonitoring,
	}

	var eventType string
	var maxEvents int
	tail := &cobra.Command{
		Use:   "tail",
		Short: "Subscribe to events (Ctrl-C to stop)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if eventType == "" {
				eventType = "*"
			}
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer func() { _ = c.Close() }()

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			ch, cancel, err := c.Events.Subscribe(ctx, eventType)
			if err != nil {
				return fmt.Errorf("events tail: %w", err)
			}
			defer cancel()

			fmt.Fprintf(os.Stderr, "tailing events type=%q (addr=%s)\n", eventType, opts.Addr)
			n := 0
			for {
				select {
				case <-ctx.Done():
					return nil
				case ev, ok := <-ch:
					if !ok {
						return nil
					}
					ts := time.Now().Format(time.RFC3339)
					if flagJSON {
						if err := printJSON(map[string]any{
							"ts": ts, "type": ev.GetType(), "source": ev.GetSource(), "bytes": len(ev.GetPayload()),
						}); err != nil {
							return err
						}
					} else {
						fmt.Printf("%s type=%s source=%s bytes=%d\n", ts, ev.GetType(), ev.GetSource(), len(ev.GetPayload()))
					}
					n++
					if maxEvents > 0 && n >= maxEvents {
						return nil
					}
				}
			}
		},
	}
	tail.Flags().StringVar(&eventType, "type", "*", "event type filter")
	tail.Flags().IntVar(&maxEvents, "max", 0, "stop after N events (0 = unlimited)")
	cmd.AddCommand(tail)
	cmd.AddCommand(newEventsStatsCmd())
	return cmd
}

func newEventsStatsCmd() *cobra.Command {
	var maxEvents int
	var sampleSeconds int
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Sample recent event type counts (admin-ui /events/stats approximation)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sampleSeconds < 1 {
				sampleSeconds = 3
			}
			if maxEvents < 1 {
				maxEvents = 500
			}
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer func() { _ = c.Close() }()

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sampleSeconds)*time.Second)
			defer cancel()

			ch, cancelSub, err := c.Events.Subscribe(ctx, "*")
			if err != nil {
				return fmt.Errorf("events stats: %w", err)
			}
			defer cancelSub()

			counts := map[string]int{}
			total := 0
			for {
				select {
				case <-ctx.Done():
					if flagJSON {
						type row struct {
							Type  string `json:"type"`
							Count int    `json:"count"`
						}
						var out []row
						for typ, n := range counts {
							out = append(out, row{Type: typ, Count: n})
						}
						return printJSON(map[string]any{"sample_seconds": sampleSeconds, "total": total, "types": out})
					}
					rows := make([][]string, 0, len(counts))
					for typ, n := range counts {
						rows = append(rows, []string{typ, fmt.Sprintf("%d", n)})
					}
					fmt.Printf("sample_seconds=%d total=%d\n", sampleSeconds, total)
					return printTable([]string{"TYPE", "COUNT"}, rows)
				case ev, ok := <-ch:
					if !ok {
						return nil
					}
					counts[ev.GetType()]++
					total++
					if total >= maxEvents {
						cancel()
					}
				}
			}
		},
	}
	cmd.Flags().IntVar(&sampleSeconds, "seconds", 3, "sample duration")
	cmd.Flags().IntVar(&maxEvents, "max", 500, "max events to count")
	return cmd
}
