package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	authv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// erasureFakeAuth records what the CLI sends. ListUserErasures and
// AckUserErasure are module-certificate-only methods; the CLI must never call
// them.
type erasureFakeAuth struct {
	authv1.UnimplementedAuthServiceServer

	mu          sync.Mutex
	statusReqs  []*authv1.GetUserErasureStatusRequest
	statusToken []string // x-auth-token per status call
	deleteToken []string // x-auth-token per delete call
	listCalls   int
	ackCalls    int

	pages     map[string]*authv1.GetUserErasureStatusResponse // keyed by page token
	byID      map[string]*authv1.ErasureStatus
	deleteID  string
	deleteErr string
}

func incomingToken(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get("x-auth-token"); len(v) > 0 {
		return v[0]
	}
	return "<none>"
}

func (f *erasureFakeAuth) GetUserErasureStatus(ctx context.Context, req *authv1.GetUserErasureStatusRequest) (*authv1.GetUserErasureStatusResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statusReqs = append(f.statusReqs, req)
	f.statusToken = append(f.statusToken, incomingToken(ctx))
	if id := req.GetErasureId(); id != "" {
		s, ok := f.byID[id]
		if !ok {
			return nil, status.Error(codes.NotFound, "unknown erasure")
		}
		return &authv1.GetUserErasureStatusResponse{Erasures: []*authv1.ErasureStatus{s}}, nil
	}
	if r, ok := f.pages[req.GetPageToken()]; ok {
		return r, nil
	}
	return &authv1.GetUserErasureStatusResponse{}, nil
}

func (f *erasureFakeAuth) ListUserErasures(context.Context, *authv1.ListUserErasuresRequest) (*authv1.ListUserErasuresResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	return nil, status.Error(codes.PermissionDenied, "module certificate required")
}

func (f *erasureFakeAuth) AckUserErasure(context.Context, *authv1.AckUserErasureRequest) (*authv1.AckUserErasureResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ackCalls++
	return nil, status.Error(codes.PermissionDenied, "module certificate required")
}

func (f *erasureFakeAuth) DeleteUser(ctx context.Context, _ *authv1.DeleteUserRequest) (*authv1.DeleteUserResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteToken = append(f.deleteToken, incomingToken(ctx))
	return &authv1.DeleteUserResponse{ErasureId: f.deleteID, Error: f.deleteErr}, nil
}

func startErasureFake(t *testing.T, fake *erasureFakeAuth) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	old := withAuthDial
	withAuthDial = func(fn func(ctx context.Context, auth authv1.AuthServiceClient) error) error {
		return fn(context.Background(), authv1.NewAuthServiceClient(conn))
	}
	t.Cleanup(func() {
		withAuthDial = old
		srv.Stop()
		_ = conn.Close()
	})
}

func setAdminToken(t *testing.T, token string) {
	t.Helper()
	t.Setenv("MUXCORE_TOKEN", token)
	t.Setenv("MUXCORE_ADMIN_TOKEN", "")
	t.Setenv("MUXCORE_TOKEN_FILE", "")
}

// runUsers executes `users <args...>` through the root command and captures
// stdout.
func runUsers(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	root := NewRoot()
	root.SetArgs(append([]string{"users"}, args...))
	var errBuf bytes.Buffer
	root.SetErr(&errBuf)
	runErr := root.Execute()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out), runErr
}

func okModule(id string, required bool) *authv1.ErasureModuleStatus {
	return &authv1.ErasureModuleStatus{
		ModuleId: id, Outcome: authv1.ErasureOutcome_ERASURE_OUTCOME_OK,
		Required: required, AckedAt: "2026-10-10T00:00:00Z",
	}
}

func pendingModule(id string) *authv1.ErasureModuleStatus {
	return &authv1.ErasureModuleStatus{ModuleId: id, Required: true}
}

func completeErasure() *authv1.ErasureStatus {
	return &authv1.ErasureStatus{
		ErasureId: "er-done", DeletedAt: "2026-10-09T10:00:00Z", Complete: true,
		Modules: []*authv1.ErasureModuleStatus{okModule("userdata-local", true), okModule("request-media", true)},
	}
}

func pendingErasure() *authv1.ErasureStatus {
	return &authv1.ErasureStatus{
		ErasureId: "er-wait", DeletedAt: "2026-10-09T11:00:00Z", Complete: false,
		Modules: []*authv1.ErasureModuleStatus{
			okModule("userdata-local", true),
			pendingModule("request-media"),
			{ModuleId: "playback-monitor", Outcome: authv1.ErasureOutcome_ERASURE_OUTCOME_FAILED, Required: true, DetailCode: "store_locked"},
			{ModuleId: "auth-oidc", Outcome: authv1.ErasureOutcome_ERASURE_OUTCOME_UNSUPPORTED},
		},
	}
}

