package cli

import (
	"context"
	"net"
	"testing"

	authv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type fakeAuthServer struct {
	authv1.UnimplementedAuthServiceServer
	createPassword string
	setPassword    string
}

func (f *fakeAuthServer) CreateUser(_ context.Context, req *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	f.createPassword = req.GetPassword()
	return &authv1.CreateUserResponse{UserId: "u1"}, nil
}

func (f *fakeAuthServer) SetPassword(_ context.Context, req *authv1.SetPasswordRequest) (*authv1.SetPasswordResponse, error) {
	f.setPassword = req.GetPassword()
	return &authv1.SetPasswordResponse{}, nil
}

func startFakeAuth(t *testing.T) (*fakeAuthServer, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	fake := &fakeAuthServer{}
	authv1.RegisterAuthServiceServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()
	dial := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(dial),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	old := withAuthDial
	withAuthDial = func(fn func(ctx context.Context, auth authv1.AuthServiceClient) error) error {
		return fn(context.Background(), authv1.NewAuthServiceClient(conn))
	}
	return fake, func() {
		withAuthDial = old
		srv.Stop()
		_ = conn.Close()
	}
}

func TestUsersCreatePasswordBufconn(t *testing.T) {
	fake, cleanup := startFakeAuth(t)
	defer cleanup()

	pass := "from-flag"
	if err := withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
		_, err := auth.CreateUser(ctx, &authv1.CreateUserRequest{Username: "alice", Password: pass})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if fake.createPassword != "from-flag" {
		t.Fatalf("create password=%q", fake.createPassword)
	}
}

func TestUsersSetPasswordBufconn(t *testing.T) {
	fake, cleanup := startFakeAuth(t)
	defer cleanup()

	pass := "rotated"
	if err := withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
		_, err := auth.SetPassword(ctx, &authv1.SetPasswordRequest{UserId: "u1", Password: pass})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if fake.setPassword != "rotated" {
		t.Fatalf("set password=%q", fake.setPassword)
	}
}

func TestReadSecretFlagOrFile(t *testing.T) {
	if got, err := readSecretFlagOrFile("inline", "", ""); err != nil || got != "inline" {
		t.Fatalf("inline=%q err=%v", got, err)
	}
}
