package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ai",
		Short:   "Greenfield AI modules (admin-ui /ai parity)",
		GroupID: groupLibrary,
	}
	cmd.AddCommand(newAIStatusCmd())
	cmd.AddCommand(newAIGenerateSubtitlesCmd())
	cmd.AddCommand(newAISyncSubtitlesCmd())
	cmd.AddCommand(newAISuggestCmd())
	cmd.AddCommand(newAITicketOpenCmd())
	cmd.AddCommand(newAIFilterProfilesCmd())
	cmd.AddCommand(newAILibrarianScanCmd())
	return cmd
}

func newAIStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show which AI capabilities are registered",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCore(func(ctx context.Context, c *client.Client) error {
				caps := []string{"ai.runtime", "ai.subtitles", "ai.recommend", "ai.tickets", "ai.filter", "ai.librarian"}
				out := map[string]any{}
				for _, cap := range caps {
					mod, err := findModuleByCapability(ctx, c, cap)
					if err != nil {
						out[cap] = map[string]any{"present": false, "error": err.Error()}
						continue
					}
					out[cap] = map[string]any{"present": true, "id": mod.GetId(), "addr": mod.GetHttpAddr()}
				}
				return printJSON(out)
			})
		},
	}
}

func withAICall(capability, method string, payload []byte) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		mod, err := findModuleByCapability(ctx, c, capability)
		if err != nil {
			return err
		}
		raw, err := meshCall(ctx, c, mod.GetId(), mod.GetHttpAddr(), method, payload)
		if err != nil {
			return err
		}
		if flagJSON || flagQuiet {
			fmt.Println(string(raw))
			return nil
		}
		fmt.Println(string(raw))
		return nil
	})
}

func newAIGenerateSubtitlesCmd() *cobra.Command {
	var mediaID, lang, hint string
	var duration float64
	cmd := &cobra.Command{
		Use:   "subtitles-generate",
		Short: "Generate subtitles when catalog search failed",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, _ := json.Marshal(map[string]any{
				"media_id": mediaID, "language": lang, "duration_sec": duration, "transcript_hint": hint,
			})
			return withAICall("ai.subtitles", "Generate", payload)
		},
	}
	cmd.Flags().StringVar(&mediaID, "media-id", "", "media id")
	cmd.Flags().StringVar(&lang, "lang", "en", "language")
	cmd.Flags().Float64Var(&duration, "duration", 0, "media duration seconds")
	cmd.Flags().StringVar(&hint, "hint", "", "transcript hint")
	return cmd
}

func newAISyncSubtitlesCmd() *cobra.Command {
	var mediaID, srt string
	var durationMs, speechMs int64
	cmd := &cobra.Command{
		Use:   "subtitles-sync",
		Short: "Align a foreign subtitle file to the media timeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, _ := json.Marshal(map[string]any{
				"media_id": mediaID, "subtitle_srt": srt,
				"media_duration_ms": durationMs, "speech_start_ms": speechMs,
			})
			return withAICall("ai.subtitles", "Sync", payload)
		},
	}
	cmd.Flags().StringVar(&mediaID, "media-id", "", "media id")
	cmd.Flags().StringVar(&srt, "srt", "", "SRT text")
	cmd.Flags().Int64Var(&durationMs, "duration-ms", 0, "media duration")
	cmd.Flags().Int64Var(&speechMs, "speech-start-ms", 0, "detected speech start")
	return cmd
}

func newAISuggestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suggest",
		Short: "Suggest titles to add from a JSON library snapshot on stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAICall("ai.recommend", "Suggest", []byte(`{"library":[]}`))
		},
	}
}

func newAITicketOpenCmd() *cobra.Command {
	var subject, body, title, mediaID string
	cmd := &cobra.Command{
		Use:   "ticket-open",
		Short: "Open a ticket and auto-resolve when possible",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, _ := json.Marshal(map[string]any{
				"subject": subject, "body": body, "title": title, "media_id": mediaID,
			})
			return withAICall("ai.tickets", "OpenTicket", payload)
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "", "subject")
	cmd.Flags().StringVar(&body, "body", "", "body")
	cmd.Flags().StringVar(&title, "title", "", "media title")
	cmd.Flags().StringVar(&mediaID, "media-id", "", "media id")
	_ = cmd.MarkFlagRequired("subject")
	return cmd
}

func newAIFilterProfilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "filter-profiles",
		Short: "List content-filter profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAICall("ai.filter", "ListProfiles", []byte(`{}`))
		},
	}
}

func newAILibrarianScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "librarian-scan",
		Short: "Scan an empty library snapshot (pipe JSON via mesh Suggest-style payload later)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAICall("ai.librarian", "Scan", []byte(`{"library":[]}`))
		},
	}
}
