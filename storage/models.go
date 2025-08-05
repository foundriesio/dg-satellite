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

type TargetTestResult struct {
	Name    string             `json:"name"`
	Status  string             `json:"status"`
	LocalTs float64            `json:"local_ts"`
	Details string             `json:"details"`
	Metrics map[string]float64 `json:"metrics"`
}

type TargetTest struct {
	Uuid        string             `json:"uuid"`
	Name        string             `json:"name"`
	TargetName  string             `json:"target_name"`
	Status      string             `json:"status"`
	CreatedOn   int64              `json:"created_on"`
	CompletedOn *int64             `json:"completed_on"`
	Details     string             `json:"details,omitempty"`
	Artifacts   []string           `json:"artifacts,omitempty"`
	Results     []TargetTestResult `json:"results,omitempty"`
}
