// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear
package gateway

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// @Summary Get a device's fioconfig configuration
// @Produce json
// @Success 200 {object} storage.FioconfigFiles
// @Success 304
// @Router  /config [get]
func (handlers) configGet(c echo.Context) error {
	d := CtxGetDevice(c.Request().Context())

	modTime, err := d.ConfigModTime()
	if err == nil {
		// get in UTC and truncate to second precision for comparison with If-Modified-Since header
		modTime = modTime.UTC().Truncate(time.Second)
		if ims := c.Request().Header.Get("If-Modified-Since"); ims != "" {
			if t, err := time.Parse(time.RFC1123, ims); err == nil {
				t = t.UTC().Truncate(time.Second)
				if !modTime.After(t) || modTime.Equal(t) {
					return c.NoContent(http.StatusNotModified)
				}
			}
		}
	}

	config, err := d.Config()
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to read device config")
	}
	c.Response().Header().Set("Date", modTime.Format(time.RFC1123))
	return c.JSON(http.StatusOK, config)
}

// @Summary Patch a device's fioconfig configuration
// @Success 201
// @Router  /config [patch]
func (handlers) configPatch(c echo.Context) error {
	return c.NoContent(http.StatusCreated)
}
