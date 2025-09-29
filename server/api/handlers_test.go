// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/api"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	t   *testing.T
	api *api.Storage
	e   *echo.Echo
	log *slog.Logger
}

func (c testClient) Do(req *http.Request) *httptest.ResponseRecorder {
	req = req.WithContext(context.CtxWithLog(req.Context(), c.log))
	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)
	return rec
}

func (c testClient) GET(resource string, status int) []byte {
	req := httptest.NewRequest(http.MethodGet, "/tmp", nil)
	rec := c.Do(req)
	require.Equal(c.t, http.StatusOK, rec.Code)
	return rec.Body.Bytes()
}

func NewTestClient(t *testing.T) *testClient {
	tmpDir := t.TempDir()
	fsS, err := storage.NewFs(tmpDir)
	require.Nil(t, err)
	db, err := storage.NewDb(filepath.Join(tmpDir, storage.DbFile))
	require.Nil(t, err)
	apiS, err := api.NewStorage(db, fsS)
	require.Nil(t, err)

	log, err := context.InitLogger("debug")
	require.Nil(t, err)

	e := server.NewEchoServer()
	RegisterHandlers(e, apiS, auth.FakeAuthUser)

	tc := testClient{
		t:   t,
		api: apiS,
		e:   e,
		log: log,
	}
	return &tc
}

func TestApi(t *testing.T) {
	tc := NewTestClient(t)
	data := tc.GET("/tmp", 200)
	require.Equal(t, "OK", string(data))
	tc.GET("/tmp?deny-has-scope=1", 403)
}
