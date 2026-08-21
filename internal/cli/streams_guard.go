package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	guardv1 "github.com/Muxcore-Media/playback-guard/proto/guardv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capPlaybackGuard = "playback.guard"

func withPlaybackGuardClient(fn func(context.Context, guardv1.PlaybackGuardServiceClient) error) error {
	return withModuleConn(capPlaybackGuard, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, guardv1.NewPlaybackGuardServiceClient(conn))
	})
}

func newStreamsGuardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guard",
		Short: "Playback guard rules, violations, and trust (admin-ui /streams/guard parity)",
	}
	cmd.AddCommand(newStreamsGuardShowCmd())
	cmd.AddCommand(newStreamsGuardViolationsCmd())
	cmd.AddCommand(newStreamsGuardAcknowledgeCmd())
	cmd.AddCommand(newStreamsGuardTrustCmd())
	cmd.AddCommand(newStreamsGuardMergeCmd())
	cmd.AddCommand(newStreamsGuardTerminateCmd())
	return cmd
}

func newStreamsGuardShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show guard rules, recent violations, and trust scores",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlaybackGuardClient(func(ctx context.Context, cli guardv1.PlaybackGuardServiceClient) error {
				rules, err := cli.ListRules(ctx, &guardv1.ListRulesRequest{})
				if err != nil {
					return fmt.Errorf("streams guard rules: %w", err)
				}
				violations, err := cli.ListViolations(ctx, &guardv1.ListViolationsRequest{Limit: 50})
				if err != nil {
					return fmt.Errorf("streams guard violations: %w", err)
				}
				trust, err := cli.ListTrustScores(ctx, &guardv1.ListTrustScoresRequest{Limit: 50})
				if err != nil {
					return fmt.Errorf("streams guard trust: %w", err)
				}
				if flagJSON {
					return printJSON(map[string]any{
						"rules":      rules.GetRules(),
						"violations": violations.GetViolations(),
						"trust":      trust.GetScores(),
					})
				}
				fmt.Println("Rules:")
				ruleRows := make([][]string, 0, len(rules.GetRules()))
				for _, r := range rules.GetRules() {
					enabled := "disabled"
					if r.GetEnabled() {
						enabled = "enabled"
					}
					ruleRows = append(ruleRows, []string{r.GetName(), r.GetType().String(), enabled})
				}
				if err := printTable([]string{"NAME", "TYPE", "STATE"}, ruleRows); err != nil {
					return err
				}
				fmt.Println("\nViolations:")
				vRows := make([][]string, 0, len(violations.GetViolations()))
				for _, v := range violations.GetViolations() {
					user := v.GetUserName()
					if user == "" {
						user = v.GetUserId()
					}
					created := ""
					if v.GetCreatedAtUnix() > 0 {
						created = time.Unix(v.GetCreatedAtUnix(), 0).UTC().Format(time.RFC3339)
					}
					vRows = append(vRows, []string{v.GetId(), user, v.GetSeverity(), v.GetSummary(), created})
				}
				return printTable([]string{"ID", "USER", "SEVERITY", "SUMMARY", "CREATED"}, vRows)
			})
		},
	}
}

func newStreamsGuardViolationsCmd() *cobra.Command {
	var limit int32
	cmd := &cobra.Command{
		Use:   "violations",
		Short: "List guard violations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 {
				limit = 100
			}
			return withPlaybackGuardClient(func(ctx context.Context, cli guardv1.PlaybackGuardServiceClient) error {
				resp, err := cli.ListViolations(ctx, &guardv1.ListViolationsRequest{Limit: limit})
				if err != nil {
					return fmt.Errorf("streams guard violations: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetViolations())
				}
				rows := make([][]string, 0, len(resp.GetViolations()))
				for _, v := range resp.GetViolations() {
					user := v.GetUserName()
					if user == "" {
						user = v.GetUserId()
					}
					rows = append(rows, []string{v.GetId(), user, v.GetSeverity(), v.GetSummary()})
				}
				return printTable([]string{"ID", "USER", "SEVERITY", "SUMMARY"}, rows)
			})
		},
	}
	cmd.Flags().Int32Var(&limit, "limit", 100, "max rows")
	return cmd
}

func newStreamsGuardAcknowledgeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "acknowledge <violation-id> [more...]",
		Short: "Acknowledge one or more violations",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlaybackGuardClient(func(ctx context.Context, cli guardv1.PlaybackGuardServiceClient) error {
				resp, err := cli.AcknowledgeViolations(ctx, &guardv1.AcknowledgeViolationsRequest{ViolationIds: args})
				if err != nil {
					return fmt.Errorf("streams guard acknowledge: %w", err)
				}
				if flagJSON {
					return printJSON(map[string]int32{"updated": resp.GetUpdated()})
				}
				if !flagQuiet {
					fmt.Printf("acknowledged %d violation(s)\n", resp.GetUpdated())
				}
				return nil
			})
		},
	}
}

