// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
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

func (s server) GetAddress() string {
	return s.echo.Listener.Addr().String()
}
