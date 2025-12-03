// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/auth/providers"
	"github.com/foundriesio/dg-satellite/server"
	apiHandlers "github.com/foundriesio/dg-satellite/server/ui/api"
	"github.com/foundriesio/dg-satellite/server/ui/daemons"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/api"
	"github.com/foundriesio/dg-satellite/storage/users"
)

const serverName = "rest-api"

type daemon interface {
	Start()
	Shutdown()
}

func NewServer(ctx context.Context, db *storage.DbHandle, fs *storage.FsHandle, port uint16, authFunc auth.AuthUserFunc) (server.Server, error) {
	strg, err := api.NewStorage(db, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s storage: %w", serverName, err)
	}
	e := server.NewEchoServer()

	authConfig, err := fs.GetAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth config: %w", err)
	}

	userStorage, err := users.NewStorage(db, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load users storage: %w", err)
	}
	provider := providers.GetProvider(authConfig.Type)
	if provider == nil {
		return nil, fmt.Errorf("authentication provider not found: %s", authConfig.Type)
	} else if err = provider.Configure(e, userStorage, authConfig); err != nil {
		return nil, fmt.Errorf("failed to configure authentication provider: %w", err)
	}

	srv := server.NewServer(ctx, e, serverName, port, nil)
	apiHandlers.RegisterHandlers(e, strg, authFunc)
	return &apiServer{server: srv, daemons: daemons.New(ctx, strg)}, nil
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
