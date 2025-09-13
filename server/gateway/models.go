// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	storage "github.com/foundriesio/dg-satellite/storage/gateway"
)

type (
	Device = storage.Device
)

type NetworkInfo struct {
	Hostname  string `json:"hostname,omitempty"`
	Mac       string `json:"mac,omitempty"`
	LocalIpv4 string `json:"local_ipv4,omitempty"`
}

type EcuInfo struct {
	TargetName string `json:"target"`
	HardwareId string `json:"hwid"`
}

type AppsStates struct {
	DeviceTime string `json:"deviceTime"`
	Ostree     string `json:"ostree"`
	Apps       map[string]struct {
		Uri      string `json:"uri"`
		State    string `json:"state"`
		Services []struct {
			Name     string `json:"name"`
			Hash     string `json:"hash"`
			Health   string `json:"health,omitempty"`
			ImageUri string `json:"image"`
			Logs     string `json:"logs,omitempty"`
			State    string `json:"state"`
			Status   string `json:"status"`
		} `json:"services"`
	} `json:"apps"`
}
