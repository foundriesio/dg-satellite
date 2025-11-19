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

func TestNewStorage(t *testing.T) {
	tmpdir := t.TempDir()
	dbFile := filepath.Join(tmpdir, "sql.db")
	db, err := storage.NewDb(dbFile)
	require.Nil(t, err)
	fs, err := storage.NewFs(tmpdir)
	require.Nil(t, err)

	require.Nil(t, fs.Certs.WriteFile("hmac.secret", []byte("random")))

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

	ul, err := users.List()
	require.Nil(t, err)
	require.Len(t, ul, 1)
	require.Equal(t, u.Username, ul[0].Username)

	u.Username = "seconduser"
	err = users.Create(&u)
	require.Nil(t, err)

	ul, err = users.List()
	require.Nil(t, err)
	require.Len(t, ul, 2)

	require.Nil(t, u.Delete())
	ul, err = users.List()
	require.Nil(t, err)
	require.Len(t, ul, 1)
	require.Equal(t, "testuser", ul[0].Username)

	ul[0].AllowedScopes = auth.ScopeDevicesD
	require.Nil(t, ul[0].Update())

	u4, err := users.Get("testuser")
	require.Nil(t, err)
	require.NotNil(t, u4)
	require.Equal(t, "devices:delete", u4.AllowedScopes.String())
}

func TestTokens(t *testing.T) {
	tmpdir := t.TempDir()
	dbFile := filepath.Join(tmpdir, "sql.db")
	db, err := storage.NewDb(dbFile)
	require.Nil(t, err)
	fs, err := storage.NewFs(tmpdir)
	require.Nil(t, err)

	require.Nil(t, fs.Certs.WriteFile("hmac.secret", []byte("random")))

	users, err := NewStorage(db, fs)
	require.Nil(t, err)
	require.NotNil(t, users)

	u := User{
		Username:      "testuser",
		Password:      "passwordhash",
		Email:         "testuser@example.com",
		AllowedScopes: auth.ScopeDevicesRU,
	}
	err = users.Create(&u)
	require.Nil(t, err)

	expires := time.Now().Add(1 * time.Hour).Unix()
	t1, err := u.GenerateToken("desc", expires, auth.ScopeDevicesR)
	require.Nil(t, err)

	time.Sleep(time.Second)
	expired := time.Now().Add(-1 * time.Hour).Unix()
	t2, err := u.GenerateToken("desc2", expired, auth.ScopeDevicesR)
	require.Nil(t, err)
	require.NotEqual(t, t1.Value, t2.Value)

	u2, err := users.GetByToken(t1.Value)
	require.Nil(t, err)
	require.NotNil(t, u2)
	require.Equal(t, u.id, u2.id)
	require.True(t, u2.AllowedScopes.Has(auth.ScopeDevicesR))
	require.False(t, u2.AllowedScopes.Has(auth.ScopeDevicesRU))

	u2, err = users.GetByToken(t2.Value)
	require.Nil(t, err)
	require.Nil(t, u2)

	tokens, err := u.ListTokens()
	require.Nil(t, err)
	require.Len(t, tokens, 2)

	require.Equal(t, t1.PublicID, tokens[0].PublicID)
	require.Equal(t, t2.PublicID, tokens[1].PublicID)
	require.Nil(t, u.DeleteToken(tokens[1].PublicID))

	tokens, err = u.ListTokens()
	require.Nil(t, err)
	require.Len(t, tokens, 1)

	require.Nil(t, u.Delete())
	tokens, err = u.ListTokens()
	require.Nil(t, err)
	require.Len(t, tokens, 0)

	_, err = u.GenerateToken("invalid scope", expires, auth.ScopeUsersC)
	require.NotNil(t, err)

	// Generate token with read-update
	t1, err = u.GenerateToken("desc", expires, auth.ScopeDevicesRU)
	require.Nil(t, err)
	// Downgrade user to devices:read
	u.AllowedScopes = auth.ScopeDevicesR
	require.Nil(t, u.Update())
	u2, err = users.GetByToken(t1.Value)
	require.Nil(t, err)
	require.True(t, u2.AllowedScopes.Has(auth.ScopeDevicesR))
	require.False(t, u2.AllowedScopes.Has(auth.ScopeDevicesRU))
}
