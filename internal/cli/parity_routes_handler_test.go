package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var handlerRoutePattern = regexp.MustCompile(`mux\.HandleFunc\("((?:GET|POST|DELETE|PUT|PATCH) [^"]+)"`)

// handlerRoutes reads admin-ui/handler/handler.go and returns every registered HTTP route.
func handlerRoutes(t *testing.T) map[string]struct{} {
	t.Helper()
	path := filepath.Join("..", "..", "..", "admin-ui", "handler", "handler.go")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open handler.go: %v", err)
	}
	defer f.Close()

	routes := make(map[string]struct{})
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := handlerRoutePattern.FindStringSubmatch(sc.Text()); len(m) == 2 {
			routes[m[1]] = struct{}{}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan handler.go: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("no routes parsed from handler.go")
	}
	return routes
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
