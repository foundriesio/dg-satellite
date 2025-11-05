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

	mtls := e.Group("/")
	mtls.Use(
		h.authDevice,
		middleware.BodyLimit("100K"), // After TLS authentication but before we read headers.
		h.checkinDevice,
	)

	mtls.POST("apps-states", h.appsStatesInfo)
	mtls.GET("device", h.deviceGet)
	mtls.POST("events", h.eventsUpload)
	mtls.POST("ostree/download-urls", h.ostreeUrls)
	mtls.GET("ostree/*", h.ostreeFileStream)
	mtls.GET("repo/timestamp.json", h.metaTimestamp)
	mtls.GET("repo/snapshot.json", h.metaSnapshot)
	mtls.GET("repo/targets.json", h.metaTargets)
	mtls.GET("repo/:root", h.metaRoot)
	mtls.PUT("system_info", h.hardwareInfo)
	mtls.PUT("system_info/config", h.akTomlInfo)
	mtls.PUT("system_info/network", h.networkInfo)
}