func newStreamsGuardTrustCmd() *cobra.Command {
	reset := &cobra.Command{
		Use:   "reset",
		Short: "Reset trust score for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, _ := cmd.Flags().GetString("user-id")
			userName, _ := cmd.Flags().GetString("user-name")
			if strings.TrimSpace(userID) == "" && strings.TrimSpace(userName) == "" {
				return fmt.Errorf("--user-id or --user-name is required")
			}
			return withPlaybackGuardClient(func(ctx context.Context, cli guardv1.PlaybackGuardServiceClient) error {
				_, err := cli.ResetTrustScore(ctx, &guardv1.ResetTrustScoreRequest{
					UserId:   userID,
					UserName: userName,
				})
				if err != nil {
					return fmt.Errorf("streams guard trust reset: %w", err)
				}
				if !flagQuiet {
					fmt.Println("trust score reset")
				}
				return nil
			})
		},
	}
	reset.Flags().String("user-id", "", "user id")
	reset.Flags().String("user-name", "", "username")
	list := &cobra.Command{
		Use:   "list",
		Short: "List user trust scores",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlaybackGuardClient(func(ctx context.Context, cli guardv1.PlaybackGuardServiceClient) error {
				resp, err := cli.ListTrustScores(ctx, &guardv1.ListTrustScoresRequest{Limit: 100})
				if err != nil {
					return fmt.Errorf("streams guard trust list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetScores())
				}
				rows := make([][]string, 0, len(resp.GetScores()))
				for _, ts := range resp.GetScores() {
					user := ts.GetUserName()
					if user == "" {
						user = ts.GetUserId()
					}
					rows = append(rows, []string{user, fmt.Sprintf("%d", ts.GetScore())})
				}
				return printTable([]string{"USER", "SCORE"}, rows)
			})
		},
	}
	cmd := &cobra.Command{Use: "trust", Short: "Trust score commands"}
	cmd.AddCommand(list, reset)
	return cmd
}

func newStreamsGuardMergeCmd() *cobra.Command {
	var sourceID, sourceName, targetID, targetName string
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge two user identities in guard state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sourceID) == "" && strings.TrimSpace(sourceName) == "" {
				return fmt.Errorf("--source-user-id or --source-user-name is required")
			}
			if strings.TrimSpace(targetID) == "" && strings.TrimSpace(targetName) == "" {
				return fmt.Errorf("--target-user-id or --target-user-name is required")
			}
			return withPlaybackGuardClient(func(ctx context.Context, cli guardv1.PlaybackGuardServiceClient) error {
				resp, err := cli.MergeUsers(ctx, &guardv1.MergeUsersRequest{
					SourceUserId:   sourceID,
					SourceUserName: sourceName,
					TargetUserId:   targetID,
					TargetUserName: targetName,
				})
				if err != nil {
					return fmt.Errorf("streams guard merge: %w", err)
				}
				if flagJSON {
					return printJSON(map[string]int32{
						"violations_updated": resp.GetViolationsUpdated(),
						"sessions_updated":   resp.GetSessionsUpdated(),
					})
				}
				if !flagQuiet {
					fmt.Printf("merged: violations=%d sessions=%d\n", resp.GetViolationsUpdated(), resp.GetSessionsUpdated())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&sourceID, "source-user-id", "", "source user id")
	cmd.Flags().StringVar(&sourceName, "source-user-name", "", "source username")
	cmd.Flags().StringVar(&targetID, "target-user-id", "", "target user id")
	cmd.Flags().StringVar(&targetName, "target-user-name", "", "target username")
	return cmd
}

func newStreamsGuardTerminateCmd() *cobra.Command {
	var serverType string
	cmd := &cobra.Command{
		Use:   "terminate <session-id>",
		Short: "Terminate an active playback session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Terminate session " + args[0] + "?"); err != nil {
				return err
			}
			return withPlaybackGuardClient(func(ctx context.Context, cli guardv1.PlaybackGuardServiceClient) error {
				resp, err := cli.TerminateSession(ctx, &guardv1.TerminateSessionRequest{
					SessionId:  args[0],
					ServerType: serverType,
					Reason:     "operator terminate from muxcorectl",
				})
				if err != nil {
					return fmt.Errorf("streams guard terminate: %w", err)
				}
				if !resp.GetOk() {
					msg := resp.GetError()
					if msg == "" {
						msg = "terminate failed"
					}
					return fmt.Errorf("%s", msg)
				}
				if !flagQuiet {
					fmt.Println("terminated")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&serverType, "server-type", "", "server type (e.g. jellyfin)")
	return cmd
}
