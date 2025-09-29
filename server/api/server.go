// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"fmt"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/api"
)

const serverName = "rest-api"

func NewServer(ctx context.Context, db *storage.DbHandle, fs *storage.FsHandle, port uint16, authFunc auth.AuthUserFunc) (*server.Server, error) {
	strg, err := api.NewStorage(db, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s storage: %w", serverName, err)
	}
	e := server.NewEchoServer()
	srv := server.NewServer(ctx, e, serverName, port, nil)
	RegisterHandlers(e, strg, authFunc)
	return &srv, nil
}
