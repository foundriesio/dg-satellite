// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package storage

// DeviceUpdateEvent represents update events that devices send the
// device-gateway.
type DeviceUpdateEvent struct {
	Id         string          `json:"id"`
	DeviceTime string          `json:"deviceTime"`
	Event      DeviceEvent     `json:"event"`
	EventType  DeviceEventType `json:"eventType"`
}

type DeviceEvent struct {
	CorrelationId string `json:"correlationId"`
	Ecu           string `json:"ecu"`
	Success       *bool  `json:"success,omitempty"`
	TargetName    string `json:"targetName"`
	Version       string `json:"version"`
	Details       string `json:"details,omitempty"`
}

type DeviceEventType struct {
	Id      string `json:"id"`
	Version int    `json:"version"`
}
