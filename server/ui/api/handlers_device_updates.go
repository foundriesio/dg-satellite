// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"net/http"

	"github.com/foundriesio/dg-satellite/storage"
	"github.com/labstack/echo/v4"
)

type UpdateEvent storage.DeviceUpdateEvent

// @Summary Get a list of updates for a device
// @Produce json
// @Success 200 []string
// @Router  /devices/:uuid/updates [get]
func (h *handlers) deviceUpdatesList(c echo.Context) error {
	uuid := c.Param("uuid")

	device, err := h.storage.DeviceGet(uuid)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device")
	}

	if device == nil {
		return c.NoContent(http.StatusNotFound)
	}
	updates, err := device.Updates()
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device updates")
	}
	return c.JSON(http.StatusOK, updates)
}

// @Summary Get details of update events for a devices
// @Produce json
// @Success 200 []UpdateEvent
// @Router  /devices/:uuid/updates/:id [get]
func (h *handlers) deviceUpdatesGet(c echo.Context) error {
	uuid := c.Param("uuid")
	updateId := c.Param("id")

	device, err := h.storage.DeviceGet(uuid)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device")
	}

	if device == nil {
		return c.NoContent(http.StatusNotFound)
	}
	events, err := device.Events(updateId)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device update events")
	}
	if len(events) == 0 {
		return c.NoContent(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, events)
}
