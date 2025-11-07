// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"os"
	"path/filepath"

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
	e.GET("/devices/:uuid/updates", h.deviceUpdatesList, requireScope(auth.ScopeDevicesR))
	e.GET("/devices/:uuid/updates/:id", h.deviceUpdatesGet, requireScope(auth.ScopeDevicesR))
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

	cr := e.Group("/v2")
	// Handle requests like
	// v2/msul-test/simple/manifests/sha256:d1e78dd150e58db366f850d93a5251e5074e72c5a5e54a0f1b5910f2fca8667b
	// v2/msul-test/simple/blobs/sha256:<>
	cr.HEAD("/:repo/:app/blobs/sha256\\::hash", h.blobHead, requireScope(auth.ScopeDevicesR))
	cr.HEAD("/:repo/:app/manifests/sha256\\::hash", h.blobHead, requireScope(auth.ScopeDevicesR))

	cr.GET("/:repo/:app/blobs/sha256\\::hash", h.blobGet, requireScope(auth.ScopeDevicesR))
	cr.GET("/:repo/:app/manifests/sha256\\::hash", h.blobGet, requireScope(auth.ScopeDevicesR))
}

func (h *handlers) blobHead(c echo.Context) error {
	fmt.Printf(">>> checkng if blob exists: %s\n", c.Param("hash"))
	return c.NoContent(200)
}

func (h *handlers) blobGet(c echo.Context) error {
	fmt.Printf(">>> getting blob: %s\n", c.Param("hash"))
	// return blob content read from a local file storage
	blobPath := filepath.Join(os.Getenv("APP_BLOBS_ROOT"), c.Param("hash"))
	return c.File(blobPath)
}
