// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	storage "github.com/foundriesio/dg-satellite/storage/api"
)

type Device = storage.Device

// @Summary Get a device by its UUID
// @Produce json
// @Success 200 Device
// @Router  /devices/:uuid [get]
func (h *handlers) deviceGet(c echo.Context) error {
	uuid := c.Param("uuid")

	device, err := h.storage.DeviceGet(uuid)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device")
	}

	if device == nil {
		return c.NoContent(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, device)
}
