package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	discoveryv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/discovery/v1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newPlaybackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "playback",
		Short:   "Playback admin settings (admin-ui /playback parity)",
		GroupID: groupPlayback,
	}
	cmd.AddCommand(newPlaybackShowCmd())
	cmd.AddCommand(newPlaybackSetCmd())
	return cmd
}

type playbackFile struct { //nolint:govet // fieldalignment: JSON field order matches playback settings file
	EnableResume     bool   `json:"enable_resume"`
	EnableTranscode  bool   `json:"enable_transcode"`
	PreferDirectPlay bool   `json:"prefer_direct_play"`
	TrickplayEnabled bool   `json:"trickplay_enabled"`
	MaxBitrateMbps   string `json:"max_bitrate_mbps"`
	FFmpegBin        string `json:"ffmpeg_bin,omitempty"`
}

func playbackPath() string {
	return envOr("ADMIN_UI_PLAYBACK_FILE", filepath.Join(os.TempDir(), "muxcore-admin-playback.json"))
}

func loadPlaybackFile() playbackFile {
	raw, err := os.ReadFile(playbackPath())
	if err != nil {
		return playbackFile{EnableResume: true, PreferDirectPlay: true, MaxBitrateMbps: "80", FFmpegBin: "ffmpeg"}
	}
	var p playbackFile
	if json.Unmarshal(raw, &p) != nil {
		return playbackFile{EnableResume: true, PreferDirectPlay: true, MaxBitrateMbps: "80", FFmpegBin: "ffmpeg"}
	}
	if p.FFmpegBin == "" {
		p.FFmpegBin = "ffmpeg"
	}
	return p
}

func savePlaybackFile(p playbackFile) error {
	if err := os.MkdirAll(filepath.Dir(playbackPath()), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := playbackPath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, playbackPath())
}

func newPlaybackShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show playback settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := loadPlaybackFile()
			if flagJSON {
				return printJSON(map[string]any{"path": playbackPath(), "settings": p})
			}
			fmt.Printf("file:               %s\n", playbackPath())
			fmt.Printf("enable_resume:      %t\n", p.EnableResume)
			fmt.Printf("enable_transcode:   %t\n", p.EnableTranscode)
			fmt.Printf("prefer_direct_play: %t\n", p.PreferDirectPlay)
			fmt.Printf("max_bitrate_mbps:   %s\n", p.MaxBitrateMbps)
			fmt.Printf("ffmpeg_bin:         %s\n", p.FFmpegBin)
			return nil
		},
	}
}

func newPlaybackSetCmd() *cobra.Command {
	var (
		resume, transcode, direct, trickplay *bool
		maxBitrate, ffmpeg                   string
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update playback settings file",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := loadPlaybackFile()
			if resume != nil {
				p.EnableResume = *resume
			}
			if transcode != nil {
				p.EnableTranscode = *transcode
			}
			if direct != nil {
				p.PreferDirectPlay = *direct
			}
			if trickplay != nil {
				p.TrickplayEnabled = *trickplay
			}
			if maxBitrate != "" {
				p.MaxBitrateMbps = maxBitrate
			}
			if ffmpeg != "" {
				p.FFmpegBin = ffmpeg
			}
			if err := savePlaybackFile(p); err != nil {
				return fmt.Errorf("playback set: %w", err)
			}
			if !flagQuiet {
				fmt.Println("saved")
			}
			return nil
		},
	}
	var resumeVal, transcodeVal, directVal, trickplayVal bool
	cmd.Flags().BoolVar(&resumeVal, "resume", true, "enable resume")
	cmd.Flags().Lookup("resume").NoOptDefVal = "true"
	cmd.Flags().BoolVar(&transcodeVal, "transcode", false, "enable transcode")
	cmd.Flags().BoolVar(&directVal, "direct-play", true, "prefer direct play")
	cmd.Flags().Lookup("direct-play").NoOptDefVal = "true"
	cmd.Flags().BoolVar(&trickplayVal, "trickplay", false, "enable trickplay")
	cmd.Flags().StringVar(&maxBitrate, "max-bitrate", "", "max bitrate mbps")
	cmd.Flags().StringVar(&ffmpeg, "ffmpeg", "", "ffmpeg binary path")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if f := cmd.Flags().Lookup("resume"); f != nil && f.Changed {
			resume = &resumeVal
		}
		if f := cmd.Flags().Lookup("transcode"); f != nil && f.Changed {
			transcode = &transcodeVal
		}
		if f := cmd.Flags().Lookup("direct-play"); f != nil && f.Changed {
			direct = &directVal
		}
		if f := cmd.Flags().Lookup("trickplay"); f != nil && f.Changed {
			trickplay = &trickplayVal
		}
		return nil
	}
	return cmd
}

func newLiveTVCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "livetv",
		Short:   "Live TV channel list (admin-ui /livetv parity)",
		GroupID: groupPlayback,
	}
	cmd.AddCommand(newLiveTVShowCmd())
	cmd.AddCommand(newLiveTVSetCmd())
	return cmd
}

type liveTVChannel struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type liveTVFile struct {
	Channels []liveTVChannel `json:"channels"`
}

func livetvPath() string {
	return envOr("ADMIN_UI_LIVETV_FILE", filepath.Join(os.TempDir(), "muxcore-admin-livetv.json"))
}

func loadLiveTVFile() liveTVFile {
	raw, err := os.ReadFile(livetvPath())
	if err != nil {
		return liveTVFile{}
	}
	var f liveTVFile
	if json.Unmarshal(raw, &f) != nil {
		return liveTVFile{}
	}
	return f
}

func saveLiveTVFile(f liveTVFile) error {
	if err := os.MkdirAll(filepath.Dir(livetvPath()), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := livetvPath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, livetvPath())
}

func newLiveTVShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show configured Live TV channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := loadLiveTVFile()
			if flagJSON {
				return printJSON(map[string]any{"path": livetvPath(), "channels": f.Channels})
			}
			fmt.Printf("file: %s (%d channels)\n", livetvPath(), len(f.Channels))
			rows := make([][]string, 0, len(f.Channels))
			for _, ch := range f.Channels {
				rows = append(rows, []string{ch.Name, ch.URL})
			}
			return printTable([]string{"NAME", "URL"}, rows)
		},
	}
}

func newLiveTVSetCmd() *cobra.Command {
	var channelsFile string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Replace channels from a JSON file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if channelsFile == "" {
				return fmt.Errorf("--file is required")
			}
			raw, err := os.ReadFile(channelsFile) //nolint:gosec // operator-selected playback channels JSON path
			if err != nil {
				return fmt.Errorf("read channels: %w", err)
			}
			var f liveTVFile
			if err := json.Unmarshal(raw, &f); err != nil {
				return fmt.Errorf("parse channels: %w", err)
			}
			if err := saveLiveTVFile(f); err != nil {
				return fmt.Errorf("livetv set: %w", err)
			}
			if !flagQuiet {
				fmt.Printf("saved %d channels\n", len(f.Channels))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&channelsFile, "file", "", "JSON file with {\"channels\":[{\"name\":\"...\",\"url\":\"...\"}]}")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newPluginsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "plugins",
		Short:   "Registered modules catalog (admin-ui /plugins parity)",
		GroupID: groupSystem,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCore(func(ctx context.Context, c *client.Client) error {
				resp, err := c.Discovery.Raw().ListAll(ctx, &discoveryv1.ListAllRequest{})
				if err != nil {
					return fmt.Errorf("plugins: %w", err)
				}
				if flagJSON {
					out := make([]map[string]any, 0, len(resp.GetEntries()))
					for _, e := range resp.GetEntries() {
						if mi := e.GetInfo(); mi != nil {
							out = append(out, map[string]any{
								"id": mi.GetId(), "name": mi.GetName(), "capabilities": mi.GetCapabilities(),
							})
						}
					}
					return printJSON(out)
				}
				rows := make([][]string, 0, len(resp.GetEntries()))
				for _, e := range resp.GetEntries() {
					mi := e.GetInfo()
					if mi == nil {
						continue
					}
					rows = append(rows, []string{mi.GetId(), mi.GetName(), strings.Join(mi.GetCapabilities(), ",")})
				}
				return printTable([]string{"ID", "NAME", "CAPABILITIES"}, rows)
			})
		},
	}
}

func newAuthCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "auth",
		Short:   "Auth module overview (admin-ui auth/SSO parity)",
		GroupID: groupAccess,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withCore(func(ctx context.Context, c *client.Client) error {
				resp, err := c.Discovery.Raw().ListAll(ctx, &discoveryv1.ListAllRequest{})
				if err != nil {
					return fmt.Errorf("auth: %w", err)
				}
				type row struct {
					ID           string   `json:"id"`
					Name         string   `json:"name"`
					Capabilities []string `json:"capabilities"`
				}
				var out []row
				for _, e := range resp.GetEntries() {
					m := e.GetInfo()
					if m == nil {
						continue
					}
					id := strings.ToLower(m.GetId())
					if strings.HasPrefix(id, "auth") || hasAuthCap(m.GetCapabilities()) {
						out = append(out, row{ID: m.GetId(), Name: m.GetName(), Capabilities: m.GetCapabilities()})
					}
				}
				if flagJSON {
					return printJSON(out)
				}
				rows := make([][]string, 0, len(out))
				for _, r := range out {
					rows = append(rows, []string{r.ID, r.Name, strings.Join(r.Capabilities, ",")})
				}
				return printTable([]string{"ID", "NAME", "CAPABILITIES"}, rows)
			})
		},
	}
}

func hasAuthCap(caps []string) bool {
	for _, cap := range caps {
		if cap == "auth" || cap == "authorizer" || strings.Contains(strings.ToLower(cap), "auth") {
			return true
		}
	}
	return false
}

func newBrandingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "branding",
		Short:   "Admin UI branding settings (admin-ui /branding parity)",
		GroupID: groupSystem,
	}
	cmd.AddCommand(newBrandingShowCmd())
	cmd.AddCommand(newBrandingSetCmd())
	return cmd
}

type brandingFile struct {
	ServerName  string `json:"server_name"`
	LoginBanner string `json:"login_banner"`
	CustomCSS   string `json:"custom_css"`
	SplashURL   string `json:"splash_url"`
}

func brandingPath() string {
	return envOr("ADMIN_UI_BRANDING_FILE", filepath.Join(os.TempDir(), "muxcore-admin-branding.json"))
}

func loadBrandingFile() brandingFile {
	raw, err := os.ReadFile(brandingPath())
	if err != nil {
		return brandingFile{ServerName: "MuxCore"}
	}
	var b brandingFile
	if json.Unmarshal(raw, &b) != nil {
		return brandingFile{ServerName: "MuxCore"}
	}
	if b.ServerName == "" {
		b.ServerName = "MuxCore"
	}
	return b
}

func saveBrandingFile(b brandingFile) error {
	if err := os.MkdirAll(filepath.Dir(brandingPath()), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	tmp := brandingPath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, brandingPath())
}

func newBrandingShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show branding settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := loadBrandingFile()
			if flagJSON {
				return printJSON(map[string]any{"path": brandingPath(), "settings": b})
			}
			fmt.Printf("file:         %s\n", brandingPath())
			fmt.Printf("server_name:  %s\n", b.ServerName)
			fmt.Printf("login_banner: %s\n", b.LoginBanner)
			fmt.Printf("splash_url:   %s\n", b.SplashURL)
			return nil
		},
	}
}

