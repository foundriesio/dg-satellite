// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"github.com/foundriesio/dg-satellite/storage/gateway"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	storage *gateway.Storage
}

func RegisterHandlers(e *echo.Echo, storage *gateway.Storage) {
	h := handlers{storage: storage}
	e.Use(h.authDevice)
	e.GET("/device", h.deviceGet)
}
