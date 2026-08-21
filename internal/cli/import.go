package cli

import (
	"context"
	"fmt"

	scannerv1 "github.com/Muxcore-Media/media-scanner/proto/scannerv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capMediaScanner = "media.scanner"

func withScannerClient(fn func(context.Context, scannerv1.ScannerServiceClient) error) error {
	return withModuleConn(capMediaScanner, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, scannerv1.NewScannerServiceClient(conn))
	})
}

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Manual import from disk (admin-ui /import parity)",
		GroupID: groupAutomation,
	}
	cmd.AddCommand(newImportCandidatesCmd())
	cmd.AddCommand(newImportPathCmd())
	return cmd
}

func newImportCandidatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "candidates",
		Short: "List import candidates on disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withScannerClient(func(ctx context.Context, cli scannerv1.ScannerServiceClient) error {
				resp, err := cli.ListImportCandidates(ctx, &scannerv1.ListImportCandidatesRequest{Limit: 100})
				if err != nil {
					return fmt.Errorf("import candidates: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetCandidates())
				}
				rows := make([][]string, 0, len(resp.GetCandidates()))
				for _, c := range resp.GetCandidates() {
					rows = append(rows, []string{c.GetPath(), c.GetName(), fmt.Sprintf("%d", c.GetSize())})
				}
				return printTable([]string{"PATH", "NAME", "BYTES"}, rows)
			})
		},
	}
}

func newImportPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path <filesystem-path>",
		Short: "Import media from a path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withScannerClient(func(ctx context.Context, cli scannerv1.ScannerServiceClient) error {
				resp, err := cli.ImportPath(ctx, &scannerv1.ImportPathRequest{Path: args[0]})
				if err != nil {
					return fmt.Errorf("import path: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("imported=%d skipped=%d found=%d\n",
						resp.GetFilesImported(), resp.GetFilesSkipped(), resp.GetFilesFound())
				}
				return nil
			})
		},
	}
}
