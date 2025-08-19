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
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/foundriesio/dg-satellite/context"
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
	log := context.CtxGetLog(args.ctx)
	fs, err := storage.NewFs(args.DataDir)
	if err != nil {
		return err
	}
	gtwTlsConfig, err := gatewayTlsConfig(fs)
	if err != nil {
		return err
	}
	gwDnsName, err := dnsNameFromCert(gtwTlsConfig.Certificates[0])
	if err != nil {
		return err
	}

	apiS, gwS, err := args.CreateStorageHandles()
	if err != nil {
		return err
	}

	apiE := server.NewEchoServer("rest-api")
	api.RegisterHandlers(apiE, apiS)
	apiServer := server.NewServer(
		args.ctx,
		apiE,
		c.ApiPort,
		nil,
	)

	gtwE := server.NewEchoServer("dg-api")
	gateway.RegisterHandlers(gtwE, gwS)
	gtwServer := server.NewServer(
		args.ctx,
		gtwE,
		c.GatewayPort,
		gtwTlsConfig,
	)

	apiErr := make(chan error)
	gtwErr := make(chan error)
	apiServer.Start(apiErr)
	gtwServer.Start(gtwErr)

	// Echo locks a mutex immediately at the Start call, and releases after port binding is done.
	// GetAddress will be locked for that duration; but we need to give it a tiny favor to start.
	time.Sleep(time.Millisecond * 2)
	apiAddress := apiServer.GetAddress()
	gatewayAddress := gtwServer.GetAddress()
	log.Info("rest api server started", "addr", apiAddress)
	log.Info("gateway server started", "addr", gatewayAddress, "dns_name", gwDnsName)
	if c.startedCb != nil {
		c.startedCb(apiAddress, gatewayAddress)
	}

	// setup channel to gracefully terminate server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-apiErr:
		return fmt.Errorf("failed to start API server: %w", err)
	case err := <-gtwErr:
		return fmt.Errorf("failed to start gateway server: %w", err)
	case <-quit:
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			if err := apiServer.Shutdown(time.Minute); err != nil {
				log.Error("unexpected error stopping rest-api server", "error", err)
			}
			wg.Done()
		}()
		go func() {
			if err := gtwServer.Shutdown(time.Minute); err != nil {
				log.Error("unexpected error stopping gateway server", "error", err)
			}
			wg.Done()
		}()
		wg.Wait()
	}
	return nil
}

func loadCas(fs *storage.FsHandle) (*x509.CertPool, error) {
	path := filepath.Join(fs.Config.CertsDir(), "cas.pem")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read CAs file: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(bytes)
	return caPool, nil
}

func loadTlsKeyPair(fs *storage.FsHandle) (tls.Certificate, error) {
	keyFile := filepath.Join(fs.Config.CertsDir(), "tls.key")
	certFile := filepath.Join(fs.Config.CertsDir(), "tls.crt")
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

func dnsNameFromCert(cert tls.Certificate) (string, error) {
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	if len(x509Cert.DNSNames) == 0 {
		return "", fmt.Errorf("no DNS names found in certificate")
	}

	return x509Cert.DNSNames[0], nil
}
