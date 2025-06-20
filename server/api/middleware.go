// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/labstack/echo/v4"
)

func authUser(authFunc auth.AuthUserFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, err := authFunc(c.Response().Writer, c.Request())
			if user == nil || err != nil {
				return err
			}
			c.Set("user", user)

			req := c.Request()
			ctx := req.Context()
			log := server.CtxGetLog(ctx).With("user", user.Id())
			ctx = server.CtxWithLog(ctx, log)
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	}
}
