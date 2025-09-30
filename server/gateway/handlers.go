// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/gateway"
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
	e.POST("/tests", h.testCreate)
	e.PUT("/tests/:testid", h.testComplete)
	e.PUT("/tests/:testid/:path", h.testArtifact)
}
