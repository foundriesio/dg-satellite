// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	storage "github.com/foundriesio/dg-satellite/storage/api"
)

type AppsStatesResp struct {
	AppsStates []storage.AppsStates `json:"apps_states"`
}

// @Summary Get a list of Apps states reported by the device
// @Produce json
// @Success 200 AppsStatesResp
// @Router  /devices/:uuid/apps-states [get]
func (h *handlers) deviceAppsStatesGet(c echo.Context) error {
	uuid := c.Param("uuid")

	device, err := h.storage.DeviceGet(uuid)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device")
	}

	if device == nil {
		return c.NoContent(http.StatusNotFound)
	}

	appsStates, err := device.AppsStates()
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device updates")
	}
	return c.JSON(http.StatusOK, AppsStatesResp{AppsStates: appsStates})
}
