// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"net/http"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/labstack/echo/v4"
)

// @Summary Get device tests
// @Produce json
// @Success 200 []storage.TargetTest
// @Router  /devices/:uuid/tests [get]
func (h *handlers) deviceTestsList(c echo.Context) error {
	uuid := c.Param("uuid")

	device, err := h.storage.DeviceGet(uuid)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	if device == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Not found")
	}

	deviceTests, err := device.Tests()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, deviceTests)
}

// @Summary Get device test
// @Produce json
// @Success 200 storage.TargetTest
// @Router  /devices/:uuid/tests/:testid [get]
func (h *handlers) deviceTestGet(c echo.Context) error {
	uuid := c.Param("uuid")
	testid := c.Param("testid")

	device, err := h.storage.DeviceGet(uuid)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	if device == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Not found")
	}

	test, err := device.Test(testid)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	} else if test == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Test not found")
	}

	return c.JSON(http.StatusOK, test)
}

func (h *handlers) deviceTestArtifact(c echo.Context) error {
	uuid := c.Param("uuid")
	testid := c.Param("testid")
	artifact := c.Param("artifact")

	device, err := h.storage.DeviceGet(uuid)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	fd, err := device.TestArtifact(testid, artifact)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	} else if fd == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Test/Artifact not found")
	}
	defer func() {
		if err := fd.Close(); err != nil {
			log := context.CtxGetLog(c.Request().Context())
			log.Error("error closing test artifact file descriptor", "errorr", err)
		}
	}()
	return c.Stream(http.StatusOK, server.GuessContentType(artifact), fd)
}
