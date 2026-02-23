// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"fmt"

	models "github.com/foundriesio/dg-satellite/storage/api"
)

type DeviceListItem = models.DeviceListItem
type Device = models.Device
type DeviceUpdateEvent = models.DeviceUpdateEvent

type DeviceApi struct {
	api *Api
}

func (a *Api) Devices() DeviceApi {
	return DeviceApi{
		api: a,
	}
}

// ListPage fetches a single page of devices. It returns the devices and
// whether more pages are available via the Link rel="next" header.
func (d DeviceApi) ListPage(page int, limit int) ([]DeviceListItem, bool, error) {
	offset := page * limit
	resource := fmt.Sprintf("/v1/devices?limit=%d&offset=%d", limit, offset)
	var devices []DeviceListItem
	headers, err := d.api.GetWithHeaders(resource, &devices)
	if err != nil {
		return nil, false, err
	}
	_, hasNext := ParseNextLink(headers.Get("Link"))
	return devices, hasNext, nil
}

func (d *DeviceApi) Get(uuid string) (*Device, error) {
	var device Device
	if err := d.api.Get("/v1/devices/"+uuid, &device); err != nil {
		return nil, err
	}
	return &device, nil
}

func (d *DeviceApi) Updates(uuid string) ([]string, error) {
	var updates []string
	return updates, d.api.Get(fmt.Sprintf("/v1/devices/%s/updates", uuid), &updates)
}

func (d *DeviceApi) UpdateEvents(uuid, updateId string) ([]DeviceUpdateEvent, error) {
	var events []DeviceUpdateEvent
	return events, d.api.Get(fmt.Sprintf("/v1/devices/%s/updates/%s", uuid, updateId), &events)
}
