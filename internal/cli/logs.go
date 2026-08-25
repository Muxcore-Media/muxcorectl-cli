package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func logDir() string {
	if v := strings.TrimSpace(os.Getenv("ADMIN_UI_LOG_DIR")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("MVP_RUN")); v != "" {
		return filepath.Join(v, "logs")
	}
	return filepath.Join(os.TempDir(), "muxcore-logs")
}

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logs",
		Short:   "Tail module log files (admin-ui /logs parity)",
		GroupID: groupMonitoring,
	}
	cmd.AddCommand(newLogsListCmd())
	cmd.AddCommand(newLogsTailCmd())
	return cmd
}

func newLogsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List log files in the log directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := logDir()
			entries, err := os.ReadDir(dir)
			if err != nil {
				return fmt.Errorf("logs list: %w", err)
			}
			type row struct {
				Name string `json:"name"`
				Size int64  `json:"size"`
			}
			var out []row
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				out = append(out, row{Name: e.Name(), Size: info.Size()})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
			if flagJSON {
				return printJSON(map[string]any{"dir": dir, "files": out})
			}
			fmt.Printf("dir: %s\n", dir)
			rows := make([][]string, 0, len(out))
			for _, f := range out {
				rows = append(rows, []string{f.Name, fmt.Sprintf("%d", f.Size)})
			}
			return printTable([]string{"FILE", "BYTES"}, rows)
		},
	}
}

func newLogsTailCmd() *cobra.Command {
	var lines int
	cmd := &cobra.Command{
		Use:   "tail <file>",
		Short: "Tail the last N lines of a log file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if lines < 1 {
				lines = 200
			}
			name := filepath.Base(args[0])
			path := filepath.Join(logDir(), name)
			f, err := os.Open(path) //nolint:gosec // operator-selected log file under MVP log dir
			if err != nil {
				return fmt.Errorf("logs tail: %w", err)
			}
			defer func() { _ = f.Close() }()
			var buf []string
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				buf = append(buf, sc.Text())
				if len(buf) > lines {
					buf = buf[1:]
				}
			}
			if err := sc.Err(); err != nil {
				return fmt.Errorf("logs tail: %w", err)
			}
			if flagJSON {
				return printJSON(map[string]any{"file": name, "lines": buf})
			}
			for _, line := range buf {
				fmt.Println(line)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&lines, "lines", 200, "number of lines to show")
	return cmd
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
