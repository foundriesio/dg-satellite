// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package dg

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/foundriesio/dg-satellite/storage"
)

type Storage struct {
	db *storage.DbHandle
	fs *storage.FsHandle

	stmtDeviceCheckIn stmtDeviceCheckIn
	stmtDeviceCreate  stmtDeviceCreate
	stmtDeviceGet     stmtDeviceGet

	maxEvents int
}

type DgDevice struct {
	storage Storage

	Uuid       string
	PubKey     string
	UpdateName string
	Deleted    bool
	LastSeen   int64
	IsProd     bool
}

func (d *DgDevice) CheckIn(targetName, tag, ostreeHash string, apps []string) error {
	appsStr := strings.Join(apps, ",")
	now := time.Now().Unix()
	return d.storage.stmtDeviceCheckIn.run(d.Uuid, targetName, tag, ostreeHash, appsStr, now)
}

func (d *DgDevice) PutFile(name string, content string) error {
	return d.storage.fs.Devices.WriteFile(d.Uuid, name, content)
}

func (d DgDevice) ProcessEvents(events []storage.DeviceUpdateEvent) error {
	var corrId string
	for _, evt := range events {
		if corrId != "" && corrId != evt.Event.CorrelationId {
			// Events ordering depends onto ModTime.
			// Make sure that a later events file gets a later ModTime.
			// Tests show that filesystem's time precision is good enough for 4 milliseconds delay.
			time.Sleep(4 * time.Millisecond)
		}
		corrId = evt.Event.CorrelationId
		name := fmt.Sprintf("%s-%s", storage.EventsPrefix, corrId)
		bytes, err := json.Marshal(evt)
		if err != nil {
			return err
		}
		if err := d.storage.fs.Devices.AppendFile(d.Uuid, name, string(bytes)+"\n"); err != nil {
			return err
		}
	}
	return d.storage.fs.Devices.RolloverFiles(d.Uuid, storage.EventsPrefix, d.storage.maxEvents)
}

func NewStorage(db *storage.DbHandle, fs *storage.FsHandle) (*Storage, error) {
	handle := Storage{
		db:        db,
		fs:        fs,
		maxEvents: 20,
	}

	if err := db.InitStmt(
		&handle.stmtDeviceCheckIn,
		&handle.stmtDeviceCreate,
		&handle.stmtDeviceGet,
	); err != nil {
		return nil, err
	}

	return &handle, nil
}

func (s Storage) DeviceCreate(uuid, pubkey string, isProd bool) (*DgDevice, error) {
	now := time.Now().Unix()
	if err := s.stmtDeviceCreate.run(uuid, pubkey, now, now, isProd); err != nil {
		return nil, err
	}

	d := DgDevice{
		storage: s,
		Uuid:    uuid,

		Deleted:  false,
		LastSeen: now,
		PubKey:   pubkey,
	}
	return &d, nil
}

func (s Storage) DeviceGet(uuid string) (*DgDevice, error) {
	d := DgDevice{storage: s, Uuid: uuid}
	if err := s.stmtDeviceGet.run(uuid, &d); err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return nil, err
	}
	return &d, nil
}

type stmtDeviceCheckIn storage.DbStmt

func (s *stmtDeviceCheckIn) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("DeviceCheckIn", `
		UPDATE devices
		SET target_name=?, tag=?, ostree_hash=?, apps=?, last_seen=?
		WHERE uuid = ?`,
	)
	return
}

func (s *stmtDeviceCheckIn) run(uuid, targetName, tag, ostreeHash, apps string, lastSeen int64) error {
	_, err := s.Stmt.Exec(targetName, tag, ostreeHash, apps, lastSeen, uuid)
	return err
}

type stmtDeviceCreate storage.DbStmt

func (s *stmtDeviceCreate) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("DeviceCreate", `
		INSERT INTO devices(uuid, pubkey, created_at, last_seen, is_prod, update_name, tag, target_name, ostree_hash, deleted)
		VALUES (?, ?, ?, ?, ?, "", "", "", "", false)`,
	)
	return
}

func (s *stmtDeviceCreate) run(uuid, pubkey string, createdAt, lastSeen int64, isProd bool) error {
	_, err := s.Stmt.Exec(uuid, pubkey, createdAt, lastSeen, isProd)
	return err
}

type stmtDeviceGet storage.DbStmt

func (s *stmtDeviceGet) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("DeviceGet", `
		SELECT deleted, pubkey, update_name, last_seen
		FROM devices
		WHERE uuid = ?`,
	)
	return
}

func (s *stmtDeviceGet) run(uuid string, d *DgDevice) error {
	return s.Stmt.QueryRow(uuid).Scan(&d.Deleted, &d.PubKey, &d.UpdateName, &d.LastSeen)
}
