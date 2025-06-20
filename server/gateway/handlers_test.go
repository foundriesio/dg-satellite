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
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/foundriesio/dg-satellite/server"
	"github.com/stretchr/testify/require"
)

type client struct {
	srv *httptest.Server
	pki *pki
}

func (c client) httpClient() *http.Client {
	client := c.srv.Client()
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig.Certificates = []tls.Certificate{c.pki.clientKp}
	return client
}

func (c client) GET(t *testing.T, resource string) []byte {
	url := c.srv.URL + resource
	res, err := c.httpClient().Get(url)
	require.Nil(t, err)
	buf, err := io.ReadAll(res.Body)
	require.Nil(t, err)
	require.Equal(t, 200, res.StatusCode, string(buf))
	return buf
}

type pki struct {
	rootKey *ecdsa.PrivateKey
	rootCa  *x509.Certificate

	clientKey  *ecdsa.PrivateKey
	clientCert *x509.Certificate
	clientKp   tls.Certificate
}

func createPKI(t *testing.T) *pki {
	pki := pki{}

	// Create root CA
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)
	pki.rootKey = key

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"example"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &pki.rootKey.PublicKey, pki.rootKey)
	require.Nil(t, err)
	pki.rootCa, err = x509.ParseCertificate(der)
	require.Nil(t, err)

	// Client client cert
	pki.clientKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.Nil(t, err)

	template = &x509.Certificate{
		Subject: pkix.Name{
			CommonName:         "client-cert",
			OrganizationalUnit: []string{"example"},
		},
		Issuer:       pki.rootCa.Subject,
		SerialNumber: big.NewInt(2019),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),

		IsCA:        false,
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err = x509.CreateCertificate(rand.Reader, template, pki.rootCa, &pki.clientKey.PublicKey, pki.rootKey)
	require.Nil(t, err)
	pki.clientCert, err = x509.ParseCertificate(der)
	require.Nil(t, err)

	// Client cert keypair
	certPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: der,
		},
	)
	privDer, err := x509.MarshalECPrivateKey(pki.clientKey)
	require.Nil(t, err)
	privPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: privDer,
		},
	)
	pki.clientKp, err = tls.X509KeyPair(certPem, privPem)
	require.Nil(t, err)

	return &pki
}

func testWrapper(t *testing.T, testFunc func(client)) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	e := server.NewEchoServer("dg-test", logger)

	srv := httptest.NewUnstartedServer(e)
	RegisterHandlers(e)

	pki := createPKI(t)

	pool := x509.NewCertPool()
	pool.AddCert(pki.rootCa)
	srv.TLS = &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  pool,
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	c := client{
		srv: srv,
		pki: pki,
	}
	testFunc(c)
}

func TestGateway(t *testing.T) {
	testWrapper(t, func(tc client) {
		_ = tc.GET(t, "/tmp")
	})
}
