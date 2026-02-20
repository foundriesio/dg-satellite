// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strings"

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

	var reader io.Reader = c.Request().Body
	defer func() {
		if err := c.Request().Body.Close(); err != nil {
			CtxGetLog(c.Request().Context()).Error("failed to close request body", "error", err)
		}
	}()

	contentType := c.Request().Header.Get("Content-Type")
	if strings.Contains(contentType, "gzip") ||
		c.Request().Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return EchoError(c, err, http.StatusBadRequest, "failed to decompress gzip stream")
		}
		defer func() {
			if err := gz.Close(); err != nil {
				CtxGetLog(c.Request().Context()).Error("failed to close gzip reader", "error", err)
			}
		}()
		reader = gz
	}

	if err := h.storage.CreateUpdate(tag, update, isProd, reader); err != nil {
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
