// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h handlers) settings(c echo.Context) error {
	return c.Render(http.StatusOK, "settings.html", h.baseCtx(c, "Settings", "settings"))
}
