// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const CsrfCookieName = "dg-satellite-csrf"
const CsrfHeaderName = "X-CSRF-Token"

// SetCsrfCookie sets a CSRF cookie on the response. It should be called
// when a new session is created. The cookie is NOT HttpOnly so that
// JavaScript can read the value to include in request headers.
func SetCsrfCookie(c echo.Context, expires time.Time) {
	c.SetCookie(&http.Cookie{
		Name:     CsrfCookieName,
		Value:    rand.Text(),
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

// CsrfCheck is a middleware that validates the CSRF token for non-safe HTTP
// methods (anything other than GET, HEAD, OPTIONS). It skips the check for
// requests that use an Authorization header (i.e. API token auth).
func CsrfCheck(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		method := c.Request().Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			return next(c)
		}

		// Skip CSRF check for token-authenticated API requests
		if c.Request().Header.Get("Authorization") != "" {
			return next(c)
		}

		cookie, err := c.Cookie(CsrfCookieName)
		if err != nil || cookie.Value == "" {
			return c.String(http.StatusForbidden, "Missing CSRF cookie")
		}

		headerToken := c.Request().Header.Get(CsrfHeaderName)
		if headerToken == "" {
			headerToken = c.FormValue("_csrf")
		}
		if headerToken == "" {
			return c.String(http.StatusForbidden, "Missing CSRF token")
		}

		if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) != 1 {
			return c.String(http.StatusForbidden, "CSRF token mismatch")
		}

		return next(c)
	}
}
