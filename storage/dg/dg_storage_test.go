// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package dg

import (
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/foundriesio/dg-satellite/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStorage(t *testing.T) {
	tmpdir := t.TempDir()
	dbFile := filepath.Join(tmpdir, "sql.db")
	db, err := storage.NewDb(dbFile)
	require.Nil(t, err)
	t.Cleanup(func() {
		require.Nil(t, db.Close())
	})
	fs, err := storage.NewFs(tmpdir)
	require.Nil(t, err)

	s, err := NewStorage(db, fs)
	require.Nil(t, err)

	d, err := s.DeviceGet("does not exist")
	require.Nil(t, err)
	require.Nil(t, d)

	uuid := "1234-567-890"
	d, err = s.DeviceCreate(uuid, "pubkey", true)
	require.Nil(t, err)

	d2, err := s.DeviceGet(uuid)
	require.Nil(t, err)
	require.Equal(t, d.PubKey, d2.PubKey)
	require.Equal(t, d.IsProd, d2.IsProd)

	time.Sleep(time.Second)
	require.Nil(t, d2.CheckIn("target", "tag", "hash", []string{}))
	d2, err = s.DeviceGet(uuid)
	require.Nil(t, err)
	require.Less(t, d.LastSeen, d2.LastSeen)

	require.Nil(t, d2.PutFile(storage.Aktoml, "test content"))
	content, err := fs.ReadFile(d2.Uuid, storage.Aktoml)
	require.Nil(t, err)
	require.Equal(t, "test content", content)
}

// Benchmark_CheckIn simulates 100 random device checking in 100_000 times
func Benchmark_CheckIn(b *testing.B) {
	tmpdir := b.TempDir()
	dbFile := filepath.Join(tmpdir, "sql.db")
	db, err := storage.NewDb(dbFile)
	require.Nil(b, err)
	b.Cleanup(func() {
		require.Nil(b, db.Close())
	})
	fs, err := storage.NewFs(tmpdir)
	require.Nil(b, err)

	s, err := NewStorage(db, fs)
	require.Nil(b, err)

	// Create fake devices
	var devices []*DgDevice
	for range 100 {
		id := uuid.New().String()
		d, err := s.DeviceCreate(id, "pubkey"+id, false)
		require.Nil(b, err)
		devices = append(devices, d)
	}

	b.StartTimer()
	for range 100000 {
		deviceIdx := rand.Intn(len(devices) - 1)
		require.Nil(b, devices[deviceIdx].CheckIn("target", "tag", "hash", []string{}))
	}
	b.StopTimer()
}
