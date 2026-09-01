package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestApproveDenyHTTP(t *testing.T) {
	flagToken = "adm"
	t.Cleanup(func() { flagToken = "" })

	for _, action := range []string{"approve", "deny"} {
		action := action
		t.Run(action, func(t *testing.T) {
			var gotAuth, gotUser, gotRoles string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				gotUser = r.Header.Get("X-MuxCore-User")
				gotRoles = r.Header.Get("X-MuxCore-Roles")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}))
			t.Cleanup(srv.Close)

			payload, _ := json.Marshal(map[string]string{"approvedBy": "ops", "by": "ops"})
			headers := withBearerAuth(map[string]string{
				"X-MuxCore-User":  "ops",
				"X-MuxCore-Roles": "admin",
			})
			_, err := httpDo(http.MethodPost, srv.URL+"/api/requests/1/"+action, payload, headers)
			if err != nil {
				t.Fatal(err)
			}
			if gotAuth != "Bearer adm" {
				t.Fatalf("Authorization=%q", gotAuth)
			}
			if gotUser != "ops" || gotRoles != "admin" {
				t.Fatalf("user=%q roles=%q", gotUser, gotRoles)
			}
		})
	}
}
