package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	discoveryv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/discovery/v1"
	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
)

func newModulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "modules",
		Short:   "Inspect registered modules on muxcored",
		GroupID: groupOverview,
	}
	cmd.AddCommand(newModulesListCmd())
	cmd.AddCommand(newModulesStatusCmd())
	return cmd
}

func newModulesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List modules known to muxcored (Discovery.ListAll)",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer func() { _ = c.Close() }()

			ctx, cancel := connect.Context(opts)
			defer cancel()

			resp, err := c.Discovery.Raw().ListAll(ctx, &discoveryv1.ListAllRequest{})
			if err != nil {
				return fmt.Errorf("modules list: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tSTATE\tHEALTH\tNODE\tVERSION\tCAPABILITIES")
			for _, e := range resp.GetEntries() {
				info := e.GetInfo()
				health := e.GetHealthError()
				if health == "" {
					health = "ok"
				}
				caps := ""
				if info != nil && len(info.GetCapabilities()) > 0 {
					caps = fmt.Sprintf("%v", info.GetCapabilities())
				}
				id, ver := "", ""
				if info != nil {
					id = info.GetId()
					ver = info.GetVersion()
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					id, e.GetState(), health, e.GetNodeId(), ver, caps)
			}
			return w.Flush()
		},
	}
}

func newModulesStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [module-id]",
		Short: "Show status for one module (or all if omitted)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dialOpts()
			c, err := connect.Dial(opts)
			if err != nil {
				return err
			}
			defer func() { _ = c.Close() }()

			ctx, cancel := connect.Context(opts)
			defer cancel()

			if len(args) == 1 {
				info, resolveErr := c.Discovery.Resolve(ctx, args[0])
				if resolveErr != nil {
					return fmt.Errorf("modules status: %w", resolveErr)
				}
				if info == nil {
					return fmt.Errorf("module %q not found", args[0])
				}
				printModuleInfo(info)
				return nil
			}

			resp, err := c.Discovery.Raw().ListAll(ctx, &discoveryv1.ListAllRequest{})
			if err != nil {
				return fmt.Errorf("modules status: %w", err)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tSTATE\tHEALTH\tNODE")
			for _, e := range resp.GetEntries() {
				info := e.GetInfo()
				id := ""
				if info != nil {
					id = info.GetId()
				}
				health := e.GetHealthError()
				if health == "" {
					health = "ok"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, e.GetState(), health, e.GetNodeId())
			}
			return w.Flush()
		},
	}
}

func printModuleInfo(info *discoveryv1.ModuleInfoProto) {
	fmt.Printf("id:           %s\n", info.GetId())
	fmt.Printf("name:         %s\n", info.GetName())
	fmt.Printf("version:      %s\n", info.GetVersion())
	fmt.Printf("state:        %s\n", info.GetState())
	health := info.GetHealthError()
	if health == "" {
		health = "ok"
	}
	fmt.Printf("health:       %s\n", health)
	fmt.Printf("http_addr:    %s\n", info.GetHttpAddr())
	fmt.Printf("roles:        %v\n", info.GetRoles())
	fmt.Printf("capabilities: %v\n", info.GetCapabilities())
	fmt.Printf("depends_on:   %v\n", info.GetDependsOn())
}
