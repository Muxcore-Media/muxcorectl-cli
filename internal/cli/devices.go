package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDevicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "devices",
		Short:   "Admin-ui browser sessions (limited CLI parity)",
		GroupID: groupAccess,
		Long: `The admin-ui /devices page lists browser cookie sessions stored in the admin-ui process memory.

muxcorectl cannot list or revoke those sessions remotely — there is no shared API. Use the web UI for device/session management, or manage API access with:

  muxcorectl users tokens list <user-id>
  muxcorectl keys list
  muxcorectl audit query`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagJSON {
				return printJSON(map[string]any{
					"supported": false,
					"reason":    "admin-ui in-memory session store; no remote API",
					"alternatives": []string{
						"users tokens list",
						"keys list",
						"audit query",
					},
				})
			}
			fmt.Println("Browser device sessions are only visible in admin-ui (/devices).")
			fmt.Println("They are stored in the admin-ui process, not muxcored.")
			fmt.Println()
			fmt.Println("Alternatives from the CLI:")
			fmt.Println("  muxcorectl users tokens list <user-id>   # user API tokens")
			fmt.Println("  muxcorectl keys list                     # global API key catalog")
			fmt.Println("  muxcorectl audit query                   # login/logout audit trail")
			return nil
		},
	}
}
