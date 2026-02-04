// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	models "github.com/foundriesio/dg-satellite/storage/api"
)

type DeviceListItem = models.DeviceListItem

type DeviceApi struct {
	api *Api
}

func (a *Api) Devices() DeviceApi {
	return DeviceApi{
		api: a,
	}
}

func (d DeviceApi) List() ([]DeviceListItem, error) {
	var devices []DeviceListItem
	return devices, d.api.Get("/v1/devices", &devices)
}
