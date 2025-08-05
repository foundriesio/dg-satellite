// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/storage/dg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type handlers struct {
	storage *dg.Storage
}

func RegisterHandlers(e *echo.Echo, storage *dg.Storage) {
	h := handlers{storage: storage}
	e.Use(h.authDevice)
	e.Use(middleware.BodyLimit("1M"))
	e.GET("/device", h.deviceGet)

	e.PUT("/system_info/config", h.putAktoml)
	e.PUT("/system_info/network", h.putNetInfo)
	e.PUT("/system_info", h.putHwInfo)
}

func ErrResponse(c echo.Context, status int, respMsg string, err error) error {
	log := context.CtxGetLog(c.Request().Context())
	if err != nil {
		log.Error(respMsg, "error", err)
	} else {
		log.Error(respMsg)
	}
	return c.String(status, respMsg)
}
