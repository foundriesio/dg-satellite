// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
		Users      []users.User
		ScopesList []string
	}{
		baseCtx:    h.baseCtx(c, "Users", "users"),
		Users:      user,
		ScopesList: users.ScopesAvailable(),
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
func (h *handlers) userScopesUpdate(c echo.Context) error {
	session := CtxGetSession(c.Request().Context())
	// Check this scope here rather than middleware so that we can return a JSON error
	// to the JS caller.
	if !session.User.AllowedScopes.Has(users.ScopeUsersRU) {
		err := fmt.Errorf("user missing required scope: %s", users.ScopeUsersRU)
		return EchoError(c, err, http.StatusForbidden, err.Error())
	}

	type request struct {
		Scopes []string `json:"scopes"`
	}
	var req request
	if err := c.Bind(&req); err != nil {
		return EchoError(c, err, http.StatusBadRequest, "Could not parse request")
	}

	if len(req.Scopes) == 0 {
		err := errors.New("at least one scope is required")
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}

	scopes, err := users.ScopesFromSlice(req.Scopes)
	if err != nil {
		return EchoError(c, err, http.StatusBadRequest, fmt.Sprintf("Invalid scope: %s", err))
	}

	username := c.Param("username")
	user, err := h.users.Get(username)
	if err != nil {
		return h.handleError(c, http.StatusNotFound, err)
	}

	user.AllowedScopes = scopes
	if err := user.Update("Scopes changed by " + session.User.Username); err != nil {
		return h.handleUnexpected(c, err)
	}
	return c.NoContent(http.StatusNoContent)
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
		err := errors.New("token description is required")
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}
	if len(req.Scopes) == 0 {
		err := errors.New("at least one scope is required")
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}

	scopes, err := users.ScopesFromSlice(req.Scopes)
	if err != nil {
		return EchoError(c, err, http.StatusBadRequest, fmt.Sprintf("Invalid scope: %s", err))
	}

	if req.Expires <= 0 || req.Expires > 365 {
		err := errors.New("expires must be between 1 and 365 days")
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}

	expires := time.Now().Add(time.Duration(req.Expires) * 24 * time.Hour)
	tok, err := session.User.GenerateToken(req.Description, expires.Unix(), scopes)
	if err != nil {
		return EchoError(c, err, http.StatusBadRequest, err.Error())
	}
	return c.String(http.StatusCreated, tok.Value)
}

func (h *handlers) userTokenDelete(c echo.Context) error {
	session := CtxGetSession(c.Request().Context())
	tokenIDStr := c.Param("tokenID")
	tokenID, err := strconv.ParseUint(tokenIDStr, 10, 64)
	if err != nil {
		return EchoError(c, err, http.StatusBadRequest, "Invalid token ID format")
	}
	if err := session.User.DeleteToken(int64(tokenID)); err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to delete token")
	}
	return c.NoContent(http.StatusNoContent)
}
