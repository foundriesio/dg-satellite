// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type handlers struct{}

func RegisterHandlers(e *echo.Echo) {
	h := handlers{}
	e.Use(authDevice)
	e.GET("/tmp", h.tmp)
}

func (handlers) tmp(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
