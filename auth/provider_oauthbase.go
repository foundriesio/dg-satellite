// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/foundriesio/dg-satellite/server/ui/web/templates"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/users"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

type authConfigOauth2 struct {
	ClientID     string
	ClientSecret string
	BaseUrl      string
}

type oauth2BaseProvider struct {
	name        string
	displayName string

	checkToken func(echo.Context, *oauth2.Token) (*users.User, error)

	newUserScopes  users.Scopes
	oauthConfig    *oauth2.Config
	users          *users.Storage
	loginTip       string
	sessionTimeout time.Duration
}

func (p oauth2BaseProvider) Name() string {
	return p.name
}

func (p *oauth2BaseProvider) configure(e *echo.Echo, usersStorage *users.Storage, cfg *storage.AuthConfig) error {
	if cfg.Type != p.Name() {
		return fmt.Errorf("invalid config type for %s provider: %s", p.Name(), cfg.Type)
	}

	var cfgOauth authConfigOauth2
	if err := json.Unmarshal(cfg.Config, &cfgOauth); err != nil {
		return fmt.Errorf("unable to unmarshal oauth2 config: %w", err)
	}

	var err error
	p.newUserScopes, err = users.ScopesFromSlice(cfg.NewUserDefaultScopes)
	if err != nil {
		return fmt.Errorf("unable to parse new user default scopes: %w", err)
	}
	p.users = usersStorage
	p.sessionTimeout = time.Duration(cfg.SessionTimeoutHours) * time.Hour

	e.GET(AuthLoginPath, p.handleLogin)
	e.GET(AuthCallbackPath, p.handleOauthCallback)
	return nil
}

func (p *oauth2BaseProvider) DropSession(c echo.Context, session *Session) {
	cookie, err := c.Cookie(AuthCookieName)
	if err != nil {
		slog.Warn("unable to read auth cookie", "error", err)
		return
	}
	if err := session.User.DeleteSession(cookie.Value); err != nil {
		slog.Warn("unable to delete session from storage", "cookie", cookie.Value, "error", err)
	}
}

func (p oauth2BaseProvider) GetUser(c echo.Context) (*users.User, error) {
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

func (p oauth2BaseProvider) GetSession(c echo.Context) (*Session, error) {
	cookie, err := c.Cookie(AuthCookieName)
	if err != nil {
		return nil, p.renderLoginPage(c, err.Error())
	} else if len(cookie.Value) == 0 {
		return nil, p.renderLoginPage(c, "")
	}
	if cookie.Expires.After(time.Now()) {
		return nil, p.renderLoginPage(c, "Cookie expired")
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
		return nil, p.renderLoginPage(c, err.Error())
	}
	return nil, p.renderLoginPage(c, "")
}

func (p oauth2BaseProvider) renderLoginPage(c echo.Context, reason string) error {
	accepts := c.Request().Header.Get("Accept")
	if !strings.Contains(accepts, "text/html") {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "authentication required",
		})
	}
	context := struct {
		Title    string
		LoginTip string
		Name     string
		Reason   string
	}{
		Title:    "Login",
		LoginTip: p.loginTip,
		Name:     p.displayName,
		Reason:   reason,
	}
	return templates.Templates.ExecuteTemplate(c.Response(), "oauth2-login.html", context)
}

func (p oauth2BaseProvider) handleLogin(c echo.Context) error {
	oauthState := generateStateOauthCookie(c)
	u := p.oauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)
	return c.Redirect(http.StatusTemporaryRedirect, u)
}

func (p oauth2BaseProvider) handleOauthCallback(c echo.Context) error {
	oauthState, err := c.Cookie("dg-oauthstate")
	if err != nil {
		return c.String(http.StatusBadRequest, "Could not read oauth cookie")
	}

	if c.FormValue("state") != oauthState.Value {
		return c.String(http.StatusBadRequest, "Invalid oauth state")
	}

	code := c.Request().URL.Query().Get("code")
	if code == "" {
		return c.String(http.StatusBadRequest, "Missing authorization code")
	}

	token, err := p.oauthConfig.Exchange(c.Request().Context(), code)
	if err != nil {
		slog.Warn("could not exchange code for token", "error", err)
		return c.String(http.StatusBadRequest, "Could not exchange code for token")
	}

	user, err := p.checkToken(c, token)
	if err != nil || user == nil {
		return err
	}

	expires := time.Now().Add(p.sessionTimeout)
	sessionId, err := user.CreateSession(c.RealIP(), expires.Unix(), user.AllowedScopes)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Could not create user session")
	}
	c.SetCookie(&http.Cookie{
		Name:    AuthCookieName,
		Value:   sessionId,
		Path:    "/",
		Expires: expires,
	})

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func generateStateOauthCookie(c echo.Context) string {
	expiration := time.Now().Add(1 * time.Hour)
	state := rand.Text()
	c.SetCookie(&http.Cookie{
		Name:    "dg-oauthstate",
		Value:   state,
		Expires: expiration,
	})
	return state
}
