// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/api"
	gatewayStorage "github.com/foundriesio/dg-satellite/storage/gateway"
)

type testClient struct {
	t   *testing.T
	fs  *storage.FsHandle
	api *storage.Storage
	gw  *gatewayStorage.Storage
	e   *echo.Echo
	log *slog.Logger
}

func (c testClient) Do(req *http.Request) *httptest.ResponseRecorder {
	req = req.WithContext(CtxWithLog(req.Context(), c.log))
	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)
	return rec
}

func (c testClient) GET(resource string, status int, headers ...string) []byte {
	req := httptest.NewRequest(http.MethodGet, resource, nil)
	c.marshalHeaders(headers, req)
	rec := c.Do(req)
	require.Equal(c.t, status, rec.Code)
	return rec.Body.Bytes()
}

func (c testClient) PUT(resource string, status int, data any, headers ...string) []byte {
	req := httptest.NewRequest(http.MethodPut, resource, c.marshalBody(data))
	c.marshalHeaders(headers, req)
	rec := c.Do(req)
	require.Equal(c.t, status, rec.Code)
	return rec.Body.Bytes()
}

func (c testClient) marshalHeaders(headers []string, req *http.Request) {
	require.Zero(c.t, len(headers)%2, "Headers must be a sequence of names and values - even number")
	for i := 0; i < len(headers)/2; i++ {
		req.Header.Add(headers[i*2], headers[i*2+1])
	}
}

func (c testClient) marshalBody(data any) io.Reader {
	if s, ok := data.(string); ok {
		return strings.NewReader(s)
	} else if b, ok := data.([]byte); ok {
		return bytes.NewReader(b)
	} else {
		b, err := json.Marshal(data)
		require.Nil(c.t, err)
		return bytes.NewReader(b)
	}
}

func NewTestClient(t *testing.T) *testClient {
	tmpDir := t.TempDir()
	fsS, err := storage.NewFs(tmpDir)
	require.Nil(t, err)
	db, err := storage.NewDb(filepath.Join(tmpDir, storage.DbFile))
	require.Nil(t, err)
	apiS, err := storage.NewStorage(db, fsS)
	require.Nil(t, err)
	gwS, err := gatewayStorage.NewStorage(db, fsS)
	require.Nil(t, err)

	log, err := context.InitLogger("debug")
	require.Nil(t, err)

	e := server.NewEchoServer()
	RegisterHandlers(e, apiS, auth.FakeAuthUser)

	tc := testClient{
		t:   t,
		fs:  fsS,
		api: apiS,
		gw:  gwS,
		e:   e,
		log: log,
	}
	return &tc
}

func TestApiDeviceList(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/devices?deny-has-scope=1", 403)

	// No devices
	data := tc.GET("/devices", 200)
	require.Equal(t, "[]\n", string(data))

	// two devices with different last seen times
	_, err := tc.gw.DeviceCreate("test-device-1", "pubkey1", true)
	require.Nil(t, err)
	time.Sleep(1 * time.Second)
	_, err = tc.gw.DeviceCreate("test-device-2", "pubkey2", false)
	require.Nil(t, err)

	data = tc.GET("/devices", 200)
	var devices []storage.Device
	require.Nil(t, json.Unmarshal(data, &devices))
	require.Len(t, devices, 2)
	assert.Equal(t, "test-device-2", devices[0].Uuid)
	assert.Equal(t, "test-device-1", devices[1].Uuid)

	// test sorting
	data = tc.GET("/devices?order-by=last-seen-asc", 200)
	require.Nil(t, json.Unmarshal(data, &devices))
	assert.Equal(t, "test-device-1", devices[0].Uuid)
	assert.Equal(t, "test-device-2", devices[1].Uuid)
}

func TestApiDeviceGet(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/devices/foo?deny-has-scope=1", 403)

	_ = tc.GET("/devices/does-not-exist", 404)

	_, err := tc.gw.DeviceCreate("test-device-1", "pubkey1", true)
	require.Nil(t, err)
	_, err = tc.gw.DeviceCreate("test-device-2", "pubkey2", false)
	require.Nil(t, err)

	data := tc.GET("/devices/test-device-1", 200)
	var device storage.Device
	require.Nil(t, json.Unmarshal(data, &device))
	assert.Equal(t, "test-device-1", device.Uuid)
	assert.Equal(t, "pubkey1", device.PubKey)

	data = tc.GET("/devices/test-device-2", 200)
	require.Nil(t, json.Unmarshal(data, &device))
	assert.Equal(t, "test-device-2", device.Uuid)
	assert.Equal(t, "pubkey2", device.PubKey)
}

