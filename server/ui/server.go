// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/server"
	apiHandlers "github.com/foundriesio/dg-satellite/server/ui/api"
	"github.com/foundriesio/dg-satellite/server/ui/daemons"
	webHandlers "github.com/foundriesio/dg-satellite/server/ui/web"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/api"
	"github.com/foundriesio/dg-satellite/storage/users"
)

const serverName = "rest-api"

type daemon interface {
	Start()
	Shutdown()
}

func NewServer(ctx context.Context, db *storage.DbHandle, fs *storage.FsHandle, port uint16) (server.Server, error) {
	strg, err := api.NewStorage(db, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s storage: %w", serverName, err)
	}
	users, err := users.NewStorage(db, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize users storage: %w", err)
	}
	e := server.NewEchoServer()

	provider, err := auth.NewProvider(e, db, fs, users)
	if err != nil {
		return nil, err
	}
	slog.Info("Using authentication provider", "name", provider.Name())

	daemons := daemons.New(ctx, strg, users)

	srv := server.NewServer(ctx, e, serverName, port, nil)
	apiHandlers.RegisterHandlers(e, strg, users, provider)
	webHandlers.RegisterHandlers(e, users, provider)
	return &apiServer{server: srv, daemons: daemons}, nil
}

type apiServer struct {
	server  server.Server
	daemons daemon
}

func (s apiServer) Start(quit chan error) {
	s.daemons.Start()
	s.server.Start(quit)
}

func (s apiServer) Shutdown(timeout time.Duration) {
	s.daemons.Shutdown()
	s.server.Shutdown(timeout)
}

func (s apiServer) GetAddress() string {
	return s.server.GetAddress()
}

func (s apiServer) GetDnsName() string {
	return s.server.GetDnsName()
}
