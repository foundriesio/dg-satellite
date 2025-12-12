// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/foundriesio/dg-satellite/storage/users"
	"github.com/labstack/echo/v4"
)

func (h handlers) usersList(c echo.Context) error {
	user, err := h.users.List()
	if err != nil {
		return h.handleUnexpected(c, err)
	}
	ctx := struct {
		baseCtx
		Users []users.User
	}{
		baseCtx: h.baseCtx(c, "Users", "users"),
		Users:   user,
	}
	return h.templates.ExecuteTemplate(c.Response(), "users.html", ctx)
}

func (h handlers) usersAuditLog(c echo.Context) error {
	username := c.Param("username")
	user, err := h.users.Get(username)
	if err != nil {
		return h.handleError(c, http.StatusNotFound, err)
	}

	log, err := user.GetAuditLog()
	if err != nil {
		return h.handleUnexpected(c, err)
	}
	return c.String(http.StatusOK, log)
}

func (h *handlers) userTokenCreate(c echo.Context) error {
	session := CtxGetSession(c.Request().Context())

	type TokenRequest struct {
		Description string   `json:"description"`
		Scopes      []string `json:"scopes"`
		Expires     int      `json:"expires"`
	}
	var req TokenRequest
	if err := c.Bind(&req); err != nil {
		return EchoError(c, err, http.StatusBadRequest, "Could not parse request")
	}

	if len(req.Description) == 0 {
		err := errors.New("Token description is required")
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}
	if len(req.Scopes) == 0 {
		err := errors.New("At least one scope is required")
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}

	scopes, err := users.ScopesFromSlice(req.Scopes)
	if err != nil {
		return EchoError(c, err, http.StatusBadRequest, fmt.Sprintf("Invalid scope: %s", err))
	}

	if req.Expires <= 0 || req.Expires > 365 {
		err := errors.New("Expires must be between 1 and 365 days")
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}

	expires := time.Now().Add(time.Duration(req.Expires) * 24 * time.Hour)
	tok, err := session.User.GenerateToken(req.Description, expires.Unix(), scopes)
	if err != nil {
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}
	return c.String(http.StatusCreated, tok.Value)
}
