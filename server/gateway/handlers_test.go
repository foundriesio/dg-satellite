// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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
	req := httptest.NewRequest(http.MethodGet, "/tmp", nil)
	rec := c.Do(req)
	require.Equal(c.t, http.StatusOK, rec.Code)
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
		gw:  gwS,
		e:   e,
		log: log,

		cert: &cert,
	}
	return &tc
}

func TestApi(t *testing.T) {
	tc := NewTestClient(t)
	data := tc.GET("/tmp", 200)
	require.Equal(t, "test-client-uuid", string(data))
}
