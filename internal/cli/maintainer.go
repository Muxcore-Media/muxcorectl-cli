package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	maintainv1 "github.com/Muxcore-Media/media-library-maintainer/proto/maintainv1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const capMediaLibraryMaintainer = "media.library.maintainer"

func withMaintainerClient(fn func(context.Context, maintainv1.MaintainerServiceClient) error) error {
	return withModuleConn(capMediaLibraryMaintainer, func(ctx context.Context, conn *grpc.ClientConn) error {
		return fn(ctx, maintainv1.NewMaintainerServiceClient(conn))
	})
}

func newMaintainerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "maintainer",
		Short:   "Library maintainer rules and cleanup (admin-ui /maintainer parity)",
		GroupID: groupAutomation,
	}
	cmd.AddCommand(newMaintainerRulesCmd())
	cmd.AddCommand(newMaintainerCandidatesCmd())
	cmd.AddCommand(newMaintainerRunsCmd())
	cmd.AddCommand(newMaintainerScanCmd())
	cmd.AddCommand(newMaintainerActCmd())
	cmd.AddCommand(newMaintainerApproveCmd())
	cmd.AddCommand(newMaintainerCancelCmd())
	cmd.AddCommand(newMaintainerRulesDeleteCmd())
	cmd.AddCommand(newMaintainerRulesAddCmd())
	cmd.AddCommand(newMaintainerRulesExportCmd())
	cmd.AddCommand(newMaintainerRulesImportCmd())
	cmd.AddCommand(newMaintainerExclusionsCmd())
	cmd.AddCommand(newMaintainerFreeUpCmd())
	return cmd
}

func newMaintainerRulesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules-delete <rule-id>",
		Short: "Delete a maintainer rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete maintainer rule " + args[0] + "?"); err != nil {
				return err
			}
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				_, err := cli.DeleteRule(ctx, &maintainv1.DeleteRuleRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("maintainer rules delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newMaintainerRulesAddCmd() *cobra.Command {
	var (
		scope, outcome, arrAction, definitionJSON, qualityProfile string
		autoActEnabled                                            bool
		autoActDelayDays                                          int32
	)
	cmd := &cobra.Command{
		Use:   "rules-add <name>",
		Short: "Create a maintainer rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				scopeVal := maintainv1.MediaScope(maintainv1.MediaScope_value[ensureEnumPrefix(scope, "MEDIA_SCOPE_")])
				outcomeVal := maintainv1.RuleOutcome(maintainv1.RuleOutcome_value[ensureEnumPrefix(outcome, "RULE_OUTCOME_")])
				actionVal := maintainv1.ArrAction(maintainv1.ArrAction_value[ensureEnumPrefix(arrAction, "ARR_ACTION_")])
				resp, err := cli.UpsertRule(ctx, &maintainv1.UpsertRuleRequest{Rule: &maintainv1.RuleGroup{
					Name: args[0], Enabled: true, Scope: scopeVal, DefinitionJson: definitionJSON,
					Outcome: outcomeVal, ArrAction: actionVal, AutoActEnabled: autoActEnabled,
					AutoActDelayDays: autoActDelayDays, MaxActionsPerRun: 50, QualityProfileId: qualityProfile,
				}})
				if err != nil {
					return fmt.Errorf("maintainer rules add: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetRule())
				}
				if !flagQuiet {
					fmt.Printf("created %s\n", resp.GetRule().GetId())
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "MEDIA_SCOPE_UNSPECIFIED", "media scope enum")
	cmd.Flags().StringVar(&outcome, "outcome", "RULE_OUTCOME_CANDIDATE", "rule outcome enum")
	cmd.Flags().StringVar(&arrAction, "arr-action", "ARR_ACTION_DELETE", "arr action enum")
	cmd.Flags().StringVar(&definitionJSON, "definition", "{}", "rule definition JSON")
	cmd.Flags().StringVar(&qualityProfile, "quality-profile", "", "quality profile id")
	cmd.Flags().BoolVar(&autoActEnabled, "auto-act", false, "enable automatic action")
	cmd.Flags().Int32Var(&autoActDelayDays, "auto-act-delay-days", 0, "auto act delay in days")
	return cmd
}

func newMaintainerRulesExportCmd() *cobra.Command {
	var outFile string
	cmd := &cobra.Command{
		Use:   "rules-export",
		Short: "Export maintainer rules as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ExportRules(ctx, &maintainv1.ExportRulesRequest{})
				if err != nil {
					return fmt.Errorf("maintainer rules export: %w", err)
				}
				if outFile != "" {
					if err := os.WriteFile(outFile, []byte(resp.GetRulesJson()), 0o600); err != nil {
						return err
					}
					if !flagQuiet {
						fmt.Printf("wrote %s\n", outFile)
					}
					return nil
				}
				if flagJSON {
					var parsed any
					if err := json.Unmarshal([]byte(resp.GetRulesJson()), &parsed); err != nil {
						return printJSON(resp.GetRulesJson())
					}
					return printJSON(parsed)
				}
				fmt.Println(resp.GetRulesJson())
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&outFile, "out", "", "write JSON to file instead of stdout")
	return cmd
}

func newMaintainerRulesImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules-import <file>",
		Short: "Import maintainer rules from JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read rules: %w", err)
			}
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ImportRules(ctx, &maintainv1.ImportRulesRequest{RulesJson: string(raw)})
				if err != nil {
					return fmt.Errorf("maintainer rules import: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("imported=%d\n", resp.GetImported())
				}
				return nil
			})
		},
	}
}

func newMaintainerExclusionsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "exclusions", Short: "Maintainer exclusion list commands"}
	list := &cobra.Command{
		Use:   "list",
		Short: "List maintainer exclusions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ListExclusionLists(ctx, &maintainv1.ListExclusionListsRequest{})
				if err != nil {
					return fmt.Errorf("maintainer exclusions: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetLists())
				}
				rows := make([][]string, 0, len(resp.GetLists()))
				for _, e := range resp.GetLists() {
					rows = append(rows, []string{e.GetId(), e.GetName(), e.GetType(), fmt.Sprintf("%d", len(e.GetTmdbIds()))})
				}
				return printTable([]string{"ID", "NAME", "TYPE", "TMDB_IDS"}, rows)
			})
		},
	}
	sync := &cobra.Command{
		Use:   "sync",
		Short: "Sync exclusion lists from external sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.SyncExclusionLists(ctx, &maintainv1.SyncExclusionListsRequest{})
				if err != nil {
					return fmt.Errorf("maintainer exclusions sync: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("lists_synced=%d ids_loaded=%d\n", resp.GetListsSynced(), resp.GetIdsLoaded())
				}
				return nil
			})
		},
	}
	add := &cobra.Command{
		Use:   "add <name>",
		Short: "Add an exclusion list source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			listType, _ := cmd.Flags().GetString("type")
			listURL, _ := cmd.Flags().GetString("list-url")
			apiKey, _ := cmd.Flags().GetString("api-key")
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.UpsertExclusionList(ctx, &maintainv1.UpsertExclusionListRequest{List: &maintainv1.ExclusionList{
					Name: args[0], Type: listType, ListUrl: listURL, ApiKey: apiKey,
				}})
				if err != nil {
					return fmt.Errorf("maintainer exclusions add: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetList())
				}
				if !flagQuiet {
					fmt.Printf("created %s\n", resp.GetList().GetId())
				}
				return nil
			})
		},
	}
	add.Flags().String("type", "trakt", "list type")
	add.Flags().String("list-url", "", "list URL")
	add.Flags().String("api-key", "", "API key")
	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an exclusion list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete exclusion list " + args[0] + "?"); err != nil {
				return err
			}
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				_, err := cli.DeleteExclusionList(ctx, &maintainv1.DeleteExclusionListRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("maintainer exclusions delete: %w", err)
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
	cmd.AddCommand(list, add, deleteCmd, sync)
	return cmd
}

