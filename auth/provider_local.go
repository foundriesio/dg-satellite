// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/server/ui/web/templates"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/users"
	"github.com/labstack/echo/v4"
)

const localLoginTemplate = "local-login.html"

type localProvider struct {
	commonProvider
	newUserScopes  users.Scopes
	sessionTimeout time.Duration
}

func (p localProvider) Name() string {
	return "local"
}

func (p *localProvider) Configure(e *echo.Echo, userStorage *users.Storage, cfg *storage.AuthConfig) error {
	var err error
	p.users = userStorage
	p.renderer = p
	p.sessionTimeout = time.Duration(cfg.SessionTimeoutHours) * time.Hour
	p.newUserScopes, err = users.ScopesFromSlice(cfg.NewUserDefaultScopes)
	if err != nil {
		return fmt.Errorf("unable to parse new user default scopes: %w", err)
	}

	e.POST("/auth/login", p.handleLogin)
	return nil
}

func (p *localProvider) handleLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	user, err := p.users.Get(username)
	if err != nil {
		return server.EchoError(c, err, http.StatusInternalServerError, "Unable to look up user")
	} else if user == nil {
		return p.renderLoginPage(c, "Invalid username or password")
	}

	if ok, err := PasswordVerify(password, user.Password); err != nil {
		return server.EchoError(c, err, http.StatusInternalServerError, "Internal error verifying password")
	} else if !ok {
		return p.renderLoginPage(c, "Invalid username or password")
	}

	expires := time.Now().Add(p.sessionTimeout)
	sessionId, err := user.CreateSession(c.RealIP(), expires.Unix(), user.AllowedScopes)
	if err != nil {
		return server.EchoError(c, err, http.StatusInternalServerError, "Could not create user session")
	}
	c.SetCookie(&http.Cookie{
		Name:     AuthCookieName,
		Value:    sessionId,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	return c.Redirect(http.StatusSeeOther, "/")
}

func (p localProvider) renderLoginPage(c echo.Context, reason string) error {
	accepts := c.Request().Header.Get("Accept")
	if !strings.Contains(accepts, "text/html") {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "authentication required",
		})
	}
	context := struct {
		Title    string
		Reason   string
		User     *users.User
		NavItems []string
	}{
		Title:  "Login",
		Reason: reason,
	}
	return templates.Templates.ExecuteTemplate(c.Response(), localLoginTemplate, context)
}

func init() {
	RegisterProvider(&localProvider{})
}
