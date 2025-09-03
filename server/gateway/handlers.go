// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"github.com/labstack/echo/v4"

	storage "github.com/foundriesio/dg-satellite/storage/gateway"
)

type handlers struct {
	storage *storage.Storage
}

func RegisterHandlers(e *echo.Echo, storage *storage.Storage) {
	h := handlers{storage: storage}
	e.Use(h.authDevice)
	e.GET("/device", h.deviceGet)
}
