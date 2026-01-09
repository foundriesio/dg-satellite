// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/foundriesio/dg-satellite/server/ui/api"
	"github.com/labstack/echo/v4"
)

func (h handlers) updatesList(c echo.Context) error {
	var ci map[string][]string
	if err := getJson(c.Request().Context(), "/v1/updates/ci", &ci); err != nil {
		return h.handleUnexpected(c, err)
	}
	var prod map[string][]string
	if err := getJson(c.Request().Context(), "/v1/updates/prod", &prod); err != nil {
		return h.handleUnexpected(c, err)
	}

	ctx := struct {
		baseCtx
		CI   map[string][]string
		Prod map[string][]string

		FioctlInstalled bool
	}{
		baseCtx: h.baseCtx(c, "Updates", "updates"),
		CI:      ci,
		Prod:    prod,

		FioctlInstalled: isFioctlInstalled(),
	}
	return h.templates.ExecuteTemplate(c.Response(), "updates.html", ctx)
}

func getExpiry(url string, apiKey string) (*time.Time, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("OSF-TOKEN", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch factory root.json: status %d, body: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Signed struct {
			Expires string `json:"expires"`
		} `json:"signed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode factory root.json response: %w", err)
	}

	expiryTime, err := time.Parse(time.RFC3339, data.Signed.Expires)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root.json expiry: %s: %w", data.Signed.Expires, err)
	}

	return &expiryTime, nil
}

func (h handlers) updatesGetMaxExpiryDays(c echo.Context) error {
	factory := c.QueryParam("factory")
	apiKey := c.QueryParam("apiKey")

	if factory == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "factory query parameter is missing"})
	}

	if apiKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "apiKey query parameter is missing"})
	}

	url := fmt.Sprintf("https://api.foundries.io/ota/repo/%s/api/v1/user_repo/root.json", factory)
	rootExpiry, err := getExpiry(url, apiKey)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get root.json expiry: %v", err))
	}

	url = fmt.Sprintf("https://api.foundries.io/ota/repo/%s/api/v1/user_repo/targets.json", factory)
	targetsExpiry, err := getExpiry(url, apiKey)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get targets.json expiry: %v", err))
	}
	expiry := *targetsExpiry
	if rootExpiry.Before(*targetsExpiry) {
		expiry = *rootExpiry
	}

	duration := time.Until(expiry)
	expiresInDays := int(duration.Hours() / 24)
	return c.String(http.StatusOK, fmt.Sprintf("%d", expiresInDays))
}

func (h handlers) updatesGet(c echo.Context) error {
	url := fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts", c.Param("prod"), c.Param("tag"), c.Param("name"))

	var rollouts []string
	if err := getJson(c.Request().Context(), url, &rollouts); err != nil {
		return h.handleUnexpected(c, err)
	}
	ctx := struct {
		baseCtx
		Tag      string
		Name     string
		Prod     string
		Rollouts []string
	}{
		baseCtx:  h.baseCtx(c, "Update Details", "updates"),
		Tag:      c.Param("tag"),
		Name:     c.Param("name"),
		Prod:     c.Param("prod"),
		Rollouts: rollouts,
	}
	return h.templates.ExecuteTemplate(c.Response(), "update.html", ctx)
}

func (h handlers) updatesRollout(c echo.Context) error {
	url := fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts/%s", c.Param("prod"), c.Param("tag"), c.Param("name"), c.Param("rollout"))

	var details api.Rollout
	if err := getJson(c.Request().Context(), url, &details); err != nil {
		return EchoError(c, err, 500, err.Error())
	}

	ctx := struct {
		baseCtx
		Tag     string
		Name    string
		Prod    string
		Rollout string
		Details api.Rollout
	}{
		baseCtx: h.baseCtx(c, "Rollout Details", "updates"),
		Tag:     c.Param("tag"),
		Name:    c.Param("name"),
		Prod:    c.Param("prod"),
		Rollout: c.Param("rollout"),
		Details: details,
	}
	return h.templates.ExecuteTemplate(c.Response(), "update_rollout.html", ctx)
}

func (h handlers) updatesTail(c echo.Context) error {
	ctx := struct {
		baseCtx
		TailUrl string
	}{
		baseCtx: h.baseCtx(c, "Rollout Progress", "updates"),
		TailUrl: fmt.Sprintf("/v1/updates/%s/%s/%s/tail", c.Param("prod"), c.Param("tag"), c.Param("name")),
	}

	return h.templates.ExecuteTemplate(c.Response(), "update_tail.html", ctx)
}

func (h handlers) updatesRolloutTail(c echo.Context) error {
	ctx := struct {
		baseCtx
		TailUrl string
	}{
		baseCtx: h.baseCtx(c, "Rollout Progress", "updates"),
		TailUrl: fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts/%s/tail", c.Param("prod"), c.Param("tag"), c.Param("name"), c.Param("rollout")),
	}

	return h.templates.ExecuteTemplate(c.Response(), "update_tail.html", ctx)
}

func isFioctlInstalled() bool {
	_, err := exec.LookPath("fioctl")
	return err == nil
}
