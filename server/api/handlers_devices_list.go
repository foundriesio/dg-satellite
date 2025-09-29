// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"net/http"

	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/api"
	"github.com/labstack/echo/v4"
)

type DeviceListOpts = storage.DeviceListOpts

// @Summary List devices
// @Param _ query DeviceListOpts false "Sorting options"
// @Accept  json
// @Produce json
// @Success 200 {array} storage.Device
// @Router  /devices [get]
func (h *handlers) deviceList(c echo.Context) error {
	opts := storage.DeviceListOpts{
		OrderBy: storage.OrderByDeviceLastSeenDsc,
		Limit:   1000,
		Offset:  0,
	}
	if err := c.Bind(&opts); err != nil {
		return server.EchoError(c, err, http.StatusInternalServerError, "Failed to parse list options")
	}

	devices, err := h.storage.DevicesList(opts)
	if err != nil {
		return server.EchoError(c, err, http.StatusInternalServerError, "Unexpected error listing devices")
	}

	// TODO handle pagination in response
	return c.JSON(http.StatusOK, devices)
}
