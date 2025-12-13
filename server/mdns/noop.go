// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package mdns

import (
	"time"

	"github.com/foundriesio/dg-satellite/context"
)

func newServer(ctx context.Context) dns {
	return &noop{}
}

type noop struct{}

func (s *noop) Start(quit chan error) {}

func (s *noop) Shutdown(timeout time.Duration) {}

func (s *noop) GetAddress() string {
	return ""
}

func (s *noop) GetDnsName() string {
	return ""
}

func (s *noop) register(service, name, addr string, port uint16) error {
	return nil
}