func newBrandingSetCmd() *cobra.Command {
	var serverName, loginBanner, splashURL string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update branding settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := loadBrandingFile()
			if serverName != "" {
				b.ServerName = serverName
			}
			if loginBanner != "" {
				b.LoginBanner = loginBanner
			}
			if splashURL != "" {
				b.SplashURL = splashURL
			}
			if err := saveBrandingFile(b); err != nil {
				return fmt.Errorf("branding set: %w", err)
			}
			if !flagQuiet {
				fmt.Println("saved")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&serverName, "server-name", "", "display server name")
	cmd.Flags().StringVar(&loginBanner, "login-banner", "", "login page banner text")
	cmd.Flags().StringVar(&splashURL, "splash-url", "", "splash image URL")
	return cmd
}

func newNetworkingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "networking",
		Short:   "Published URLs and proxy settings (admin-ui /networking parity)",
		GroupID: groupSystem,
	}
	cmd.AddCommand(newNetworkingShowCmd())
	cmd.AddCommand(newNetworkingSetCmd())
	return cmd
}

type networkingFile struct {
	PublicURL      string `json:"public_url"`
	TrustedProxies string `json:"trusted_proxies"`
	PublishedHosts string `json:"published_hosts"`
	HTTPPort       string `json:"http_port"`
	HTTPSPort      string `json:"https_port"`
}

func networkingPath() string {
	return envOr("ADMIN_UI_NETWORKING_FILE", filepath.Join(os.TempDir(), "muxcore-admin-networking.json"))
}

func loadNetworkingFile() networkingFile {
	raw, err := os.ReadFile(networkingPath())
	if err != nil {
		return networkingFile{HTTPPort: "80", HTTPSPort: "443"}
	}
	var n networkingFile
	if json.Unmarshal(raw, &n) != nil {
		return networkingFile{HTTPPort: "80", HTTPSPort: "443"}
	}
	return n
}

func saveNetworkingFile(n networkingFile) error {
	if err := os.MkdirAll(filepath.Dir(networkingPath()), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return err
	}
	tmp := networkingPath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, networkingPath())
}

func newNetworkingShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show networking settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			n := loadNetworkingFile()
			if flagJSON {
				return printJSON(map[string]any{"path": networkingPath(), "settings": n})
			}
			fmt.Printf("file:            %s\n", networkingPath())
			fmt.Printf("public_url:      %s\n", n.PublicURL)
			fmt.Printf("published_hosts: %s\n", n.PublishedHosts)
			fmt.Printf("trusted_proxies: %s\n", n.TrustedProxies)
			fmt.Printf("http_port:       %s\n", n.HTTPPort)
			fmt.Printf("https_port:      %s\n", n.HTTPSPort)
			return nil
		},
	}
}

func newNetworkingSetCmd() *cobra.Command {
	var publicURL, publishedHosts, trustedProxies, httpPort, httpsPort string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update networking settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			n := loadNetworkingFile()
			if publicURL != "" {
				n.PublicURL = publicURL
			}
			if publishedHosts != "" {
				n.PublishedHosts = publishedHosts
			}
			if trustedProxies != "" {
				n.TrustedProxies = trustedProxies
			}
			if httpPort != "" {
				n.HTTPPort = httpPort
			}
			if httpsPort != "" {
				n.HTTPSPort = httpsPort
			}
			if err := saveNetworkingFile(n); err != nil {
				return fmt.Errorf("networking set: %w", err)
			}
			if !flagQuiet {
				fmt.Println("saved")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&publicURL, "public-url", "", "public base URL")
	cmd.Flags().StringVar(&publishedHosts, "published-hosts", "", "comma-separated published hosts")
	cmd.Flags().StringVar(&trustedProxies, "trusted-proxies", "", "trusted proxy CIDRs")
	cmd.Flags().StringVar(&httpPort, "http-port", "", "HTTP port")
	cmd.Flags().StringVar(&httpsPort, "https-port", "", "HTTPS port")
	return cmd
}
