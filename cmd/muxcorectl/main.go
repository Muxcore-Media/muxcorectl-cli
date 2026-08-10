package main

import (
	"fmt"
	"os"

	"github.com/Muxcore-Media/muxcorectl-cli/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
