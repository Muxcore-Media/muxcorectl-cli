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
		Use:   "events",
		Short: "Event bus commands",
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
			defer c.Close()

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
					fmt.Printf("%s type=%s source=%s bytes=%d\n", ts, ev.GetType(), ev.GetSource(), len(ev.GetPayload()))
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
	return cmd
}
