package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	discoveryv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/discovery/v1"
	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
)

func newSchedulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedules",
		Short:   "Manage scheduler-cron tasks via its HTTP API",
		GroupID: groupAutomation,
	}
	cmd.PersistentFlags().String("scheduler-url", "", "scheduler-cron base URL (default: discover HttpAddr)")
	cmd.AddCommand(newSchedulesListCmd())
	cmd.AddCommand(newSchedulesStatusCmd())
	cmd.AddCommand(newSchedulesCancelCmd())
	cmd.AddCommand(newSchedulesAddCmd())
	return cmd
}

func newSchedulesListCmd() *cobra.Command {
	var nameFilter string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List scheduled tasks (GET /list)",
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := resolveSchedulerURL(cmd)
			if err != nil {
				return err
			}
			u := base + "/list"
			if nameFilter != "" {
				u += "?name=" + nameFilter
			}
			body, err := schedulerDo(http.MethodGet, u, nil)
			if err != nil {
				return err
			}
			var tasks []map[string]any
			if err := json.Unmarshal(body, &tasks); err != nil {
				return fmt.Errorf("decode list: %w", err)
			}
			if flagJSON {
				return printJSON(tasks)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tCRON\tSTATUS\tONCE\tLAST_FIRED")
			for _, t := range tasks {
				_, _ = fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
					t["id"], t["name"], t["cron_expr"], t["status"], t["once"], t["last_fired_at"])
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&nameFilter, "name", "", "substring filter on task name")
	return cmd
}

func newSchedulesStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <task-id>",
		Short: "Show one task (GET /status/{id})",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := resolveSchedulerURL(cmd)
			if err != nil {
				return err
			}
			body, err := schedulerDo(http.MethodGet, base+"/status/"+args[0], nil)
			if err != nil {
				return err
			}
			if flagJSON {
				var out any
				if err := json.Unmarshal(body, &out); err != nil {
					return fmt.Errorf("decode status: %w", err)
				}
				return printJSON(out)
			}
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, body, "", "  "); err != nil {
				_, _ = fmt.Println(string(body))
				return err
			}
			fmt.Println(pretty.String())
			return nil
		},
	}
}

func newSchedulesCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Cancel a task (DELETE /cancel/{id})",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := resolveSchedulerURL(cmd)
			if err != nil {
				return err
			}
			body, err := schedulerDo(http.MethodDelete, base+"/cancel/"+args[0], nil)
			if err != nil {
				return err
			}
			if flagJSON {
				var out any
				if err := json.Unmarshal(body, &out); err != nil {
					return printJSON(map[string]string{"status": strings.TrimSpace(string(body))})
				}
				return printJSON(out)
			}
			fmt.Println(strings.TrimSpace(string(body)))
			return nil
		},
	}
}

func newSchedulesAddCmd() *cobra.Command {
	var (
		name     string
		cronExpr string
		once     bool
		timeout  string
		webhook  string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Register a task (POST /schedule)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || cronExpr == "" {
				return fmt.Errorf("--name and --cron are required")
			}
			base, err := resolveSchedulerURL(cmd)
			if err != nil {
				return err
			}
			payload := map[string]any{
				"name":      name,
				"cron_expr": cronExpr,
				"once":      once,
			}
			if timeout != "" {
				payload["timeout"] = timeout
			}
			if webhook != "" {
				payload["meta"] = map[string]any{"webhook_url": webhook}
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			body, err := schedulerDo(http.MethodPost, base+"/schedule", raw)
			if err != nil {
				return err
			}
			if flagJSON {
				var out any
				if err := json.Unmarshal(body, &out); err != nil {
					return printJSON(map[string]string{"result": strings.TrimSpace(string(body))})
				}
				return printJSON(out)
			}
			fmt.Println(strings.TrimSpace(string(body)))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "task name")
	cmd.Flags().StringVar(&cronExpr, "cron", "", "cron expression (e.g. '@hourly' or '0 * * * *')")
	cmd.Flags().BoolVar(&once, "once", false, "disarm after first fire")
	cmd.Flags().StringVar(&timeout, "timeout", "", "webhook timeout duration (e.g. 10s)")
	cmd.Flags().StringVar(&webhook, "webhook", "", "optional webhook_url stored in meta")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("cron")
	return cmd
}

func resolveSchedulerURL(cmd *cobra.Command) (string, error) {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.PersistentFlags().Lookup("scheduler-url"); f != nil && strings.TrimSpace(f.Value.String()) != "" {
			return strings.TrimRight(strings.TrimSpace(f.Value.String()), "/"), nil
		}
		if f := c.Flags().Lookup("scheduler-url"); f != nil && strings.TrimSpace(f.Value.String()) != "" {
			return strings.TrimRight(strings.TrimSpace(f.Value.String()), "/"), nil
		}
	}
	if u := strings.TrimSpace(os.Getenv("SCHEDULER_URL")); u != "" {
		return strings.TrimRight(u, "/"), nil
	}
	opts := dialOpts()
	c, err := connect.Dial(opts)
	if err != nil {
		return "", err
	}
	defer func() { _ = c.Close() }()

	ctx, cancel := connect.Context(opts)
	defer cancel()

	resp, err := c.Discovery.Raw().ListAll(ctx, &discoveryv1.ListAllRequest{})
	if err != nil {
		return "", fmt.Errorf("discover scheduler: %w", err)
	}
	var info *discoveryv1.ModuleInfoProto
	for _, e := range resp.GetEntries() {
		mi := e.GetInfo()
		if mi == nil {
			continue
		}
		if mi.GetId() == "scheduler-cron" {
			info = mi
			break
		}
	}
	if info == nil {
		for _, e := range resp.GetEntries() {
			mi := e.GetInfo()
			if mi == nil {
				continue
			}
			for _, cap := range mi.GetCapabilities() {
				if cap == "scheduler" || cap == "scheduler.cron" {
					info = mi
					break
				}
			}
			if info != nil {
				break
			}
		}
	}
	if info == nil {
		return "", fmt.Errorf("scheduler-cron not registered; pass --scheduler-url or set SCHEDULER_URL")
	}
	addr := normalizeHTTPBase(info.GetHttpAddr())
	if addr == "" {
		return "", fmt.Errorf("scheduler module %q has empty HttpAddr; pass --scheduler-url", info.GetId())
	}
	return addr, nil
}

func schedulerDo(method, url string, body []byte) ([]byte, error) {
	opts := dialOpts()
	ctx, cancel := connect.Context(opts)
	defer cancel()

	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth := bearerAuthHeader(); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedules: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("schedules: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	return out, nil
}
