// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"fmt"

	"github.com/foundriesio/dg-satellite/server/ui/api"
	"github.com/labstack/echo/v4"
)

func (h handlers) updatesList(c echo.Context) error {
	var ci map[string][]string
	if err := getJson(c.Request().Context(), "/v1/updates/ci", &ci); err != nil {
		return h.handleUnexpected(c, err)
	}
	var prod map[string][]string
	if err := getJson(c.Request().Context(), "/v1/updates/prod", &prod); err != nil {
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
	if err := getJson(c.Request().Context(), url, &rollouts); err != nil {
		return h.handleUnexpected(c, err)
	}
	ctx := struct {
		baseCtx
		Tag      string
		Name     string
		Prod     string
		Rollouts []string
	}{
		baseCtx:  h.baseCtx(c, "Update Details", "updates"),
		Tag:      c.Param("tag"),
		Name:     c.Param("name"),
		Prod:     c.Param("prod"),
		Rollouts: rollouts,
	}
	return h.templates.ExecuteTemplate(c.Response(), "update.html", ctx)
}

func (h handlers) updatesRollout(c echo.Context) error {
	url := fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts/%s", c.Param("prod"), c.Param("tag"), c.Param("name"), c.Param("rollout"))

	var details api.Rollout
	if err := getJson(c.Request().Context(), url, &details); err != nil {
		return EchoError(c, err, 500, err.Error())
	}

	ctx := struct {
		baseCtx
		Tag     string
		Name    string
		Prod    string
		Rollout string
		Details api.Rollout
	}{
		baseCtx: h.baseCtx(c, "Rollout Details", "updates"),
		Tag:     c.Param("tag"),
		Name:    c.Param("name"),
		Prod:    c.Param("prod"),
		Rollout: c.Param("rollout"),
		Details: details,
	}
	return h.templates.ExecuteTemplate(c.Response(), "update_rollout.html", ctx)
}

func (h handlers) updatesTail(c echo.Context) error {
	ctx := struct {
		baseCtx
		TailUrl string
	}{
		baseCtx: h.baseCtx(c, "Rollout Progress", "updates"),
		TailUrl: fmt.Sprintf("/v1/updates/%s/%s/%s/tail", c.Param("prod"), c.Param("tag"), c.Param("name")),
	}

	return h.templates.ExecuteTemplate(c.Response(), "update_tail.html", ctx)
}

func (h handlers) updatesRolloutTail(c echo.Context) error {
	ctx := struct {
		baseCtx
		TailUrl string
	}{
		baseCtx: h.baseCtx(c, "Rollout Progress", "updates"),
		TailUrl: fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts/%s/tail", c.Param("prod"), c.Param("tag"), c.Param("name"), c.Param("rollout")),
	}

	return h.templates.ExecuteTemplate(c.Response(), "update_tail.html", ctx)
}
