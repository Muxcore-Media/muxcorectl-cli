package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	authv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/auth/v1"
	"github.com/spf13/cobra"
)

func newKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Short:   "API keys across all users (admin-ui /keys parity)",
		GroupID: groupAccess,
	}
	cmd.AddCommand(newKeysListCmd())
	cmd.AddCommand(newKeysRevokeCmd())
	return cmd
}

func newKeysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List API tokens for all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				users, err := auth.ListUsers(ctx, &authv1.ListUsersRequest{})
				if err != nil {
					return fmt.Errorf("keys list users: %w", err)
				}
				type row struct {
					UserID   string `json:"user_id"`
					Username string `json:"username"`
					TokenID  string `json:"token_id"`
					Name     string `json:"name"`
					Created  string `json:"created_at"`
				}
				var out []row
				for _, u := range users.GetUsers() {
					tok, err := auth.ListAPITokens(ctx, &authv1.ListAPITokensRequest{UserId: u.GetId()})
					if err != nil {
						continue
					}
					for _, t := range tok.GetTokens() {
						out = append(out, row{
							UserID: u.GetId(), Username: u.GetUsername(),
							TokenID: t.GetId(), Name: t.GetName(), Created: t.GetCreatedAt(),
						})
					}
				}
				if flagJSON {
					return printJSON(out)
				}
				rows := make([][]string, 0, len(out))
				for _, r := range out {
					rows = append(rows, []string{r.Username, r.TokenID, r.Name, r.Created})
				}
				return printTable([]string{"USER", "TOKEN_ID", "NAME", "CREATED"}, rows)
			})
		},
	}
}

func newKeysRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <token-id>",
		Short: "Revoke an API token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Revoke token " + args[0] + "?"); err != nil {
				return err
			}
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				_, err := auth.DeleteAPIToken(ctx, &authv1.DeleteAPITokenRequest{TokenId: args[0]})
				if err != nil {
					return fmt.Errorf("keys revoke: %w", err)
				}
				if !flagQuiet {
					fmt.Println("revoked")
				}
				return nil
			})
		},
	}
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "config",
		Short:   "Show MUXCORE_/ADMIN_UI_ environment (admin-ui /config parity)",
		GroupID: groupSystem,
		RunE: func(cmd *cobra.Command, args []string) error {
			type entry struct {
				Key       string `json:"key"`
				Value     string `json:"value,omitempty"`
				Redacted  bool   `json:"redacted,omitempty"`
			}
			var out []entry
			for _, env := range os.Environ() {
				key, val, ok := strings.Cut(env, "=")
				if !ok {
					continue
				}
				if !strings.HasPrefix(key, "ADMIN_UI_") && !strings.HasPrefix(key, "MUXCORE_") {
					continue
				}
				upper := strings.ToUpper(key)
				redacted := strings.Contains(upper, "KEY") ||
					strings.Contains(upper, "SECRET") ||
					strings.Contains(upper, "TOKEN") ||
					strings.Contains(upper, "PASSWORD") ||
					strings.Contains(upper, "CERT") ||
					val == ""
				if redacted {
					out = append(out, entry{Key: key, Redacted: true})
				} else {
					out = append(out, entry{Key: key, Value: val})
				}
			}
			if flagJSON {
				return printJSON(out)
			}
			rows := make([][]string, 0, len(out))
			for _, e := range out {
				val := e.Value
				if e.Redacted {
					val = "(redacted)"
				}
				rows = append(rows, []string{e.Key, val})
			}
			return printTable([]string{"KEY", "VALUE"}, rows)
		},
	}
}

func newTasksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tasks",
		Short:   "Scheduled tasks (admin-ui /tasks parity; alias of schedules)",
		GroupID: groupAutomation,
	}
	cmd.PersistentFlags().String("scheduler-url", "", "scheduler-cron base URL (default: discover HttpAddr)")
	cmd.AddCommand(newSchedulesListCmd())
	cmd.AddCommand(newSchedulesStatusCmd())
	cmd.AddCommand(newSchedulesCancelCmd())
	return cmd
}
