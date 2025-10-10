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
	// In updates APIs :prod path element can be either "prod" or "ci".
	upd := e.Group("/updates/:prod")
	upd.Use(validateUpdateParams)
	upd.GET("", h.updateList, requireScope(auth.ScopeDevicesR))
	upd.GET("/:tag", h.updateList, requireScope(auth.ScopeDevicesR))
	// TODO: What data would we want to show for an update?
	// upd.GET("/:tag/:update", h.updateGet, requireScope(auth.ScopeDevicesR))
	upd.GET("/:tag/:update/rollouts", h.rolloutList, requireScope(auth.ScopeDevicesR))
	upd.GET("/:tag/:update/rollouts/:rollout", h.rolloutGet, requireScope(auth.ScopeDevicesR))
	upd.PUT("/:tag/:update/rollouts/:rollout", h.rolloutPut, requireScope(auth.ScopeDevicesRU))
}
