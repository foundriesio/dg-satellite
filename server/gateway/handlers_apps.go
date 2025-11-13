// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"crypto/rand"
	"net/http"

	"github.com/labstack/echo/v4"
)

// @Summary Get access to apps proxy URL
// @Produce json
// @Success 201 {string}
// @Router  /app-proxy-url [post]
func (h handlers) appsProxyUrl(c echo.Context) error {
	d := CtxGetDevice(c.Request().Context())
	token := rand.Text()[:10] // 10 chars, 5 bits of entropy per char = A big number
	h.tokenCache.Set(token, d.Uuid, 0)
	url := h.url + "/registry?token=" + token
	return c.String(http.StatusCreated, url)
}

func (h handlers) blobHead(c echo.Context) error {
	// kind of a hack, but HEAD requests are sort of dumb
	// the "GET" must still check for 404 which is how we handle this below
	return c.NoContent(200)
}

func (h handlers) blobGet(c echo.Context) error {
	device := CtxGetDevice(c.Request().Context())
	path := device.GetAppsFilePath("blobs/sha256/" + c.Param("hash"))
	return c.File(path)
}
