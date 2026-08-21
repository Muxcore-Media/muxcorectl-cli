package cli

import (
	"context"
	"fmt"

	lifecyclev1 "github.com/Muxcore-Media/core/proto/gen/muxcore/lifecycle/v1"
	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func newLifecycleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lifecycle",
		Short:   "Start, stop, and restart modules (admin-ui /modules parity)",
		GroupID: groupOverview,
	}
	cmd.AddCommand(newLifecycleListCmd())
	cmd.AddCommand(newLifecycleSpawnCmd())
	cmd.AddCommand(newLifecycleStopCmd())
	cmd.AddCommand(newLifecycleRestartCmd())
	return cmd
}

func newLifecycleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List modules with lifecycle state and uptime",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := dialOpts()
			conn, err := dialLifecycle(opts)
			if err != nil {
				return err
			}
			defer conn.Close()
			ctx, cancel := connect.Context(opts)
			defer cancel()
			client := lifecyclev1.NewModuleLifecycleServiceClient(conn)
			resp, err := client.ListModules(ctx, &lifecyclev1.ListModulesRequest{})
			if err != nil {
				return fmt.Errorf("lifecycle list: %w", err)
			}
			if flagJSON {
				return printJSON(resp.GetModules())
			}
			rows := make([][]string, 0, len(resp.GetModules()))
			for _, m := range resp.GetModules() {
				rows = append(rows, []string{
					m.GetModuleId(),
					m.GetState(),
					m.GetVersion(),
					fmt.Sprintf("%d", m.GetUptimeSeconds()),
					fmt.Sprintf("%v", m.GetCapabilities()),
				})
			}
			return printTable([]string{"ID", "STATE", "VERSION", "UPTIME_S", "CAPABILITIES"}, rows)
		},
	}
}

func newLifecycleSpawnCmd() *cobra.Command {
	var tag, repo, version, instanceID string
	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a module from a deployed tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tag == "" || repo == "" || version == "" {
				return fmt.Errorf("--tag, --repo, and --version are required")
			}
			return lifecycleAction("", func(ctx context.Context, c lifecyclev1.ModuleLifecycleServiceClient) error {
				resp, err := c.SpawnModule(ctx, &lifecyclev1.SpawnModuleRequest{
					TagName: tag, Repo: repo, Version: version, InstanceId: instanceID,
				})
				if err != nil {
					return err
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("spawn accepted=%t module=%s\n", resp.GetAccepted(), resp.GetModuleId())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "deployed spool tag name")
	cmd.Flags().StringVar(&repo, "repo", "", "module repository URL")
	cmd.Flags().StringVar(&version, "version", "", "module version")
	cmd.Flags().StringVar(&instanceID, "instance-id", "", "optional instance id")
	_ = cmd.MarkFlagRequired("tag")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("version")
	return cmd
}

func newLifecycleStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <module-id>",
		Short: "Stop a running module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Stop module " + args[0] + "?"); err != nil {
				return err
			}
			return lifecycleAction(args[0], func(ctx context.Context, c lifecyclev1.ModuleLifecycleServiceClient) error {
				resp, err := c.StopModule(ctx, &lifecyclev1.StopModuleRequest{ModuleId: args[0]})
				if err != nil {
					return err
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("stopped %s\n", args[0])
				}
				return nil
			})
		},
	}
}

func newLifecycleRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <module-id>",
		Short: "Restart a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return lifecycleAction(args[0], func(ctx context.Context, c lifecyclev1.ModuleLifecycleServiceClient) error {
				resp, err := c.RestartModule(ctx, &lifecyclev1.RestartModuleRequest{ModuleId: args[0]})
				if err != nil {
					return err
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("restarted %s accepted=%t\n", args[0], resp.GetAccepted())
				}
				return nil
			})
		},
	}
}

func lifecycleAction(moduleID string, fn func(context.Context, lifecyclev1.ModuleLifecycleServiceClient) error) error {
	opts := dialOpts()
	conn, err := dialLifecycle(opts)
	if err != nil {
		return err
	}
	defer conn.Close()
	ctx, cancel := connect.Context(opts)
	defer cancel()
	client := lifecyclev1.NewModuleLifecycleServiceClient(conn)
	if err := fn(ctx, client); err != nil {
		return fmt.Errorf("lifecycle %s: %w", moduleID, err)
	}
	return nil
}

func dialLifecycle(opts connect.Options) (*grpc.ClientConn, error) {
	var dialOpts []grpc.DialOption
	if opts.Insecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		return nil, fmt.Errorf("TLS dial not configured; use --insecure for local stacks")
	}
	if opts.Token != "" {
		token := opts.Token
		dialOpts = append(dialOpts,
			grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOpts ...grpc.CallOption) error {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
				return invoker(ctx, method, req, reply, cc, callOpts...)
			}),
		)
	}
	return grpc.NewClient(opts.Addr, dialOpts...)
}
