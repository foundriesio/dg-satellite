// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"errors"
	"net/http"
	"os"
	"regexp"
	"slices"

	"github.com/labstack/echo/v4"

	storage "github.com/foundriesio/dg-satellite/storage/api"
)

type Rollout = storage.Rollout

// @Summary List updates
// @Produce json
// @Success 200 {object} map[string][]string
// @Router  /updates/{prod}/{tag} [get]
func (h *handlers) updateList(c echo.Context) error {
	ctx := c.Request().Context()
	isProd := CtxGetIsProd(ctx)
	tag := c.Param("tag")

	if updates, err := h.storage.ListUpdates(tag, isProd); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to look up updates")
	} else {
		if updates == nil {
			updates = map[string][]string{}
		}
		return c.JSON(http.StatusOK, updates)
	}
}

// @Summary List update rollouts
// @Produce json
// @Success 200 {array} string
// @Router  /updates/{prod}/{tag}/{update}/rollouts [get]
func (h *handlers) rolloutList(c echo.Context) error {
	ctx := c.Request().Context()
	isProd := CtxGetIsProd(ctx)
	tag := c.Param("tag")
	updateName := c.Param("update")

	if rollouts, err := h.storage.ListRollouts(tag, updateName, isProd); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to look up update rollouts")
	} else {
		if rollouts == nil {
			rollouts = []string{}
		}
		return c.JSON(http.StatusOK, rollouts)
	}
}

// @Summary Get update rollout
// @Produce json
// @Success 200 {object} Rollout
// @Router  /updates/{prod}/{tag}/{update}/rollouts/{rollout} [get]
func (h *handlers) rolloutGet(c echo.Context) error {
	ctx := c.Request().Context()
	isProd := CtxGetIsProd(ctx)
	tag := c.Param("tag")
	updateName := c.Param("update")
	rolloutName := c.Param("rollout")

	if rollout, err := h.storage.GetRollout(tag, updateName, rolloutName, isProd); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return EchoError(c, err, http.StatusNotFound, "Not found rollout")
		} else {
			return EchoError(c, err, http.StatusInternalServerError, "Failed to look up update rollout")
		}
	} else {
		return c.JSON(http.StatusOK, rollout)
	}
}

// @Summary Create update rollout
// @Accept json
// @Param data body Rollout true "Rollout data"
// @Produce json
// @Success 202
// @Router  /updates/{prod}/{tag}/{update}/rollouts/{rollout} [put]
func (h *handlers) rolloutPut(c echo.Context) error {
	ctx := c.Request().Context()
	isProd := CtxGetIsProd(ctx)
	tag := c.Param("tag")
	updateName := c.Param("update")
	rolloutName := c.Param("rollout")
	var (
		rollout Rollout
		err     error
	)
	if err = c.Bind(&rollout); err != nil {
		return EchoError(c, err, http.StatusBadRequest, "Bad JSON body")
	}
	if len(rollout.Uuids) == 0 && len(rollout.Groups) == 0 {
		return c.String(http.StatusBadRequest, "Either uuids or groups must be set")
	}
	if len(rollout.Effect) > 0 {
		return c.String(http.StatusBadRequest, "Effective uuids are readonly")
	}

	// Check if update with this name exists
	if updates, err := h.storage.ListUpdates(tag, isProd); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to check if update exists")
	} else if tagUpdates, ok := updates[tag]; !ok || !slices.Contains(tagUpdates, updateName) {
		return c.String(http.StatusNotFound, "Update with this name does not exist")
	}

	// Check if rollout with this name already exists
	if _, err = h.storage.GetRollout(tag, updateName, rolloutName, isProd); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return EchoError(c, err, http.StatusInternalServerError, "Failed to check if rollout exists")
		}
	} else {
		return c.String(http.StatusConflict, "Rollout with this name already exists")
	}

	// TODO: This is not atomic. Improvement would involve a daemon goroutine watching for data corruption.
	if err = h.storage.SaveRollout(tag, updateName, rolloutName, isProd, rollout); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to save rollout to disk")
	}
	// TODO: This may be slow.  Consider spawning a goroutine, probably in a worker pool.
	if rollout.Effect, err = h.storage.SetUpdateName(tag, updateName, rollout.Uuids, rollout.Groups); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to update devices for rollout")
	}
	if err = h.storage.SaveRollout(tag, updateName, rolloutName, isProd, rollout); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to save rollout to disk")
	}
	return c.NoContent(http.StatusAccepted)
}

func validateUpdateParams(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		ctx := req.Context()
		var isProd bool
		if !parseProdParam(c.Param("prod"), &isProd) {
			return c.NoContent(http.StatusNotFound)
		} else if tag := c.Param("tag"); len(tag) > 0 && !validateTag(tag) {
			return echo.NewHTTPError(http.StatusNotFound, "Tag must match a given regexp: "+validTagRegex)
		} else if update := c.Param("update"); len(update) > 0 && !validateUpdate(update) {
			return echo.NewHTTPError(http.StatusNotFound, "Update name must match a given regexp: "+validUpdateRegex)
		} else if rollout := c.Param("rollout"); len(rollout) > 0 && !validateRollout(rollout) {
			return echo.NewHTTPError(http.StatusNotFound, "Rollout name must match a given regexp: "+validRolloutRegex)
		}
		ctx = CtxWithIsProd(ctx, isProd)
		c.SetRequest(req.WithContext(ctx))
		return next(c)
	}
}

const (
	validTagRegex     = `^[a-zA-Z0-9_\-\.\+]+$`
	validUpdateRegex  = `^[a-zA-Z0-9_\-\.]+$`
	validRolloutRegex = validUpdateRegex
)

var (
	validateTag     = regexp.MustCompile(validTagRegex).MatchString
	validateUpdate  = regexp.MustCompile(validUpdateRegex).MatchString
	validateRollout = regexp.MustCompile(validRolloutRegex).MatchString
)

func parseProdParam(param string, isProd *bool) (ok bool) {
	ok = true
	switch param {
	case "prod":
		*isProd = true
	case "ci":
		*isProd = false
	default:
		ok = false
	}
	return
}
