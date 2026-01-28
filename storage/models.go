// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

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

type DeviceStatus struct {
	Uuid          string `json:"uuid"`
	CorrelationId string `json:"correlationId"`
	TargetName    string `json:"target-name"`
	Status        string `json:"status"`
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
