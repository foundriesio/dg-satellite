// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth_test

import (
	"slices"
	"testing"

	"github.com/foundriesio/dg-satellite/auth"
)

func TestScopesFromString(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		scopes  string
		want    auth.Scopes
		has     []auth.Scopes
		wantErr bool
	}{
		{
			name:    "Invalid resource",
			scopes:  "device:read,users:read-update",
			wantErr: true,
		},
		{
			name:    "Invalid scope",
			scopes:  "devices:read,users:read-updat",
			wantErr: true,
		},
		{
			name:    "Nonexistent scope",
			scopes:  "devices:read,updates:delete",
			wantErr: true,
		},
		{
			name:   "Handle white space",
			scopes: "devices:read, users:read-update",
			want:   auth.ScopeDevicesR | auth.ScopeUsersRU,
			has:    []auth.Scopes{auth.ScopeDevicesR, auth.ScopeUsersR},
		},
		{
			name:   "Normalize supersets",
			scopes: "devices:read, devices:read-update,updates:read",
			want:   auth.ScopeDevicesRU | auth.ScopeUpdatesR,
			has:    []auth.Scopes{auth.ScopeDevicesR, auth.ScopeDevicesRU, auth.ScopeUpdatesR},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := auth.ScopesFromString(tt.scopes)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ScopesFromString() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ScopesFromString() succeeded unexpectedly")
			}

			if got != tt.want {
				t.Errorf("ScopesFromString() = %v, want %v", got, tt.want)
			}

			for _, h := range tt.has {
				if !got.Has(h) {
					t.Errorf("ScopesFromString().Has(%v) = false, want true", h)
				}
			}
			if got.Has(auth.ScopeDevicesD) {
				t.Errorf("ScopesFromString().Has(devices:delete) = true, want false")
			}
		})
	}
}

func TestScopes_ToSlice(t *testing.T) {
	tests := []struct {
		name   string // description of this test case
		scopes auth.Scopes
		want   []string
	}{
		{
			name:   "devices:read and users:read-update",
			scopes: auth.ScopeDevicesR | auth.ScopeUsersRU,
			want:   []string{"devices:read", "users:read-update"},
		},
		{
			name:   "devices:read, users:read, users:delete",
			scopes: auth.ScopeDevicesR | auth.ScopeUsersR | auth.ScopeUsersD,
			want:   []string{"devices:read", "users:delete", "users:read"},
		},
		{
			name:   "users:read-update, users:create, updates:read-update",
			scopes: auth.ScopeUsersRU | auth.ScopeUsersC | auth.ScopeUpdatesRU,
			want:   []string{"updates:read-update", "users:create", "users:read-update"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asSlice := tt.scopes.ToSlice()
			if !slices.Equal(asSlice, tt.want) {
				t.Fatalf("ToSlice() = %v, want %v", asSlice, tt.want)
			}

			got, err := auth.ScopesFromSlice(asSlice)
			if err != nil {
				t.Fatalf("ScopesFromSlice() failed: %v", err)
			}
			if got != tt.scopes {
				t.Errorf("ScopesFromSlice() = %v, want %v", got, tt.scopes)
			}
		})
	}
}
