// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// @Summary Create update events
// @Produce json
// @Success 200 {object} DockerCreds
// @Router  /hub-creds [get]
func (h handlers) dockerCreds(c echo.Context) error {
	ctx := c.Request().Context()
	d := CtxGetDevice(ctx)
	// All we need as a token data is a device UUID, so that the docker auth part can check the device validity.
	// As a token is encrypted, opaques, and expires; there is no need for any other data.
	if token, err := h.auth.NewToken(d.Uuid, 8*time.Hour); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to generate auth token")
	} else {
		creds := DockerCreds{Username: d.Uuid, Secret: token}
		return c.JSON(http.StatusOK, creds)
	}
}
