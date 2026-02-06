// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
const localPasswordChangeTemplate = "local-password-change.html"

type authConfigLocal struct {
	MinPasswordLength int
	PasswordHistory   int
	PasswordAgeDays   int
}

type localProvider struct {
	commonProvider
	authConfig     *authConfigLocal
	newUserScopes  users.Scopes
	sessionTimeout time.Duration
}

type localProviderUserData struct {
	PasswordTimestamp int64
	PasswordHistory   []string
}

func (p localProvider) Name() string {
	return "local"
}

func (p *localProvider) Configure(e *echo.Echo, userStorage *users.Storage, cfg *storage.AuthConfig) error {
	if err := json.Unmarshal(cfg.Config, &p.authConfig); err != nil {
		return fmt.Errorf("unable to unmarshal local config: %w", err)
	}
	var err error
	p.users = userStorage
	p.renderer = p
	p.sessionTimeout = time.Duration(cfg.SessionTimeoutHours) * time.Hour
	p.newUserScopes, err = users.ScopesFromSlice(cfg.NewUserDefaultScopes)
	if err != nil {
		return fmt.Errorf("unable to parse new user default scopes: %w", err)
	}

	e.POST("/auth/login", p.handleLogin)
	e.POST("/users/:username/password", p.handlePasswordChange)
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

func (p localProvider) GetSession(c echo.Context) (*Session, error) {
	// A user can login and have a valid session. However, if the password has
	// expired due to password ageing, we want to force them to change their
	// password before allowing them to access any other pages.
	// To accomplish this, we check the password age in GetSession, and if
	// the password has expired, we force a password-change page. The only
	// page/handler we allow with an expired password is the password-change handler
	session, err := p.commonProvider.GetSession(c)
	if err != nil || session == nil {
		return session, err
	}

	passwordPage := "/users/" + session.User.Username + "/password"
	if p.authConfig.PasswordAgeDays > 0 && c.Request().URL.Path != passwordPage {
		var localData localProviderUserData
		if err := json.Unmarshal(session.User.AuthProviderData, &localData); err == nil {
			passwordAge := time.Now().Unix() - localData.PasswordTimestamp
			maxAge := int64(p.authConfig.PasswordAgeDays * 24 * 60 * 60)
			if localData.PasswordTimestamp == 0 || passwordAge > maxAge {
				return nil, p.handlePasswordPage(c, session)
			}
		} else {
			slog.Warn("unable to unmarshal auth provider data", "user", session.User.Username, "error", err)
		}
	}

	return session, nil
}

func (p *localProvider) handlePasswordPage(c echo.Context, session *Session) error {
	context := struct {
		Title    string
		Message  string
		User     *users.User
		NavItems []string
	}{
		Title:   "Change Password",
		Message: "Your password has expired. Please choose a new password.",
		User:    session.User,
	}
	return templates.Templates.ExecuteTemplate(c.Response(), localPasswordChangeTemplate, context)
}

func (p *localProvider) handlePasswordChange(c echo.Context) error {
	session, err := p.GetSession(c)
	if err != nil {
		return server.EchoError(c, err, http.StatusInternalServerError, err.Error())
	} else if session == nil {
		err := errors.New("authentication required")
		return server.EchoError(c, err, http.StatusUnauthorized, "authentication required")
	}
	u := session.User

	if u.Username != c.Param("username") {
		err := errors.New("users can only change their own password")
		return server.EchoError(c, err, http.StatusForbidden, err.Error())
	}

	curPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	if curPassword == "" || newPassword == "" {
		return server.EchoError(c, errors.New("missing form values"), http.StatusBadRequest, "Missing form values")
	}
	if curPassword == newPassword {
		return server.EchoError(c, errors.New("new password must be different"), http.StatusBadRequest, "New password must be different from current password")
	}

	if ok, err := PasswordVerify(curPassword, u.Password); err != nil {
		return server.EchoError(c, err, http.StatusInternalServerError, "Internal error verifying password")
	} else if !ok {
		return server.EchoError(c, errors.New("current password is incorrect"), http.StatusBadRequest, "Current password is incorrect")
	}

	if rc, err := p.setPassword(u, newPassword); err != nil {
		return server.EchoError(c, err, rc, err.Error())
	}
	return c.String(http.StatusOK, "")
}

func (p localProvider) setPassword(u *users.User, password string) (int, error) {
	var localData localProviderUserData
	if err := json.Unmarshal(u.AuthProviderData, &localData); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to unmarshal auth provider data: %w", err)
	}

	if p.authConfig.MinPasswordLength > 0 && len(password) < p.authConfig.MinPasswordLength {
		return http.StatusBadRequest, fmt.Errorf("new password must be at least %d characters", p.authConfig.MinPasswordLength)
	}

	hashed, err := PasswordHash(password)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to hash password: %w", err)
	}

	if p.authConfig.PasswordHistory > 0 {
		for _, oldHash := range localData.PasswordHistory {
			if ok, err := PasswordVerify(password, oldHash); err != nil {
				return http.StatusInternalServerError, fmt.Errorf("unable to verify password history: %w", err)
			} else if ok {
				return http.StatusBadRequest, fmt.Errorf("new password cannot be the same as any of the last %d passwords", p.authConfig.PasswordHistory)
			}
		}

		localData.PasswordHistory = append(localData.PasswordHistory, u.Password)
		// Keep only the most recent N-1 passwords in history, since the current password should count as well.
		toRemove := len(localData.PasswordHistory) - (p.authConfig.PasswordHistory - 1)
		if toRemove > 0 {
			localData.PasswordHistory = localData.PasswordHistory[toRemove:]
		}
	}

	u.Password = hashed

	localData.PasswordTimestamp = time.Now().Unix()
	u.AuthProviderData, err = json.Marshal(localData)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to marshal auth provider data: %w", err)
	}

	if err := u.Update("Password changed"); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to update user: %w", err)
	}
	return 0, nil
}

func init() {
	RegisterProvider(&localProvider{})
}
