// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/foundriesio/dg-satellite/storage"
)

type OrderBy string

const (
	OrderByDeviceLastSeenDsc OrderBy = "last-seen-desc"
	OrderByDeviceLastSeenAsc OrderBy = "last-seen-asc"
	OrderByDeviceCreatedDsc  OrderBy = "created-at-desc"
	OrderByDeviceCreatedAsc  OrderBy = "created-at-asc"
)

var orderByDeviceMap = map[OrderBy]string{
	OrderByDeviceCreatedAsc:  "created_at ASC",
	OrderByDeviceCreatedDsc:  "created_at DESC",
	OrderByDeviceLastSeenAsc: "last_seen ASC",
	OrderByDeviceLastSeenDsc: "last_seen DESC",
}

// DeviceListOpts lets you set the order devices will be returned
// by the `List` api
type DeviceListOpts struct {
	OrderBy OrderBy `query:"order-by" example:"1"    default:"1"`
	Limit   int     `query:"limit"    example:"100"  default:"1000"`
	Offset  int     `query:"offset"   example:"1"    default:"0"`
}

type DeviceListItem struct {
	Uuid      string `json:"uuid"`
	CreatedAt int64  `json:"created-at"`
	LastSeen  int64  `json:"last-seen"`
	Target    string `json:"target"`
	IsProd    bool   `json:"is-prod"`
}

type Device struct {
	DeviceListItem

	OstreeHash string `json:"ostree-hash"`
	PubKey     string `json:"pubkey"`
	Tag        string `json:"tag"`
	UpdateName string `json:"update-name"`

	Aktoml  string `json:"aktualizr-toml"`
	HwInfo  string `json:"hardware-info"`
	NetInfo string `json:"network-info"`

	storage Storage
}

type Storage struct {
	db *storage.DbHandle
	fs *storage.FsHandle

	stmtDeviceGet       stmtDeviceGet
	stmtDeviceList      map[OrderBy]stmtDeviceList
	stmtDeviceSetUpdate stmtDeviceSetUpdate
}

func NewStorage(db *storage.DbHandle, fs *storage.FsHandle) (*Storage, error) {
	handle := Storage{db: db, fs: fs}

	if err := db.InitStmt(&handle.stmtDeviceGet, &handle.stmtDeviceSetUpdate); err != nil {
		return nil, err
	}

	handle.stmtDeviceList = make(map[OrderBy]stmtDeviceList, len(orderByDeviceMap))
	for orderBy, orderByStr := range orderByDeviceMap {
		stmt := stmtDeviceList{}
		if err := stmt.Init(*db, orderByStr); err != nil {
			return nil, err
		}
		handle.stmtDeviceList[orderBy] = stmt
	}

	return &handle, nil
}

func (s Storage) DevicesList(opts DeviceListOpts) ([]DeviceListItem, error) {
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = OrderByDeviceLastSeenDsc
	}
	stmt, ok := s.stmtDeviceList[orderBy]
	if !ok {
		return nil, fmt.Errorf("invalid order by arg: %s", opts.OrderBy)
	}

	devices := make([]DeviceListItem, 0, opts.Limit)
	if err := stmt.run(opts.Limit, opts.Offset, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

func (s Storage) DeviceGet(uuid string) (*Device, error) {
	d := Device{storage: s, DeviceListItem: DeviceListItem{Uuid: uuid}}
	if err := s.stmtDeviceGet.run(
		uuid, &d.CreatedAt, &d.LastSeen, &d.PubKey, &d.UpdateName, &d.Tag, &d.Target, &d.OstreeHash, &d.IsProd,
	); err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return nil, err
	}

	var err error
	if d.Aktoml, err = s.fs.Devices.ReadFile(d.Uuid, storage.AktomlFile); err != nil {
		return nil, err
	}
	if d.HwInfo, err = s.fs.Devices.ReadFile(d.Uuid, storage.HwInfoFile); err != nil {
		return nil, err
	}
	if d.NetInfo, err = s.fs.Devices.ReadFile(d.Uuid, storage.NetInfoFile); err != nil {
		return nil, err
	}

	return &d, nil
}

func (s Storage) SetUpdateName(uuids []string, updateName string) error {
	return s.stmtDeviceSetUpdate.run(uuids, updateName)
}

type stmtDeviceGet storage.DbStmt

func (s *stmtDeviceGet) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceGet", `
		SELECT
			created_at, last_seen, pubkey, update_name, tag, target_name, ostree_hash, is_prod
		FROM devices
		WHERE uuid = ? AND deleted=false`,
	)
	return
}

func (s *stmtDeviceGet) run(
	uuid string,
	createdAt, lastSeen *int64,
	pubkey, updateName, tag, targetName, ostreeHash *string,
	isProd *bool,
) error {
	return s.Stmt.QueryRow(uuid).Scan(createdAt, lastSeen, pubkey, updateName, tag, targetName, ostreeHash, isProd)
}

type stmtDeviceList storage.DbStmt

func (s *stmtDeviceList) Init(db storage.DbHandle, orderBy string) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceList", fmt.Sprintf(`
		SELECT
			uuid, created_at, last_seen, target_name, is_prod
		FROM devices
		WHERE deleted=false
		ORDER BY %s LIMIT ? OFFSET ?`, orderBy),
	)
	return
}

func (s *stmtDeviceList) run(limit, offset int, dl *[]DeviceListItem) error {
	if rows, err := s.Stmt.Query(limit, offset); err != nil {
		return err
	} else {
		for rows.Next() {
			var d DeviceListItem
			if err = rows.Scan(&d.Uuid, &d.CreatedAt, &d.LastSeen, &d.Target, &d.IsProd); err != nil {
				return err
			}
			*dl = append(*dl, d)
		}
		if err = rows.Err(); err != nil {
			return err
		}
	}
	return nil
}

type stmtDeviceSetUpdate storage.DbStmt

func (s *stmtDeviceSetUpdate) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceSetUpdateName", `
		UPDATE devices
		SET update_name=?
		WHERE uuid IN (SELECT value from json_each(?))`,
	)
	return
}

func (s *stmtDeviceSetUpdate) run(uuids []string, updateName string) error {
	uuidsStr, err := json.Marshal(uuids)
	if err != nil {
		return fmt.Errorf("unexpected error marshalling UUIDs to JSON: %w", err)
	}
	_, err = s.Stmt.Exec(updateName, uuidsStr)
	return err
}

func (d Device) Updates() ([]string, error) {
	names, err := d.storage.fs.Devices.ListFiles(d.Uuid, storage.EventsPrefix, true)
	if err != nil {
		return nil, err
	}
	for i, name := range names {
		names[i] = name[len(storage.EventsPrefix)+1:]
	}
	return names, nil
}

func (d Device) Events(updateId string) ([]storage.DeviceUpdateEvent, error) {
	name := fmt.Sprintf("%s-%s", storage.EventsPrefix, updateId)
	content, err := d.storage.fs.Devices.ReadFile(d.Uuid, name)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(content, "\n")
	events := make([]storage.DeviceUpdateEvent, 0, len(lines))
	for _, line := range lines {
		if len(line) > 0 {
			var evt storage.DeviceUpdateEvent
			if err := json.Unmarshal([]byte(line), &evt); err != nil {
				return nil, fmt.Errorf("unexpected error unmarshalling event json: %w", err)
			}
			events = append(events, evt)
		}
	}
	return events, nil
}
