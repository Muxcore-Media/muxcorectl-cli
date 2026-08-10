package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print muxcorectl version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("muxcorectl %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
		},
	}
}
