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
