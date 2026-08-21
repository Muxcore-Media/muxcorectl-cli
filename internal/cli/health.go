package cli

import (
	"context"
	"fmt"

	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "health",
		Short:   "Dashboard health summary (admin-ui / parity)",
		GroupID: groupOverview,
	}
	cmd.AddCommand(newHealthStatusCmd())
	cmd.AddCommand(newHealthMonitorCmd())
	return cmd
}

func newHealthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show cluster and module health overview",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCore(func(ctx context.Context, c *client.Client) error {
				members, leader, err := c.Discovery.Members(ctx)
				if err != nil {
					return fmt.Errorf("health status: %w", err)
				}
				libraries, _ := listMediaLibraries(ctx, c)
				type summary struct {
					Leader         string `json:"leader"`
					Nodes          int    `json:"nodes"`
					Libraries      int    `json:"libraries"`
					ModulesOnLeader int   `json:"modules_on_leader"`
				}
				s := summary{
					Leader:    leader,
					Nodes:     len(members),
					Libraries: len(libraries),
				}
				for _, m := range members {
					if m.GetId() == leader {
						s.ModulesOnLeader = len(m.GetModules())
						break
					}
				}
				if flagJSON {
					return printJSON(s)
				}
				fmt.Printf("leader:            %s\n", s.Leader)
				fmt.Printf("cluster nodes:     %d\n", s.Nodes)
				fmt.Printf("media libraries:   %d\n", s.Libraries)
				fmt.Printf("modules on leader: %d\n", s.ModulesOnLeader)
				return nil
			})
		},
	}
}
