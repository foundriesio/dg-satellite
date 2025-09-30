// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import "regexp"

// DeviceUpdateEvent represents update events that devices send the
// device-gateway.
type DeviceUpdateEvent struct {
	Id         string          `json:"id"`
	DeviceTime string          `json:"deviceTime"`
	Event      DeviceEvent     `json:"event"`
	EventType  DeviceEventType `json:"eventType"`
}

var ValidCorrelationId = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`).MatchString

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

type DeviceStatus struct {
	Uuid          string `json:"uuid"`
	CorrelationId string `json:"correlationId"`
	TargetName    string `json:"target-name"`
	Status        string `json:"status"`
	DeviceTime    string `json:"deviceTime"`
}

func (e DeviceUpdateEvent) ParseStatus() *DeviceStatus {
	var status string
	switch e.EventType.Id {
	case "MetadataUpdateCompleted":
		if e.Event.Success != nil && !*e.Event.Success {
			status = "Metadata update failed"
		} else {
			status = "Metadata update completed"
		}
	case "EcuDownloadStarted":
		status = "Download started"
	case "EcuDownloadCompleted":
		if e.Event.Success != nil && !*e.Event.Success {
			status = "Download failed"
		} else {
			status = "Download completed"
		}
	case "EcuInstallationStarted":
		status = "Install started"
	case "EcuInstallationApplied":
		status = "Install applied, awaiting update finalization"
	case "EcuInstallationCompleted":
		if e.Event.Success != nil && !*e.Event.Success {
			status = "Install failed"
		} else {
			status = "Install completed"
		}
	}
	if len(status) > 0 {
		return &DeviceStatus{
			CorrelationId: e.Event.CorrelationId,
			TargetName:    e.Event.TargetName,
			Status:        status,
			DeviceTime:    e.DeviceTime,
		}
	} else {
		return nil
	}
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

var TestIdRegex = regexp.MustCompile(`^[A-Za-z0-9\-\_]{15,48}$`)

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
