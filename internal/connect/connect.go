// Package connect dials a running muxcored gRPC endpoint for muxcorectl.
package connect

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Muxcore-Media/core/sdk/go/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Options control how muxcorectl dials muxcored.
type Options struct {
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
	insecure := envTruthy("MUXCORE_INSECURE_DISABLE_TLS")
	token := os.Getenv("MUXCORE_TOKEN")
	if token == "" {
		token = os.Getenv("MUXCORE_ADMIN_TOKEN")
	}
	return Options{
		Addr:     addr,
		Insecure: insecure,
		Token:    token,
		Timeout:  15 * time.Second,
	}
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// Dial connects to muxcored using the core SDK client.
// Prefer MUXCORE_INSECURE_DISABLE_TLS / --insecure for local laptop stacks.
func Dial(opts Options) (*client.Client, error) {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:9090"
	}
	var dialOpts []client.Option
	if opts.Insecure {
		dialOpts = append(dialOpts, client.WithInsecure())
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
