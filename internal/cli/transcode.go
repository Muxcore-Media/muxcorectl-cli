package cli

import (
	"context"
	"fmt"

	transcodev1 "github.com/Muxcore-Media/media-transcoder/proto/transcodev1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

const (
	capMediaTranscoder = "media.transcoder"
	capTranscoder      = "transcoder"
)

func withTranscoderClient(fn func(context.Context, transcodev1.TranscodeServiceClient) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		var modErr error
		for _, cap := range []string{capMediaTranscoder, capTranscoder} {
			mod, err := findModuleByCapability(ctx, c, cap)
			if err != nil {
				modErr = err
				continue
			}
			conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
			if err != nil {
				return err
			}
			defer conn.Close()
			return fn(ctx, transcodev1.NewTranscodeServiceClient(conn))
		}
		if modErr != nil {
			return modErr
		}
		return fmt.Errorf("no module with capability %q or %q", capMediaTranscoder, capTranscoder)
	})
}

func newTranscodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transcode",
		Short:   "Transcode profiles, setups, and pipeline runs (admin-ui /transcode parity)",
		GroupID: groupPlayback,
	}
	cmd.AddCommand(newTranscodeProfilesCmd())
	cmd.AddCommand(newTranscodeSetupsCmd())
	cmd.AddCommand(newTranscodeRunsCmd())
	cmd.AddCommand(newTranscodeApproveCmd())
	cmd.AddCommand(newTranscodeRejectCmd())
	cmd.AddCommand(newTranscodeApplyTemplateCmd())
	return cmd
}

func newTranscodeProfilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profiles",
		Short: "List transcode profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTranscoderClient(func(ctx context.Context, cli transcodev1.TranscodeServiceClient) error {
				resp, err := cli.ListProfiles(ctx, &transcodev1.ListProfilesRequest{})
				if err != nil {
					return fmt.Errorf("transcode profiles: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetProfiles())
				}
				rows := make([][]string, 0, len(resp.GetProfiles()))
				for _, p := range resp.GetProfiles() {
					rows = append(rows, []string{p.GetId(), p.GetName()})
				}
				return printTable([]string{"ID", "NAME"}, rows)
			})
		},
	}
}

func newTranscodeSetupsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setups",
		Short: "List transcode pipeline setups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTranscoderClient(func(ctx context.Context, cli transcodev1.TranscodeServiceClient) error {
				resp, err := cli.ListSetups(ctx, &transcodev1.ListSetupsRequest{})
				if err != nil {
					return fmt.Errorf("transcode setups: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetSetups())
				}
				rows := make([][]string, 0, len(resp.GetSetups()))
				for _, s := range resp.GetSetups() {
					rows = append(rows, []string{s.GetId(), s.GetName(), fmt.Sprintf("%t", s.GetEnabled())})
				}
				return printTable([]string{"ID", "NAME", "ENABLED"}, rows)
			})
		},
	}
}

func newTranscodeRunsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "runs",
		Short: "List pipeline runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTranscoderClient(func(ctx context.Context, cli transcodev1.TranscodeServiceClient) error {
				resp, err := cli.ListPipelineRuns(ctx, &transcodev1.ListPipelineRunsRequest{Page: 1, PageSize: 25})
				if err != nil {
					return fmt.Errorf("transcode runs: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetRuns())
				}
				rows := make([][]string, 0, len(resp.GetRuns()))
				for _, run := range resp.GetRuns() {
					rows = append(rows, []string{run.GetId(), run.GetStatus(), run.GetSetupName(), run.GetCreatedAt()})
				}
				return printTable([]string{"ID", "STATUS", "SETUP", "STARTED"}, rows)
			})
		},
	}
}

func newTranscodeApproveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <run-id>",
		Short: "Approve a pending pipeline run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTranscoderClient(func(ctx context.Context, cli transcodev1.TranscodeServiceClient) error {
				resp, err := cli.ApprovePipelineRun(ctx, &transcodev1.ApprovePipelineRunRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("transcode approve: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Println("approved")
				}
				return nil
			})
		},
	}
}

func newTranscodeRejectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reject <run-id>",
		Short: "Reject a pending pipeline run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTranscoderClient(func(ctx context.Context, cli transcodev1.TranscodeServiceClient) error {
				resp, err := cli.RejectPipelineRun(ctx, &transcodev1.RejectPipelineRunRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("transcode reject: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Println("rejected")
				}
				return nil
			})
		},
	}
}

func newTranscodeApplyTemplateCmd() *cobra.Command {
	var templateID string
	var replaceSteps bool
	cmd := &cobra.Command{
		Use:   "apply-template <setup-id>",
		Short: "Apply a step template to a transcode setup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if templateID == "" {
				return fmt.Errorf("--template-id is required")
			}
			return withTranscoderClient(func(ctx context.Context, cli transcodev1.TranscodeServiceClient) error {
				resp, err := cli.ApplyStepTemplate(ctx, &transcodev1.ApplyStepTemplateRequest{
					SetupId:      args[0],
					TemplateId:   templateID,
					ReplaceSteps: replaceSteps,
				})
				if err != nil {
					return fmt.Errorf("transcode apply-template: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Println("applied")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&templateID, "template-id", "", "step template id")
	cmd.Flags().BoolVar(&replaceSteps, "replace-steps", false, "replace existing steps")
	return cmd
}
