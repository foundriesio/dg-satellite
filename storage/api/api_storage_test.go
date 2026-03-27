// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/gateway"
)

func TestStorage(t *testing.T) {
	tmpdir := t.TempDir()
	dbFile := filepath.Join(tmpdir, "sql.db")
	db, err := storage.NewDb(dbFile)
	require.Nil(t, err)
	fs, err := storage.NewFs(tmpdir)
	require.Nil(t, err)

	s, err := NewStorage(db, fs)
	require.Nil(t, err)

	dg, err := gateway.NewStorage(db, fs)
	require.Nil(t, err)

	// Test 404 type operation
	d, err := s.DeviceGet("does not exist")
	require.Nil(t, err)
	require.Nil(t, d)

	// Test we can list when there are no devices
	opts := DeviceListOpts{}
	devices, err := s.DevicesList(opts)
	require.Nil(t, err)
	require.Equal(t, 0, len(devices))

	// Create two devices to list/get on
	d2, err := dg.DeviceCreate("uuid-1", "pubkey-value-1", false)
	require.Nil(t, err)
	require.Nil(t, d2.PutFile(storage.AktomlFile, "aktoml content"))
	require.Nil(t, d2.CheckIn("target", "tag", "hash", ""))
	time.Sleep(time.Second)
	_, err = dg.DeviceCreate("uuid-2", "pubkey-value-2", false)
	require.Nil(t, err)

	uuids, err := s.SetUpdateName("tag", "update42", false, []string{"uuid-1", "uuid-2"}, nil)
	require.Nil(t, err)
	require.Equal(t, 1, len(uuids))
	assert.Equal(t, "uuid-1", uuids[0])

	opts.Limit = 2
	opts.OrderBy = OrderByDeviceCreatedAsc
	devices, err = s.DevicesList(opts)
	require.Nil(t, err)
	require.Equal(t, 2, len(devices))
	assert.Equal(t, "uuid-1", devices[0].Uuid)
	assert.Equal(t, "uuid-2", devices[1].Uuid)

	opts.OrderBy = OrderByDeviceCreatedDsc
	devices, err = s.DevicesList(opts)
	require.Nil(t, err)
	require.Equal(t, 2, len(devices))
	assert.Equal(t, "uuid-2", devices[0].Uuid)

	d, err = s.DeviceGet("uuid-1")
	require.Nil(t, err)
	assert.False(t, d.IsProd)
	assert.Equal(t, "hash", d.OstreeHash)
	assert.Equal(t, "tag", d.Tag)
	assert.Equal(t, "pubkey-value-1", d.PubKey)
	assert.Equal(t, "update42", d.UpdateName)
	assert.Equal(t, "aktoml content", d.Aktoml)
}

func TestDeviceDelete(t *testing.T) {
	tmpdir := t.TempDir()
	dbFile := filepath.Join(tmpdir, "sql.db")
	db, err := storage.NewDb(dbFile)
	require.Nil(t, err)
	fs, err := storage.NewFs(tmpdir)
	require.Nil(t, err)

	s, err := NewStorage(db, fs)
	require.Nil(t, err)

	dg, err := gateway.NewStorage(db, fs)
	require.Nil(t, err)

	// Create a device
	_, err = dg.DeviceCreate("uuid-del", "pubkey-del", false)
	require.Nil(t, err)

	// Verify it exists
	d, err := s.DeviceGet("uuid-del")
	require.Nil(t, err)
	require.NotNil(t, d)

	// Delete it
	require.Nil(t, d.Delete())

	// Verify it no longer shows up in Get or List
	d, err = s.DeviceGet("uuid-del")
	require.Nil(t, err)
	require.Nil(t, d, "deleted device should not be returned by DeviceGet")

	devices, err := s.DevicesList(DeviceListOpts{Limit: 100})
	require.Nil(t, err)
	for _, dev := range devices {
		assert.NotEqual(t, "uuid-del", dev.Uuid, "deleted device should not appear in DevicesList")
	}
}
