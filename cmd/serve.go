// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/server/api"
	"github.com/foundriesio/dg-satellite/server/gateway"
	"github.com/foundriesio/dg-satellite/storage"
)

type ServeCmd struct {
	startedCb func(apiAddress, gatewayAddress string)

	ApiPort     uint16 `default:"8080"`
	GatewayPort uint16 `default:"8443"`
}

func (c *ServeCmd) Run(args CommonArgs) error {
	fs, err := storage.NewFs(args.DataDir)
	if err != nil {
		return err
	}
	gtwTlsConfig, err := gatewayTlsConfig(fs)
	if err != nil {
		return err
	}

	apiS, gwS, err := args.CreateStorageHandles()
	if err != nil {
		return err
	}

	apiE := server.NewEchoServer()
	api.RegisterHandlers(apiE, apiS)
	apiServer := server.NewServer(
		args.ctx,
		apiE,
		"rest-api",
		c.ApiPort,
		nil,
	)

	gtwE := server.NewEchoServer()
	gateway.RegisterHandlers(gtwE, gwS)
	gtwServer := server.NewServer(
		args.ctx,
		gtwE,
		"gateway-api",
		c.GatewayPort,
		gtwTlsConfig,
	)

	quitErr := make(chan error, 2)
	apiServer.Start(quitErr)
	gtwServer.Start(quitErr)

	if c.startedCb != nil {
		// Testing code, see serve_test.go
		time.Sleep(time.Millisecond * 2)
		c.startedCb(apiServer.GetAddress(), gtwServer.GetAddress())
	}

	// setup channel to gracefully terminate server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err = <-quitErr:
	case <-quit:
		break
	}

	var wg sync.WaitGroup
	wg.Add(2)
	for _, srv := range []server.Server{apiServer, gtwServer} {
		go func() {
			srv.Shutdown(time.Minute)
			wg.Done()
		}()
	}
	wg.Wait()

	return err
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

func gatewayTlsConfig(fs *storage.FsHandle) (*tls.Config, error) {
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