func TestApiUpdateList(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/updates/ci?deny-has-scope=1", 403)
	tc.GET("/updates/ci/tag?deny-has-scope=1", 403)
	tc.GET("/updates/prod?deny-has-scope=1", 403)
	tc.GET("/updates/prod/tag?deny-has-scope=1", 403)

	tc.GET("/updates/non-prod", 404)
	tc.GET("/updates/non-prod/tag", 404)

	s := func(data []byte) string {
		return strings.TrimSpace(string(data))
	}

	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag1", "update1", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag1", "update2", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag2", "update1", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag2", "update3", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag1", "update2", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag4", "update42", "rollout1", "foo"))

	data := tc.GET("/updates/ci", 200)
	assert.Equal(t, `{"tag1":["update1","update2"],"tag2":["update1","update3"]}`, s(data))
	data = tc.GET("/updates/ci/tag1", 200)
	assert.Equal(t, `{"tag1":["update1","update2"]}`, s(data))
	data = tc.GET("/updates/ci/tag2", 200)
	assert.Equal(t, `{"tag2":["update1","update3"]}`, s(data))
	data = tc.GET("/updates/ci/tag4", 200) // tag not exists
	assert.Equal(t, "{}", s(data))
	data = tc.GET("/updates/prod", 200)
	assert.Equal(t, `{"tag1":["update2"],"tag4":["update42"]}`, s(data))
	data = tc.GET("/updates/prod/tag1", 200)
	assert.Equal(t, `{"tag1":["update2"]}`, s(data))
	data = tc.GET("/updates/prod/tag2", 200) // tag not exists
	assert.Equal(t, "{}", s(data))
	data = tc.GET("/updates/prod/tag4", 200)
	assert.Equal(t, `{"tag4":["update42"]}`, s(data))

	// Synthetic tag validation - create a bad tag on disk - request must still return 404
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("bad^tag", "update42", "rollout1", "foo"))
	tc.GET("/updates/prod/bad^tag", 404)
}

func TestApiRolloutList(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/updates/ci/tag/update/rollouts?deny-has-scope=1", 403)
	tc.GET("/updates/prod/tag/update/rollouts?deny-has-scope=1", 403)

	tc.GET("/updates/non-prod/tag/update/rollouts", 404)

	s := func(data []byte) string {
		return strings.TrimSpace(string(data))
	}

	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag1", "update1", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag1", "update1", "rollout2", "foo"))
	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag2", "update1", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag1", "update2", "rollout4", "foo"))

	data := tc.GET("/updates/ci/tag1/update1/rollouts", 200)
	assert.Equal(t, `["rollout1","rollout2"]`, s(data))
	data = tc.GET("/updates/ci/tag2/update1/rollouts", 200)
	assert.Equal(t, `["rollout1"]`, s(data))
	data = tc.GET("/updates/ci/tag2/update2/rollouts", 200) // update not exists
	assert.Equal(t, "[]", s(data))
	data = tc.GET("/updates/ci/tag3/update1/rollouts", 200) // tag not exists
	assert.Equal(t, "[]", s(data))
	data = tc.GET("/updates/prod/tag1/update2/rollouts", 200)
	assert.Equal(t, `["rollout4"]`, s(data))
	data = tc.GET("/updates/ci/tag2/update2/rollouts", 200) // tag not exists
	assert.Equal(t, "[]", s(data))

	// Synthetic tag/update validation - create a bad tag/update on disk - request must still return 404
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("bad^tag", "update42", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag", "update=bad", "rollout1", "foo"))
	tc.GET("/updates/prod/bad^tag/update42/rollouts", 404)
	tc.GET("/updates/prod/tag/update=bad/rollouts", 404)
}

func TestApiRolloutGet(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/updates/ci/tag/update/rollouts/rolling?deny-has-scope=1", 403)
	tc.GET("/updates/prod/tag/update/rollouts/stones?deny-has-scope=1", 403)

	tc.GET("/updates/non-prod/tag/update/rollouts/rocks", 404)

	s := func(data []byte) string {
		return strings.TrimSpace(string(data))
	}

	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag1", "update1", "rollout1", `{"uuids":["123","xyz"]}`))
	require.Nil(t, tc.fs.Updates.Ci.Rollouts.WriteFile("tag1", "update2", "rollout2", `{"groups":["test","dev"]}`))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag", "update", "rollout", `{"uuids":["uh"],"groups":["oh"]}`))

	data := tc.GET("/updates/ci/tag1/update1/rollouts/rollout1", 200)
	assert.Equal(t, `{"uuids":["123","xyz"]}`, s(data))
	data = tc.GET("/updates/ci/tag1/update2/rollouts/rollout2", 200)
	assert.Equal(t, `{"groups":["test","dev"]}`, s(data))
	tc.GET("/updates/ci/tag1/update2/rollouts/rollout3", 404) // rollout not exists
	tc.GET("/updates/ci/tag1/update3/rollouts/rollout1", 404) // update not exists
	tc.GET("/updates/ci/tag2/update1/rollouts/rollout1", 404) // tag not exists
	data = tc.GET("/updates/prod/tag/update/rollouts/rollout", 200)
	assert.Equal(t, `{"uuids":["uh"],"groups":["oh"]}`, s(data))

	// Synthetic tag/update/rollout validation - create a bad tag/update/rollout on disk - request must still return 404
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("bad^tag", "update42", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag", "update=bad", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag", "update", "omg+", "foo"))
	tc.GET("/updates/prod/bad^tag/update42/rollouts/rollout1", 404)
	tc.GET("/updates/prod/tag/update=bad/rollouts/rollout1", 404)
	tc.GET("/updates/prod/tag/update/rollouts/omg+", 404)
}

