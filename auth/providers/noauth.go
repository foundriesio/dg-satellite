// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package providers

import (
	"log/slog"
	"net/http"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/users"
	"github.com/labstack/echo/v4"
)

type noauthProvider struct {
	users  *users.Storage
	scopes auth.Scopes
}

func (noauthProvider) Name() string {
	return "noauth"
}

func (p *noauthProvider) Configure(e *echo.Echo, storage *users.Storage, authConfig *storage.AuthConfig) (err error) {
	p.users = storage
	p.scopes, err = auth.ScopesFromSlice(authConfig.NewUserDefaultScopes)
	return
}

func (p noauthProvider) GetUser(c echo.Context) (*users.User, error) {
	user, err := p.users.Get("root")
	if err != nil {
		return nil, err
	} else if user == nil {
		slog.Info("noauth: creating root user")
		user = &users.User{
			Username:      "root",
			AllowedScopes: p.scopes,
		}
		err = p.users.Create(user)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (p noauthProvider) GetSession(c echo.Context) (*Session, error) {
	user, err := p.GetUser(c)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, nil
	}
	return &Session{
		BaseUrl: c.Scheme() + "://" + c.Request().Host,
		User:    user,
		Client:  http.DefaultClient,
	}, nil
}

func (noauthProvider) DropSession(c echo.Context) {
}

func init() {
	Providers = append(Providers, &noauthProvider{})
}
