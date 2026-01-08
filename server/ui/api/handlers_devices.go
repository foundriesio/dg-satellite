// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"fmt"
	"net/http"
	"regexp"

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

type LabelsReq struct {
	Upserts map[string]string
	Deletes []string
}

type LabelsPutReq map[string]*string

// @Summary List devices
// @Param _ query DeviceListOpts false "Sorting options"
// @Accept  json
// @Produce json
// @Success 200 {array} DeviceListItem
// @Router  /devices [get]
func (h *handlers) deviceList(c echo.Context) error {
	opts := storage.DeviceListOpts{
		OrderBy: storage.OrderByDeviceNameAsc,
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

// @Summary Patch device labels
// @Accept json
// @Param data body LabelsReq true "Labels to upsert or delete"
// @Success 200
// @Router  /devices/:uuid/labels [patch]
func (h *handlers) deviceLabelsPatch(c echo.Context) error {
	return h.handleDevice(c, func(device *Device) error {
		var labelsReq LabelsReq
		if err := c.Bind(&labelsReq); err != nil {
			return EchoError(c, err, http.StatusBadRequest, "Bad JSON body")
		}
		if labels, err := parseLabels(labelsReq); err != nil {
			return EchoError(c, err, http.StatusBadRequest, err.Error())
		} else if err = h.storage.PatchDeviceLabels(labels, []string{device.Uuid}); err != nil {
			if storage.IsDbError(err, storage.ErrDbConstraintUnique) {
				return EchoError(c, err, http.StatusConflict, "A device with the same 'name' label value already exists")
			}
			return EchoError(c, err, http.StatusInternalServerError, "Failed to update device labels")
		}
		return c.NoContent(http.StatusOK)
	})
}

// @Summary Put device labels
// @Accept json
// @Param data body LabelsPutReq true "Labels to set"
// @Success 200
// @Router  /devices/:uuid/labels [put]
func (h *handlers) deviceLabelsPut(c echo.Context) error {
	return h.handleDevice(c, func(device *Device) error {
		var labels LabelsPutReq
		if err := c.Bind(&labels); err != nil {
			return EchoError(c, err, http.StatusBadRequest, "Bad JSON body")
		}
		if err := validateLabels(labels); err != nil {
			return EchoError(c, err, http.StatusBadRequest, err.Error())
		}

		for k := range device.Labels {
			if _, ok := labels[k]; !ok {
				labels[k] = nil
			}
		}

		if err := h.storage.PatchDeviceLabels(labels, []string{device.Uuid}); err != nil {
			if storage.IsDbError(err, storage.ErrDbConstraintUnique) {
				return EchoError(c, err, http.StatusConflict, "A device with the same 'name' label value already exists")
			}
			return EchoError(c, err, http.StatusInternalServerError, "Failed to update device labels")
		}
		return c.NoContent(http.StatusOK)
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

const (
	// Together with a 2048 limit on total labels JSONB size,
	// these constraints allow at least 24 labels per device (realistic limit is around 60-70).
	maxLabelName  = 20
	maxLabelValue = 60
	// Label names are lowercase only; label values are case-sensitive.
	validLabelNameRegex  = `^[a-z0-9_\-\.]+$`
	validLabelValueRegex = `^[a-zA-Z0-9_\-\.]+$`
)

var (
	validateLabelName  = regexp.MustCompile(validLabelNameRegex).MatchString
	validateLabelValue = regexp.MustCompile(validLabelValueRegex).MatchString
)

func parseLabels(req LabelsReq) (map[string]*string, error) {
	if len(req.Upserts) == 0 && len(req.Deletes) == 0 {
		return nil, fmt.Errorf("at least one label change must be requested")
	}
	labels := make(map[string]*string, len(req.Upserts)+len(req.Deletes))
	for k, v := range req.Upserts {
		labels[k] = &v
	}
	for _, k := range req.Deletes {
		if _, ok := labels[k]; ok {
			return nil, fmt.Errorf("a label %s cannot be both updated and deleted at once", k)
		}
		labels[k] = nil
	}
	if err := validateLabels(labels); err != nil {
		return nil, err
	}
	return labels, nil
}

func validateLabels(labels map[string]*string) error {
	for k, v := range labels {
		switch {
		case len(k) > maxLabelName:
			return fmt.Errorf("label %s exceeds maximum label name limit %d", k, maxLabelName)
		case v != nil && len(*v) > maxLabelValue:
			return fmt.Errorf("label %s exceeds maximum label value limit %d", k, maxLabelValue)
		case !validateLabelName(k):
			return fmt.Errorf("label %s name must match a given regexp: %s", k, validLabelNameRegex)
		case v != nil && !validateLabelValue(*v):
			return fmt.Errorf("label %s value must match a given regexp: %s", k, validLabelValueRegex)
		}
	}
	return nil
}
