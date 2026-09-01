package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Muxcore-Media/core/sdk/go/client"
)

func resolveModuleHTTP(ctx context.Context, c *client.Client, capability, envKey string) (string, error) {
	if u := strings.TrimSpace(os.Getenv(envKey)); u != "" {
		return strings.TrimRight(u, "/"), nil
	}
	mod, err := findModuleByCapability(ctx, c, capability)
	if err != nil {
		return "", err
	}
	base := normalizeHTTPBase(mod.GetHttpAddr())
	if base == "" {
		addr := normalizeDialAddr(mod.GetId(), mod.GetHttpAddr())
		if addr == "" {
			return "", fmt.Errorf("module %q has no HTTP address; set %s", mod.GetId(), envKey)
		}
		base = "http://" + addr
	}
	return base, nil
}

func httpDo(method, url string, body []byte, headers map[string]string) ([]byte, error) {
	opts := dialOpts()
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range withBearerAuth(headers) {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func withRequestBase(fn func(base string) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		base, err := resolveModuleHTTP(ctx, c, capMediaRequest, "MUXCORE_REQUEST_URL")
		if err != nil {
			return err
		}
		return fn(base)
	})
}

const capMediaRequest = "media.request"
