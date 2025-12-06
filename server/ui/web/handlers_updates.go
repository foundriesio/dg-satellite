// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"github.com/labstack/echo/v4"
)

func (h handlers) updatesList(c echo.Context) error {
	var ci map[string][]string
	if err := CtxGetJson(c.Request().Context(), "/v1/updates/ci", &ci); err != nil {
		return h.handleUnexpected(c, err)
	}
	var prod map[string][]string
	if err := CtxGetJson(c.Request().Context(), "/v1/updates/prod", &prod); err != nil {
		return h.handleUnexpected(c, err)
	}

	ctx := struct {
		baseCtx
		CI   map[string][]string
		Prod map[string][]string
	}{
		baseCtx: h.baseCtx(c, "Updates", "updates"),
		CI:      ci,
		Prod:    prod,
	}
	return h.templates.ExecuteTemplate(c.Response(), "updates.html", ctx)
}
