// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package login

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/foundriesio/dg-satellite/cli/config"
)

var LoginCmd = &cobra.Command{
	Use:   "login <context-name> <server-url>",
	Short: "Configure authentication for a server",
	Long: `Login to a Satellite Server by configuring a context with authentication.

This command will guide you through the authentication process and save
the configuration to ~/.config/satcli.yaml.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contextName := args[0]
		serverURL := args[1]

		token, _ := cmd.Flags().GetString("token")
		setDefault, _ := cmd.Flags().GetBool("set-default")
		scopes, _ := cmd.Flags().GetString("scopes")
		expiresInDays, _ := cmd.Flags().GetInt("expires-in-days")

		return login(contextName, serverURL, token, scopes, expiresInDays, setDefault)
	},
}

func init() {
	LoginCmd.Flags().String("token", "", "API token for authentication (skips OAuth2 device flow)")
	LoginCmd.Flags().Bool("set-default", true, "Set this context as the default")
	LoginCmd.Flags().String("scopes", "devices:read-update,updates:read-update", "Comma-separated list of OAuth2 scopes to request (optional)")
	LoginCmd.Flags().Int("expires-in-days", 90, "Number of days until the access token expires")
}

func login(contextName, serverURL, token, scopes string, expiresInDays int, setDefault bool) error {
	if token != "" {
		return saveToken(contextName, serverURL, token, setDefault)
	}

	fmt.Println("Initiating OAuth2 device authorization flow...")
	expires := time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour).Unix()
	return oauth2DeviceFlow(contextName, serverURL, scopes, expires, setDefault)
}

func saveToken(contextName, serverURL, token string, setDefault bool) error {
	// Load existing config or create new one
	cfg, err := config.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			cfg = &config.Config{
				Contexts: make(map[string]config.Context),
			}
		} else {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]config.Context)
	}
	cfg.Contexts[contextName] = config.Context{
		URL:   serverURL,
		Token: token,
	}

	if setDefault {
		cfg.ActiveContext = contextName
	}

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Successfully configured context '%s'\n", contextName)
	fmt.Printf("  Server URL: %s\n", serverURL)
	if setDefault {
		fmt.Printf("  Set as default context\n")
	}

	return nil
}

type deviceCodeRequest struct {
	Scopes  string `json:"scope"`
	Expires int64  `json:"token_expires"`
}

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	Expires                 int64  `json:"expires"`
	Interval                int    `json:"interval"`
}

type deviceTokenRequest struct {
	DeviceCode string `json:"device_code"`
	GrantType  string `json:"grant_type"`
}

type deviceTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Expires     int64  `json:"expires"`
	Scopes      string `json:"scope"`
}

type oauth2Error struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func (e *oauth2Error) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.ErrorDescription)
	}
	return e.ErrorCode
}

func oauth2DeviceFlow(contextName, serverURL, scopes string, expires int64, setDefault bool) error {
	// Step 1: Request device code
	codeReq := deviceCodeRequest{
		Scopes:  scopes,
		Expires: expires,
	}
	jsonData, err := json.Marshal(codeReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(serverURL+"/oauth2/device/code", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to request device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get device code (status %d): %s", resp.StatusCode, string(body))
	}

	var codeResp deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&codeResp); err != nil {
		return fmt.Errorf("failed to decode device code response: %w", err)
	}

	// Step 2: Display user code and verification URI
	fmt.Println()
	fmt.Println("------------------------------------------------")
	fmt.Printf("  Visit: %s\n", codeResp.VerificationURI)
	fmt.Println()
	fmt.Printf("  Enter code: %s\n", codeResp.UserCode)
	fmt.Println("------------------------------------------------")
	fmt.Println()
	fmt.Println("Waiting for authorization...")

	// Step 3: Poll for token
	pollInterval := time.Duration(codeResp.Interval) * time.Second
	expiresAt := time.Now().Add(time.Duration(codeResp.Expires) * time.Second)

	for time.Now().Before(expiresAt) {
		time.Sleep(pollInterval)

		token, err := pollForToken(serverURL, codeResp.DeviceCode)
		if err == nil {
			// Success! Save the token
			fmt.Println()
			fmt.Println("✓ Authorization successful!")
			return saveToken(contextName, serverURL, token, setDefault)
		}

		// Check if we should continue polling
		if oauth2Err, ok := err.(*oauth2Error); ok {
			switch oauth2Err.ErrorCode {
			case "authorization_pending":
				// Continue polling
				continue
			case "access_denied":
				return fmt.Errorf("authorization was denied")
			case "expired_token":
				return fmt.Errorf("authorization code expired")
			default:
				return fmt.Errorf("OAuth2 error: %s - %s", oauth2Err.ErrorCode, oauth2Err.ErrorDescription)
			}
		}

		return fmt.Errorf("failed to get token: %w", err)
	}

	return fmt.Errorf("authorization timed out")
}

func pollForToken(serverURL, deviceCode string) (string, error) {
	tokenReq := deviceTokenRequest{
		DeviceCode: deviceCode,
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
	}

	jsonData, err := json.Marshal(tokenReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(serverURL+"/oauth2/device/token", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 200 {
		var tokenResp deviceTokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return "", fmt.Errorf("failed to decode token response: %w", err)
		}
		return tokenResp.AccessToken, nil
	}

	var errResp oauth2Error
	if err := json.Unmarshal(body, &errResp); err != nil {
		return "", fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(body))
	}

	return "", &errResp
}
