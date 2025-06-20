// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/server/gateway"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type ServeCmd struct {
	ApiPort     uint16 `default:"8080"`
	GatewayPort uint16 `default:"8443"`

	quit          chan os.Signal
	apiServer     *echo.Echo
	gatewayServer *echo.Echo
}

func (c *ServeCmd) Run(args CommonArgs) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	caPool, err := args.loadCas()
	if err != nil {
		return err
	}
	kp, err := args.loadTlsKeyPair()
	if err != nil {
		return err
	}

	// setup channel to gracefully terminate server
	c.quit = make(chan os.Signal, 1)
	signal.Notify(c.quit, syscall.SIGTERM)
	serveErr := make(chan error)

	c.apiServer = server.NewEchoServer("rest-api", logger)
	c.gatewayServer = server.NewEchoServer("device-gateway", logger)
	gateway.RegisterHandlers(c.gatewayServer)

	go func() {
		if err := c.apiServer.Start(fmt.Sprintf(":%d", c.ApiPort)); err != http.ErrServerClosed {
			serveErr <- err
		}
	}()

	go func() {
		s := c.gatewayServer.TLSServer
		s.Addr = fmt.Sprintf(":%d", c.GatewayPort)
		s.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{kp},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
			ClientCAs:    caPool,
		}
		if err := c.gatewayServer.StartServer(s); err != http.ErrServerClosed {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		panic(err)
	case <-c.quit:
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		if err := c.apiServer.Shutdown(ctx); err != nil {
			log.Error("Unexpected error stopping rest-api server", "error", err)
		}
		if err := c.gatewayServer.Shutdown(ctx); err != nil {
			log.Error("Unexpected error stopping device-gateway server", "error", err)
		}
	}

	return nil
}

func (c CommonArgs) loadCas() (*x509.CertPool, error) {
	path := filepath.Join(c.CertsDir(), "cas.pem")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read CAs file: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(bytes)
	return caPool, nil
}

func (c CommonArgs) loadTlsKeyPair() (tls.Certificate, error) {
	keyFile := filepath.Join(c.CertsDir(), "tls.key")
	certFile := filepath.Join(c.CertsDir(), "tls.crt")
	return tls.LoadX509KeyPair(certFile, keyFile)
}
