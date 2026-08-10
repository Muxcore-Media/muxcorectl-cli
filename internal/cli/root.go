// Package cli implements the muxcorectl cobra command tree.
package cli

import (
	"time"

	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"github.com/spf13/cobra"
)

// Version is overridden at link time via -X github.com/Muxcore-Media/muxcorectl-cli/internal/cli.Version.
var Version = "dev"

// root persistent flags shared by all RPCs.
var (
	flagAddr     string
	flagInsecure bool
	flagToken    string
	flagTimeout  time.Duration
)

// NewRoot builds the muxcorectl command tree.
func NewRoot() *cobra.Command {
	env := connect.FromEnv()

	root := &cobra.Command{
		Use:           "muxcorectl",
		Short:         "Operator CLI for a running MuxCore (muxcored) node",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagAddr, "addr", env.Addr, "muxcored gRPC address (MUXCORE_GRPC_ADDR)")
	root.PersistentFlags().BoolVar(&flagInsecure, "insecure", env.Insecure, "disable TLS (MUXCORE_INSECURE_DISABLE_TLS)")
	root.PersistentFlags().StringVar(&flagToken, "token", env.Token, "bearer token (MUXCORE_TOKEN)")
	root.PersistentFlags().DurationVar(&flagTimeout, "timeout", env.Timeout, "per-RPC timeout")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newModulesCmd())
	root.AddCommand(newClusterCmd())
	root.AddCommand(newEventsCmd())
	root.AddCommand(newStorageCmd())
	root.AddCommand(newAuditCmd())
	root.AddCommand(newSpoolCmd())
	root.AddCommand(newSchedulesCmd())

	return root
}

func dialOpts() connect.Options {
	return connect.Options{
		Addr:     flagAddr,
		Insecure: flagInsecure,
		Token:    flagToken,
		Timeout:  flagTimeout,
	}
}
