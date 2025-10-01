// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"github.com/labstack/echo/v4"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/api"
)

type handlers struct {
	storage *storage.Storage
}

var EchoError = server.EchoError

func RegisterHandlers(e *echo.Echo, storage *storage.Storage, authFunc auth.AuthUserFunc) {
	h := handlers{storage: storage}
	e.Use(authUser(authFunc))

	e.GET("/devices", h.deviceList, requireScope(auth.ScopeDevicesR))
	e.GET("/devices/:uuid", h.deviceGet, requireScope(auth.ScopeDevicesR))
}
