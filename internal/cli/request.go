package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

func newRequestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "request",
		Short:   "Search and add media requests (admin-ui /request parity)",
		GroupID: groupAutomation,
	}
	cmd.AddCommand(newRequestListCmd())
	cmd.AddCommand(newRequestSearchCmd())
	cmd.AddCommand(newRequestAddCmd())
	cmd.AddCommand(newRequestApproveCmd())
	cmd.AddCommand(newRequestDenyCmd())
	return cmd
}

func newRequestListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List media requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRequestBase(func(base string) error {
				body, err := httpDo(http.MethodGet, base+"/api/requests", nil, nil)
				if err != nil {
					return fmt.Errorf("request list: %w", err)
				}
				var rows []map[string]any
				if err := json.Unmarshal(body, &rows); err != nil {
					return fmt.Errorf("request list decode: %w", err)
				}
				if flagJSON {
					return printJSON(rows)
				}
				out := make([][]string, 0, len(rows))
				for _, r := range rows {
					out = append(out, []string{
						fmt.Sprint(r["id"]),
						fmt.Sprint(r["title"]),
						fmt.Sprint(r["status"]),
						fmt.Sprint(r["itemType"]),
						fmt.Sprint(r["requestedBy"]),
					})
				}
				return printTable([]string{"ID", "TITLE", "STATUS", "TYPE", "REQUESTED_BY"}, out)
			})
		},
	}
}

func newRequestSearchCmd() *cobra.Command {
	var mediaType string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search TMDB for movies or TV shows",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if mediaType == "" {
				mediaType = "movie"
			}
			return withRequestBase(func(base string) error {
				path := base + "/api/search?q=" + url.QueryEscape(args[0]) + "&type=" + url.QueryEscape(mediaType)
				body, err := httpDo(http.MethodGet, path, nil, nil)
				if err != nil {
					return fmt.Errorf("request search: %w", err)
				}
				var payload struct {
					Results []map[string]any `json:"results"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					return fmt.Errorf("request search decode: %w", err)
				}
				if flagJSON {
					return printJSON(payload.Results)
				}
				out := make([][]string, 0, len(payload.Results))
				for _, r := range payload.Results {
					out = append(out, []string{
						fmt.Sprint(r["id"]),
						fmt.Sprint(r["title"]),
						fmt.Sprint(r["year"]),
						fmt.Sprint(r["type"]),
					})
				}
				return printTable([]string{"TMDB_ID", "TITLE", "YEAR", "TYPE"}, out)
			})
		},
	}
	cmd.Flags().StringVar(&mediaType, "type", "movie", "movie or tv")
	return cmd
}

func newRequestAddCmd() *cobra.Command {
	var (
		tmdbID   int
		title    string
		year     int
		itemType string
		overview string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a media request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tmdbID == 0 || title == "" {
				return fmt.Errorf("--tmdb-id and --title are required")
			}
			if itemType == "" {
				itemType = "movie"
			}
			headers, ident, err := requestIdentityHeaders()
			if err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]any{
				"tmdbId": tmdbID, "title": title, "year": year,
				"overview": overview, "type": itemType,
				"requestedBy": ident.Username, "isAdmin": ident.IsAdmin,
			})
			return withRequestBase(func(base string) error {
				body, err := httpDo(http.MethodPost, base+"/api/request", payload, headers)
				if err != nil {
					return fmt.Errorf("request add: %w", err)
				}
				if flagJSON {
					var out any
					_ = json.Unmarshal(body, &out)
					return printJSON(out)
				}
				if !flagQuiet {
					fmt.Println("request created")
				}
				return nil
			})
		},
	}
	cmd.Flags().IntVar(&tmdbID, "tmdb-id", 0, "TMDB id")
	cmd.Flags().StringVar(&title, "title", "", "title")
	cmd.Flags().IntVar(&year, "year", 0, "release year")
	cmd.Flags().StringVar(&itemType, "type", "movie", "movie or tv")
	cmd.Flags().StringVar(&overview, "overview", "", "optional overview text")
	_ = cmd.MarkFlagRequired("tmdb-id")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newRequestApproveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <request-id>",
		Short: "Approve a pending request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return requestDecide(args[0], "approve")
		},
	}
}

func newRequestDenyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deny <request-id>",
		Short: "Deny a pending request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return requestDecide(args[0], "deny")
		},
	}
}

func requestDecide(id, action string) error {
	headers, ident, err := requestIdentityHeaders()
	if err != nil {
		return err
	}
	by := ident.Username
	if by == "" {
		by = "muxcorectl"
	}
	payload, _ := json.Marshal(map[string]string{"approvedBy": by, "by": by})
	return withRequestBase(func(base string) error {
		_, err := httpDo(http.MethodPost, base+"/api/requests/"+url.PathEscape(id)+"/"+action, payload, headers)
		if err != nil {
			return fmt.Errorf("request %s: %w", action, err)
		}
		if !flagQuiet {
			fmt.Println(action + "d")
		}
		return nil
	})
}
