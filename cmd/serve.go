// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
)

type ServeCmd struct {
	started sync.WaitGroup

	ApiPort    uint16 `default:"8080"`
	ApiAddress string
}

func NewServeCmd() (s ServeCmd) {
	s.started.Add(1)
	return
}

func (c *ServeCmd) WaitUntilStarted() {
	c.started.Wait()
}

func (c *ServeCmd) Run(ctx context.Context, args CommonArgs) error {
	log := context.CtxGetLog(ctx)
	apiServer := server.NewServer(
		ctx,
		server.NewEchoServer("rest-api"),
		c.ApiPort,
	)

	apiErr := make(chan error)
	apiServer.Start(apiErr)

	// Echo locks a mutex immediately at the Start call, and releases after port binding is done.
	// GetAddress will be locked for that duration; but we need to give it a tiny favor to start.
	time.Sleep(time.Millisecond * 2)
	c.ApiAddress = apiServer.GetAddress()
	log.Info("rest api server started", "addr", c.ApiAddress)
	c.started.Done()

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
