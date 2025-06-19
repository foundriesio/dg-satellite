// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/foundriesio/dg-satellite/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type ServeCmd struct {
	ApiPort uint16 `default:"8080"`

	quit      chan os.Signal
	apiServer *echo.Echo
}

func (c *ServeCmd) Run(args CommonArgs) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// setup channel to gracefully terminate server
	c.quit = make(chan os.Signal, 1)
	signal.Notify(c.quit, syscall.SIGTERM)
	serveErr := make(chan error)

	c.apiServer = server.NewEchoServer("rest-api", logger)

	go func() {
		if err := c.apiServer.Start(fmt.Sprintf(":%d", c.ApiPort)); err != http.ErrServerClosed {
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
	}

	return nil
}
