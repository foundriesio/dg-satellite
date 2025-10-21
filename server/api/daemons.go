// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	storage "github.com/foundriesio/dg-satellite/storage/api"
)

type daemonFunc func(stop chan bool)

type daemons struct {
	context Context
	storage *storage.Storage
	daemons []daemonFunc
	stops   []chan bool
}

func NewDaemons(context Context, storage *storage.Storage) *daemons {
	d := &daemons{context: context, storage: storage}
	d.daemons = []daemonFunc{
		d.rolloutWatchdog(true),
		d.rolloutWatchdog(false),
	}
	return d
}

func (d *daemons) Start() {
	for _, f := range d.daemons {
		stop := make(chan bool)
		d.stops = append(d.stops, stop)
		go f(stop)
	}
}

func (d *daemons) Shutdown() {
	for _, s := range d.stops {
		s <- true
	}
}
