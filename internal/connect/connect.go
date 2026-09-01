// Package connect dials a running muxcored gRPC endpoint for muxcorectl.
package connect

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Muxcore-Media/core/sdk/go/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// Options control how muxcorectl dials muxcored.
type Options struct { //nolint:govet // fieldalignment: dial options grouped for readability
	Addr     string
	Insecure bool
	Token    string
	Timeout  time.Duration
}

// FromEnv fills defaults from environment variables used by the laptop MVP.
func FromEnv() Options {
	addr := "127.0.0.1:9090"
	if v := os.Getenv("MUXCORE_GRPC_ADDR"); v != "" {
		addr = v
	}
	insecureDev := envTruthy("MUXCORE_INSECURE_DISABLE_TLS")
	token := os.Getenv("MUXCORE_TOKEN")
	if token == "" {
		token = os.Getenv("MUXCORE_ADMIN_TOKEN")
	}
	return Options{
		Addr:     addr,
		Insecure: insecureDev,
		Token:    token,
		Timeout:  15 * time.Second,
	}
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func tlsGRPCDialOptions() ([]grpc.DialOption, error) {
	certFile := strings.TrimSpace(os.Getenv("MUXCORE_TLS_CERT"))
	keyFile := strings.TrimSpace(os.Getenv("MUXCORE_TLS_KEY"))
	caFile := strings.TrimSpace(os.Getenv("MUXCORE_TLS_CA"))
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("TLS not configured; set MUXCORE_TLS_CERT and MUXCORE_TLS_KEY, or use --insecure")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client TLS cert/key: %w", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if caFile != "" {
		pemBytes, err := os.ReadFile(caFile) //nolint:gosec // operator-configured CA path
		if err != nil {
			return nil, fmt.Errorf("read TLS CA: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemBytes) {
			return nil, fmt.Errorf("parse TLS CA from %q", caFile)
		}
		tlsConfig.RootCAs = pool
	}
	return []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}, nil
}

// Dial connects to muxcored using the core SDK client.
// Prefer MUXCORE_INSECURE_DISABLE_TLS / --insecure for local laptop stacks.
func Dial(opts Options) (*client.Client, error) {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:9090"
	}
	var dialOpts []client.Option
	if opts.Insecure || envTruthy("MUXCORE_INSECURE_DISABLE_TLS") {
		dialOpts = append(dialOpts, client.WithInsecure())
	} else {
		tlsOpts, err := tlsGRPCDialOptions()
		if err != nil {
			return nil, err
		}
		for _, o := range tlsOpts {
			dialOpts = append(dialOpts, client.WithGRPCOption(o))
		}
	}
	if opts.Token != "" {
		token := opts.Token
		dialOpts = append(dialOpts,
			client.WithGRPCOption(grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOpts ...grpc.CallOption) error {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
				return invoker(ctx, method, req, reply, cc, callOpts...)
			})),
			client.WithGRPCOption(grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, callOpts ...grpc.CallOption) (grpc.ClientStream, error) {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
				return streamer(ctx, desc, cc, method, callOpts...)
			})),
		)
	}
	c, err := client.Dial(opts.Addr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", opts.Addr, err)
	}
	return c, nil
}

// Context returns a timeout context for a single CLI RPC.
func Context(opts Options) (context.Context, context.CancelFunc) {
	d := opts.Timeout
	if d <= 0 {
		d = 15 * time.Second
	}
	return context.WithTimeout(context.Background(), d)
}
