// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	storage "github.com/foundriesio/dg-satellite/storage/api"
)

// @Summary Upload factory/group/device configs from an archive
// @Tags    Config
// @Accept  application/x-tar
// @Success 200
// @Router  /configs [put]
func (h *handlers) configsUpload(c echo.Context) error {
	req := c.Request()

	payload := req.Body
	defer payload.Close() //nolint:errcheck

	var brokenErr *storage.ErrConfigUploadBroken
	if err := h.storage.UploadConfigs(payload); err == nil {
		return c.String(http.StatusOK, "Configs uploaded successfully")
	} else if errors.As(err, &brokenErr) {
		// This is practically impossible.
		// But if it happens - there is a problem at filesystem level, and the user must intervene.
		CtxGetLog(req.Context()).Error("configs folder is broken by upload", "upload", brokenErr.UploadPath, "error", err)
		return c.String(http.StatusServiceUnavailable, fmt.Sprintf(`
Configs upload broke the configs directory.
Neither old nor new configs are now active.
It can be fixed by uploading the same file again.
If an error persists, a problem needs to be fixed manually.
Inspect the contents of '%s' where both the uploaded and backup configs are stored.
One of them should be moved to the configs directory at '%s'.`,
			brokenErr.UploadPath,
			brokenErr.ConfigsPath,
		))

	} else {
		return EchoError(c, err, http.StatusInternalServerError, "Configs upload failed")
	}
}
