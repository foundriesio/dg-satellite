// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	storage "github.com/foundriesio/dg-satellite/storage/gateway"
)

type testClient struct {
	t   *testing.T
	fs  *storage.FsHandle
	gw  *storage.Storage
	e   *echo.Echo
	log *slog.Logger

	uuid string
	cert *x509.Certificate
}

func (c testClient) Do(req *http.Request) *httptest.ResponseRecorder {
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{c.cert},
	}
	req = req.WithContext(context.CtxWithLog(req.Context(), c.log))
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

func (c testClient) POST(resource string, status int, data any, headers ...string) []byte {
	req := httptest.NewRequest(http.MethodPost, resource, c.marshalBody(data))
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
	db, err := storage.NewDb(fsS.Config.DbFile())
	require.Nil(t, err)
	gwS, err := storage.NewStorage(db, fsS)
	require.Nil(t, err)

	log, err := context.InitLogger("debug")
	require.Nil(t, err)

	e := server.NewEchoServer()
	RegisterHandlers(e, gwS)

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)

	uuid := "test-client-uuid"
	cert := x509.Certificate{
		Subject:   pkix.Name{CommonName: uuid},
		PublicKey: priv.Public(),
	}
	tc := testClient{
		t:   t,
		gw:  gwS,
		fs:  fsS,
		e:   e,
		log: log,

		uuid: uuid,
		cert: &cert,
	}
	return &tc
}

func TestApiDevice(t *testing.T) {
	lastSeen := time.Now().Add(-1 * time.Second).Unix()
	tc := NewTestClient(t)
	deviceBytes := tc.GET("/device", 200)
	var device storage.Device
	require.Nil(t, json.Unmarshal(deviceBytes, &device))
	assert.Equal(t, tc.cert.Subject.CommonName, device.Uuid)
	assert.Less(t, lastSeen, device.LastSeen)
}

func TestCheckIn(t *testing.T) {
	apps := "a,b,c"
	hash := "abcd"
	tag := "tag"
	target := "target"
	tc := NewTestClient(t)
	deviceBytes := tc.GET(
		"/device", 200, "x-ats-dockerapps", apps, "x-ats-ostreehash", hash, "x-ats-tags", tag, "x-ats-target", target)

	var d *storage.Device
	require.Nil(t, json.Unmarshal(deviceBytes, &d))
	assert.Equal(t, apps, d.Apps)
	assert.Equal(t, hash, d.OstreeHash)
	assert.Equal(t, tag, d.Tag)
	assert.Equal(t, target, d.TargetName)

	d, err := tc.gw.DeviceGet(tc.uuid)
	require.Nil(t, err)
	assert.Equal(t, apps, d.Apps)
	assert.Equal(t, hash, d.OstreeHash)
	assert.Equal(t, tag, d.Tag)
	assert.Equal(t, target, d.TargetName)

	// Check that fields are not erased on a partial update
	tag = "switch"
	apps = "a,b,d"
	_ = tc.GET("/device", 200, "x-ats-dockerapps", apps, "x-ats-tags", tag)

	d, err = tc.gw.DeviceGet(tc.uuid)
	require.Nil(t, err)
	assert.Equal(t, apps, d.Apps)
	assert.Equal(t, hash, d.OstreeHash)
	assert.Equal(t, tag, d.Tag)
	assert.Equal(t, target, d.TargetName)
}

func TestInfo(t *testing.T) {
	akInfo := "[config]\nkey=value"
	ecInfo := `{"abc":{"hwid":"beef","target":"rack"}}`
	ecInfoBad := `{"hw":"target"}`
	hwInfo := `{"key":"value"}`
	hwInfoBad := `{key=value}`
	nwInfo := `{"hostname":"example.org"}`
	nwInfoBad := `{"hostname":123}`
	stInfo := `{"deviceTime":"2025-09-12T10:00:00Z"}`
	stInfo1 := `{"deviceTime":"2025-09-12T10:00:05Z"}`
	stInfoBad := `{"deviceTime":"2025-09-12 10:00:00"}`

	tc := NewTestClient(t)
	_ = tc.PUT("/system_info", 200, hwInfo)
	_ = tc.PUT("/system_info", 400, hwInfoBad)
	_ = tc.PUT("/system_info/config", 200, akInfo)
	_ = tc.PUT("/system_info/network", 200, nwInfo)
	_ = tc.PUT("/system_info/network", 400, nwInfoBad)
	_ = tc.PUT("/ecus", 200, ecInfo)
	_ = tc.PUT("/ecus", 400, ecInfoBad)
	_ = tc.POST("/apps-states", 200, stInfo)
	_ = tc.POST("/apps-states", 200, stInfo1)
	_ = tc.POST("/apps-states", 400, stInfoBad)

	data, err := tc.fs.Devices.ReadFile(tc.uuid, storage.AktomlFile)
	assert.Nil(t, err)
	assert.Equal(t, akInfo, data)
	data, err = tc.fs.Devices.ReadFile(tc.uuid, storage.HwInfoFile)
	assert.Nil(t, err)
	assert.Equal(t, hwInfo, data)
	data, err = tc.fs.Devices.ReadFile(tc.uuid, storage.NetInfoFile)
	assert.Nil(t, err)
	assert.Equal(t, nwInfo, data)
	data, err = tc.fs.Devices.ReadFile(tc.uuid, storage.EcusInfoFile)
	assert.Nil(t, err)
	assert.Equal(t, ecInfo, data)

	states, err := tc.fs.Devices.ListFiles(tc.uuid, storage.StatesPrefix, true)
	require.Nil(t, err)
	assert.Equal(t, 2, len(states))
	exp := []string{stInfo, stInfo1}
	for idx, name := range states {
		data, err = tc.fs.Devices.ReadFile(tc.uuid, name)
		assert.Nil(t, err)
		assert.Equal(t, exp[idx], data)
	}

	// apps states rollover
	for i := 0; i < 15; i++ {
		_ = tc.POST("/apps-states", 200, stInfo1)
	}
	states, err = tc.fs.Devices.ListFiles(tc.uuid, storage.StatesPrefix, true)
	require.Nil(t, err)
	assert.Equal(t, 10, len(states))
	for _, name := range states {
		data, err = tc.fs.Devices.ReadFile(tc.uuid, name)
		assert.Nil(t, err)
		assert.Equal(t, stInfo1, data)
	}
}
