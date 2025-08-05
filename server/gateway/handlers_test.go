// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/dg"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	t   *testing.T
	gw  *dg.Storage
	fs  *storage.FsHandle
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

func (c testClient) GET(resource string, status int) []byte {
	req := httptest.NewRequest(http.MethodGet, resource, nil)
	rec := c.Do(req)
	require.Equal(c.t, status, rec.Code)
	return rec.Body.Bytes()
}

func (c testClient) PUT(resource string, data []byte, status int) []byte {
	reader := bytes.NewReader(data)
	req := httptest.NewRequest(http.MethodPut, resource, reader)
	rec := c.Do(req)
	require.Equal(c.t, status, rec.Code)
	return rec.Body.Bytes()
}

func (c testClient) POST(resource string, data []byte, status int) []byte {
	reader := bytes.NewReader(data)
	req := httptest.NewRequest(http.MethodPost, resource, reader)
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
	gwS, err := dg.NewStorage(db, fsS)
	require.Nil(t, err)

	log, err := context.InitLogger("debug")
	require.Nil(t, err)

	e := server.NewEchoServer("api-test")
	RegisterHandlers(e, gwS)

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)

	cert := x509.Certificate{
		Subject:   pkix.Name{CommonName: "test-client-uuid"},
		PublicKey: priv.Public(),
	}
	tc := testClient{
		t:   t,
		fs:  fsS,
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
	var device dg.DgDevice
	require.Nil(t, json.Unmarshal(deviceBytes, &device))
	require.Equal(t, tc.cert.Subject.CommonName, device.Uuid)
	require.Less(t, lastSeen, device.LastSeen)
}

func TestApiSysInfo(t *testing.T) {
	tc := NewTestClient(t)
	tc.PUT("/system_info/config", []byte("test-aktoml"), 204)
	tc.PUT("/system_info/network", []byte("network-info"), 204)
	tc.PUT("/system_info", []byte("lshw info"), 204)

	content, err := tc.fs.ReadFile(tc.cert.Subject.CommonName, storage.Aktoml)
	require.Nil(t, err)
	require.Equal(t, "test-aktoml", content)

	content, err = tc.fs.ReadFile(tc.cert.Subject.CommonName, storage.NetInfo)
	require.Nil(t, err)
	require.Equal(t, "network-info", content)

	content, err = tc.fs.ReadFile(tc.cert.Subject.CommonName, storage.HwInfo)
	require.Nil(t, err)
	require.Equal(t, "lshw info", content)
}

func TestApiFiotest(t *testing.T) {
	tc := NewTestClient(t)

	content := `{"name": "test-1"}`
	resp := tc.POST("/tests", []byte(content), 201)
	testid := string(resp)

	content = `{
			"status": "PASSED",
			"details": "detail x",
			"results": [
				{
					"name": "tr-1",
					"status": "FAILED"
				},
				{
					"name": "tr-2",
					"status": "PASSED",
					"local_ts": 1597802911.1365469,
					"details": "tr2-detail",
					"metrics": {
						"m1": 12,
						"m2": 42.1
					}
				}
			],
			"artifacts": ["console.txt"]
		}`
	out := tc.PUT("/tests/"+testid, []byte(content), 200)

	type signedUrl struct {
		Url         string `json:"url"`
		ContentType string `json:"content-type"`
	}
	var urls map[string]signedUrl
	require.Nil(t, json.Unmarshal(out, &urls))
	for name, signed := range urls {
		tc.PUT(signed.Url, []byte(name+"BLAH"), 200)
	}
	prefix := storage.TestArtifactsPrefix + "-" + testid + "_"
	files, err := tc.fs.ListFiles(tc.cert.Subject.CommonName, prefix, true)
	require.Nil(t, err)
	require.Len(t, files, 1)
	require.Equal(t, prefix+"console.txt", files[0])
}
