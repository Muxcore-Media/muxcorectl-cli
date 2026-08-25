package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	auditv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/audit/v1"
	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
)

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "audit",
		Short:   "Audit log commands",
		GroupID: groupMonitoring,
	}
	cmd.AddCommand(newAuditQueryCmd())
	cmd.AddCommand(newAuditExportCmd())
	return cmd
}

func newAuditQueryCmd() *cobra.Command {
	var (
		actor, action, resource, traceID string
		fromTime, toTime                 string
		maxResults                       int32
	)
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer func() { _ = c.Close() }()

			ctx, cancel := connect.Context(opts)
			defer cancel()

			entries, err := c.Audit.Query(ctx, actor, action, resource, traceID, fromTime, toTime, maxResults)
			if err != nil {
				return fmt.Errorf("audit query: %w", err)
			}
			if flagJSON {
				return printJSON(entries)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tTIME\tACTOR\tACTION\tRESOURCE\tRESOURCE_ID")
			for _, e := range entries {
				ts := time.Unix(e.GetTimestamp(), 0).UTC().Format(time.RFC3339)
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					e.GetId(), ts, e.GetActor(), e.GetAction(), e.GetResource(), e.GetResourceId())
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "filter by actor")
	cmd.Flags().StringVar(&action, "action", "", "filter by action")
	cmd.Flags().StringVar(&resource, "resource", "", "filter by resource")
	cmd.Flags().StringVar(&traceID, "trace-id", "", "filter by trace id")
	cmd.Flags().StringVar(&fromTime, "from", "", "RFC3339 lower bound")
	cmd.Flags().StringVar(&toTime, "to", "", "RFC3339 upper bound")
	cmd.Flags().Int32Var(&maxResults, "max", 50, "max results (0 = server default)")
	return cmd
}

func newAuditExportCmd() *cobra.Command {
	var (
		format, actor, action, resource string
		fromTime, toTime                string
		outFile                         string
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Stream audit log to stdout or a file (json or csv)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if format == "" {
				format = "json"
			}
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer func() { _ = c.Close() }()
			ctx, cancel := connect.Context(opts)
			defer cancel()

			stream, err := c.Audit.Raw().Export(ctx, &auditv1.AuditExportRequest{
				Format: format, Actor: actor, Action: action,
				Resource: resource, FromTime: fromTime, ToTime: toTime,
			})
			if err != nil {
				return fmt.Errorf("audit export: %w", err)
			}
			var w io.Writer = os.Stdout
			if outFile != "" {
				f, err := os.Create(outFile) //nolint:gosec // operator-selected export path
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()
				w = f
			}
			for {
				chunk, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return fmt.Errorf("audit export recv: %w", err)
				}
				if _, err := w.Write(chunk.GetData()); err != nil {
					return err
				}
				if chunk.GetComplete() {
					break
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "json", "json or csv")
	cmd.Flags().StringVar(&actor, "actor", "", "filter by actor")
	cmd.Flags().StringVar(&action, "action", "", "filter by action")
	cmd.Flags().StringVar(&resource, "resource", "", "filter by resource")
	cmd.Flags().StringVar(&fromTime, "from", "", "RFC3339 lower bound")
	cmd.Flags().StringVar(&toTime, "to", "", "RFC3339 upper bound")
	cmd.Flags().StringVar(&outFile, "output", "", "write to file instead of stdout")
	return cmd
}
