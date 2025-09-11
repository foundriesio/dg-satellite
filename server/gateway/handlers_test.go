// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
	gw  *storage.Storage
	e   *echo.Echo
	log *slog.Logger

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

func (c testClient) marshalHeaders(headers []string, req *http.Request) {
	require.Zero(c.t, len(headers)%2, "Headers must be a sequence of names and values - even number")
	for i := 0; i < len(headers)/2; i++ {
		req.Header.Add(headers[i*2], headers[i*2+1])
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

	cert := x509.Certificate{
		Subject:   pkix.Name{CommonName: "test-client-uuid"},
		PublicKey: priv.Public(),
	}
	tc := testClient{
		t:   t,
		gw:  gwS,
		e:   e,
		log: log,

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

	d, err := tc.gw.DeviceGet(d.Uuid)
	require.Nil(t, err)
	assert.Equal(t, apps, d.Apps)
	assert.Equal(t, hash, d.OstreeHash)
	assert.Equal(t, tag, d.Tag)
	assert.Equal(t, target, d.TargetName)

	// Check that fields are not erased on a partial update
	tag = "switch"
	apps = "a,b,d"
	_ = tc.GET("/device", 200, "x-ats-dockerapps", apps, "x-ats-tags", tag)

	d, err = tc.gw.DeviceGet(d.Uuid)
	require.Nil(t, err)
	assert.Equal(t, apps, d.Apps)
	assert.Equal(t, hash, d.OstreeHash)
	assert.Equal(t, tag, d.Tag)
	assert.Equal(t, target, d.TargetName)
}
