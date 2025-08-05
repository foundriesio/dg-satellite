// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// normal uuid's are 36 characters. We'll allow 20-48 alphanumeric with dashes and underscores
var testIdRegex = regexp.MustCompile(`^[a-z0-9\-\_]{20,48}$`)

type testCreateBody struct {
	Name   string `json:"name"`
	TestId string `json:"test-id"`
}

// @Summary Create a test
// @Accept  json
// @Param   test body testCreateBody true "Test body"
// @Param   x-ats-target header string true "Target name"
// @Produce plain
// @Success 200 "test-id"
// @Router  /tests [post]
func (h handlers) testCreate(c echo.Context) error {
	d := getDevice(c)
	bytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return ErrResponse(c, http.StatusBadRequest, "Failed to read request body", err)
	}

	target := c.Request().Header.Get("x-ats-target")

	var test testCreateBody
	if err := json.Unmarshal(bytes, &test); err != nil {
		return ErrResponse(c, http.StatusInternalServerError, "Failed to parse request body", err)
	}

	if len(test.TestId) > 0 && !testIdRegex.MatchString(test.TestId) {
		msg := fmt.Sprintf("test-id(%s) must match pattern: %s", test.TestId, testIdRegex.String())
		return ErrResponse(c, http.StatusBadRequest, msg, nil)
	} else if len(test.TestId) == 0 {
		test.TestId = uuid.New().String()
	}
	if err = d.TestCreate(target, test.Name, test.TestId); err != nil {
		return ErrResponse(c, http.StatusInternalServerError, "Failed to save test", err)
	}

	return c.String(http.StatusCreated, test.TestId)
}

type testCompleteBody struct {
	Status    string                     `json:"status"`
	Details   string                     `json:"details"`
	Results   []storage.TargetTestResult `json:"results"`
	Artifacts []string                   `json:"artifacts"`
}

type signedUrl struct {
	Url         string `json:"url"`
	ContentType string `json:"content-type"`
}

// @Summary Complete a test
// @Accept  json
// @Param   test body testCompleteBody true "Test details"
// @Produce json
// @Success 200 {object} map[string]signedUrl
// @Router  /tests/{test-id} [put]
func (h handlers) testComplete(c echo.Context) error {
	ctx := c.Request().Context()
	d := getDevice(c)
	testid := c.Param("testid")

	log := context.CtxGetLog(ctx)
	log = log.With("testid", testid)
	ctx = context.CtxWithLog(ctx, log)
	c.SetRequest(c.Request().WithContext(ctx))

	bytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return ErrResponse(c, http.StatusBadRequest, "Failed to read request body", err)
	}
	var test testCompleteBody
	if err := json.Unmarshal(bytes, &test); err != nil {
		return ErrResponse(c, http.StatusInternalServerError, "Failed to parse request body", err)
	}

	if test.Status == "" {
		test.Status = "PASSED"
	}

	if err = d.TestComplete(testid, test.Status, test.Details, test.Results); err != nil {
		return ErrResponse(c, http.StatusInternalServerError, "Failed to save test", err)
	}

	// NOTE: c.Request().URL doesn't include the base host info of the request
	// so we have use Request().Host
	baseUrl := "https://" + c.Request().Host + c.Request().URL.Path
	urls := make(map[string]signedUrl)
	for _, p := range test.Artifacts {
		urls[p] = signedUrl{
			Url:         baseUrl + "/" + p,
			ContentType: server.GuessContentType(p),
		}
	}

	return c.JSON(http.StatusOK, urls)
}

func (h handlers) testArtifact(c echo.Context) error {
	ctx := c.Request().Context()
	d := getDevice(c)
	testid := c.Param("testid")
	path := c.Param("path")

	log := context.CtxGetLog(ctx)
	log = log.With("testid", testid)
	ctx = context.CtxWithLog(ctx, log)
	c.SetRequest(c.Request().WithContext(ctx))

	if err := d.TestStoreArtifact(testid, path, c.Request().Body); err != nil {
		return ErrResponse(c, http.StatusInternalServerError, "Failed to save test artifact", err)
	}
	return c.String(http.StatusOK, "OK")
}
