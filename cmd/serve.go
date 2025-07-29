// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/foundriesio/dg-satellite/server"
)

type ServeCmd struct {
	ApiPort    uint16 `default:"8080"`
	ApiAddress func() string
}

func (c *ServeCmd) Run(args CommonArgs) error {
	apiServer := server.NewServer(
		context.Background(),
		server.NewEchoServer(),
		c.ApiPort,
	)
	c.ApiAddress = apiServer.GetAddress

	apiErr := make(chan error)
	apiServer.Start(apiErr)

	// setup channel to gracefully terminate server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-apiErr:
		return fmt.Errorf("failed to start API server: %w", err)
	case <-quit:
		if err := apiServer.Shutdown(time.Minute); err != nil {
			return fmt.Errorf("unexpected error stopping rest-api server %w", err)
		}
	}
	return nil
}
