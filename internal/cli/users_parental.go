package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type parentalSettings struct {
	MaxParentalRating string `json:"max_parental_rating"`
	BlockedTags       string `json:"blocked_tags"`
	AllowedTags       string `json:"allowed_tags"`
	AllowUnrated      bool   `json:"allow_unrated"`
}

func parentalPath() string {
	return adminDataFile("ADMIN_UI_PARENTAL_FILE", "parental.json")
}

func loadParentalMap() map[string]parentalSettings {
	raw, err := os.ReadFile(parentalPath())
	if err != nil {
		return map[string]parentalSettings{}
	}
	var m map[string]parentalSettings
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return map[string]parentalSettings{}
	}
	return m
}

func saveParentalMap(m map[string]parentalSettings) error {
	if err := os.MkdirAll(filepath.Dir(parentalPath()), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := parentalPath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, parentalPath())
}

func newUsersParentalCmd() *cobra.Command {
	show := &cobra.Command{
		Use:   "show <user-id>",
		Short: "Show parental control settings for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := loadParentalMap()[args[0]]
			if flagJSON {
				return printJSON(map[string]any{"user_id": args[0], "path": parentalPath(), "settings": p})
			}
			fmt.Printf("file:         %s\n", parentalPath())
			fmt.Printf("max_rating:   %s\n", p.MaxParentalRating)
			fmt.Printf("blocked_tags: %s\n", p.BlockedTags)
			fmt.Printf("allowed_tags: %s\n", p.AllowedTags)
			fmt.Printf("allow_unrated: %t\n", p.AllowUnrated)
			return nil
		},
	}

	set := &cobra.Command{
		Use:   "set <user-id>",
		Short: "Update parental control settings for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			maxRating, _ := cmd.Flags().GetString("max-rating")
			blockedTags, _ := cmd.Flags().GetString("blocked-tags")
			allowedTags, _ := cmd.Flags().GetString("allowed-tags")
			allowUnrated, _ := cmd.Flags().GetBool("allow-unrated")
			m := loadParentalMap()
			m[args[0]] = parentalSettings{
				MaxParentalRating: strings.TrimSpace(maxRating),
				BlockedTags:       strings.TrimSpace(blockedTags),
				AllowedTags:       strings.TrimSpace(allowedTags),
				AllowUnrated:      allowUnrated,
			}
			if err := saveParentalMap(m); err != nil {
				return fmt.Errorf("users parental set: %w", err)
			}
			if !flagQuiet {
				fmt.Println("saved")
			}
			return nil
		},
	}
	set.Flags().String("max-rating", "", "maximum parental rating")
	set.Flags().String("blocked-tags", "", "comma-separated blocked tags")
	set.Flags().String("allowed-tags", "", "comma-separated allowed tags")
	set.Flags().Bool("allow-unrated", false, "allow unrated content")

	cmd := &cobra.Command{
		Use:   "parental",
		Short: "Per-user parental controls (admin-ui /users/{id}/parental parity)",
	}
	cmd.AddCommand(show, set)
	return cmd
}
