// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/gateway"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type handlers struct {
	storage *storage.Storage
}

var (
	EchoError     = server.EchoError
	ReadBody      = server.ReadBody
	ReadJsonBody  = server.ReadJsonBody
	ParseJsonBody = server.ParseJsonBody
)

func RegisterHandlers(e *echo.Echo, storage *storage.Storage) {
	h := handlers{storage: storage}
	e.Use(h.authDevice)
	e.Use(middleware.BodyLimit("100K")) // After TLS authentication but before we read headers.
	e.Use(h.checkinDevice)
	e.POST("/apps-states", h.appsStatesInfo)
	e.GET("/device", h.deviceGet)
	e.POST("/events", h.eventsUpload)
	e.POST("/ostree/download-urls", h.ostreeUrls)
	e.GET("/ostree/*", h.ostreeFileStream)
	e.GET("/repo/timestamp.json", h.metaTimestamp)
	e.GET("/repo/snapshot.json", h.metaSnapshot)
	e.GET("/repo/targets.json", h.metaTargets)
	e.GET("/repo/:root", h.metaRoot)
	e.PUT("/system_info", h.hardwareInfo)
	e.PUT("/system_info/config", h.akTomlInfo)
	e.PUT("/system_info/network", h.networkInfo)

	cr := e.Group("/v2")
	// Handle requests like
	// v2/msul-test/simple/manifests/sha256:d1e78dd150e58db366f850d93a5251e5074e72c5a5e54a0f1b5910f2fca8667b
	// v2/msul-test/simple/blobs/sha256:<>
	cr.HEAD("/:repo/:app/blobs/sha256\\::hash", h.blobHead)
	cr.HEAD("/:repo/:app/manifests/sha256\\::hash", h.blobHead)

	cr.GET("/:repo/:app/blobs/sha256\\::hash", h.blobGet)
	cr.GET("/:repo/:app/manifests/sha256\\::hash", h.blobGet)
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
