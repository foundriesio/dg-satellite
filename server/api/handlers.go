// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/storage/api"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	storage *api.Storage
}

func RegisterHandlers(e *echo.Echo, storage *api.Storage, authUserFunc auth.AuthUserFunc) {
	h := handlers{storage: storage}
	e.Use(authUser(authUserFunc))

	e.GET("/devices", h.deviceList, requireScope(auth.ScopeDevicesR))
	e.GET("/devices/:uuid", h.deviceGet, requireScope(auth.ScopeDevicesR))

	e.GET("/devices/:uuid/tests", h.deviceTestsList, requireScope(auth.ScopeDevicesR))
	e.GET("/devices/:uuid/tests/:testid", h.deviceTestGet, requireScope(auth.ScopeDevicesR))
	e.GET("/devices/:uuid/tests/:testid/:artifact", h.deviceTestArtifact, requireScope(auth.ScopeDevicesR))
}
