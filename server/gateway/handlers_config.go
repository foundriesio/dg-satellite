// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type ConfigFile struct {
	Value       string
	Unencrypted *bool    `json:"Unencrypted,omitempty"`
	OnChanged   []string `json:"OnChanged,omitempty"`
}

// @Summary Get device's current configuration
// @Produce json
// @Success 200 {object} map[string]ConfigFile
// @Router  /config [get]
func (h handlers) configGet(c echo.Context) error {
	req := c.Request()
	ctx := req.Context()
	log := CtxGetLog(ctx)
	d := CtxGetDevice(ctx)
	configs, timestamp, err := d.GetConfigs()
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "failed to fetch config")
	} else if timestamp == 0 {
		return c.NoContent(http.StatusNoContent)
	}

	cts := time.Unix(timestamp, 0)
	ifModifiedSince := req.Header.Get("If-Modified-Since")
	if len(ifModifiedSince) > 0 {
		if dts, err := time.Parse(time.RFC1123, ifModifiedSince); err != nil {
			log.Warn("Unable to parse If-Modified-Since", "error", err, "if-modified-since", ifModifiedSince)
		} else if dts.Before(cts) {
			return c.String(http.StatusNotModified, "")
		}
	}

	files := make(map[string]ConfigFile)
	for _, rawConfig := range configs {
		var cfg map[string]ConfigFile
		if len(rawConfig) == 0 {
			continue
		} else if err = json.Unmarshal([]byte(rawConfig), &cfg); err != nil {
			return EchoError(c, err, http.StatusInternalServerError, "failed to parse config JSON")
		}
		for k, v := range cfg {
			files[k] = v
		}
	}
	c.Response().Header().Set("Date", cts.Format(time.RFC1123))
	return c.JSON(http.StatusOK, files)
}
