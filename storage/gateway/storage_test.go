// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package gateway

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/foundriesio/dg-satellite/storage"
	"github.com/foundriesio/dg-satellite/storage/api"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type UpdateEvents []storage.DeviceUpdateEvent

func (ue UpdateEvents) generate(pack string) UpdateEvents {
	num := rand.Intn(3) + 2

	corId := uuid.New().String()
	events := make([]storage.DeviceUpdateEvent, num)
	for i := 0; i < num; i++ {
		events[i] = storage.DeviceUpdateEvent{
			Id:         fmt.Sprintf("%d_%s", i, corId),
			DeviceTime: "2023-12-12T12:00:00",
			Event: storage.DeviceEvent{
				CorrelationId: corId,
				Ecu:           "",
				Success:       nil,
				TargetName:    "intel-corei7-64-lmp-23",
				Version:       "23",
				Details:       pack,
			},
			EventType: storage.DeviceEventType{
				Id:      corId,
				Version: 0,
			},
		}
	}
	return events
}

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
	require.Nil(t, d2.CheckIn("target", "tag", "hash", ""))
	d2, err = s.DeviceGet(uuid)
	require.Nil(t, err)
	require.Less(t, d.LastSeen, d2.LastSeen)

	require.Nil(t, d2.PutFile(storage.AktomlFile, "test content"))
	content, err := fs.Devices.ReadFile(d2.Uuid, storage.AktomlFile)
	require.Nil(t, err)
	require.Equal(t, "test content", content)
}

func Test_ProcessEvents(t *testing.T) {
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

	// Create fake device
	id := uuid.New().String()
	d, err := s.DeviceCreate(id, "pubkey", false)
	require.Nil(t, err)

	var events UpdateEvents
	for i := 0; i < s.maxEvents+3; i++ {
		pack := fmt.Sprintf("test-%d", i)
		events = events.generate(pack)
		require.Nil(t, d.ProcessEvents(events))
		time.Sleep(4 * time.Millisecond)
	}

	validate := func(files []string, skip int) {
		require.Equal(t, s.maxEvents, len(files))
		for i, name := range files {
			pack := fmt.Sprintf("test-%d", i+skip) // Some initial events must get stripped
			content, err := fs.Devices.ReadFile(d.Uuid, name)
			require.Nil(t, err)
			for _, line := range strings.Split(content, "\n") {
				if len(line) == 0 {
					continue
				}
				var evt storage.DeviceUpdateEvent
				require.Nil(t, json.Unmarshal([]byte(line), &evt))
				require.Equal(t, pack, evt.Event.Details)
			}
		}
	}

	files, err := fs.Devices.ListFiles(d.Uuid, storage.EventsPrefix, true)
	require.Nil(t, err)
	validate(files, 3)

	// Special case - some events roll over to the next pack.
	lastEventCorrId := events[0].Event.CorrelationId
	lastEventPack := events[0].Event.Details
	newPack := fmt.Sprintf("test-%d", s.maxEvents+3)
	events = events.generate(newPack)
	events[0].Event.CorrelationId = lastEventCorrId
	events[0].Event.Details = lastEventPack
	require.Nil(t, d.ProcessEvents(events))

	files, err = fs.Devices.ListFiles(d.Uuid, storage.EventsPrefix, true)
	require.Nil(t, err)
	validate(files, 4)

	// TODO: Add fine-grained unit tests for SaveAppsStates
}

func Benchmark_ProcessEvents(b *testing.B) {
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
	var devices []*Device
	for i := 0; i < 10; i++ {
		id := uuid.New().String()
		d, err := s.DeviceCreate(id, "pubkey", false)
		require.Nil(b, err)
		devices = append(devices, d)
	}
	require.Nil(b, err)

	b.StartTimer()
	var events UpdateEvents
	for i := 0; i < 100000; i++ {
		events = events.generate("test")
		deviceIdx := rand.Intn(len(devices) - 1)
		require.Nil(b, devices[deviceIdx].ProcessEvents(events))
	}
	b.StopTimer()
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
	var devices []*Device
	for range 100 {
		id := uuid.New().String()
		d, err := s.DeviceCreate(id, "pubkey"+id, false)
		require.Nil(b, err)
		devices = append(devices, d)
	}

	b.StartTimer()
	for range 100000 {
		deviceIdx := rand.Intn(len(devices) - 1)
		require.Nil(b, devices[deviceIdx].CheckIn("target", "tag", "hash", ""))
	}
	b.StopTimer()
}

func Test_Fiotest(t *testing.T) {
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

	// Create fake device
	id := uuid.New().String()
	d, err := s.DeviceCreate(id, "pubkey", false)
	require.Nil(t, err)

	require.Nil(t, d.TestCreate("intel-corei7-64-lmp-23", "test1", "test1-id"))
	require.Nil(t, d.TestCreate("intel-corei7-64-lmp-23", "test1", "test2-id"))

	require.Nil(t, d.TestComplete("test1-id", "PASSED", "details", nil))

	results := []storage.TargetTestResult{
		{
			Name:    "res1",
			Status:  "PASSED",
			Details: "details",
		},
	}
	require.Nil(t, d.TestComplete("test2-id", "FAILED", "details", results))

	// A little lazy, but test the REST API code from here as well
	api, err := api.NewStorage(db, fs)
	require.Nil(t, err)

	apiD, err := api.DeviceGet(d.Uuid)
	require.Nil(t, err)
	tests, err := apiD.Tests()
	require.Nil(t, err)
	require.Len(t, tests, 2)

	require.Equal(t, "test1-id", tests[0].Uuid)
	require.Equal(t, "test1", tests[0].Name)
	require.Equal(t, "intel-corei7-64-lmp-23", tests[0].TargetName)
	require.Equal(t, "PASSED", tests[0].Status)
	require.NotNil(t, tests[0].CompletedOn)
	require.Len(t, tests[0].Results, 0)

	require.Equal(t, "test2-id", tests[1].Uuid)
	require.Equal(t, "test1", tests[1].Name)
	require.Equal(t, "intel-corei7-64-lmp-23", tests[1].TargetName)
	require.Equal(t, "FAILED", tests[1].Status)
	require.NotNil(t, tests[0].CompletedOn)
	require.Len(t, tests[1].Results, 1)
	require.Equal(t, "res1", tests[1].Results[0].Name)

	require.Nil(t, d.TestStoreArtifact("test1-id", "artifact.txt", strings.NewReader("artifact content")))
	path := apiD.TestArtifactPath("test1-id", "artifact.txt")
	require.Nil(t, err)
	content, err := os.ReadFile(path)
	require.Nil(t, err)
	require.Equal(t, "artifact content", string(content))
}
