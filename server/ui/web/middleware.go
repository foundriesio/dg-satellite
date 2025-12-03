// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/foundriesio/dg-satellite/context"
)

func (h handlers) requireSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, err := h.provider.GetSession(c)
		if err != nil {
			return EchoError(c, err, http.StatusInternalServerError, err.Error())
		} else if session == nil {
			return nil // The provider sent the response (e.g., redirect to login)
		}

		ctx := c.Request().Context()
		log := context.CtxGetLog(ctx).With("user", session.User.Username)
		ctx = context.CtxWithLog(ctx, log)
		ctx = CtxWithSession(ctx, session)
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}
