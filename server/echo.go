// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package server

import (
	"github.com/labstack/echo/v4"
)

func NewEchoServer() *echo.Echo {
	server := echo.New()
	server.HideBanner = true
	server.HidePort = true
	return server
}
