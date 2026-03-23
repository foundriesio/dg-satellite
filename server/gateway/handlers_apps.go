// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"crypto/rand"
	"net/http"
	"path/filepath"
	"strings"

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
	path := c.Param("*")
	if filepath.Clean(path) != path {
		return c.String(http.StatusNotFound, "Not Found")
	}
	hash := parseRegistryHash(c.Param("*"))
	if len(hash) == 0 || strings.Contains(hash, "..") {
		return c.String(http.StatusNotFound, "Not Found")
	}
	device := CtxGetDevice(c.Request().Context())
	path = device.GetAppsFilePath("blobs/sha256/" + hash)
	return c.File(path)
}

// parseRegistryHash extracts the sha256 hash from a registry wildcard path.
// The path is expected to end with /blobs/sha256:<hash> or /manifests/sha256:<hash>,
// where the portion before that can contain slashes (e.g. repo/app/with/slashes/blobs/sha256:<hash>).
func parseRegistryHash(path string) string {
	for _, marker := range []string{"/blobs/sha256:", "/manifests/sha256:"} {
		idx := strings.LastIndex(path, marker)
		if idx != -1 {
			return path[idx+len(marker):]
		}
	}
	return ""
}
