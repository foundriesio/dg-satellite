// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/storage/api"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	storage *api.Storage
}

func RegisterHandlers(e *echo.Echo, storage *api.Storage, authFunc auth.AuthUserFunc) {
	h := handlers{storage: storage}
	e.Use(authUser(authFunc))

	e.GET("/devices", h.deviceList, requireScope(auth.ScopeDevicesR))
	e.GET("/devices/:uuid", h.deviceGet, requireScope(auth.ScopeDevicesR))

	e.GET("/devices/:uuid/tests", h.deviceTestsList, requireScope(auth.ScopeDevicesR))
	e.GET("/devices/:uuid/tests/:testid", h.deviceTestGet, requireScope(auth.ScopeDevicesR))
	e.GET("/devices/:uuid/tests/:testid/:artifact", h.deviceTestArtifact, requireScope(auth.ScopeDevicesR))
}
