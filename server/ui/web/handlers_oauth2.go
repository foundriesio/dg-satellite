// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/foundriesio/dg-satellite/storage/users"
	"github.com/labstack/echo/v4"
)

func (h handlers) authDevice(c echo.Context) error {
	userCode := c.QueryParam("user_code")

	data := struct {
		baseCtx
		UserCode string
	}{
		baseCtx:  h.baseCtx(c, "API Activation", "settings"),
		UserCode: userCode,
	}

	return c.Render(http.StatusOK, "device_auth.html", data)
}

func (h handlers) authDeviceConfirm(c echo.Context) error {
	userCode := c.QueryParam("user_code")
	if userCode == "" {
		return EchoError(c, nil, http.StatusBadRequest, "user_code is required")
	}

	auth, err := h.users.GetDeviceAuthByUserCode(userCode)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to get device authorization")
	}
	if auth == nil {
		return EchoError(c, errors.New("Invalid user code"), http.StatusNotFound, "Invalid user code")
	}

	if auth.ExpiresAt < time.Now().Unix() {
		return EchoError(c, errors.New("This authorization code has expired"), http.StatusBadRequest, "This authorization code has expired")
	}
	if auth.Authorized {
		return EchoError(c, errors.New("This device has already been authorized"), http.StatusBadRequest, "This device has already been authorized")
	}
	if auth.Denied {
		return EchoError(c, errors.New("This device has already been denied"), http.StatusBadRequest, "This device has already been denied")
	}

	// Get the current user from session to intersect scopes
	session := CtxGetSession(c.Request().Context())
	if session == nil || session.User == nil {
		return EchoError(c, nil, http.StatusUnauthorized, "Not authenticated")
	}

	scopes, err := users.ScopesFromString(auth.Scopes)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to parse requested scopes: "+auth.Scopes)
	}
	allowedScopes := scopes & session.User.AllowedScopes

	data := struct {
		baseCtx
		UserCode     string
		Scopes       string
		TokenExpires int64
	}{
		baseCtx:      h.baseCtx(c, "API Activation", "settings"),
		UserCode:     userCode,
		Scopes:       allowedScopes.String(),
		TokenExpires: auth.TokenExpires,
	}

	return c.Render(http.StatusOK, "device_auth_confirm.html", data)
}

func (h handlers) authDeviceAuthorize(c echo.Context) error {
	userCode := c.FormValue("user_code")
	if userCode == "" {
		return EchoError(c, nil, http.StatusBadRequest, "user_code is required")
	}
	description := c.FormValue("token_description")

	auth, err := h.users.GetDeviceAuthByUserCode(userCode)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to get device authorization")
	}
	if auth == nil {
		return EchoError(c, errors.New("Invalid user code"), http.StatusNotFound, "Invalid user code")
	}

	if auth.ExpiresAt < time.Now().Unix() {
		return EchoError(c, errors.New("This authorization code has expired"), http.StatusBadRequest, "This authorization code has expired")
	}

	if auth.Authorized {
		return EchoError(c, errors.New("This device has already been authorized"), http.StatusBadRequest, "This device has already been authorized")
	}
	if auth.Denied {
		return EchoError(c, errors.New("The deivce has aleady been denied"), http.StatusBadRequest, "The device has already been denied")
	}

	session := CtxGetSession(c.Request().Context())
	if err := session.User.ApproveAuthorization(auth.DeviceCode, description); err != nil {
		return h.handleUnexpected(c, err)
	}

	data := struct {
		baseCtx
		Message string
	}{
		baseCtx: h.baseCtx(c, "API Activation", "settings"),
		Message: "Activation successful! You can close this window and return to your device.",
	}
	return c.Render(http.StatusOK, "device_auth_success.html", data)
}

// authDeviceDeny handles the user denying the device authorization
// POST /auth/device/deny
func (h handlers) authDeviceDeny(c echo.Context) error {
	userCode := c.FormValue("user_code")
	if userCode == "" {
		return EchoError(c, nil, http.StatusBadRequest, "user_code is required")
	}

	auth, err := h.users.GetDeviceAuthByUserCode(userCode)
	if err != nil {
		return h.handleError(c, http.StatusInternalServerError, errors.New("Failed to get device authorization"))
	}
	if auth == nil {
		return h.handleError(c, http.StatusNotFound, errors.New("Invalid user code"))
	}

	if auth.ExpiresAt < time.Now().Unix() {
		return h.handleError(c, http.StatusBadRequest, errors.New("This authorization code has expired"))
	}

	if auth.Authorized {
		return h.handleError(c, http.StatusBadRequest, errors.New("This device has already been authorized"))
	}
	if auth.Denied {
		return h.handleError(c, http.StatusBadRequest, errors.New("This device has already been denied"))
	}

	session := CtxGetSession(c.Request().Context())
	if err := session.User.DenyDeviceAuth(auth.DeviceCode); err != nil {
		return h.handleUnexpected(c, err)
	}

	data := struct {
		baseCtx
		Message string
	}{
		baseCtx: h.baseCtx(c, "API Activation", "settings"),
		Message: "Activation denied. You can close this window.",
	}
	return c.Render(http.StatusOK, "device_auth_success.html", data)
}
