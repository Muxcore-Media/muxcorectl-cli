package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	discoveryv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/discovery/v1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

const (
	capSettings  = "settings"
	methodGet    = "Settings"
	methodUpdate = "UpdateSetting"
)

type settingDefJSON struct { //nolint:govet // fieldalignment: JSON keys match admin-ui settings schema
	Key         string   `json:"Key"`
	Label       string   `json:"Label"`
	Type        string   `json:"Type"`
	Default     string   `json:"Default"`
	Value       string   `json:"Value"`
	Description string   `json:"Description"`
	Required    bool     `json:"Required"`
	Options     []string `json:"Options"`
	Group       string   `json:"Group"`
}

type updateSettingReq struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func newSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Short:   "View and change module settings (admin-ui /settings parity)",
		GroupID: groupSystem,
	}
	cmd.AddCommand(newSettingsListCmd())
	cmd.AddCommand(newSettingsGetCmd())
	cmd.AddCommand(newSettingsSetCmd())
	return cmd
}

func newSettingsListCmd() *cobra.Command {
	var moduleFilter string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all settings from modules with the settings capability",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCore(func(ctx context.Context, c *client.Client) error {
				mods, err := settingsModules(ctx, c, moduleFilter)
				if err != nil {
					return err
				}
				type row struct {
					Module string `json:"module"`
					Group  string `json:"group"`
					Key    string `json:"key"`
					Label  string `json:"label"`
					Value  string `json:"value"`
					Type   string `json:"type"`
				}
				var out []row
				var skipped []string
				for _, mod := range mods {
					defs, err := fetchSettings(ctx, c, mod)
					if err != nil {
						skipped = append(skipped, mod.GetId()+": "+err.Error())
						continue
					}
					for _, d := range defs {
						out = append(out, row{
							Module: mod.GetId(),
							Group:  d.Group,
							Key:    d.Key,
							Label:  d.Label,
							Value:  d.Value,
							Type:   d.Type,
						})
					}
				}
				if len(out) == 0 && len(skipped) > 0 {
					return fmt.Errorf("no settings loaded (%s)", strings.Join(skipped, "; "))
				}
				if len(skipped) > 0 && !flagQuiet {
					for _, s := range skipped {
						fmt.Fprintf(os.Stderr, "warning: skipped %s\n", s)
					}
				}
				if flagJSON {
					return printJSON(out)
				}
				rows := make([][]string, 0, len(out))
				for _, r := range out {
					rows = append(rows, []string{r.Module, r.Group, r.Key, r.Value, r.Type})
				}
				return printTable([]string{"MODULE", "GROUP", "KEY", "VALUE", "TYPE"}, rows)
			})
		},
	}
	cmd.Flags().StringVar(&moduleFilter, "module", "", "filter to one module id")
	return cmd
}

func newSettingsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <module> <key>",
		Short: "Show one setting value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleID, key := args[0], args[1]
			return withCore(func(ctx context.Context, c *client.Client) error {
				mod, err := findModuleByID(ctx, c, moduleID)
				if err != nil {
					return err
				}
				defs, err := fetchSettings(ctx, c, mod)
				if err != nil {
					return err
				}
				for _, d := range defs {
					if d.Key == key {
						if flagJSON {
							return printJSON(d)
						}
						fmt.Printf("%s=%s\n", d.Key, d.Value)
						return nil
					}
				}
				return fmt.Errorf("setting %q not found on module %q", key, moduleID)
			})
		},
	}
	return cmd
}

func newSettingsSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <module> <key> <value>",
		Short: "Update a module setting",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleID, key, value := args[0], args[1], args[2]
			return withCore(func(ctx context.Context, c *client.Client) error {
				mod, err := findModuleByID(ctx, c, moduleID)
				if err != nil {
					return err
				}
				payload, err := json.Marshal(updateSettingReq{Key: key, Value: value})
				if err != nil {
					return err
				}
				_, err = meshCall(ctx, c, mod.GetId(), normalizeDialAddr(mod.GetId(), mod.GetHttpAddr()), methodUpdate, payload)
				if err != nil {
					return fmt.Errorf("settings set: %w", err)
				}
				if flagJSON {
					return printJSON(map[string]string{"module": moduleID, "key": key, "value": value})
				}
				if !flagQuiet {
					fmt.Printf("updated %s.%s=%s\n", moduleID, key, value)
				}
				return nil
			})
		},
	}
	return cmd
}

func settingsModules(ctx context.Context, c *client.Client, filter string) ([]*discoveryv1.ModuleInfoProto, error) {
	mods, err := c.Discovery.FindByCapability(ctx, capSettings)
	if err != nil {
		return nil, fmt.Errorf("settings: %w", err)
	}
	out := make([]*discoveryv1.ModuleInfoProto, 0, len(mods))
	for _, mod := range mods {
		if mod.GetId() == "" {
			continue
		}
		if filter != "" && mod.GetId() != filter {
			continue
		}
		out = append(out, mod)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].GetId() < out[j].GetId()
	})
	if len(out) == 0 {
		if filter != "" {
			return nil, fmt.Errorf("no settings module %q", filter)
		}
		return nil, fmt.Errorf("no modules advertise capability %q", capSettings)
	}
	return out, nil
}

func fetchSettings(ctx context.Context, c *client.Client, mod *discoveryv1.ModuleInfoProto) ([]settingDefJSON, error) {
	raw, err := meshCall(ctx, c, mod.GetId(), normalizeDialAddr(mod.GetId(), mod.GetHttpAddr()), methodGet, nil)
	if err != nil {
		return nil, fmt.Errorf("settings list %s: %w", mod.GetId(), err)
	}
	var defs []settingDefJSON
	if err := json.Unmarshal(raw, &defs); err != nil {
		return nil, fmt.Errorf("settings decode %s: %w", mod.GetId(), err)
	}
	return defs, nil
}
