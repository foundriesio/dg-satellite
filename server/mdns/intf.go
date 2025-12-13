// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package mdns

import (
	"fmt"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
)

type ServerParams struct {
	Addr string
	Port uint16
	Tld  string
}

func NewServer(ctx context.Context, gatewayParams ServerParams) (server.Server, error) {
	srv := newServer(ctx)
	for _, service := range []string{"api", "hub", "ostree"} {
		if err := srv.register(service, gatewayParams.Tld, gatewayParams.Addr, gatewayParams.Port); err != nil {
			return nil, fmt.Errorf("failed to register mdns service %s: %w", service, err)
		}
	}
	return srv, nil
}

type configurator interface {
	register(service, name, addr string, port uint16) error
}

type dns interface {
	server.Server
	configurator
}
