// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package auth

import (
	"net/http"
)

// Scope helps simplify how we check for role-based access. For example,
// if you want to list devices you need `devices:read` *or* `devices:read-update`.
// Scope allows us to define a "devices-list" that covers them both
type Scope []string

var ScopeDevicesR = Scope{"devices:read", "devices:read-update"}
var ScopeDevicesRU = Scope{"devices:read-update"}
var ScopeDevicesD = Scope{"devices:delete"}

type User interface {
	Id() string
	HasScope(Scope) error
}

// AuthUserFunc allows us to define a generic way for middleware to do
// authentication and authorization based on the incoming http request.
// The function returns nil if the user wasn't authenticated implying
// this function returned the proper error to the caller.
type AuthUserFunc func(w http.ResponseWriter, r *http.Request) (User, error)
