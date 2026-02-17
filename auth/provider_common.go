// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/foundriesio/dg-satellite/storage/users"
	"github.com/labstack/echo/v4"
)

type loginPageRenderer interface {
	renderLoginPage(c echo.Context, reason string) error
}

type commonProvider struct {
	users    *users.Storage
	renderer loginPageRenderer
}

func (p *commonProvider) DropSession(c echo.Context, session *Session) {
	cookie, err := c.Cookie(AuthCookieName)
	if err != nil {
		slog.Warn("unable to read auth cookie", "error", err)
		return
	}
	if err := session.User.DeleteSession(cookie.Value); err != nil {
		slog.Warn("unable to delete session from storage", "cookie", cookie.Value, "error", err)
	}
}

func (p *commonProvider) GetUser(c echo.Context) (*users.User, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if len(authHeader) > 0 {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return nil, fmt.Errorf("invalid authorization header")
		}
		user, err := p.users.GetByToken(parts[1])
		if err != nil {
			slog.Warn("unable to get user by token", "error", err)
			return nil, c.String(http.StatusInternalServerError, "Could not get user by token")
		} else if user == nil {
			return nil, c.String(http.StatusUnauthorized, "Invalid token")
		}
		return user, nil
	}

	session, err := p.GetSession(c)
	if err != nil || session == nil {
		return nil, err
	}
	return session.User, nil
}

func (p *commonProvider) GetSession(c echo.Context) (*Session, error) {
	cookie, err := c.Cookie(AuthCookieName)
	if err != nil {
		return nil, p.renderer.renderLoginPage(c, err.Error())
	} else if len(cookie.Value) == 0 {
		return nil, p.renderer.renderLoginPage(c, "")
	}
	if cookie.Expires.After(time.Now()) {
		return nil, p.renderer.renderLoginPage(c, "Cookie expired")
	}
	sessionID := cookie.Value
	user, err := p.users.GetBySession(sessionID)
	if user != nil {
		session := &Session{
			BaseUrl: c.Scheme() + "://" + c.Request().Host,
			User:    user,
			Client:  newHttpClientWithSessionCookie(cookie),
		}
		return session, nil
	}
	if err != nil {
		return nil, p.renderer.renderLoginPage(c, err.Error())
	}
	return nil, p.renderer.renderLoginPage(c, "")
}
