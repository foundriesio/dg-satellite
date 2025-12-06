// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"encoding/json"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server/ui/api"
	"github.com/labstack/echo/v4"
)

func (h handlers) devicesList(c echo.Context) error {
	var devices []api.DeviceListItem
	if err := CtxGetJson(c.Request().Context(), "/v1/devices", &devices); err != nil {
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

type ipInfo struct {
	Hostname string `json:"hostname"`
	IP       string `json:"local_ipv4"`
	Mac      string `json:"mac"`
}

func (h handlers) devicesGet(c echo.Context) error {
	var device api.Device
	if err := CtxGetJson(c.Request().Context(), "/v1/devices/"+c.Param("uuid"), &device); err != nil {
		return h.handleUnexpected(c, err)
	}

	var info ipInfo
	infoPtr := &info
	if err := json.Unmarshal([]byte(device.NetInfo), &info); err != nil {
		context.CtxGetLog(c.Request().Context()).Warn("failed to parse device netinfo", "err", err)
		infoPtr = nil
	}

	var hw map[string]any
	if err := json.Unmarshal([]byte(device.HwInfo), &hw); err != nil {
		context.CtxGetLog(c.Request().Context()).Warn("failed to parse device hardware info", "err", err)
	} else {
		indentBytes, err := json.MarshalIndent(hw, "", "  ")
		if err != nil {
			context.CtxGetLog(c.Request().Context()).Warn("failed to re-marshal device hardware info", "err", err)
		} else {
			device.HwInfo = string(indentBytes)
		}
	}

	var updates []string
	if err := CtxGetJson(c.Request().Context(), "/v1/devices/"+c.Param("uuid")+"/updates", &updates); err != nil {
		return h.handleUnexpected(c, err)
	}

	ctx := struct {
		baseCtx
		Device  api.Device
		IpInfo  *ipInfo
		HwInfo  map[string]any
		Updates []string
	}{
		baseCtx: h.baseCtx(c, "Device - "+device.Uuid, "devices"),
		Device:  device,
		IpInfo:  infoPtr,
		HwInfo:  hw,
		Updates: updates,
	}
	return h.templates.ExecuteTemplate(c.Response(), "device.html", ctx)
}
