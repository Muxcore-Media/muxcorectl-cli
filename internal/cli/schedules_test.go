package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeHTTPBase(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{":9200", "http://127.0.0.1:9200"},
		{"0.0.0.0:9200", "http://127.0.0.1:9200"},
		{"127.0.0.1:9200", "http://127.0.0.1:9200"},
		{"http://127.0.0.1:9200/", "http://127.0.0.1:9200"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := normalizeHTTPBase(tc.in); got != tc.want {
			t.Fatalf("normalizeHTTPBase(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSchedulerDoList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/list" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "t1", "name": "n1", "cron_expr": "@hourly", "status": "armed", "once": false},
		})
	}))
	t.Cleanup(srv.Close)

	body, err := schedulerDo(http.MethodGet, srv.URL+"/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"id":"t1"`) && !strings.Contains(string(body), `"id": "t1"`) {
		t.Fatalf("body=%s", body)
	}
}

func TestSchedulerDoAddCancel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/schedule", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"task_id": "abc"})
	})
	mux.HandleFunc("/cancel/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	body, err := schedulerDo(http.MethodPost, srv.URL+"/schedule", []byte(`{"name":"x","cron_expr":"@hourly"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "abc") {
		t.Fatalf("add body=%s", body)
	}
	body, err = schedulerDo(http.MethodDelete, srv.URL+"/cancel/abc", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "cancelled") {
		t.Fatalf("cancel body=%s", body)
	}
}

func TestSchedulerTokenHeader(t *testing.T) {
	var got string
	var status = http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		w.WriteHeader(status)
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MUXCORE_TOKEN", "")
	t.Setenv("MUXCORE_TOKEN_FILE", "")
	t.Setenv("MUXCORE_ADMIN_TOKEN", "")
	flagToken = ""
	t.Setenv("MUXCORE_SCHEDULER_TOKEN", "")
	t.Setenv("SCHEDULER_HTTP_TOKEN", "")
	flagSchedulerToken = ""
	t.Cleanup(func() { flagSchedulerToken = "" })

	if _, err := schedulerDo(http.MethodGet, srv.URL+"/list", nil); err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected no Authorization, got %q", got)
	}

	t.Setenv("SCHEDULER_HTTP_TOKEN", "fallback")
	_, _ = schedulerDo(http.MethodGet, srv.URL+"/list", nil)
	if got != "Bearer fallback" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("MUXCORE_SCHEDULER_TOKEN", "primary")
	_, _ = schedulerDo(http.MethodGet, srv.URL+"/list", nil)
	if got != "Bearer primary" {
		t.Fatalf("got %q", got)
	}
	flagSchedulerToken = "flagtok"
	_, _ = schedulerDo(http.MethodGet, srv.URL+"/list", nil)
	if got != "Bearer flagtok" {
		t.Fatalf("got %q", got)
	}

	status = http.StatusUnauthorized
	_, err := schedulerDo(http.MethodGet, srv.URL+"/list", nil)
	if err == nil || !strings.Contains(err.Error(), "--scheduler-token") || !strings.Contains(err.Error(), "MUXCORE_SCHEDULER_TOKEN") {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(err.Error(), "flagtok") {
		t.Fatal("token leaked in error")
	}
}
