// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/api"
	"github.com/foundriesio/dg-satellite/storage/gateway"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	t     *testing.T
	api   *api.Storage
	gwapi *gateway.Storage
	e     *echo.Echo
	log   *slog.Logger
}

func (c testClient) Do(req *http.Request) *httptest.ResponseRecorder {
	req = req.WithContext(context.CtxWithLog(req.Context(), c.log))
	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)
	return rec
}

func (c testClient) GET(resource string, status int) []byte {
	req := httptest.NewRequest(http.MethodGet, resource, nil)
	rec := c.Do(req)
	require.Equal(c.t, status, rec.Code)
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
	gwapi, err := gateway.NewStorage(db, fsS)
	require.Nil(t, err)

	log, err := context.InitLogger("debug")
	require.Nil(t, err)

	e := server.NewEchoServer()
	RegisterHandlers(e, apiS, auth.FakeAuthUser)

	tc := testClient{
		t:     t,
		api:   apiS,
		gwapi: gwapi,
		e:     e,
		log:   log,
	}
	return &tc
}

func TestApiList(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/devices?deny-has-scope=1", 403)

	// No devices
	data := tc.GET("/devices", 200)
	require.Equal(t, "[]\n", string(data))

	// two devices with different last seen times
	_, err := tc.gwapi.DeviceCreate("test-device-1", "pubkey1", true)
	require.Nil(t, err)
	time.Sleep(1 * time.Second)
	_, err = tc.gwapi.DeviceCreate("test-device-2", "pubkey2", false)
	require.Nil(t, err)

	data = tc.GET("/devices", 200)
	var devices []api.Device
	require.Nil(t, json.Unmarshal(data, &devices))
	require.Len(t, devices, 2)
	require.Equal(t, "test-device-2", devices[0].Uuid)
	require.Equal(t, "test-device-1", devices[1].Uuid)

	// test sorting
	data = tc.GET("/devices?order-by=last-seen-asc", 200)
	require.Nil(t, json.Unmarshal(data, &devices))
	require.Equal(t, "test-device-1", devices[0].Uuid)
	require.Equal(t, "test-device-2", devices[1].Uuid)
}
