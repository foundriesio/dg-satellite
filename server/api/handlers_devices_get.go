// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// @Summary Get a device by its UUID
// @Produce json
// @Success 200 storage.DeviceItem
// @Router  /devices/:uuid [get]
func (h *handlers) deviceGet(c echo.Context) error {
	uuid := c.Param("uuid")

	device, err := h.storage.DeviceGet(uuid)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	if device == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Not found")
	}
	return c.JSON(http.StatusOK, device)
}
