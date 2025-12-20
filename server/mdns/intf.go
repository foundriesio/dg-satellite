// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package mdns

import (
	"errors"
	"fmt"
	"net"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
)

var (
	ErrNotFoundInterface   = errors.New("interface not found")
	ErrZeroInterfaces      = errors.New("zero non-loopback interfaces found")
	ErrAmbiguousInterfaces = errors.New("more than one non-loopback interfaces found")
)

type ServerParams struct {
	// If Addr is not specified, use Intf to auto-detect the IP address.
	// If Intf is not specified, auto-detect a non-loopback interface to use.
	// If there is more than one non-loopback interface, return an error.
	Addr string
	Intf string
	Port uint16
	Tld  string
}

func NewServer(ctx context.Context, gatewayParams ServerParams) (server.Server, error) {
	const errGateway string = "failed to configure gateway mDNS record: %w"
	log := context.CtxGetLog(ctx)
	srv := newServer(ctx)

	gatewayAddr := gatewayParams.Addr
	if len(gatewayAddr) != 0 {
		log.Info("Using user provided IP address to set up mDNS gateway records", "addr", gatewayAddr)
	} else {
		var err error
		if gatewayAddr, err = getLocalAddr(gatewayParams.Intf); err != nil {
			return nil, fmt.Errorf(errGateway, err)
		}
		log.Info("Auto-detected IP address to set up mDNS gateway records",
			"addr", gatewayAddr, "intf", gatewayParams.Intf)
	}

	for _, service := range []string{"api", "hub", "ostree"} {
		if err := srv.register(service, gatewayParams.Tld, gatewayAddr, gatewayParams.Port); err != nil {
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

func getLocalAddr(intfName string) (string, error) {
	var (
		interfaceList []net.Interface
		allowLoopback bool
	)
	if len(intfName) > 0 {
		allowLoopback = true
		if intf, err := net.InterfaceByName(intfName); err != nil {
			return "", ErrNotFoundInterface
		} else {
			interfaceList = append(interfaceList, *intf)
		}
	} else {
		var err error
		interfaceList, err = net.Interfaces()
		if err != nil {
			return "", err
		}
	}

	var (
		found bool
		res   string
	)
	for _, intf := range interfaceList {
		addrList, err := intf.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrList {
			if ip, ok := addr.(*net.IPNet); ok {
				if !allowLoopback && ip.IP.IsLoopback() {
					continue
				}
				if found {
					return "", ErrAmbiguousInterfaces
				}
				if res != "" {
					if ip.IP.To4() == nil {
						// If both IPv4 and IPv6 are set - prefer IPv4.
						// This is what is usually needed in the link-local network.
						// And it allows to make the mdns work faster.
						continue
					}
				}
				res = ip.IP.String()
			}
		}
		found = res != ""
	}
	if !found {
		return "", ErrZeroInterfaces
	}
	return res, nil
}
