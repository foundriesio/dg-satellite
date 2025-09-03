// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// @Summary Get server side information on device
// @Produce json
// @Success 200 {object} Device
// @Router  /device [get]
func (handlers) deviceGet(c echo.Context) error {
	d := getDevice(c)
	return c.JSON(http.StatusOK, d)
}
