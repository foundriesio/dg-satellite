// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"net/http"
)

type fakeUser struct {
	denyHasScope bool
}

func (fakeUser) Id() string {
	return "fake-user"
}

func (u fakeUser) Scopes() Scopes {
	if u.denyHasScope {
		return 0
	}
	return 0xFFFFFFFFFFFFFFFF
}

func FakeAuthUser(w http.ResponseWriter, r *http.Request) (User, error) {
	deny := len(r.URL.Query().Get("deny-has-scope")) > 0
	return &fakeUser{denyHasScope: deny}, nil
}
