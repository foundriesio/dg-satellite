// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
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
	"github.com/foundriesio/dg-satellite/storage/dg"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	t     *testing.T
	api   *api.Storage
	dgApi *dg.Storage
	fs    *storage.FsHandle
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
	db, err := storage.NewDb(filepath.Join(tmpDir, "db.sqlite"))
	require.Nil(t, err)
	apiS, err := api.NewStorage(db, fsS)
	require.Nil(t, err)
	dgApi, err := dg.NewStorage(db, fsS)
	require.Nil(t, err)

	log, err := context.InitLogger("debug")
	require.Nil(t, err)

	e := server.NewEchoServer("api-test")
	RegisterHandlers(e, apiS, auth.FakeAuthUser)

	tc := testClient{
		t:     t,
		api:   apiS,
		dgApi: dgApi,
		fs:    fsS,
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
	_, err := tc.dgApi.DeviceCreate("test-device-1", "pubkey1", true)
	require.Nil(t, err)
	time.Sleep(1 * time.Second)
	_, err = tc.dgApi.DeviceCreate("test-device-2", "pubkey2", false)
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

func TestApiGet(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/devices/foo?deny-has-scope=1", 403)

	_ = tc.GET("/devices/does-not-exist", 404)

	_, err := tc.dgApi.DeviceCreate("test-device-1", "pubkey1", true)
	require.Nil(t, err)
	_, err = tc.dgApi.DeviceCreate("test-device-2", "pubkey2", false)
	require.Nil(t, err)

	data := tc.GET("/devices/test-device-1", 200)
	var device api.Device
	require.Nil(t, json.Unmarshal(data, &device))
	require.Equal(t, "test-device-1", device.Uuid)
	require.Equal(t, "pubkey1", device.PubKey)

	data = tc.GET("/devices/test-device-2", 200)
	require.Nil(t, json.Unmarshal(data, &device))
	require.Equal(t, "test-device-2", device.Uuid)
	require.Equal(t, "pubkey2", device.PubKey)

	// Test sys-info files
	require.Nil(t, tc.fs.WriteFile("test-device-1", storage.Aktoml, []byte("test-aktoml")))
	require.Nil(t, tc.fs.WriteFile("test-device-1", storage.NetInfo, []byte("netinfo")))
	require.Nil(t, tc.fs.WriteFile("test-device-1", storage.HwInfo, []byte("lshw")))
	data = tc.GET("/devices/test-device-1", 200)
	require.Nil(t, json.Unmarshal(data, &device))
	require.Equal(t, "test-aktoml", device.Aktoml)
	require.Equal(t, "netinfo", device.NetInfo)
	require.Equal(t, "lshw", device.HwInfo)
}
