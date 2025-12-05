// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package daemons

import (
	"github.com/foundriesio/dg-satellite/storage/users"
)

func WithUserGc(users *users.Storage) Option {
	return func(d *daemons) {
		gcFunc := func(stop chan bool) {
			users.StartGc()
			<-stop
			users.StopGc()
		}
		d.daemons = append(d.daemons, gcFunc)
	}
}
