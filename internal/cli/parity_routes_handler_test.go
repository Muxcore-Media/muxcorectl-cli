package cli

import (
	"bufio"
	_ "embed"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

//go:embed testdata/admin_ui_handler_routes.txt
var embeddedHandlerRoutes string

var handlerRoutePattern = regexp.MustCompile(`mux\.HandleFunc\("((?:GET|POST|DELETE|PUT|PATCH) [^"]+)"`)

func handlerGoCandidates() []string {
	return []string{
		filepath.Join("..", "..", "admin-ui", "handler", "handler.go"),
		filepath.Join("..", "..", "..", "admin-ui", "handler", "handler.go"),
	}
}

func routesFromLines(t *testing.T, lines []string) map[string]struct{} {
	t.Helper()
	routes := make(map[string]struct{})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := handlerRoutePattern.FindStringSubmatch(line); len(m) == 2 {
			routes[m[1]] = struct{}{}
			continue
		}
		routes[line] = struct{}{}
	}
	if len(routes) == 0 {
		t.Fatal("no routes parsed")
	}
	return routes
}

// handlerRoutes reads admin-ui/handler/handler.go when present in the umbrella
// checkout, otherwise falls back to the embedded route snapshot for standalone CI.
func handlerRoutes(t *testing.T) map[string]struct{} {
	t.Helper()
	for _, path := range handlerGoCandidates() {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		routes := make(map[string]struct{})
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if m := handlerRoutePattern.FindStringSubmatch(sc.Text()); len(m) == 2 {
				routes[m[1]] = struct{}{}
			}
		}
		_ = f.Close()
		if err := sc.Err(); err != nil {
			t.Fatalf("scan handler.go: %v", err)
		}
		if len(routes) > 0 {
			return routes
		}
	}
	return routesFromLines(t, strings.Split(embeddedHandlerRoutes, "\n"))
}

func TestAdminUIRoutesMatchHandler(t *testing.T) {
	handler := handlerRoutes(t)
	for route := range handler {
		if _, ok := adminUIRouteCoverage[route]; !ok {
			t.Errorf("admin-ui route %q missing from adminUIRouteCoverage", route)
		}
	}
	for route := range adminUIRouteCoverage {
		if _, ok := handler[route]; !ok {
			t.Errorf("adminUIRouteCoverage has stale route %q not in handler.go", route)
		}
	}
}

func TestAdminUIRouteUnsupportedDocumented(t *testing.T) {
	// Browser-only or admin-ui in-memory flows must stay explicitly empty.
	for _, route := range []string{
		"GET /login",
		"POST /login",
		"GET /logout",
		"GET /api/auth/passkey/register/{id}/begin",
		"POST /api/auth/passkey/register/{id}/complete",
		"GET /auth/callback",
		"GET /auth/status",
		"GET /branding.css",
		"POST /devices/{token}/revoke",
	} {
		if adminUIRouteCoverage[route] != "" {
			t.Errorf("route %q should remain intentionally unsupported (empty mapping)", route)
		}
	}
}
