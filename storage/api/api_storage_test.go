// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/dg"
	"github.com/stretchr/testify/require"
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

	dg, err := dg.NewStorage(db, fs)
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
	require.Nil(t, d2.PutFile(storage.Aktoml, []byte("aktoml content")))
	time.Sleep(time.Second)
	_, err = dg.DeviceCreate("uuid-2", "pubkey-value-2", false)
	require.Nil(t, err)

	require.Nil(t, s.SetUpdateName([]string{"uuid-1", "uuid-2"}, "update42"))

	opts.Limit = 2
	opts.OrderBy = OrderByDeviceCreatedAsc
	devices, err = s.DevicesList(opts)
	require.Nil(t, err)
	require.Equal(t, 2, len(devices))
	require.Equal(t, "uuid-1", devices[0].Uuid)
	require.Equal(t, "uuid-2", devices[1].Uuid)

	opts.OrderBy = OrderByDeviceCreatedDsc
	devices, err = s.DevicesList(opts)
	require.Nil(t, err)
	require.Equal(t, 2, len(devices))
	require.Equal(t, "uuid-2", devices[0].Uuid)

	d, err = s.DeviceGet("uuid-1")
	require.Nil(t, err)
	require.False(t, d.IsProd)
	require.Equal(t, "", d.OstreeHash)
	require.Equal(t, "pubkey-value-1", d.PubKey)
	require.Equal(t, "update42", d.UpdateName)
	require.Equal(t, "aktoml content", d.Aktoml)
}
