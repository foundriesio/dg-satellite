// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"net/http"

	"github.com/foundriesio/dg-satellite/storage/dg"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	storage *dg.Storage
}

func RegisterHandlers(e *echo.Echo, storage *dg.Storage) {
	h := handlers{storage: storage}
	e.Use(h.authDevice)
	e.GET("/tmp", h.tmp)
}

func (h handlers) tmp(c echo.Context) error {
	d := getDevice(c)
	return c.String(http.StatusOK, d.Uuid)
}