func assertNoModuleOnlyCalls(t *testing.T, f *erasureFakeAuth) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listCalls != 0 || f.ackCalls != 0 {
		t.Fatalf("ListUserErasures calls=%d AckUserErasure calls=%d, want 0/0", f.listCalls, f.ackCalls)
	}
}

func TestUsersErasuresDefaultsToPendingAndSendsAdminBearer(t *testing.T) {
	fake := &erasureFakeAuth{pages: map[string]*authv1.GetUserErasureStatusResponse{
		"": {Erasures: []*authv1.ErasureStatus{pendingErasure()}},
	}}
	startErasureFake(t, fake)
	setAdminToken(t, "admin-session-token")

	out, err := runUsers(t, "erasures")
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.statusReqs) != 1 {
		t.Fatalf("status calls=%d", len(fake.statusReqs))
	}
	req := fake.statusReqs[0]
	if !req.GetPendingOnly() || req.GetErasureId() != "" {
		t.Fatalf("request=%v, want pending_only and no erasure_id", req)
	}
	if fake.statusToken[0] != "admin-session-token" {
		t.Fatalf("x-auth-token=%q", fake.statusToken[0])
	}
	assertNoModuleOnlyCalls(t, fake)

	for _, want := range []string{
		"ERASURE_ID", "er-wait", "no",
		"userdata-local", "ok",
		"request-media", "pending",
		"playback-monitor", "failed",
		"auth-oidc", "unsupported",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n")[1:] {
		if f := strings.Fields(line); f[2] != "no" {
			t.Fatalf("pending erasure rendered complete: %q", line)
		}
	}
}

func TestUsersErasuresAllFlagListsEverything(t *testing.T) {
	fake := &erasureFakeAuth{pages: map[string]*authv1.GetUserErasureStatusResponse{
		"": {Erasures: []*authv1.ErasureStatus{completeErasure(), pendingErasure()}},
	}}
	startErasureFake(t, fake)
	setAdminToken(t, "tok")

	out, err := runUsers(t, "erasures", "--all")
	if err != nil {
		t.Fatal(err)
	}
	if fake.statusReqs[0].GetPendingOnly() {
		t.Fatalf("--all must not set pending_only: %v", fake.statusReqs[0])
	}
	assertNoModuleOnlyCalls(t, fake)

	rows := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n")[1:] {
		f := strings.Fields(line)
		rows[f[0]+"/"+f[3]] = f
	}
	if got := rows["er-done/userdata-local"]; len(got) != 6 || got[2] != "yes" || got[4] != "ok" {
		t.Fatalf("complete row=%v\n%s", got, out)
	}
	if got := rows["er-wait/request-media"]; len(got) != 6 || got[2] != "no" || got[4] != "pending" {
		t.Fatalf("pending row=%v\n%s", got, out)
	}
}

