package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	discoveryv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/discovery/v1"
	meshv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/mesh/v1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/Muxcore-Media/muxcorectl-cli/internal/connect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// normalizeDialAddr maps discovery HttpAddr (:port) to a dialable target.
// When MUXCORE_MESH_DIAL_LOCAL=true (host MVP), bare ports become 127.0.0.1:port.
func normalizeDialAddr(moduleID, httpAddr string) string {
	httpAddr = strings.TrimSpace(httpAddr)
	if httpAddr == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(httpAddr)
	if err != nil {
		if strings.HasPrefix(httpAddr, ":") {
			port = strings.TrimPrefix(httpAddr, ":")
			host = ""
		} else {
			return httpAddr
		}
	}
	if host != "" && host != "0.0.0.0" && host != "::" {
		return net.JoinHostPort(host, port)
	}
	if os.Getenv("MUXCORE_MESH_DIAL_LOCAL") == "true" {
		return net.JoinHostPort("127.0.0.1", port)
	}
	if moduleID != "" {
		return net.JoinHostPort(moduleID, port)
	}
	return net.JoinHostPort("127.0.0.1", port)
}

func normalizeHTTPBase(httpAddr string) string {
	httpAddr = strings.TrimSpace(httpAddr)
	if httpAddr == "" {
		return ""
	}
	if strings.HasPrefix(httpAddr, "http://") || strings.HasPrefix(httpAddr, "https://") {
		return strings.TrimRight(httpAddr, "/")
	}
	host, port, err := net.SplitHostPort(httpAddr)
	if err != nil {
		if strings.HasPrefix(httpAddr, ":") {
			port = strings.TrimPrefix(httpAddr, ":")
			host = ""
		} else {
			return "http://" + strings.TrimRight(httpAddr, "/")
		}
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func withCore(fn func(ctx context.Context, c *client.Client) error) error {
	opts := dialOpts()
	c, err := connect.Dial(opts)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	ctx, cancel := connect.Context(opts)
	defer cancel()
	return fn(ctx, c)
}

func findModuleByCapability(ctx context.Context, c *client.Client, capability string) (*discoveryv1.ModuleInfoProto, error) {
	mods, err := c.Discovery.FindByCapability(ctx, capability)
	if err != nil {
		return nil, fmt.Errorf("find capability %q: %w", capability, err)
	}
	if len(mods) == 0 {
		return nil, fmt.Errorf("no module with capability %q", capability)
	}
	return mods[0], nil
}

func findModuleByID(ctx context.Context, c *client.Client, moduleID string) (*discoveryv1.ModuleInfoProto, error) {
	info, err := c.Discovery.Resolve(ctx, moduleID)
	if err != nil {
		return nil, fmt.Errorf("resolve module %q: %w", moduleID, err)
	}
	if info == nil {
		return nil, fmt.Errorf("module %q not found", moduleID)
	}
	return info, nil
}

func dialModuleGRPC(moduleID, httpAddr string) (*grpc.ClientConn, error) {
	addr := normalizeDialAddr(moduleID, httpAddr)
	if addr == "" {
		return nil, fmt.Errorf("module %q has no dial address", moduleID)
	}
	dialOpts, err := meshGRPCDialOptions()
	if err != nil {
		return nil, err
	}
	if token := resolveOperatorToken(); token != "" {
		dialOpts = append(dialOpts,
			grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOpts ...grpc.CallOption) error {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
				return invoker(ctx, method, req, reply, cc, callOpts...)
			}),
			grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, callOpts ...grpc.CallOption) (grpc.ClientStream, error) {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
				return streamer(ctx, desc, cc, method, callOpts...)
			}),
		)
	}
	return grpc.NewClient(addr, dialOpts...)
}

func meshCall(ctx context.Context, c *client.Client, moduleID, httpAddr, method string, payload []byte) ([]byte, error) {
	if httpAddr != "" {
		conn, err := dialModuleGRPC(moduleID, httpAddr)
		if err == nil {
			defer func() { _ = conn.Close() }()
			mc := meshv1.NewModuleMeshClient(conn)
			resp, err := mc.Call(ctx, &meshv1.CallRequest{
				TargetModule: moduleID,
				Method:       method,
				Payload:      payload,
			})
			if err == nil && resp.GetError() == "" {
				return resp.GetPayload(), nil
			}
		}
	}
	return c.Mesh.Call(ctx, moduleID, method, payload)
}

func listMediaLibraries(ctx context.Context, c *client.Client) ([]*discoveryv1.ModuleInfoProto, error) {
	return c.Discovery.FindByCapability(ctx, "media.library")
}

// withModuleConn dials the first module advertising capability and runs fn with a gRPC conn.
func withModuleConn(capability string, fn func(ctx context.Context, conn *grpc.ClientConn) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		mod, err := findModuleByCapability(ctx, c, capability)
		if err != nil {
			return err
		}
		conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		return fn(ctx, conn)
	})
}
