// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"

	"github.com/foundriesio/dg-satellite/context"
)

type Server struct {
	context context.Context
	name    string
	echo    *echo.Echo
	server  *http.Server
}

func NewServer(ctx context.Context, echo *echo.Echo, name string, port uint16, tlsConfig *tls.Config) Server {
	log := context.CtxGetLog(ctx).With("server", name)
	ctx = context.CtxWithLog(ctx, log)
	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		BaseContext: func(net.Listener) context.Context { return ctx },
		ConnContext: adjustConnContext,
		TLSConfig:   tlsConfig,
	}
	return Server{context: ctx, name: name, echo: echo, server: srv}
}

func (s Server) Start(quit chan error) {
	go func() {
		if err := s.echo.StartServer(s.server); err != nil && err != http.ErrServerClosed {
			quit <- err
		}
	}()
}

func (s Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(s.context, timeout)
	defer cancel()
	return s.echo.Shutdown(ctx)
}

func (s Server) GetAddress() (ret string) {
	// ListenerAddr waits for the server to start before returning
	if addr := s.echo.TLSListenerAddr(); addr != nil {
		// Addr can be nil when server fails to start
		ret = addr.String()
	} else if addr := s.echo.ListenerAddr(); addr != nil {
		ret = addr.String()
	}
	return
}

func (s Server) GetDnsName() (ret string) {
	log := context.CtxGetLog(s.context)
	if s.server.TLSConfig == nil || len(s.server.TLSConfig.Certificates) == 0 {
		log.Error("programming error", "error", errTlsNotConfigured)
	} else if cert := s.server.TLSConfig.Certificates[0].Leaf; cert == nil {
		log.Error("programming error", "error", errTlsLeafCert)
	} else if len(cert.DNSNames) > 0 {
		ret = cert.DNSNames[0]
	}
	return
}

func adjustConnContext(ctx context.Context, conn net.Conn) context.Context {
	cid := random.String(10) // No need for uuid, save some space
	log := context.CtxGetLog(ctx).With("conn_id", cid)
	// There is nothing meaningful to log before the TLS connection
	return context.CtxWithLog(ctx, log)
}

var (
	errTlsNotConfigured = errors.New("server TLS not configured properly")
	errTlsLeafCert      = errors.New("since Golang 1.23, LoadX509KeyPair always sets TLS leaf cert")
)
