package cli

import (
	"context"
	"fmt"
	"time"

	backupv1 "github.com/Muxcore-Media/backup-local/muxcore/backup/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capBackup = "backup"

func withBackupClient(fn func(context.Context, backupv1.BackupServiceClient) error) error {
	return withModuleConn(capBackup, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, backupv1.NewBackupServiceClient(conn))
	})
}

func newBackupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "backups",
		Short:   "Create and restore backups (admin-ui /backups parity)",
		GroupID: groupSystem,
	}
	cmd.AddCommand(newBackupsListCmd())
	cmd.AddCommand(newBackupsCreateCmd())
	cmd.AddCommand(newBackupsDeleteCmd())
	cmd.AddCommand(newBackupsRestoreCmd())
	return cmd
}

func newBackupsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBackupClient(func(ctx context.Context, cli backupv1.BackupServiceClient) error {
				resp, err := cli.ListBackups(ctx, &backupv1.ListBackupsRequest{})
				if err != nil {
					return fmt.Errorf("backups list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetBackups())
				}
				rows := make([][]string, 0, len(resp.GetBackups()))
				for _, b := range resp.GetBackups() {
					ts := time.Unix(b.GetTimestampUnix(), 0).UTC().Format(time.RFC3339)
					rows = append(rows, []string{
						b.GetId(), ts, fmt.Sprintf("%d", b.GetSizeBytes()), b.GetChecksumSha256(),
					})
				}
				return printTable([]string{"ID", "TIME", "BYTES", "SHA256"}, rows)
			})
		},
	}
}

func newBackupsCreateCmd() *cobra.Command {
	var paths []string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new backup archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBackupClient(func(ctx context.Context, cli backupv1.BackupServiceClient) error {
				resp, err := cli.CreateBackup(ctx, &backupv1.CreateBackupRequest{SourcePaths: paths})
				if err != nil {
					return fmt.Errorf("backups create: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetBackup())
				}
				if !flagQuiet {
					fmt.Printf("created %s (%d bytes)\n", resp.GetBackup().GetId(), resp.GetBackup().GetSizeBytes())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringSliceVar(&paths, "path", nil, "extra source paths to include")
	return cmd
}

func newBackupsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <backup-id>",
		Short: "Delete a backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete backup " + args[0] + "?"); err != nil {
				return err
			}
			return withBackupClient(func(ctx context.Context, cli backupv1.BackupServiceClient) error {
				_, err := cli.DeleteBackup(ctx, &backupv1.DeleteBackupRequest{BackupId: args[0]})
				if err != nil {
					return fmt.Errorf("backups delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newBackupsRestoreCmd() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "restore <backup-id>",
		Short: "Restore a backup to a target directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if target == "" {
				return fmt.Errorf("--target is required")
			}
			if err := confirmAction("Restore backup " + args[0] + " to " + target + "?"); err != nil {
				return err
			}
			return withBackupClient(func(ctx context.Context, cli backupv1.BackupServiceClient) error {
				resp, err := cli.RestoreBackup(ctx, &backupv1.RestoreBackupRequest{
					BackupId: args[0], TargetPath: target,
				})
				if err != nil {
					return fmt.Errorf("backups restore: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("restored %d files to %s\n", resp.GetFilesRestored(), target)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "directory to extract into")
	_ = cmd.MarkFlagRequired("target")
	return cmd
}
