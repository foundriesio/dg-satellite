// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

//go:build mdns

// Note: mDNS requires multicasting, which is generally disabled in any public cloud infrastructure.
// However, it is usually fine to use it in private infrastructure, if the network is configured accordingly.

package mdns

import (
	"time"

	"github.com/hashicorp/mdns"

	"github.com/foundriesio/dg-satellite/context"
)

func newServer(ctx context.Context) dns {
	log := context.CtxGetLog(ctx).With("server", "mdns")
	return &mDNS{
		context: ctx,
		config: &mdns.Config{
			Logger: context.StdLogAdapter(log, false),
		},
	}
}

type mDNS struct {
	context context.Context
	config  *mdns.Config
	server  *mdns.Server
}

func (s *mDNS) Start(quit chan error) {
	log := context.CtxGetLog(s.context)
	// NewServer binds synchronously and starts listening in background (returns).
	var err error
	if s.server, err = mdns.NewServer(s.config); err != nil {
		log.Error("failed to start server", "error", err)
		quit <- err
	} else {
		log.Info("multicast server started")
	}
}

func (s *mDNS) Shutdown(timeout time.Duration) {
	fin := make(chan bool, 1)
	go func() {
		if err := s.server.Shutdown(); err != nil {
			context.CtxGetLog(s.context).Error("error stopping server", "error", err)
		}
		fin <- true
	}()
	select {
	case <-fin:
	case <-time.After(timeout):
	}
}

func (s *mDNS) GetAddress() string {
	// Port 5353
	// IPv4: 224.0.0.251
	// IPv6: ff02::fb
	return "ff02::fb"
}

func (s *mDNS) GetDnsName() string {
	// makes no sense for mDNS; just required by the
	return ""
}

func (s *mDNS) register(service, domain, addr string, port uint16) error {
	// TODO: This is where things get complex. Implemented in the next commit.
	return nil
}
