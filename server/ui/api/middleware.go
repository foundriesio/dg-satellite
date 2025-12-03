// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"net/http"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/auth/providers"
	"github.com/foundriesio/dg-satellite/storage/users"
	"github.com/labstack/echo/v4"
)

func requireScope(scope auth.Scopes) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get("user").(*users.User)
			if !user.AllowedScopes.Has(scope) {
				msg := "User missing required scope(s): " + scope.String()
				return c.String(http.StatusForbidden, msg)
			}
			return next(c)
		}
	}
}

func authUser(provider providers.Provider) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, err := provider.GetUser(c)
			if user == nil || err != nil {
				return err
			}
			c.Set("user", user)

			req := c.Request()
			ctx := req.Context()
			log := CtxGetLog(ctx).With("user", user.Username)
			ctx = CtxWithLog(ctx, log)
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	}
}
