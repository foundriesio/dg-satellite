// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"fmt"

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

func (h handlers) updatesGet(c echo.Context) error {
	url := fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts", c.Param("prod"), c.Param("tag"), c.Param("name"))

	var rollouts []string
	if err := CtxGetJson(c.Request().Context(), url, &rollouts); err != nil {
		return h.handleUnexpected(c, err)
	}
	ctx := struct {
		baseCtx
		Tag      string
		Name     string
		Rollouts []string
	}{
		baseCtx:  h.baseCtx(c, "Update Details", "update"),
		Tag:      c.Param("tag"),
		Name:     c.Param("name"),
		Rollouts: rollouts,
	}
	return h.templates.ExecuteTemplate(c.Response(), "update.html", ctx)
}
