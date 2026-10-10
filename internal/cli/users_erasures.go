package cli

import (
	"context"
	"fmt"
	"strings"

	authv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/auth/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
)

// ADR-0035 §1/§2 + ADR-0026: DeleteUser and GetUserErasureStatus take the
// end-user admin session in x-auth-token. The Authorization header that
// dialModuleGRPC adds is not read for these methods, so the token is also
// attached explicitly. The CLI never calls ListUserErasures / AckUserErasure
// (module-certificate only) and owns no personal-data store, so it runs no
// reconciler.
const authTokenMetadataKey = "x-auth-token" //nolint:gosec // G101: gRPC metadata key name, not a secret

// withAdminToken attaches the operator token as x-auth-token. It reports
// whether a token was available.
func withAdminToken(ctx context.Context) (context.Context, bool) {
	tok := resolveOperatorToken()
	if tok == "" {
		return ctx, false
	}
	return metadata.AppendToOutgoingContext(ctx, authTokenMetadataKey, tok), true
}

// Outcome labels printed by `users erasures`. UNSPECIFIED is a required module
// that has not acknowledged yet (core v0.6.17 proto comment).
const (
	erasureOutcomeOK          = "ok"
	erasureOutcomeFailed      = "failed"
	erasureOutcomeUnsupported = "unsupported"
	erasureOutcomePending     = "pending"
)

func erasureOutcomeLabel(o authv1.ErasureOutcome) string {
	switch o {
	case authv1.ErasureOutcome_ERASURE_OUTCOME_OK:
		return erasureOutcomeOK
	case authv1.ErasureOutcome_ERASURE_OUTCOME_FAILED:
		return erasureOutcomeFailed
	case authv1.ErasureOutcome_ERASURE_OUTCOME_UNSUPPORTED:
		return erasureOutcomeUnsupported
	default:
		return erasureOutcomePending
	}
}

type erasureModuleView struct {
	ModuleID   string `json:"module_id"`
	Outcome    string `json:"outcome"`
	DetailCode string `json:"detail_code,omitempty"`
	AckedAt    string `json:"acked_at,omitempty"`
	Required   bool   `json:"required"`
}

type erasureView struct {
	ErasureID string              `json:"erasure_id"`
	DeletedAt string              `json:"deleted_at"`
	Modules   []erasureModuleView `json:"modules"`
	Complete  bool                `json:"complete"`
}

// newErasureView renders one ErasureStatus. An erasure is reported complete
// only when the provider says so (every AUTH_ERASURE_REQUIRED module
// acknowledged OK, ADR-0035 §5) and every module it lists is OK, so a failed,
// unsupported or pending module is never hidden behind a "complete" row.
func newErasureView(s *authv1.ErasureStatus) erasureView {
	v := erasureView{
		ErasureID: s.GetErasureId(),
		DeletedAt: s.GetDeletedAt(),
		Modules:   make([]erasureModuleView, 0, len(s.GetModules())),
	}
	allOK := true
	for _, m := range s.GetModules() {
		label := erasureOutcomeLabel(m.GetOutcome())
		if label != erasureOutcomeOK {
			allOK = false
		}
		v.Modules = append(v.Modules, erasureModuleView{
			ModuleID:   m.GetModuleId(),
			Outcome:    label,
			Required:   m.GetRequired(),
			DetailCode: m.GetDetailCode(),
			AckedAt:    m.GetAckedAt(),
		})
	}
	v.Complete = s.GetComplete() && allOK
	return v
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// erasureTableRows flattens erasures to one row per (erasure, module) so the
// output stays grep/awk friendly. An erasure with no listed modules gets a
// single row with "-" placeholders.
func erasureTableRows(views []erasureView) [][]string {
	rows := make([][]string, 0, len(views))
	for _, v := range views {
		if len(v.Modules) == 0 {
			rows = append(rows, []string{v.ErasureID, v.DeletedAt, yesNo(v.Complete), "-", "-", "-"})
			continue
		}
		for _, m := range v.Modules {
			rows = append(rows, []string{v.ErasureID, v.DeletedAt, yesNo(v.Complete), m.ModuleID, m.Outcome, yesNo(m.Required)})
		}
	}
	return rows
}

func newUsersErasuresCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "erasures [erasure-id]",
		Short: "Show user-erasure completion (ADR-0035)",
		Long: `Show per-module completion of user erasures (ADR-0035), as reported by the
identity provider's GetUserErasureStatus.

With no argument, lists erasures that are not yet complete; --all lists every
erasure in the tenant. With an <erasure-id> (printed by 'users delete'), shows
that one erasure; an unknown id is an error.

The call needs an administrator's end-user session token (--token,
--token-file, MUXCORE_TOKEN or MUXCORE_TOKEN_FILE); a module certificate alone
is rejected by the provider.

An erasure is complete only when the provider reports it complete and every
module it lists acknowledged ok. Outcomes: ok, failed, unsupported, pending
(a required module that has not acknowledged yet).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &authv1.GetUserErasureStatusRequest{}
			switch {
			case len(args) == 1:
				id := strings.TrimSpace(args[0])
				if id == "" {
					return fmt.Errorf("users erasures: erasure id must not be empty")
				}
				req.ErasureId = id
			default:
				req.PendingOnly = !all
			}
			if _, ok := withAdminToken(context.Background()); !ok {
				return fmt.Errorf("users erasures: administrator token required (--token, --token-file, MUXCORE_TOKEN, or MUXCORE_TOKEN_FILE)")
			}
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				ctx, _ = withAdminToken(ctx)
				views := []erasureView{}
				seen := map[string]bool{}
				for {
					resp, err := auth.GetUserErasureStatus(ctx, req)
					if err != nil {
						return fmt.Errorf("users erasures: %w", err)
					}
					for _, s := range resp.GetErasures() {
						views = append(views, newErasureView(s))
					}
					next := resp.GetNextPageToken()
					if next == "" || len(args) == 1 {
						break
					}
					if seen[next] {
						return fmt.Errorf("users erasures: provider repeated page token %q", next)
					}
					seen[next] = true
					req.PageToken = next
				}
				if flagJSON {
					return printJSON(views)
				}
				return printTable(
					[]string{"ERASURE_ID", "DELETED_AT", "COMPLETE", "MODULE", "OUTCOME", "REQUIRED"},
					erasureTableRows(views),
				)
			})
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "list every erasure, not only those that are not yet complete")
	return cmd
}
