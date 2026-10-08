package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
)

// Parental restrictions are authoritative in userdata-local (ADR-0030) and
// enforced by the BFF (ADR-0031). admin-ui edits them and migrates the legacy
// parental.json. This CLI no longer writes that file: it is read by nothing
// that enforces, so a write would report success for a restriction that does
// not exist (and used to drop kids_mode / pin_hash).

const (
	parentalAdminUIPath    = "/users/{id}/parental"
	parentalMigrateUIPath  = "/users/parental/migrate"
	parentalLegacyBanner   = "LEGACY, NON-AUTHORITATIVE parental.json (ADR-0031): this file is not enforced; what it shows is not what is enforced."
	parentalSetRemovedText = "users parental set: removed. Parental restrictions are now managed in admin-ui " +
		"(provider-backed userdata-local policy, ADR-0030/ADR-0031), not in parental.json. " +
		"Edit a user's restrictions at admin-ui " + parentalAdminUIPath + " and import existing legacy settings " +
		"once at admin-ui " + parentalMigrateUIPath + ". Nothing was written"
)

// legacyParentalEntry mirrors one entry of admin-ui's legacy parental.json,
// including the fields the old CLI struct dropped (kids_mode, pin_hash).
type legacyParentalEntry struct {
	MaxParentalRating string `json:"max_parental_rating"`
	BlockedTags       string `json:"blocked_tags"`
	AllowedTags       string `json:"allowed_tags"`
	PINHash           string `json:"pin_hash"`
	AllowUnrated      bool   `json:"allow_unrated"`
	KidsMode          bool   `json:"kids_mode"`
}

// legacyParentalView is what `show` prints. It never carries the PIN hash,
// only whether one is present.
type legacyParentalView struct {
	MaxParentalRating string `json:"max_parental_rating"`
	BlockedTags       string `json:"blocked_tags"`
	AllowedTags       string `json:"allowed_tags"`
	AllowUnrated      bool   `json:"allow_unrated"`
	KidsMode          bool   `json:"kids_mode"`
	PINHashPresent    bool   `json:"pin_hash_present"`
}

func parentalPath() string {
	return adminDataFile("ADMIN_UI_PARENTAL_FILE", "parental.json")
}

// loadLegacyParental reads parental.json. A missing, unreadable or corrupt
// file is an error so callers never present it as an empty result.
func loadLegacyParental() (map[string]legacyParentalEntry, error) {
	path := parentalPath()
	raw, err := os.ReadFile(path) //nolint:gosec // operator-selected legacy parental.json path (ADMIN_UI_PARENTAL_FILE)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("legacy parental file %s does not exist (nothing to show; restrictions are managed in admin-ui)", path)
		}
		return nil, fmt.Errorf("legacy parental file %s is unreadable: %w", path, err)
	}
	var m map[string]legacyParentalEntry
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("legacy parental file %s is corrupt: %w", path, err)
	}
	if m == nil {
		return nil, fmt.Errorf("legacy parental file %s is corrupt: expected a JSON object keyed by user id", path)
	}
	return m, nil
}

func newUsersParentalCmd() *cobra.Command {
	show := &cobra.Command{
		Use:   "show <user-id>",
		Short: "Show the LEGACY parental.json entry for a user (not enforced)",
		Long: `Show the legacy admin-ui parental.json entry for a user, read-only.

` + parentalLegacyBanner + `
Enforced restrictions live in userdata-local and are edited in admin-ui at
` + parentalAdminUIPath + `. The PIN hash is never printed; only whether one is present.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := loadLegacyParental()
			if err != nil {
				return fmt.Errorf("users parental show: %w", err)
			}
			e, found := m[args[0]]
			view := legacyParentalView{
				MaxParentalRating: e.MaxParentalRating,
				BlockedTags:       e.BlockedTags,
				AllowedTags:       e.AllowedTags,
				AllowUnrated:      e.AllowUnrated,
				KidsMode:          e.KidsMode,
				PINHashPresent:    e.PINHash != "",
			}
			if flagJSON {
				return printJSON(map[string]any{
					"legacy":        true,
					"authoritative": false,
					"notice":        parentalLegacyBanner,
					"user_id":       args[0],
					"path":          parentalPath(),
					"found":         found,
					"settings":      view,
				})
			}
			fmt.Println(parentalLegacyBanner)
			fmt.Printf("file:          %s\n", parentalPath())
			if !found {
				fmt.Printf("no legacy entry for user %q in this file\n", args[0])
				return nil
			}
			fmt.Printf("max_rating:    %s\n", view.MaxParentalRating)
			fmt.Printf("blocked_tags:  %s\n", view.BlockedTags)
			fmt.Printf("allowed_tags:  %s\n", view.AllowedTags)
			fmt.Printf("allow_unrated: %t\n", view.AllowUnrated)
			fmt.Printf("kids_mode:     %t\n", view.KidsMode)
			fmt.Printf("pin_hash:      %s\n", map[bool]string{true: "present (not shown)", false: "absent"}[view.PINHashPresent])
			return nil
		},
	}

	set := &cobra.Command{
		Use:   "set <user-id>",
		Short: "Removed: parental restrictions are managed in admin-ui",
		Long: `Removed. This command no longer writes anything and always fails.

Parental restrictions are authoritative in userdata-local (ADR-0030) and
enforced by the BFF (ADR-0031). Manage them in admin-ui at
` + parentalAdminUIPath + `; import existing legacy parental.json settings once at
` + parentalMigrateUIPath + `.`,
		// Accept any args/flags so an old invocation reaches the explanation
		// instead of failing on a flag-parse error.
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, a := range args {
				if a == "-h" || a == "--help" {
					return cmd.Help()
				}
			}
			return errors.New(parentalSetRemovedText)
		},
	}

	cmd := &cobra.Command{
		Use:   "parental",
		Short: "Legacy parental.json viewer (restrictions are managed in admin-ui)",
		Long: `Parental restrictions are managed in admin-ui (userdata-local policy,
ADR-0030/ADR-0031) at ` + parentalAdminUIPath + `.

'show' reads the legacy, non-enforced parental.json; 'set' has been removed.`,
	}
	cmd.AddCommand(show, set)
	return cmd
}
