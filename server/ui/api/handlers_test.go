// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/foundriesio/dg-satellite/server/ui/daemons"
	"github.com/foundriesio/dg-satellite/storage"
	apiStorage "github.com/foundriesio/dg-satellite/storage/api"
	gatewayStorage "github.com/foundriesio/dg-satellite/storage/gateway"
)

func generateUpdateEvents(corId, pack string, num int) []storage.DeviceUpdateEvent {
	events := make([]storage.DeviceUpdateEvent, num)
	for i := 0; i < num; i++ {
		events[i] = storage.DeviceUpdateEvent{
			Id:         fmt.Sprintf("%d_%s", i, corId),
			DeviceTime: "2023-12-12T12:00:00",
			Event: storage.DeviceEvent{
				CorrelationId: corId,
				Ecu:           "",
				Success:       nil,
				TargetName:    "intel-corei7-64-lmp-23",
				Version:       "23",
				Details:       pack,
			},
			EventType: storage.DeviceEventType{
				Id:      "EcuDownloadStarted",
				Version: 0,
			},
		}
	}
	return events
}

type testClient struct {
	t   *testing.T
	ctx Context
	fs  *apiStorage.FsHandle
	api *apiStorage.Storage
	gw  *gatewayStorage.Storage
	e   *echo.Echo
}

func (c testClient) Do(req *http.Request) *httptest.ResponseRecorder {
	req = req.WithContext(c.ctx)
	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)
	return rec
}

func (c testClient) DoAsync(req *http.Request, done chan<- bool) *httptest.ResponseRecorder {
	req = req.WithContext(c.ctx)
	rec := httptest.NewRecorder()
	go func() {
		c.e.ServeHTTP(rec, req)
		if done != nil {
			done <- true
			close(done)
		}
	}()
	return rec
}

func (c testClient) assertDone(done <-chan bool) {
	select {
	case <-done:
		break
	default:
		require.Fail(c.t, "Must be done")
	}
}

