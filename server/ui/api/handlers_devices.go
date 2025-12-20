// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	storage "github.com/foundriesio/dg-satellite/storage/api"
)

type (
	Device            = storage.Device
	DeviceListItem    = storage.DeviceListItem
	DeviceListOpts    = storage.DeviceListOpts
	DeviceUpdateEvent = storage.DeviceUpdateEvent
)

type AppsStatesResp struct {
	AppsStates []storage.AppsStates `json:"apps_states"`
}

// @Summary List devices
// @Param _ query DeviceListOpts false "Sorting options"
// @Accept  json
// @Produce json
// @Success 200 {array} DeviceListItem
// @Router  /devices [get]
func (h *handlers) deviceList(c echo.Context) error {
	opts := storage.DeviceListOpts{
		OrderBy: storage.OrderByDeviceLastSeenDsc,
		Limit:   1000,
		Offset:  0,
	}
	if err := c.Bind(&opts); err != nil {
		return EchoError(c, err, http.StatusBadRequest, "Failed to parse list options")
	}

	devices, err := h.storage.DevicesList(opts)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Unexpected error listing devices")
	}

	// TODO handle pagination in response
	return c.JSON(http.StatusOK, devices)
}

// @Summary Get a device by its UUID
// @Produce json
// @Success 200 Device
// @Router  /devices/:uuid [get]
func (h *handlers) deviceGet(c echo.Context) error {
	return h.handleDevice(c, func(device *Device) error {
		return c.JSON(http.StatusOK, device)
	})
}

// @Summary Get a list of updates for a device
// @Produce json
// @Success 200 []string
// @Router  /devices/:uuid/updates [get]
func (h *handlers) deviceUpdatesList(c echo.Context) error {
	return h.handleDevice(c, func(device *Device) error {
		updates, err := device.Updates()
		if err != nil {
			return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device updates")
		}
		return c.JSON(http.StatusOK, updates)
	})
}

// @Summary Get details of update events for a devices
// @Produce json
// @Success 200 []DeviceUpdateEvent
// @Router  /devices/:uuid/updates/:id [get]
func (h *handlers) deviceUpdatesGet(c echo.Context) error {
	return h.handleDevice(c, func(device *Device) error {
		updateId := c.Param("id")
		events, err := device.Events(updateId)
		if err != nil {
			return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device update events")
		}
		if len(events) == 0 {
			return c.NoContent(http.StatusNotFound)
		}
		return c.JSON(http.StatusOK, events)
	})
}

// @Summary Get a list of Apps states reported by the device
// @Produce json
// @Success 200 AppsStatesResp
// @Router  /devices/:uuid/apps-states [get]
func (h *handlers) deviceAppsStatesGet(c echo.Context) error {
	return h.handleDevice(c, func(device *Device) error {
		appsStates, err := device.AppsStates()
		if err != nil {
			return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device updates")
		}
		return c.JSON(http.StatusOK, AppsStatesResp{AppsStates: appsStates})
	})
}

func (h *handlers) handleDevice(c echo.Context, next func(*Device) error) error {
	uuid := c.Param("uuid")
	if device, err := h.storage.DeviceGet(uuid); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to lookup device")
	} else if device == nil {
		return c.NoContent(http.StatusNotFound)
	} else {
		return next(device)
	}
}
