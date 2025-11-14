// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package users

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/stretchr/testify/require"
)

func TestUsers(t *testing.T) {
	tmpdir := t.TempDir()
	dbFile := filepath.Join(tmpdir, "sql.db")
	db, err := storage.NewDb(dbFile)
	require.Nil(t, err)
	fs, err := storage.NewFs(tmpdir)
	require.Nil(t, err)

	users, err := NewStorage(db, fs)
	require.Nil(t, err)
	require.NotNil(t, users)

	u := User{
		Username:      "testuser",
		Password:      "passwordhash",
		Email:         "testuser@example.com",
		AllowedScopes: auth.ScopeDevicesR | auth.ScopeUsersRU,
	}
	now := time.Now().Unix()
	err = users.Create(&u)
	require.Nil(t, err)
	require.NotZero(t, u.id)
	require.InDelta(t, now, int64(u.Created), 5)

	u2, err := users.Get("testuser")
	require.Nil(t, err)
	require.NotNil(t, u2)
	require.Equal(t, u.id, u2.id)
	require.Equal(t, u.Username, u2.Username)
	require.Equal(t, u.Password, u2.Password)
	require.Equal(t, u.Email, u2.Email)
	require.Equal(t, u.AllowedScopes, u2.AllowedScopes)

	require.True(t, u2.AllowedScopes.Has(auth.ScopeDevicesR))
	require.False(t, u2.AllowedScopes.Has(auth.ScopeDevicesD))
	require.Equal(t, []string{"devices:read", "users:read-update"}, u2.AllowedScopes.ToSlice())

	require.NotNil(t, users.Create(u2), "duplicate username should fail")

	u3, err := users.Get("nonexistent")
	require.Nil(t, err)
	require.Nil(t, u3)
}
