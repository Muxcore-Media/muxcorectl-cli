package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	authv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/auth/v1"
	"github.com/Muxcore-Media/core/sdk/go/client"
	"github.com/spf13/cobra"
)

func newUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Short:   "Manage local auth users (admin-ui /users parity)",
		GroupID: groupAccess,
	}
	cmd.AddCommand(newUsersListCmd())
	cmd.AddCommand(newUsersCreateCmd())
	cmd.AddCommand(newUsersDeleteCmd())
	cmd.AddCommand(newUsersPasswordCmd())
	cmd.AddCommand(newUsersRolesCmd())
	cmd.AddCommand(newUsersTOTPCmd())
	cmd.AddCommand(newUsersTokensCmd())
	cmd.AddCommand(newUsersPasskeysCmd())
	cmd.AddCommand(newUsersParentalCmd())
	return cmd
}

func withAuth(fn func(ctx context.Context, auth authv1.AuthServiceClient) error) error {
	return withCore(func(ctx context.Context, c *client.Client) error {
		mod, err := findModuleByCapability(ctx, c, "auth")
		if err != nil {
			return err
		}
		conn, err := dialModuleGRPC(mod.GetId(), mod.GetHttpAddr())
		if err != nil {
			return err
		}
		defer conn.Close()
		return fn(ctx, authv1.NewAuthServiceClient(conn))
	})
}

func newUsersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.ListUsers(ctx, &authv1.ListUsersRequest{})
				if err != nil {
					return fmt.Errorf("users list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetUsers())
				}
				rows := make([][]string, 0, len(resp.GetUsers()))
				for _, u := range resp.GetUsers() {
					totp := "off"
					if u.GetTotpEnabled() {
						totp = "on"
					}
					rows = append(rows, []string{u.GetId(), u.GetUsername(), strings.Join(u.GetRoles(), ","), totp, u.GetTenantId()})
				}
				return printTable([]string{"ID", "USERNAME", "ROLES", "TOTP", "TENANT"}, rows)
			})
		},
	}
}

func newUsersCreateCmd() *cobra.Command {
	var password string
	cmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a user (prompts for password unless --password is set)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			pass := password
			if pass == "" {
				var err error
				pass, err = readPassword("Password: ")
				if err != nil {
					return err
				}
				confirm, err := readPassword("Confirm: ")
				if err != nil {
					return err
				}
				if pass != confirm {
					return fmt.Errorf("passwords do not match")
				}
			}
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.CreateUser(ctx, &authv1.CreateUserRequest{
					Username: username,
					Password: pass,
				})
				if err != nil {
					return fmt.Errorf("users create: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if flagJSON {
					return printJSON(map[string]string{"id": resp.GetUserId(), "username": username})
				}
				fmt.Printf("created user %s (%s)\n", username, resp.GetUserId())
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&password, "password", "", "password (non-interactive)")
	return cmd
}

func newUsersDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <user-id>",
		Short: "Delete a user by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("Delete user " + args[0] + "?"); err != nil {
				return err
			}
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.DeleteUser(ctx, &authv1.DeleteUserRequest{UserId: args[0]})
				if err != nil {
					return fmt.Errorf("users delete: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
}

func newUsersPasswordCmd() *cobra.Command {
	var password string
	cmd := &cobra.Command{
		Use:   "password <user-id>",
		Short: "Set a user's password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pass := password
			if pass == "" {
				var err error
				pass, err = readPassword("New password: ")
				if err != nil {
					return err
				}
			}
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.SetPassword(ctx, &authv1.SetPasswordRequest{
					UserId:   args[0],
					Password: pass,
				})
				if err != nil {
					return fmt.Errorf("users password: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if !flagQuiet {
					fmt.Println("password updated")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&password, "password", "", "new password (non-interactive)")
	return cmd
}

func newUsersRolesCmd() *cobra.Command {
	set := &cobra.Command{
		Use:   "set <user-id> <role...>",
		Short: "Replace all roles for a user",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID := args[0]
			roles := args[1:]
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.SetRoles(ctx, &authv1.SetRolesRequest{UserId: userID, Roles: roles})
				if err != nil {
					return fmt.Errorf("users roles set: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if !flagQuiet {
					fmt.Printf("roles set to %v\n", roles)
				}
				return nil
			})
		},
	}
	add := &cobra.Command{
		Use:   "add <user-id> <role>",
		Short: "Add one role to a user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				roles, err := userRoles(ctx, auth, args[0])
				if err != nil {
					return err
				}
				for _, r := range roles {
					if r == args[1] {
						if !flagQuiet {
							fmt.Println("role already present")
						}
						return nil
					}
				}
				roles = append(roles, args[1])
				resp, err := auth.SetRoles(ctx, &authv1.SetRolesRequest{UserId: args[0], Roles: roles})
				if err != nil {
					return fmt.Errorf("users roles add: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if !flagQuiet {
					fmt.Printf("added role %q\n", args[1])
				}
				return nil
			})
		},
	}
	rm := &cobra.Command{
		Use:   "remove <user-id> <role>",
		Short: "Remove one role from a user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				roles, err := userRoles(ctx, auth, args[0])
				if err != nil {
					return err
				}
				next := make([]string, 0, len(roles))
				for _, r := range roles {
					if r != args[1] {
						next = append(next, r)
					}
				}
				resp, err := auth.SetRoles(ctx, &authv1.SetRolesRequest{UserId: args[0], Roles: next})
				if err != nil {
					return fmt.Errorf("users roles remove: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if !flagQuiet {
					fmt.Printf("removed role %q\n", args[1])
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{Use: "roles", Short: "Manage user roles"}
	cmd.AddCommand(set, add, rm)
	return cmd
}

func userRoles(ctx context.Context, auth authv1.AuthServiceClient, userID string) ([]string, error) {
	resp, err := auth.ListUsers(ctx, &authv1.ListUsersRequest{})
	if err != nil {
		return nil, fmt.Errorf("users list: %w", err)
	}
	for _, u := range resp.GetUsers() {
		if u.GetId() == userID {
			return u.GetRoles(), nil
		}
	}
	return nil, fmt.Errorf("user %q not found", userID)
}

func newUsersTOTPCmd() *cobra.Command {
	status := &cobra.Command{
		Use:   "status <user-id>",
		Short: "Show TOTP status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.TOTPStatus(ctx, &authv1.TOTPStatusRequest{UserId: args[0]})
				if err != nil {
					return fmt.Errorf("users totp status: %w", err)
				}
				if flagJSON {
					return printJSON(resp)
				}
				if resp.GetEnabled() {
					fmt.Println("enabled")
				} else {
					fmt.Println("disabled")
				}
				return nil
			})
		},
	}
	enable := &cobra.Command{
		Use:   "enable <user-id>",
		Short: "Enable TOTP for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.EnableTOTP(ctx, &authv1.EnableTOTPRequest{UserId: args[0]})
				if err != nil {
					return fmt.Errorf("users totp enable: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if flagJSON {
					return printJSON(resp)
				}
				fmt.Printf("secret: %s\n", resp.GetSecret())
				if resp.GetQrCodeUrl() != "" {
					fmt.Printf("qr: %s\n", resp.GetQrCodeUrl())
				}
				return nil
			})
		},
	}
	disable := &cobra.Command{
		Use:   "disable <user-id>",
		Short: "Disable TOTP for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.DisableTOTP(ctx, &authv1.DisableTOTPRequest{UserId: args[0]})
				if err != nil {
					return fmt.Errorf("users totp disable: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if !flagQuiet {
					fmt.Println("disabled")
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{Use: "totp", Short: "Manage TOTP two-factor auth"}
	cmd.AddCommand(status, enable, disable)
	return cmd
}

func newUsersTokensCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list <user-id>",
		Short: "List API tokens for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.ListAPITokens(ctx, &authv1.ListAPITokensRequest{UserId: args[0]})
				if err != nil {
					return fmt.Errorf("users tokens list: %w", err)
				}
				if flagJSON {
					return printJSON(resp.GetTokens())
				}
				rows := make([][]string, 0, len(resp.GetTokens()))
				for _, t := range resp.GetTokens() {
					rows = append(rows, []string{t.GetId(), t.GetName(), t.GetPrefix()})
				}
				return printTable([]string{"ID", "NAME", "PREFIX"}, rows)
			})
		},
	}
	create := &cobra.Command{
		Use:   "create <user-id> <name>",
		Short: "Create an API token",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.CreateAPIToken(ctx, &authv1.CreateAPITokenRequest{
					UserId: args[0],
					Name:   args[1],
				})
				if err != nil {
					return fmt.Errorf("users tokens create: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if flagJSON {
					return printJSON(map[string]string{"id": resp.GetTokenId(), "token": resp.GetToken()})
				}
				fmt.Printf("token: %s\n", resp.GetToken())
				fmt.Println("(store this token — it will not be shown again)")
				return nil
			})
		},
	}
	rm := &cobra.Command{
		Use:   "delete <token-id>",
		Short: "Delete an API token by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.DeleteAPIToken(ctx, &authv1.DeleteAPITokenRequest{TokenId: args[0]})
				if err != nil {
					return fmt.Errorf("users tokens delete: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{Use: "tokens", Short: "Manage user API tokens"}
	cmd.AddCommand(list, create, rm)
	return cmd
}

func newUsersPasskeysCmd() *cobra.Command {
	list := &cobra.Command{
		Use:   "list <user-id>",
		Short: "List WebAuthn passkeys for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.ListWebAuthnCredentials(ctx, &authv1.ListWebAuthnCredentialsRequest{UserId: args[0]})
				if err != nil {
					return fmt.Errorf("users passkeys list: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if flagJSON {
					return printJSON(resp.GetCredentials())
				}
				rows := make([][]string, 0, len(resp.GetCredentials()))
				for _, c := range resp.GetCredentials() {
					rows = append(rows, []string{
						c.GetId(),
						c.GetCredentialType(),
						c.GetCreatedAt(),
						c.GetLastUsedAt(),
					})
				}
				return printTable([]string{"ID", "TYPE", "CREATED", "LAST_USED"}, rows)
			})
		},
	}
	rm := &cobra.Command{
		Use:   "delete <user-id> <credential-id>",
		Short: "Delete a WebAuthn passkey",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmAction("delete passkey " + args[1]); err != nil {
				return err
			}
			return withAuth(func(ctx context.Context, auth authv1.AuthServiceClient) error {
				resp, err := auth.DeleteWebAuthnCredential(ctx, &authv1.DeleteWebAuthnCredentialRequest{
					UserId:       args[0],
					CredentialId: args[1],
				})
				if err != nil {
					return fmt.Errorf("users passkeys delete: %w", err)
				}
				if resp.GetError() != "" {
					return fmt.Errorf("%s", resp.GetError())
				}
				if !flagQuiet {
					fmt.Println("deleted")
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "passkeys",
		Short: "Manage WebAuthn passkeys (registration is browser-only)",
	}
	cmd.AddCommand(list, rm)
	return cmd
}

func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