func (c testClient) assertNotDone(done <-chan bool) {
	select {
	case <-done:
		require.Fail(c.t, "Must be not done")
	default:
		break
	}
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
	ctx := context.Background()
	tmpDir := t.TempDir()
	fsS, err := apiStorage.NewFs(tmpDir)
	require.Nil(t, err)
	db, err := apiStorage.NewDb(filepath.Join(tmpDir, apiStorage.DbFile))
	require.Nil(t, err)
	apiS, err := apiStorage.NewStorage(db, fsS)
	require.Nil(t, err)
	gwS, err := gatewayStorage.NewStorage(db, fsS)
	require.Nil(t, err)

	log, err := context.InitLogger("debug")
	require.Nil(t, err)
	ctx = CtxWithLog(ctx, log)

	e := server.NewEchoServer()
	RegisterHandlers(e, apiS, auth.FakeAuthUser)

	tc := testClient{
		t:   t,
		ctx: ctx,
		fs:  fsS,
		api: apiS,
		gw:  gwS,
		e:   e,
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
	var devices []apiStorage.Device
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
	var device apiStorage.Device
	require.Nil(t, json.Unmarshal(data, &device))
	assert.Equal(t, "test-device-1", device.Uuid)
	assert.Equal(t, "pubkey1", device.PubKey)

	data = tc.GET("/devices/test-device-2", 200)
	require.Nil(t, json.Unmarshal(data, &device))
	assert.Equal(t, "test-device-2", device.Uuid)
	assert.Equal(t, "pubkey2", device.PubKey)
}

func TestApiDeviceUpdateEvents(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/devices/foo/updates?deny-has-scope=1", 403)

	_ = tc.GET("/devices/updates/does-not-exist", 404)

	d, err := tc.gw.DeviceCreate("test-device-1", "pubkey1", true)
	require.Nil(t, err)

	data := tc.GET("/devices/test-device-1/updates", 200)
	var updates []string
	require.Nil(t, json.Unmarshal(data, &updates))
	require.Len(t, updates, 0)

	events := generateUpdateEvents("uuid-1", "first", 2)
	require.Nil(t, d.ProcessEvents(events))
	events = generateUpdateEvents("uuid-2", "second", 3)
	require.Nil(t, d.ProcessEvents(events))

	data = tc.GET("/devices/test-device-1/updates", 200)
	require.Nil(t, json.Unmarshal(data, &updates))
	require.Len(t, updates, 2)

	data = tc.GET("/devices/test-device-1/updates/"+updates[1], 200)
	require.Nil(t, json.Unmarshal(data, &events))
	assert.Equal(t, "second", events[0].Event.Details)

	data = tc.GET("/devices/test-device-1/updates/"+updates[0], 200)
	require.Nil(t, json.Unmarshal(data, &events))
	assert.Equal(t, "first", events[0].Event.Details)

	_ = tc.GET("/devices/test-device-1/updates/doesnoexist", 404)
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
	assert.Equal(t, `{"uuids":["123","xyz"],"committed":false}`, s(data))
	data = tc.GET("/updates/ci/tag1/update2/rollouts/rollout2", 200)
	assert.Equal(t, `{"groups":["test","dev"],"committed":false}`, s(data))
	tc.GET("/updates/ci/tag1/update2/rollouts/rollout3", 404) // rollout not exists
	tc.GET("/updates/ci/tag1/update3/rollouts/rollout1", 404) // update not exists
	tc.GET("/updates/ci/tag2/update1/rollouts/rollout1", 404) // tag not exists
	data = tc.GET("/updates/prod/tag/update/rollouts/rollout", 200)
	assert.Equal(t, `{"uuids":["uh"],"groups":["oh"],"committed":false}`, s(data))

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
	d, err = tc.gw.DeviceCreate("ci4", "pubkey1", false)
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

	require.Nil(t, tc.api.SetGroupName("grp1", []string{"prod3", "prod4", "ci4"}))

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
	time.Sleep(50 * time.Millisecond) // Allow async database updates to finish

	data := tc.GET("/updates/ci/tag1/update1/rollouts/rocks", 200)
	assert.Equal(t, `{"uuids":["ci1","ci2","ci3"],"effective-uuids":["ci1","ci2"],"committed":true}`, s(data))
	data = tc.GET("/updates/prod/tag2/update2/rollouts/rocks", 200)
	assert.Equal(t, `{"uuids":["prod2"],"groups":["grp1"],"effective-uuids":["prod2","prod3"],"committed":true}`, s(data))
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

func TestApiRolloutDaemon(t *testing.T) {
	tc := NewTestClient(t)
	daemons := daemons.New(tc.ctx, tc.api, daemons.WithRolloverInterval(20*time.Millisecond))
	daemons.Start()
	defer daemons.Shutdown()

	require.Nil(t, tc.fs.Updates.Ci.Ostree.WriteFile("tag1", "update1", "foo", "bar"))
	require.Nil(t, tc.fs.Updates.Prod.Ostree.WriteFile("tag2", "update2", "foo", "bar"))
	d, err := tc.gw.DeviceCreate("ci1", "pubkey1", false)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag1", "", ""))
	d, err = tc.gw.DeviceCreate("prod1", "pubkey2", true)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag2", "", ""))

	s := func(data []byte) string {
		return strings.TrimSpace(string(data))
	}

	// Emulate a non-committed rollout (file present, database not updated).
	require.Nil(t, tc.api.CreateRollout("tag1", "update1", "roll1", false, Rollout{Uuids: []string{"ci1"}}))
	require.Nil(t, tc.api.CreateRollout("tag2", "update2", "roll2", true, Rollout{Uuids: []string{"prod1"}}))

	// Before the watchdog daemon processing, rollouts are not yet committed.
	data := tc.GET("/updates/ci/tag1/update1/rollouts/roll1", 200)
	assert.Equal(t, `{"uuids":["ci1"],"committed":false}`, s(data))
	data = tc.GET("/updates/prod/tag2/update2/rollouts/roll2", 200)
	assert.Equal(t, `{"uuids":["prod1"],"committed":false}`, s(data))
	dev, err := tc.api.DeviceGet("ci1")
	assert.Nil(t, err)
	assert.Equal(t, "", dev.UpdateName)
	dev, err = tc.api.DeviceGet("prod1")
	assert.Nil(t, err)
	assert.Equal(t, "", dev.UpdateName)

	// After the watchdog daemon processing, rollouts are committed.
	time.Sleep(50 * time.Millisecond)
	data = tc.GET("/updates/ci/tag1/update1/rollouts/roll1", 200)
	assert.Equal(t, `{"uuids":["ci1"],"effective-uuids":["ci1"],"committed":true}`, s(data))
	data = tc.GET("/updates/prod/tag2/update2/rollouts/roll2", 200)
	assert.Equal(t, `{"uuids":["prod1"],"effective-uuids":["prod1"],"committed":true}`, s(data))
	dev, err = tc.api.DeviceGet("ci1")
	assert.Nil(t, err)
	assert.Equal(t, "update1", dev.UpdateName)
	dev, err = tc.api.DeviceGet("prod1")
	assert.Nil(t, err)
	assert.Equal(t, "update2", dev.UpdateName)
}

func TestApiUpdateTail(t *testing.T) {
	tc := NewTestClient(t)
	tc.GET("/updates/prod/tag1/update1/tail?deny-has-scope=1", 403)

	d, err := tc.gw.DeviceCreate("test-device-1", "pubkey1", true)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag1", "", ""))
	d, err = tc.gw.DeviceCreate("test-device-2", "pubkey1", true)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag1", "", ""))
	d, err = tc.gw.DeviceCreate("test-device-3", "pubkey1", true)
	require.Nil(t, err)
	require.Nil(t, d.CheckIn("", "tag1", "", ""))
	_, err = tc.api.SetUpdateName("tag1", "update1", true, []string{"test-device-1", "test-device-2"}, nil)
	require.Nil(t, err)

	d1, err := tc.gw.DeviceGet("test-device-1")
	require.Nil(t, err)
	d2, err := tc.gw.DeviceGet("test-device-2")
	require.Nil(t, err)
	d3, err := tc.gw.DeviceGet("test-device-3")
	require.Nil(t, err)

	// Emulate a real HTTP client holding connection - something a test client apparently does not do.
	ctx, cancel := context.WithCancel(tc.ctx)
	tc.ctx = ctx

	// Before any events appear, check the correct error event is received.
	done := make(chan bool)
	rec := tc.DoAsync(httptest.NewRequest(http.MethodGet, "/updates/prod/tag1/update1/tail", nil), done)
	time.Sleep(10 * time.Millisecond)
	expectedStream := `event: error
id: 0
retry: 1000
data: No rollout logs for this update yet.

`
	require.Equal(t, 200, rec.Code)
	require.Equal(t, expectedStream, rec.Body.String())
	tc.assertDone(done)

	events := generateUpdateEvents("uuid-1", "first", 1)
	require.Nil(t, d1.ProcessEvents(events))
	events = generateUpdateEvents("uuid-2", "second", 1)
	require.Nil(t, d2.ProcessEvents(events))
	events = generateUpdateEvents("uuid-3", "third", 1)
	require.Nil(t, d3.ProcessEvents(events))

	// Check that the original response did not change, meaning that it was closed by server.
	time.Sleep(10 * time.Millisecond)
	require.Equal(t, expectedStream, rec.Body.String())

	// rec1 is plain request, rec2 is request with resumption.
	done1 := make(chan bool)
	rec1 := tc.DoAsync(httptest.NewRequest(http.MethodGet, "/updates/prod/tag1/update1/tail", nil), done1)
	done2 := make(chan bool)
	req2 := httptest.NewRequest(http.MethodGet, "/updates/prod/tag1/update1/tail", nil)
	req2.Header.Add("Last-Event-ID", "1")
	rec2 := tc.DoAsync(req2, done2)
	time.Sleep(10 * time.Millisecond)
	// A previous error line should not appear in the new response.
	expectedStream1 := `event: log
id: 1
data: {"uuid":"test-device-1","correlationId":"uuid-1","target-name":"intel-corei7-64-lmp-23","status":"Download started"}

`
	expectedStream2 := `event: log
id: 2
data: {"uuid":"test-device-2","correlationId":"uuid-2","target-name":"intel-corei7-64-lmp-23","status":"Download started"}

`
	expectedStream1 += expectedStream2
	require.Equal(t, 200, rec1.Code)
	require.Equal(t, expectedStream1, rec1.Body.String())
	require.Equal(t, 200, rec2.Code)
	require.Equal(t, expectedStream2, rec2.Body.String())

	// Write to the file and check the new response bytes within the same connections.
	events = generateUpdateEvents("uuid-1", "forth", 1)
	require.Nil(t, d1.ProcessEvents(events))
	time.Sleep(10 * time.Millisecond)
	expectedStreamX := `event: log
id: 3
data: {"uuid":"test-device-1","correlationId":"uuid-1","target-name":"intel-corei7-64-lmp-23","status":"Download started"}

`
	expectedStream1 += expectedStreamX
	expectedStream2 += expectedStreamX
	require.Equal(t, expectedStream1, rec1.Body.String())
	require.Equal(t, expectedStream2, rec2.Body.String())
	tc.assertNotDone(done1)
	tc.assertNotDone(done2)

	// keepalive test
	saved := keepaliveResponseInterval
	keepaliveResponseInterval = 20 * time.Millisecond
	defer func() { keepaliveResponseInterval = saved }()
	done3 := make(chan bool)
	rec3 := tc.DoAsync(httptest.NewRequest(http.MethodGet, "/updates/prod/tag1/update1/tail", nil), done3)
	time.Sleep(50 * time.Millisecond)
	expectedStream3 := expectedStream1 + keepaliveResponseText + keepaliveResponseText
	require.Equal(t, 200, rec3.Code)
	require.Equal(t, expectedStream3, rec3.Body.String())
	require.Nil(t, d1.ProcessEvents(events))
	time.Sleep(30 * time.Millisecond)
	expectedStreamY := strings.Replace(expectedStreamX, "id: 3", "id: 4", 1)
	expectedStream3 += expectedStreamY + keepaliveResponseText
	require.Equal(t, expectedStream3, rec3.Body.String())
	tc.assertNotDone(done3)

	cancel() // This is where we disconnect, closing all holding handlers.
	time.Sleep(10 * time.Millisecond)
	tc.assertDone(done1)
	tc.assertDone(done2)
	tc.assertDone(done3)

	// TODO: Add rollout tail tests
}
