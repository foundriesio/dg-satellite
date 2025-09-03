// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/gateway"
)

const serverName = "gateway-api"

func NewServer(ctx context.Context, db *storage.DbHandle, fs *storage.FsHandle, port uint16) (*server.Server, error) {
	tlsCfg, err := loadTlsConfig(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s TLS config: %w", serverName, err)
	}
	strg, err := gateway.NewStorage(db, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s storage: %w", serverName, err)
	}
	e := server.NewEchoServer()
	srv := server.NewServer(ctx, e, serverName, port, tlsCfg)
	RegisterHandlers(e, strg)
	return &srv, nil
}

func loadTlsConfig(fs *storage.FsHandle) (*tls.Config, error) {
	caPool, err := loadCas(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load gateway cert: %w", err)
	}
	kp, err := loadTlsKeyPair(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load gateway key: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{kp},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
		ClientCAs:    caPool,
	}
	return cfg, nil
}

func loadCas(fs *storage.FsHandle) (*x509.CertPool, error) {
	bytes, err := fs.Certs.ReadFile(storage.CertsCasPemFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read CAs file: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(bytes)
	return caPool, nil
}

func loadTlsKeyPair(fs *storage.FsHandle) (tls.Certificate, error) {
	keyFile := fs.Certs.FilePath(storage.CertsTlsKeyFile)
	certFile := fs.Certs.FilePath(storage.CertsTlsPemFile)
	return tls.LoadX509KeyPair(certFile, keyFile)
}