func TestApiRolloutPut(t *testing.T) {
	tc := NewTestClient(t)
	tc.PUT("/updates/ci/tag/update/rollouts/rolling?deny-has-scope=1", 403, "{}")
	tc.PUT("/updates/prod/tag/update/rollouts/stones?deny-has-scope=1", 403, "{}")

	tc.PUT("/updates/non-prod/tag/update/rollouts/rocks", 404, "{}")

	tc.PUT("/updates/prod/tag/update/rollouts/rocks", 400, "{")
	tc.PUT("/updates/prod/tag/update/rollouts/rocks", 400, "{}")

	require.Nil(t, tc.fs.Updates.Ci.Ostree.WriteFile("tag1", "update1", "foo", "bar"))
	require.Nil(t, tc.fs.Updates.Prod.Ostree.WriteFile("tag2", "update2", "foo", "bar"))
	d, err := tc.gw.DeviceCreate("ci1", "pubkey1", false)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag1", "", ""))
	d, err = tc.gw.DeviceCreate("ci2", "pubkey1", false)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag1", "", ""))
	d, err = tc.gw.DeviceCreate("ci3", "pubkey1", false)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag2", "", ""))
	d, err = tc.gw.DeviceCreate("prod1", "pubkey2", true)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag2", "", ""))
	d, err = tc.gw.DeviceCreate("prod2", "pubkey2", true)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag2", "", ""))
	d, err = tc.gw.DeviceCreate("prod3", "pubkey2", true)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag2", "", ""))
	d, err = tc.gw.DeviceCreate("prod4", "pubkey2", true)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag3", "", ""))

	require.Nil(t, tc.api.SetGroupName("grp1", []string{"prod3", "prod4"}))

	tc.PUT("/updates/ci/tag1/update1/rollouts/rocks", 202,
		`{"uuids":["ci1","ci2","ci3"]}`, "content-type", "application/json")
	tc.PUT("/updates/ci/tag1/update2/rollouts/rocks", 404,
		`{"uuids":["ci1","ci2"]}`, "content-type", "application/json")
	tc.PUT("/updates/ci/tag1/update1/rollouts/rocks", 409,
		`{"uuids":["ci1"]}`, "content-type", "application/json")
	tc.PUT("/updates/prod/tag2/update2/rollouts/rocks", 202,
		`{"uuids":["prod2"],"groups":["grp1"]}`, "content-type", "application/json")
	tc.PUT("/updates/prod/tag1/update1/rollouts/rocks", 404,
		`{"uuids":["prod2"],"groups":["grp1"]}`, "content-type", "application/json")

	s := func(data []byte) string {
		return strings.TrimSpace(string(data))
	}

	data := tc.GET("/updates/ci/tag1/update1/rollouts/rocks", 200)
	assert.Equal(t, `{"uuids":["ci1","ci2","ci3"],"effective-uuids":["ci1","ci2"]}`, s(data))
	data = tc.GET("/updates/prod/tag2/update2/rollouts/rocks", 200)
	assert.Equal(t, `{"uuids":["prod2"],"groups":["grp1"],"effective-uuids":["prod2","prod3"]}`, s(data))
	dev, err := tc.api.DeviceGet("ci1")
	assert.Nil(t, err)
	assert.Equal(t, "update1", dev.UpdateName)
	dev, err = tc.api.DeviceGet("ci2")
	assert.Nil(t, err)
	assert.Equal(t, "update1", dev.UpdateName)
	dev, err = tc.api.DeviceGet("prod1")
	assert.Nil(t, err)
	assert.Equal(t, "", dev.UpdateName)
	dev, err = tc.api.DeviceGet("prod2")
	assert.Nil(t, err)
	assert.Equal(t, "update2", dev.UpdateName)
	dev, err = tc.api.DeviceGet("prod3")
	assert.Nil(t, err)
	assert.Equal(t, "update2", dev.UpdateName)

	// Synthetic tag/update/rollout validation - create a bad tag/update/rollout on disk - request must still return 404
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("bad^tag", "update42", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag", "update=bad", "rollout1", "foo"))
	require.Nil(t, tc.fs.Updates.Prod.Rollouts.WriteFile("tag", "update", "omg+", "foo"))
	tc.PUT("/updates/prod/bad^tag/update42/rollouts/gogogo", 404, "foo")
	tc.PUT("/updates/prod/tag/update=bad/rollouts/gogogo", 404, "foo")
	tc.PUT("/updates/prod/tag/update/rollouts/omg+", 404, "foo")
}
