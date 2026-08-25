package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newInvitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "invites",
		Short:   "Manage signup invites (admin-ui /invites parity)",
		GroupID: groupAccess,
	}
	cmd.AddCommand(newInvitesListCmd())
	cmd.AddCommand(newInvitesCreateCmd())
	cmd.AddCommand(newInvitesRevokeCmd())
	return cmd
}

func newInvitesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List invite links",
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := authHTTPBase()
			if err != nil {
				return err
			}
			body, err := authHTTPDo(http.MethodGet, base+"/api/invites", nil, nil)
			if err != nil {
				return err
			}
			var payload struct {
				Invites []map[string]any `json:"invites"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				return fmt.Errorf("invites list decode: %w", err)
			}
			if flagJSON {
				return printJSON(payload.Invites)
			}
			rows := make([][]string, 0, len(payload.Invites))
			for _, inv := range payload.Invites {
				rows = append(rows, []string{
					fmt.Sprint(inv["id"]),
					fmt.Sprint(inv["prefix"]),
					fmt.Sprint(inv["role"]),
					fmt.Sprint(inv["use_count"]),
					fmt.Sprint(inv["max_uses"]),
					fmt.Sprint(inv["expires_at"]),
				})
			}
			return printTable([]string{"ID", "PREFIX", "ROLE", "USES", "MAX", "EXPIRES"}, rows)
		},
	}
}

func newInvitesCreateCmd() *cobra.Command {
	var role string
	var maxUses, ttlHours int
	var tenantID string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new invite link",
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := authHTTPBase()
			if err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]any{
				"createdBy": "muxcorectl",
				"role":      role,
				"maxUses":   maxUses,
				"ttlHours":  ttlHours,
				"tenantId":  tenantID,
			})
			headers := map[string]string{}
			if tenantID != "" {
				headers["X-Tenant-ID"] = tenantID
				headers["X-Auth-Claims-Tenant"] = tenantID
			}
			body, err := authHTTPDo(http.MethodPost, base+"/api/invites", payload, headers)
			if err != nil {
				return err
			}
			var inv struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal(body, &inv); err != nil {
				return fmt.Errorf("invites create decode: %w", err)
			}
			link := base + "/invite?token=" + url.QueryEscape(inv.Token)
			if flagJSON {
				return printJSON(map[string]string{"link": link, "token": inv.Token})
			}
			fmt.Println(link)
			return nil
		},
	}
	cmd.Flags().StringVar(&role, "role", "user", "role granted by invite")
	cmd.Flags().IntVar(&maxUses, "max-uses", 1, "maximum uses (0 = unlimited)")
	cmd.Flags().IntVar(&ttlHours, "ttl-hours", 168, "hours until expiry")
	cmd.Flags().StringVar(&tenantID, "tenant", "", "tenant id for multi-tenant installs")
	return cmd
}

func newInvitesRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <invite-id>",
		Short: "Revoke an invite",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := authHTTPBase()
			if err != nil {
				return err
			}
			_, err = authHTTPDo(http.MethodDelete, base+"/api/invites/"+url.PathEscape(args[0]), nil, nil)
			if err != nil {
				return err
			}
			if !flagQuiet {
				fmt.Println("revoked")
			}
			return nil
		},
	}
}

func authHTTPBase() (string, error) {
	var base string
	err := withCore(func(ctx context.Context, c *client.Client) error {
		var err error
		base, err = resolveModuleHTTP(ctx, c, "auth", "MUXCORE_AUTH_URL")
		return err
	})
	return base, err
}

func authHTTPDo(method, rawURL string, body []byte, headers map[string]string) ([]byte, error) {
	return httpDo(method, rawURL, body, headers)
}
