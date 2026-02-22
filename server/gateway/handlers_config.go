// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"encoding/json"
	"net/http"

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
	d := CtxGetDevice(ctx)
	configs, err := d.GetConfigs()
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "failed to fetch config")
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
	return c.JSON(http.StatusOK, files)
}
