// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"io"
	"net/http"

	"github.com/foundriesio/dg-satellite/storage"
	"github.com/labstack/echo/v4"
)

func (h *handlers) putFile(name string, c echo.Context) error {
	device := getDevice(c)

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return ErrResponse(c, http.StatusInternalServerError, "Failed to read request body", err)
	}
	if err := device.PutFile(name, body); err != nil {
		return ErrResponse(c, http.StatusInternalServerError, "Failed to save "+name, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// @Summary Upload aktualizr-toml settings
// @Success 204
// @Router  /system_info/config [put]
func (h *handlers) putAktoml(c echo.Context) error {
	return h.putFile(storage.Aktoml, c)
}

// @Summary Upload device ipv4 local network info
// @Success 204
// @Router  /system_info/network [put]
func (h *handlers) putNetInfo(c echo.Context) error {
	return h.putFile(storage.NetInfo, c)
}

// @Summary Upload content of `lshw` command.
// @Success 204
// @Router  /system_info [put]
func (h *handlers) putHwInfo(c echo.Context) error {
	return h.putFile(storage.HwInfo, c)
}
