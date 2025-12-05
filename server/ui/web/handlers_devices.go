// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"github.com/foundriesio/dg-satellite/server/ui/api"
	"github.com/labstack/echo/v4"
)

func (h handlers) devicesList(c echo.Context) error {
	var devices []api.DeviceListItem
	if err := getJson(c.Request().Context(), "/v1/devices", &devices); err != nil {
		return h.handleUnexpected(c, err)
	}

	ctx := struct {
		baseCtx
		Devices []api.DeviceListItem
	}{
		baseCtx: h.baseCtx(c, "Devices", "devices"),
		Devices: devices,
	}
	return h.templates.ExecuteTemplate(c.Response(), "devices_list.html", ctx)
}
