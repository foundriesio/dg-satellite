// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/labstack/echo/v4"
)

func getJson(ctx context.Context, resource string, result any) error {
	s := CtxGetSession(ctx)

	req, err := http.NewRequest("GET", s.BaseUrl+resource, nil)
	if err != nil {
		return err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			context.CtxGetLog(ctx).Error("unable to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: HTTP_%d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func proxyApi(c echo.Context, resource string) error {
	s := CtxGetSession(c.Request().Context())

	req, err := http.NewRequest(c.Request().Method, s.BaseUrl+resource, c.Request().Body)
	if err != nil {
		return err
	}
	for key, values := range c.Request().Header {
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}

	apiResp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := apiResp.Body.Close(); err != nil {
			context.CtxGetLog(c.Request().Context()).Error("unable to close response body", "error", err)
		}
	}()

	for key, values := range apiResp.Header {
		for _, v := range values {
			c.Response().Header().Add(key, v)
		}
	}
	c.Response().WriteHeader(apiResp.StatusCode)
	_, err = io.Copy(c.Response(), apiResp.Body)
	return err
}
