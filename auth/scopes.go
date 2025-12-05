// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"fmt"
	"sort"
	"strings"
)

// Scopes is a bitmask representation of RBAC access. Scopes are done in groups
// of four bits to denote Read, Update, Create, Delete. This 64-bit value gives
// room for 16 different resources.
type Scopes uint64

const (
	scopeR Scopes = 1 << 0
	scopeU Scopes = 1 << 1
	scopeC Scopes = 1 << 2
	scopeD Scopes = 1 << 3

	scopeShiftDevices Scopes = 0
	scopeShiftUpdates Scopes = 4
	scopeShiftUsers   Scopes = 8

	ScopeDevicesR  = scopeR << scopeShiftDevices
	ScopeDevicesRU = (scopeU | scopeR) << scopeShiftDevices
	ScopeDevicesD  = scopeD << scopeShiftDevices

	ScopeUpdatesR  = scopeR << scopeShiftUpdates
	ScopeUpdatesRU = (scopeU | scopeR) << scopeShiftUpdates

	ScopeUsersR  = scopeR << scopeShiftUsers
	ScopeUsersRU = (scopeU | scopeR) << scopeShiftUsers
	ScopeUsersC  = scopeC << scopeShiftUsers
	ScopeUsersD  = scopeD << scopeShiftUsers
)

var maskToString = map[Scopes]string{
	ScopeDevicesR:  "devices:read",
	ScopeDevicesRU: "devices:read-update",
	ScopeDevicesD:  "devices:delete",

	ScopeUpdatesR:  "updates:read",
	ScopeUpdatesRU: "updates:read-update",

	ScopeUsersR:  "users:read",
	ScopeUsersRU: "users:read-update",
	ScopeUsersC:  "users:create",
	ScopeUsersD:  "users:delete",
}

var stringToMask = map[string]Scopes{}
var allScopes []string

func init() {
	for k, v := range maskToString {
		stringToMask[v] = k
		allScopes = append(allScopes, v)
	}
	sort.Strings(allScopes)
}

// ScopesAvailable returns a list of all available scopes as strings for display purposes.
func ScopesAvailable() []string {
	return allScopes
}

// ScopesFromString parses a comma-separated list of scopes into a Scopes bitmask.
func ScopesFromString(scopes string) (Scopes, error) {
	return ScopesFromSlice(strings.Split(scopes, ","))
}

// ScopesFromSlice parses a slice of scope strings into a Scopes bitmask.
func ScopesFromSlice(scopes []string) (Scopes, error) {
	var s Scopes
	for _, scope := range scopes {
		if v, ok := stringToMask[strings.TrimSpace(scope)]; ok {
			s |= v
		} else {
			return 0, fmt.Errorf("invalid scope: `%s`", scope)
		}
	}
	return s, nil
}

func (s Scopes) String() string {
	return strings.Join(s.ToSlice(), ",")
}

func (s Scopes) ToSlice() []string {
	var result []string
	for k, v := range maskToString {
		if s&k == k {
			result = append(result, v)
		}
	}
	sort.Strings(result)

	// Remove "read" entries when "read-update" entries are present for the same prefix
	filteredResult := make([]string, 0, len(result))
	for i, scope := range result {
		if strings.HasSuffix(scope, ":read") {
			// Look ahead to see if the next element is the read-update version
			if i+1 < len(result) && result[i+1] == scope+"-update" {
				continue
			}
		}
		filteredResult = append(filteredResult, scope)
	}

	return filteredResult
}

func (s Scopes) Has(scope Scopes) bool {
	return s&scope == scope
}
