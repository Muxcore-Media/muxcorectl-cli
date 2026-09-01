package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPDoBearerAuth(t *testing.T) {
	const want = "Bearer sekret"
	flagToken = "sekret"
	t.Cleanup(func() { flagToken = "" })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != want {
			t.Fatalf("Authorization=%q want %q", got, want)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	body, err := httpDo(http.MethodGet, srv.URL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "ok") {
		t.Fatalf("body=%s", body)
	}
}

func TestSchedulerDoBearerAuth(t *testing.T) {
	const want = "Bearer sched"
	flagToken = "sched"
	t.Cleanup(func() { flagToken = "" })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != want {
			t.Fatalf("Authorization=%q want %q", got, want)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	if _, err := schedulerDo(http.MethodGet, srv.URL, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAuthHTTPDoBearerAuth(t *testing.T) {
	const want = "Bearer invite"
	flagToken = "invite"
	t.Cleanup(func() { flagToken = "" })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != want {
			t.Fatalf("Authorization=%q want %q", got, want)
		}
		_, _ = w.Write([]byte(`{"invites":[]}`))
	}))
	t.Cleanup(srv.Close)

	if _, err := authHTTPDo(http.MethodGet, srv.URL, nil, nil); err != nil {
		t.Fatal(err)
	}
}
