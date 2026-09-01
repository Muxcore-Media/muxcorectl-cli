package cli

import (
	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate shell completion scripts",
		Long: `Install shell completion for muxcorectl's command tree.

Examples:
  muxcorectl completion bash > /etc/bash_completion.d/muxcorectl
  muxcorectl completion zsh > "${fpath[1]}/_muxcorectl"
  muxcorectl completion fish > ~/.config/fish/completions/muxcorectl.fish`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return root.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return root.GenFishCompletion(cmd.OutOrStdout(), true)
			default:
				return cmd.Help()
			}
		},
	}
}