func newMaintainerFreeUpCmd() *cobra.Command {
	var target float64
	cmd := &cobra.Command{
		Use:   "free-up",
		Short: "Run maintainer free-up actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Run maintainer free-up now?"); err != nil {
				return err
			}
			if target <= 0 {
				target = 15
			}
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ActNow(ctx, &maintainv1.ActNowRequest{FreeUp: true, TargetFreePercent: target})
				if err != nil {
					return fmt.Errorf("maintainer free-up: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("actions_taken=%d target=%.1f%%\n", resp.GetActionsTaken(), target)
				}
				return nil
			})
		},
	}
	cmd.Flags().Float64Var(&target, "target", 15, "target free disk percent")
	return cmd
}

func ensureEnumPrefix(value, prefix string) string {
	v := strings.ToUpper(strings.TrimSpace(value))
	if strings.HasPrefix(v, prefix) {
		return v
	}
	return prefix + v
}

func newMaintainerRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules",
		Short: "List maintainer rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ListRules(ctx, &maintainv1.ListRulesRequest{})
				if err != nil {
					return fmt.Errorf("maintainer rules: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetRules())
				}
				rows := make([][]string, 0, len(resp.GetRules()))
				for _, r := range resp.GetRules() {
					rows = append(rows, []string{
						r.GetId(), r.GetName(), fmt.Sprintf("%t", r.GetEnabled()),
					})
				}
				return printTable([]string{"ID", "NAME", "ENABLED"}, rows)
			})
		},
	}
}

func newMaintainerCandidatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "candidates",
		Short: "List pending maintainer candidates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ListCandidates(ctx, &maintainv1.ListCandidatesRequest{Page: 1, PageSize: 50})
				if err != nil {
					return fmt.Errorf("maintainer candidates: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetCandidates())
				}
				rows := make([][]string, 0, len(resp.GetCandidates()))
				for _, c := range resp.GetCandidates() {
					rows = append(rows, []string{c.GetId(), c.GetTitle(), c.GetStatus().String()})
				}
				return printTable([]string{"ID", "TITLE", "STATUS"}, rows)
			})
		},
	}
}

func newMaintainerRunsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "runs",
		Short: "List recent maintainer runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ListRuns(ctx, &maintainv1.ListRunsRequest{Page: 1, PageSize: 20})
				if err != nil {
					return fmt.Errorf("maintainer runs: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetRuns())
				}
				rows := make([][]string, 0, len(resp.GetRuns()))
				for _, run := range resp.GetRuns() {
					rows = append(rows, []string{
						run.GetId(), run.GetKind(), run.GetStatus(),
						fmt.Sprintf("%d", run.GetCandidatesFound()), run.GetStartedAt(),
					})
				}
				return printTable([]string{"ID", "KIND", "STATUS", "FOUND", "STARTED"}, rows)
			})
		},
	}
}

func newMaintainerScanCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Run a maintainer scan now",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ScanNow(ctx, &maintainv1.ScanNowRequest{DryRun: dryRun})
				if err != nil {
					return fmt.Errorf("maintainer scan: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("candidates_found=%d dry_run=%t\n", resp.GetCandidatesFound(), dryRun)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "scan without recording actions")
	return cmd
}

func newMaintainerActCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "act",
		Short: "Execute pending maintainer actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Execute maintainer actions now?"); err != nil {
				return err
			}
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ActNow(ctx, &maintainv1.ActNowRequest{DryRun: dryRun})
				if err != nil {
					return fmt.Errorf("maintainer act: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if !flagQuiet {
					fmt.Printf("actions_taken=%d\n", resp.GetActionsTaken())
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview actions without applying")
	return cmd
}

func newMaintainerApproveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <candidate-id>",
		Short: "Approve a maintainer candidate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				resp, err := cli.ApproveCandidate(ctx, &maintainv1.ApproveCandidateRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("maintainer approve: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetCandidate())
				}
				if !flagQuiet {
					fmt.Println("approved")
				}
				return nil
			})
		},
	}
}

func newMaintainerCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <candidate-id>",
		Short: "Cancel a maintainer candidate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMaintainerClient(func(ctx context.Context, cli maintainv1.MaintainerServiceClient) error {
				_, err := cli.CancelCandidate(ctx, &maintainv1.CancelCandidateRequest{Id: args[0]})
				if err != nil {
					return fmt.Errorf("maintainer cancel: %w", err)
				}
				if !flagQuiet {
					fmt.Println("cancelled")
				}
				return nil
			})
		},
	}
}
