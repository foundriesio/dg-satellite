// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	storage "github.com/foundriesio/dg-satellite/storage/api"
)

type UpdateTufResp map[string]map[string]any

// @Summary Create an update from a tar or tar+gz stream
// @Accept  application/x-tar,application/gzip
// @Success 201
// @Router  /updates/{prod}/{tag}/{update} [post]
func (h handlers) updateCreate(c echo.Context) error {
	tag := c.Param("tag")
	update := c.Param("update")
	isProd := CtxGetIsProd(c.Request().Context())

	payload := c.Request().Body
	defer payload.Close() //nolint:errcheck

	if err := h.storage.CreateUpdate(tag, update, isProd, payload); err != nil {
		if errors.Is(err, storage.ErrInvalidUpdate) {
			return EchoError(c, err, http.StatusBadRequest, err.Error())
		}
		return EchoError(c, err, http.StatusInternalServerError, "failed to create update")
	}

	return c.NoContent(http.StatusCreated)
}

// @Summary Returns the TUF metadata for the update
// @Produce json
// @Success 200 {object} UpdateTufResp
// @Router  /updates/{prod}/{tag}/{update}/rollouts [get]
func (h handlers) updateGetTuf(c echo.Context) error {
	tag := c.Param("tag")
	update := c.Param("update")
	isProd := CtxGetIsProd(c.Request().Context())

	metas, err := h.storage.GetUpdateTufMetadata(tag, update, isProd)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "failed to get update TUF metadata")
	}

	return c.JSON(http.StatusOK, metas)
}
