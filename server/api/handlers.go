// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"errors"
	"net/http"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/labstack/echo/v4"
)

type handlers struct{}

func RegisterHandlers(e *echo.Echo, authUserFunc auth.AuthUserFunc) {
	h := handlers{}
	e.Use(authUser(authUserFunc))
	e.GET("/tmp", h.tmp)
}

// checkScope returns non-nil if the scope check failed. By returning
// non-nil to echo, we will still return the correct response but we will
// also *log* the reason the scope check failed.
func checkScope(c echo.Context, scope auth.Scope) error {
	user := c.Get("user").(auth.User)
	if err := user.HasScope(scope); err != nil {
		if err2 := c.String(http.StatusForbidden, err.Error()); err2 != nil {
			return errors.Join(err, err2)
		}
		return err
	}
	return nil
}

func (handlers) tmp(c echo.Context) error {
	if err := checkScope(c, auth.ScopeDevicesR); err != nil {
		return err
	}
	return c.String(http.StatusOK, "OK")
}
