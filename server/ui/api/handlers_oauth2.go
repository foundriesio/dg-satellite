// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/foundriesio/dg-satellite/storage/users"
)

type oauth2Handlers struct {
	users *users.Storage
}

type deviceCodeRequest struct {
	Scopes       string `json:"scope"` // backward compatabile with lmp-device-register
	TokenExpires int64  `json:"token_expires"`
}

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	Expires                 int64  `json:"expires"`
	Interval                int    `json:"interval"`
}

type deviceTokenRequest struct {
	DeviceCode string `json:"device_code"`
	GrantType  string `json:"grant_type"`
}

type deviceTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Expires     int64  `json:"expires"`
	Scopes      string `json:"scope"`
}

type oauth2Error struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// @Summary Initiate OAuth2 Device Authorization
// @Accept json
// @Param data body deviceCodeRequest true "Device code request"
// @Produce json
// @Success 200
// @Router  /oauth2/device/code [post]
func (h oauth2Handlers) oauth2DeviceCode(c echo.Context) error {
	var req deviceCodeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "invalid_request",
			ErrorDescription: "Invalid request body",
		})
	}

	_, err := users.ScopesFromString(req.Scopes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "invalid_scope",
			ErrorDescription: fmt.Sprintf("Invalid scopes: %v", err),
		})
	}

	if req.TokenExpires <= time.Now().Unix() {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "invalid_request",
			ErrorDescription: "token_expires must be in the future",
		})
	}

	expires := time.Now().Add(10 * time.Minute).Unix()
	deviceCode, userCode, err := h.users.CreateDeviceAuth(expires, req.TokenExpires, req.Scopes)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to create device authorization")
	}

	baseURL := c.Scheme() + "://" + c.Request().Host
	verificationURI := baseURL + "/auth/activate"
	verificationURIComplete := fmt.Sprintf("%s?user_code=%s", verificationURI, userCode)

	return c.JSON(http.StatusOK, deviceCodeResponse{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		Expires:                 expires,
		Interval:                5, // Poll every 5 seconds
	})
}

// oauth2DeviceToken handles device polling for the access token
// POST /oauth2/device/token
func (h oauth2Handlers) oauth2DeviceToken(c echo.Context) error {
	var req deviceTokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "invalid_request",
			ErrorDescription: "Invalid request body",
		})
	}

	if req.GrantType != "urn:ietf:params:oauth:grant-type:device_code" {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "unsupported_grant_type",
			ErrorDescription: "Only device_code grant type is supported",
		})
	}

	if req.DeviceCode == "" {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "invalid_request",
			ErrorDescription: "device_code is required",
		})
	}

	auth, err := h.users.GetDeviceAuthByDeviceCode(req.DeviceCode)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to get authorization")
	}
	if auth == nil {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "invalid_grant",
			ErrorDescription: "Invalid device code",
		})
	}

	if auth.ExpiresAt < time.Now().Unix() {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "expired_token",
			ErrorDescription: "Device code has expired",
		})
	}

	if auth.Denied {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "access_denied",
			ErrorDescription: "User denied the authorization request",
		})
	}

	if !auth.Authorized {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "authorization_pending",
			ErrorDescription: "User has not yet authorized this device",
		})
	}

	if auth.UserID == nil {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "invalid_grant",
			ErrorDescription: "No user associated with this authorization",
		})
	}

	user, err := h.users.GetByID(*auth.UserID)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to get user for authorization")
	}
	if user == nil {
		return c.JSON(http.StatusBadRequest, oauth2Error{
			Error:            "invalid_grant",
			ErrorDescription: "User for authorization not found",
		})
	}

	scopes, err := users.ScopesFromString(auth.Scopes)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to parse scopes")
	}

	if len(auth.TokenDescription) == 0 {
		auth.TokenDescription = "OAuth2 authorization"
	}
	scopes = user.AllowedScopes & scopes
	token, err := user.GenerateToken(auth.TokenDescription, auth.TokenExpires, scopes)
	if err != nil {
		return EchoError(c, err, http.StatusInternalServerError, "Failed to generate token")
	}

	return c.JSON(http.StatusOK, deviceTokenResponse{
		AccessToken: token.Value,
		TokenType:   "Bearer",
		Expires:     auth.TokenExpires,
		Scopes:      scopes.String(),
	})
}
