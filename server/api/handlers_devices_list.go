// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"net/http"

	storage "github.com/foundriesio/dg-satellite/storage/api"
	"github.com/labstack/echo/v4"
)

// @Summary List devices
// @Param _ query storage.DeviceListOpts false "Sorting options"
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
		return c.String(http.StatusBadRequest, err.Error())
	}

	devices, err := h.storage.DevicesList(opts)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	// TODO handle pagination in response
	return c.JSON(http.StatusOK, devices)
}
