// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

//go:build mdns

// Note: mDNS requires multicasting, which is generally disabled in any public cloud infrastructure.
// However, it is usually fine to use it in private infrastructure, if the network is configured accordingly.

package mdns

import (
	"net"
	"time"

	"github.com/hashicorp/mdns"
	dnsLib "github.com/miekg/dns"

	"github.com/foundriesio/dg-satellite/context"
)

func newServer(ctx context.Context) dns {
	log := context.CtxGetLog(ctx).With("server", "mdns")
	mzone := &multiZone{}
	return &mDNS{
		context: ctx,
		config: &mdns.Config{
			Logger: context.StdLogAdapter(log, false),
			Zone:   mzone, // An zone interface reference that mdns uses
		},
		mzone: mzone, // A zone struct that mDNS populates
	}
}

type mDNS struct {
	context context.Context
	config  *mdns.Config
	server  *mdns.Server
	mzone   *multiZone
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
	hostName := service + "." + domain
	if srv, err := mdns.NewMDNSService(
		// Instance and service names, used by the hashicorp/mdns to fill in the SRV record.
		// We don't use them, as what they try to do is mimic a cloud.
		// We use the host name to register A records instead.
		// Maybe, we need to write our own zone logic to register a correct SRV record.
		"satellite",
		"satellite",
		// Top-level domain name.
		domain,
		// Host name - this is how we want our service to actually resolve.
		// HashiCorp assumes this is the local host name; that does not suit our use case.
		hostName,
		int(port),
		[]net.IP{net.ParseIP(addr)},
		nil, // optional text record, unused for us.
	); err != nil {
		return err
	} else {
		s.mzone.zones = append(s.mzone.zones, srv)
		context.CtxGetLog(s.context).Info("Registered new mDNS service", "host", hostName, "addr", addr)
		return nil
	}
}

type multiZone struct {
	zones []mdns.Zone
}

func (m *multiZone) Records(q dnsLib.Question) (res []dnsLib.RR) {
	for _, z := range m.zones {
		if rr := z.Records(q); len(rr) > 0 {
			res = append(res, rr...)
		}
	}
	return
}
