package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	authv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/auth/v1"
)

type requestIdentity struct {
	Username string
	Roles    []string
	IsAdmin  bool
}

func resolveOperatorToken() string {
	if strings.TrimSpace(flagToken) != "" {
		return strings.TrimSpace(flagToken)
	}
	if v := strings.TrimSpace(os.Getenv("MUXCORE_TOKEN_FILE")); v != "" {
		if b, err := os.ReadFile(v); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	if v := strings.TrimSpace(os.Getenv("MUXCORE_TOKEN")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("MUXCORE_ADMIN_TOKEN"))
}

func bearerAuthHeader() string {
	if tok := resolveOperatorToken(); tok != "" {
		return "Bearer " + tok
	}
	return ""
}

func withBearerAuth(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		out[k] = v
	}
	if auth := bearerAuthHeader(); auth != "" {
		out["Authorization"] = auth
	}
	return out
}

func requestIdentityFromToken() (requestIdentity, error) {
	tok := resolveOperatorToken()
	if tok == "" {
		return requestIdentity{}, fmt.Errorf("token required (--token, --token-file, MUXCORE_TOKEN, or MUXCORE_TOKEN_FILE)")
	}
	var ident requestIdentity
	err := withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
		resp, err := auth.Validate(ctx, &authv1.ValidateRequest{Token: tok})
		if err != nil {
			return fmt.Errorf("validate token: %w", err)
		}
		if resp.GetError() != "" {
			return fmt.Errorf("%s", resp.GetError())
		}
		if !resp.GetValid() {
			return fmt.Errorf("invalid token")
		}
		ident.Username = resp.GetUsername()
		if ident.Username == "" {
			ident.Username = resp.GetUserId()
		}
		ident.Roles = append([]string(nil), resp.GetRoles()...)
		for _, role := range ident.Roles {
			if strings.EqualFold(role, "admin") {
				ident.IsAdmin = true
				break
			}
		}
		return nil
	})
	return ident, err
}

func requestIdentityHeaders() (map[string]string, requestIdentity, error) {
	ident, err := requestIdentityFromToken()
	if err != nil {
		return nil, ident, err
	}
	headers := withBearerAuth(nil)
	if ident.Username != "" {
		headers["X-MuxCore-User"] = ident.Username
	}
	if len(ident.Roles) > 0 {
		headers["X-MuxCore-Roles"] = strings.Join(ident.Roles, ",")
	}
	return headers, ident, nil
}

// resolveTokenFile loads --token-file / MUXCORE_TOKEN_FILE when --token is unset.
func resolveTokenFile() error {
	if strings.TrimSpace(flagToken) != "" {
		return nil
	}
	path := strings.TrimSpace(flagTokenFile)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("MUXCORE_TOKEN_FILE"))
	}
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read token file: %w", err)
	}
	flagToken = strings.TrimSpace(string(b))
	return nil
}

func readSecretFlagOrFile(flagValue, filePath, prompt string) (string, error) {
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue), nil
	}
	if strings.TrimSpace(filePath) != "" {
		b, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read password file: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	return readPassword(prompt)
}
