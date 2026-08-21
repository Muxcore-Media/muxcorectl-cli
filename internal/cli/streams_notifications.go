package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func withPlaybackMonitorHTTP(fn func(base string) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		base, err := resolveModuleHTTP(ctx, c, capPlaybackMonitor, "MUXCORE_PLAYBACK_MONITOR_URL")
		if err != nil {
			return err
		}
		return fn(base)
	})
}

func monitorHTTP(method, path string, body []byte) ([]byte, error) {
	var out []byte
	err := withPlaybackMonitorHTTP(func(base string) error {
		var err error
		out, err = httpDo(method, base+path, body, nil)
		return err
	})
	return out, err
}

func newStreamsNotificationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notifications",
		Short: "Playback notification rules and destinations (admin-ui /streams/notifications parity)",
	}
	cmd.AddCommand(newStreamsNotificationsRulesCmd())
	cmd.AddCommand(newStreamsNotificationsDestinationsCmd())
	return cmd
}

func newStreamsNotificationsRulesCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List notification rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			eventType, _ := cmd.Flags().GetString("event-type")
			path := "/notification/rules"
			if eventType != "" {
				path += "?event_type=" + url.QueryEscape(eventType)
			}
			raw, err := monitorHTTP("GET", path, nil)
			if err != nil {
				return fmt.Errorf("streams notifications rules list: %w", err)
			}
			var resp struct {
				Rules []map[string]any `json:"rules"`
			}
			if err := json.Unmarshal(raw, &resp); err != nil {
				return fmt.Errorf("streams notifications rules decode: %w", err)
			}
			if flagJSON {
				return printJSON(resp.Rules)
			}
			rows := make([][]string, 0, len(resp.Rules))
			for _, r := range resp.Rules {
				rows = append(rows, []string{
					fmt.Sprint(r["id"]),
					fmt.Sprint(r["name"]),
					fmt.Sprint(r["event_type"]),
					fmt.Sprint(r["enabled"]),
				})
			}
			return printTable([]string{"ID", "NAME", "EVENT", "ENABLED"}, rows)
		},
	}
	list.Flags().String("event-type", "", "filter by event type")

	create := &cobra.Command{
		Use:   "create",
		Short: "Create a notification rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			eventType, _ := cmd.Flags().GetString("event-type")
			title, _ := cmd.Flags().GetString("title-template")
			message, _ := cmd.Flags().GetString("message-template")
			severity, _ := cmd.Flags().GetString("severity")
			enabled, _ := cmd.Flags().GetBool("enabled")
			transcodeOnly, _ := cmd.Flags().GetBool("transcode-only")
			destIDs, _ := cmd.Flags().GetStringSlice("destination-id")
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			body, _ := json.Marshal(map[string]any{
				"name":             name,
				"event_type":       eventType,
				"title_template":   title,
				"message_template": message,
				"severity":         severity,
				"enabled":          enabled,
				"destination_ids":  destIDs,
				"filters":          map[string]any{"transcode_only": transcodeOnly},
			})
			if _, err := monitorHTTP("POST", "/notification/rules", body); err != nil {
				return fmt.Errorf("streams notifications rules create: %w", err)
			}
			if !flagQuiet {
				fmt.Println("created")
			}
			return nil
		},
	}
	create.Flags().String("name", "", "rule name")
	create.Flags().String("event-type", "session.start", "event type")
	create.Flags().String("title-template", "", "notification title template")
	create.Flags().String("message-template", "", "notification message template")
	create.Flags().String("severity", "info", "severity")
	create.Flags().Bool("enabled", true, "enable rule")
	create.Flags().Bool("transcode-only", false, "only transcode sessions")
	create.Flags().StringSlice("destination-id", nil, "destination ids")

	deleteCmd := &cobra.Command{
		Use:   "delete <rule-id>",
		Short: "Delete a notification rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete notification rule " + args[0] + "?"); err != nil {
				return err
			}
			if _, err := monitorHTTP("DELETE", "/notification/rules/"+url.PathEscape(args[0]), nil); err != nil {
				return fmt.Errorf("streams notifications rules delete: %w", err)
			}
			if !flagQuiet {
				fmt.Println("deleted")
			}
			return nil
		},
	}

	cmd := &cobra.Command{Use: "rules", Short: "Notification rule commands"}
	cmd.AddCommand(list, create, deleteCmd)
	return cmd
}

func newStreamsNotificationsDestinationsCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List notification destinations",
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := monitorHTTP("GET", "/notification/destinations", nil)
			if err != nil {
				return fmt.Errorf("streams notifications destinations list: %w", err)
			}
			var resp struct {
				Destinations []map[string]any `json:"destinations"`
			}
			if err := json.Unmarshal(raw, &resp); err != nil {
				return fmt.Errorf("streams notifications destinations decode: %w", err)
			}
			if flagJSON {
				return printJSON(resp.Destinations)
			}
			rows := make([][]string, 0, len(resp.Destinations))
			for _, d := range resp.Destinations {
				rows = append(rows, []string{
					fmt.Sprint(d["id"]),
					fmt.Sprint(d["name"]),
					fmt.Sprint(d["type"]),
					fmt.Sprint(d["enabled"]),
				})
			}
			return printTable([]string{"ID", "NAME", "TYPE", "ENABLED"}, rows)
		},
	}

	create := &cobra.Command{
		Use:   "create",
		Short: "Create a notification destination",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			destType, _ := cmd.Flags().GetString("type")
			webhookURL, _ := cmd.Flags().GetString("webhook-url")
			appriseURLs, _ := cmd.Flags().GetString("apprise-urls")
			enabled, _ := cmd.Flags().GetBool("enabled")
			events, _ := cmd.Flags().GetStringSlice("event")
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			config := map[string]string{}
			switch destType {
			case "apprise":
				config["urls"] = appriseURLs
			default:
				config["webhook_url"] = webhookURL
			}
			body, _ := json.Marshal(map[string]any{
				"name":    name,
				"type":    destType,
				"enabled": enabled,
				"config":  config,
				"events":  events,
			})
			if _, err := monitorHTTP("POST", "/notification/destinations", body); err != nil {
				return fmt.Errorf("streams notifications destinations create: %w", err)
			}
			if !flagQuiet {
				fmt.Println("created")
			}
			return nil
		},
	}
	create.Flags().String("name", "", "destination name")
	create.Flags().String("type", "webhook", "destination type (webhook or apprise)")
	create.Flags().String("webhook-url", "", "webhook URL")
	create.Flags().String("apprise-urls", "", "Apprise URLs (comma-separated)")
	create.Flags().Bool("enabled", true, "enable destination")
	create.Flags().StringSlice("event", nil, "subscribed events")

	deleteCmd := &cobra.Command{
		Use:   "delete <destination-id>",
		Short: "Delete a notification destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete destination " + args[0] + "?"); err != nil {
				return err
			}
			if _, err := monitorHTTP("DELETE", "/notification/destinations/"+url.PathEscape(args[0]), nil); err != nil {
				return fmt.Errorf("streams notifications destinations delete: %w", err)
			}
			if !flagQuiet {
				fmt.Println("deleted")
			}
			return nil
		},
	}

	test := &cobra.Command{
		Use:   "test <destination-id>",
		Short: "Send a test notification to a destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := monitorHTTP("POST", "/notification/destinations/"+url.PathEscape(args[0])+"/test", nil); err != nil {
				return fmt.Errorf("streams notifications destinations test: %w", err)
			}
			if !flagQuiet {
				fmt.Println("test sent")
			}
			return nil
		},
	}

	cmd := &cobra.Command{Use: "destinations", Short: "Notification destination commands"}
	cmd.AddCommand(list, create, deleteCmd, test)
	return cmd
}
