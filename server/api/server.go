// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"fmt"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/api"
)

const serverName = "rest-api"

func NewServer(ctx context.Context, db *storage.DbHandle, fs *storage.FsHandle, port uint16) (*server.Server, error) {
	strg, err := api.NewStorage(db, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s storage: %w", serverName, err)
	}
	e := server.NewEchoServer()
	srv := server.NewServer(ctx, e, serverName, port, nil)
	RegisterHandlers(e, strg)
	return &srv, nil
}
