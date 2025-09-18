// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"github.com/labstack/echo/v4"

	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/gateway"
)

type handlers struct {
	auth    server.Authenticator
	storage *storage.Storage
}

var (
	EchoError     = server.EchoError
	ReadBody      = server.ReadBody
	ReadJsonBody  = server.ReadJsonBody
	ParseJsonBody = server.ParseJsonBody
)

func RegisterHandlers(e *echo.Echo, storage *storage.Storage, auth server.Authenticator) {
	h := handlers{auth: auth, storage: storage}
	e.Use(h.authDevice)
	e.Use(h.checkinDevice)
	e.POST("/apps-states", h.appsStatesInfo)
	e.GET("/hub-creds", h.dockerCreds)
	e.GET("/device", h.deviceGet)
	e.PUT("/ecus", h.ecusInfo)
	e.POST("/events", h.eventsUpload)
	e.PUT("/system_info", h.hardwareInfo)
	e.PUT("/system_info/config", h.akTomlInfo)
	e.PUT("/system_info/network", h.networkInfo)
}
