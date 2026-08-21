package cli

import (
	"context"
	"fmt"

	monitorv1 "github.com/Muxcore-Media/playback-monitor/proto/monitorv1"
	"github.com/spf13/cobra"
)

type streamMapPin struct {
	ID      string  `json:"id"`
	User    string  `json:"user"`
	Title   string  `json:"title"`
	State   string  `json:"state"`
	IP      string  `json:"ip"`
	City    string  `json:"city"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

func newStreamsMapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "map",
		Short: "Geo pins for active streams (admin-ui /streams/map parity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withPlaybackMonitorClient(func(ctx context.Context, cli monitorv1.PlaybackMonitorServiceClient) error {
				resp, err := cli.ListActiveSessions(ctx, &monitorv1.ListActiveSessionsRequest{Limit: 200})
				if err != nil {
					return fmt.Errorf("streams map: %w", err)
				}
				pins := mapPinsFromSessions(resp.GetSessions())
				if flagJSON {
					return printJSON(map[string]any{"pins": pins, "count": len(pins)})
				}
				rows := make([][]string, 0, len(pins))
				for _, p := range pins {
					rows = append(rows, []string{p.User, p.Title, p.City, p.Country, fmt.Sprintf("%.4f,%.4f", p.Lat, p.Lon)})
				}
				fmt.Printf("pins=%d\n", len(pins))
				return printTable([]string{"USER", "TITLE", "CITY", "COUNTRY", "LAT,LON"}, rows)
			})
		},
	}
}

func mapPinsFromSessions(sessions []*monitorv1.SessionRecord) []streamMapPin {
	out := make([]streamMapPin, 0)
	for _, s := range sessions {
		if s == nil {
			continue
		}
		lat, lon := s.GetGeoLat(), s.GetGeoLon()
		if lat == 0 && lon == 0 {
			continue
		}
		if s.GetGeoCountry() == "Local Network" {
			continue
		}
		user := sessionUser(s)
		out = append(out, streamMapPin{
			ID:      s.GetId(),
			User:    user,
			Title:   s.GetTitle(),
			State:   s.GetState().String(),
			IP:      s.GetIpAddress(),
			City:    s.GetGeoCity(),
			Country: s.GetGeoCountry(),
			Lat:     lat,
			Lon:     lon,
		})
	}
	return out
}
