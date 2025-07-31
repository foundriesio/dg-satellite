// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package server

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"

	"github.com/foundriesio/dg-satellite/context"
)

type server struct {
	context context.Context
	echo    *echo.Echo
	server  *http.Server
}

func NewServer(ctx context.Context, echo *echo.Echo, port uint16) server {
	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		BaseContext: func(net.Listener) context.Context { return ctx },
		ConnContext: adjustConnContext,
	}
	return server{context: ctx, echo: echo, server: srv}
}

func (s server) Start(quit chan error) {
	go func() {
		if err := s.echo.StartServer(s.server); err != nil && err != http.ErrServerClosed {
			quit <- err
		}
	}()
}

func (s server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(s.context, timeout)
	defer cancel()
	return s.echo.Shutdown(ctx)
}

func (s server) GetAddress() (ret string) {
	// ListenerAddr waits for the server to start before returning
	if addr := s.echo.ListenerAddr(); addr != nil {
		// Addr can be nil when server fails to start
		ret = addr.String()
	}
	return
}

func adjustConnContext(ctx context.Context, conn net.Conn) context.Context {
	cid := random.String(10) // No need for uuid, save some space
	log := context.CtxGetLog(ctx).With("conn_id", cid)
	// There is nothing meaningful to log before the TLS connection
	return context.CtxWithLog(ctx, log)
}
