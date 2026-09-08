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
	flagAddr      string
	flagInsecure  bool
	flagToken     string
	flagTokenFile string
	flagTimeout   time.Duration
)

// NewRoot builds the muxcorectl command tree.
func NewRoot() *cobra.Command {
	env := connect.FromEnv()

	root := &cobra.Command{
		Use:   "muxcorectl",
		Short: "Admin and operator CLI for a running MuxCore node",
		Long: `muxcorectl talks to muxcored over gRPC and module mesh APIs.

It mirrors the admin-ui web dashboard: modules, marketplace, settings, users,
media libraries, invites, audit, storage, and more.

Quick start (local laptop stack):
  export MUXCORE_INSECURE_DISABLE_TLS=true
  export MUXCORE_TOKEN="$(cat _mvp/run/admin.token)"
  muxcorectl modules list
  muxcorectl settings list
  muxcorectl media libraries

Tips:
  --json          machine-readable output for scripts
  --yes           skip confirmation prompts
  --help on any command for examples`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := resolveTokenFile(); err != nil {
				return err
			}
			if flagToken == "" {
				flagToken = resolveOperatorToken()
			}
			return nil
		},
	}

	root.AddGroup(
		&cobra.Group{ID: groupOverview, Title: "Overview:"},
		&cobra.Group{ID: groupMonitoring, Title: "Monitoring:"},
		&cobra.Group{ID: groupLibrary, Title: "Library:"},
		&cobra.Group{ID: groupAutomation, Title: "Automation:"},
		&cobra.Group{ID: groupPlayback, Title: "Playback:"},
		&cobra.Group{ID: groupAccess, Title: "Access:"},
		&cobra.Group{ID: groupSystem, Title: "System:"},
	)

	root.PersistentFlags().StringVar(&flagAddr, "addr", env.Addr, "muxcored gRPC address (MUXCORE_GRPC_ADDR)")
	root.PersistentFlags().BoolVar(&flagInsecure, "insecure", env.Insecure, "disable TLS (MUXCORE_INSECURE_DISABLE_TLS)")
	root.PersistentFlags().StringVar(&flagToken, "token", "", "bearer token (prefer MUXCORE_TOKEN env or --token-file)")
	root.PersistentFlags().StringVar(&flagTokenFile, "token-file", "", "read bearer token from file (MUXCORE_TOKEN_FILE)")
	root.PersistentFlags().DurationVar(&flagTimeout, "timeout", env.Timeout, "per-RPC timeout")
	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "emit JSON instead of tables")
	root.PersistentFlags().BoolVar(&flagQuiet, "quiet", false, "suppress non-essential output")
	root.PersistentFlags().BoolVar(&flagYes, "yes", false, "assume yes to confirmation prompts")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newCompletionCmd())
	root.AddCommand(newModulesCmd())
	root.AddCommand(newClusterCmd())
	root.AddCommand(newHealthCmd())
	root.AddCommand(newLifecycleCmd())
	root.AddCommand(newMarketplaceCmd())
	root.AddCommand(newEventsCmd())
	root.AddCommand(newLogsCmd())
	root.AddCommand(newAuditCmd())
	root.AddCommand(newStorageCmd())
	root.AddCommand(newSpoolCmd())
	root.AddCommand(newSchedulesCmd())
	root.AddCommand(newSettingsCmd())
	root.AddCommand(newUsersCmd())
	root.AddCommand(newDevicesCmd())
	root.AddCommand(newInvitesCmd())
	root.AddCommand(newMediaCmd())
	root.AddCommand(newMetadataCmd())
	root.AddCommand(newFormatsCmd())
	root.AddCommand(newRootsCmd())
	root.AddCommand(newRenameCmd())
	root.AddCommand(newRequestCmd())
	root.AddCommand(newQueueCmd())
	root.AddCommand(newAutomationCmd())
	root.AddCommand(newCalendarCmd())
	root.AddCommand(newSubtitlesCmd())
	root.AddCommand(newBackupsCmd())
	root.AddCommand(newJellyfinCmd())
	root.AddCommand(newMaintainerCmd())
	root.AddCommand(newListSyncCmd())
	root.AddCommand(newImportCmd())
	root.AddCommand(newStreamsCmd())
	root.AddCommand(newTranscodeCmd())
	root.AddCommand(newPlaybackCmd())
	root.AddCommand(newLiveTVCmd())
	root.AddCommand(newTasksCmd())
	root.AddCommand(newKeysCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newPluginsCmd())
	root.AddCommand(newBrandingCmd())
	root.AddCommand(newNetworkingCmd())
	root.AddCommand(newActivityCmd())
	root.AddCommand(newMigrateCmd())
	root.AddCommand(newAICmd())

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