func TestUsersErasuresSingleID(t *testing.T) {
	fake := &erasureFakeAuth{byID: map[string]*authv1.ErasureStatus{"er-done": completeErasure()}}
	startErasureFake(t, fake)
	setAdminToken(t, "tok")

	out, err := runUsers(t, "erasures", "er-done")
	if err != nil {
		t.Fatal(err)
	}
	req := fake.statusReqs[0]
	if req.GetErasureId() != "er-done" || req.GetPendingOnly() {
		t.Fatalf("request=%v, want erasure_id only", req)
	}
	if fake.statusToken[0] != "tok" {
		t.Fatalf("x-auth-token=%q", fake.statusToken[0])
	}
	assertNoModuleOnlyCalls(t, fake)
	if !strings.Contains(out, "er-done") || !strings.Contains(out, "yes") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestUsersErasuresUnknownIDIsAnError(t *testing.T) {
	fake := &erasureFakeAuth{byID: map[string]*authv1.ErasureStatus{}}
	startErasureFake(t, fake)
	setAdminToken(t, "tok")

	_, err := runUsers(t, "erasures", "nope")
	if err == nil || !strings.Contains(err.Error(), "NotFound") {
		t.Fatalf("err=%v, want NotFound", err)
	}
	assertNoModuleOnlyCalls(t, fake)
}

func TestUsersErasuresNeverCompleteWhenAListedModuleIsNotOK(t *testing.T) {
	// The provider says complete, but a listed module is failed: do not show
	// a complete row.
	e := completeErasure()
	e.Modules = append(e.Modules, &authv1.ErasureModuleStatus{
		ModuleId: "playback-guard", Outcome: authv1.ErasureOutcome_ERASURE_OUTCOME_FAILED,
	})
	if v := newErasureView(e); v.Complete {
		t.Fatalf("view=%+v, want not complete", v)
	}
	// Provider says not complete although every listed module is OK.
	e = completeErasure()
	e.Complete = false
	if v := newErasureView(e); v.Complete {
		t.Fatalf("view=%+v, want not complete", v)
	}
	if v := newErasureView(completeErasure()); !v.Complete {
		t.Fatalf("view=%+v, want complete", v)
	}
}

func TestUsersErasuresJSON(t *testing.T) {
	fake := &erasureFakeAuth{pages: map[string]*authv1.GetUserErasureStatusResponse{
		"": {Erasures: []*authv1.ErasureStatus{completeErasure(), pendingErasure()}},
	}}
	startErasureFake(t, fake)
	setAdminToken(t, "tok")

	out, err := runUsers(t, "erasures", "--all", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got []erasureView
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if len(got) != 2 || !got[0].Complete || got[1].Complete {
		t.Fatalf("got=%+v", got)
	}
	var outcomes []string
	for _, m := range got[1].Modules {
		outcomes = append(outcomes, m.ModuleID+"="+m.Outcome)
	}
	want := "userdata-local=ok request-media=pending playback-monitor=failed auth-oidc=unsupported"
	if strings.Join(outcomes, " ") != want {
		t.Fatalf("outcomes=%q want %q", strings.Join(outcomes, " "), want)
	}
	if got[1].Modules[2].DetailCode != "store_locked" {
		t.Fatalf("detail=%q", got[1].Modules[2].DetailCode)
	}
}

func TestUsersErasuresEmptyJSONIsArray(t *testing.T) {
	startErasureFake(t, &erasureFakeAuth{})
	setAdminToken(t, "tok")
	out, err := runUsers(t, "erasures", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("out=%q", out)
	}
}

func TestUsersErasuresFollowsPages(t *testing.T) {
	fake := &erasureFakeAuth{pages: map[string]*authv1.GetUserErasureStatusResponse{
		"":   {Erasures: []*authv1.ErasureStatus{pendingErasure()}, NextPageToken: "p2"},
		"p2": {Erasures: []*authv1.ErasureStatus{completeErasure()}},
	}}
	startErasureFake(t, fake)
	setAdminToken(t, "tok")

	out, err := runUsers(t, "erasures", "--all")
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.statusReqs) != 2 || fake.statusReqs[1].GetPageToken() != "p2" {
		t.Fatalf("requests=%v", fake.statusReqs)
	}
	for _, tok := range fake.statusToken {
		if tok != "tok" {
			t.Fatalf("x-auth-token=%q", tok)
		}
	}
	if !strings.Contains(out, "er-wait") || !strings.Contains(out, "er-done") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestUsersErasuresPageTokenLoopIsAnError(t *testing.T) {
	fake := &erasureFakeAuth{pages: map[string]*authv1.GetUserErasureStatusResponse{
		"":   {NextPageToken: "p2"},
		"p2": {NextPageToken: "p2"},
	}}
	startErasureFake(t, fake)
	setAdminToken(t, "tok")
	if _, err := runUsers(t, "erasures"); err == nil || !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("err=%v", err)
	}
}

func TestUsersErasuresRequiresTokenAndMakesNoCall(t *testing.T) {
	fake := &erasureFakeAuth{}
	startErasureFake(t, fake)
	setAdminToken(t, "")

	_, err := runUsers(t, "erasures")
	if err == nil || !strings.Contains(err.Error(), "token required") {
		t.Fatalf("err=%v", err)
	}
	if len(fake.statusReqs) != 0 {
		t.Fatalf("status calls=%d, want 0", len(fake.statusReqs))
	}
}

func TestUsersDeleteSendsAdminBearerAndPrintsErasureID(t *testing.T) {
	fake := &erasureFakeAuth{deleteID: "er-123"}
	startErasureFake(t, fake)
	setAdminToken(t, "admin-session-token")

	out, err := runUsers(t, "delete", "u1", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.deleteToken) != 1 || fake.deleteToken[0] != "admin-session-token" {
		t.Fatalf("delete x-auth-token=%v", fake.deleteToken)
	}
	if !strings.Contains(out, "deleted\n") || !strings.Contains(out, "erasure_id: er-123") {
		t.Fatalf("out=%q", out)
	}
	assertNoModuleOnlyCalls(t, fake)
}

func TestUsersDeleteWithoutErasureIDKeepsOldOutput(t *testing.T) {
	// Providers that predate ADR-0035 return no erasure_id.
	fake := &erasureFakeAuth{}
	startErasureFake(t, fake)
	setAdminToken(t, "tok")

	out, err := runUsers(t, "delete", "u1", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if out != "deleted\n" {
		t.Fatalf("out=%q", out)
	}
}

func TestUsersDeleteSurfacesProviderError(t *testing.T) {
	fake := &erasureFakeAuth{deleteErr: "cannot delete the last admin"}
	startErasureFake(t, fake)
	setAdminToken(t, "tok")

	out, err := runUsers(t, "delete", "u1", "--yes")
	if err == nil || !strings.Contains(err.Error(), "last admin") {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if strings.Contains(out, "deleted") {
		t.Fatalf("out=%q", out)
	}
}
