// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/users"
	"github.com/labstack/echo/v4"
)

// Session represents an authenticated web UI session.
type Session struct {
	BaseUrl string
	User    *users.User
	// Client is an HTTP client that includes the session cookie for making
	// authenticated requests against the REST api
	Client *http.Client
}

// Provider defines the interface that an authentication provider must implement
// to support a web server's authentication needs. This interface works for basic
// username/password authentication as well as OAuth2-based authentication.
type Provider interface {
	Name() string

	// Configure can be used to:
	//  - set up routes on the Echo instance
	//  - initialize any provider-specific settings
	Configure(e *echo.Echo, users *users.Storage, authConfig *storage.AuthConfig) error

	// GetUser retrieves the user based on either an API token or session cookie.
	GetUser(c echo.Context) (*users.User, error)

	// GetSession retrieves the session associated with the given context.
	GetSession(c echo.Context) (*Session, error)
	DropSession(c echo.Context)
}

var Providers []Provider

func GetProvider(name string) Provider {
	for _, p := range Providers {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// GetJson performs a request to the specified resource and decodes the JSON
// response into the provided result interface.
func (s Session) GetJson(path string, result any) error {
	req, err := http.NewRequest("GET", s.BaseUrl+path, nil)
	if err != nil {
		return err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("unable to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: HTTP_%d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
