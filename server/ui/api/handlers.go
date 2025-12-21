// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"github.com/labstack/echo/v4"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/api"
	"github.com/foundriesio/dg-satellite/storage/users"
)

type handlers struct {
	storage *storage.Storage
}

var EchoError = server.EchoError

func RegisterHandlers(e *echo.Echo, storage *storage.Storage, a auth.Provider) {
	h := handlers{storage: storage}
	g := e.Group("/v1")
	g.Use(authUser(a))

	g.GET("/devices", h.deviceList, requireScope(users.ScopeDevicesR))
	g.GET("/devices/:uuid", h.deviceGet, requireScope(users.ScopeDevicesR))
	g.GET("/devices/:uuid/apps-states", h.deviceAppsStatesGet, requireScope(users.ScopeDevicesR))
	g.GET("/devices/:uuid/updates", h.deviceUpdatesList, requireScope(users.ScopeDevicesR))
	g.GET("/devices/:uuid/updates/:id", h.deviceUpdatesGet, requireScope(users.ScopeDevicesR))
	g.PATCH("/devices/:uuid/labels", h.deviceLabelsPatch, requireScope(users.ScopeDevicesRU))
	// In updates APIs :prod path element can be either "prod" or "ci".
	upd := g.Group("/updates/:prod")
	upd.Use(validateUpdateParams)
	upd.GET("", h.updateList, requireScope(users.ScopeDevicesR))
	upd.GET("/:tag", h.updateList, requireScope(users.ScopeDevicesR))
	// TODO: What data would we want to show for an update?
	// upd.GET("/:tag/:update", h.updateGet, requireScope(users.ScopeDevicesR))
	upd.GET("/:tag/:update/rollouts", h.rolloutList, requireScope(users.ScopeDevicesR))
	upd.GET("/:tag/:update/rollouts/:rollout", h.rolloutGet, requireScope(users.ScopeDevicesR))
	upd.PUT("/:tag/:update/rollouts/:rollout", h.rolloutPut, requireScope(users.ScopeDevicesRU))
	upd.GET("/:tag/:update/rollouts/:rollout/tail", h.rolloutTail, requireScope(users.ScopeDevicesR))
	upd.GET("/:tag/:update/tail", h.updateTail, requireScope(users.ScopeDevicesR))
}
