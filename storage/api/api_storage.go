// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"archive/tar"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/foundriesio/dg-satellite/storage"
)

type (
	Labels  map[string]string
	OrderBy string

	FsHandle = storage.FsHandle

	AppsStates        = storage.AppsStates
	DeviceStatus      = storage.DeviceStatus
	DeviceUpdateEvent = storage.DeviceUpdateEvent
)

const (
	OrderByDeviceLastSeenDsc OrderBy = "last-seen-desc"
	OrderByDeviceLastSeenAsc OrderBy = "last-seen-asc"
	OrderByDeviceCreatedDsc  OrderBy = "created-at-desc"
	OrderByDeviceCreatedAsc  OrderBy = "created-at-asc"
	OrderByDeviceNameAsc     OrderBy = "name-asc"
	OrderByDeviceNameDesc    OrderBy = "name-desc"
	OrderByDeviceUuidAsc     OrderBy = "uuid-asc"
	OrderByDeviceUuidDesc    OrderBy = "uuid-desc"
)

var orderByDeviceMap = map[OrderBy]string{
	OrderByDeviceCreatedAsc:  "created_at ASC",
	OrderByDeviceCreatedDsc:  "created_at DESC",
	OrderByDeviceLastSeenAsc: "last_seen ASC",
	OrderByDeviceLastSeenDsc: "last_seen DESC",
	// Devices with name always come before devices without name
	OrderByDeviceNameAsc:  "name ASC NULLS LAST, uuid ASC",
	OrderByDeviceNameDesc: "name DESC NULLS LAST, uuid DESC",
	OrderByDeviceUuidAsc:  "uuid ASC",
	OrderByDeviceUuidDesc: "uuid DESC",
}

var (
	NewDb = storage.NewDb
	NewFs = storage.NewFs

	DbFile = storage.DbFile

	IsDbError             = storage.IsDbError
	ErrDbConstraintUnique = storage.ErrDbConstraintUnique

	ErrInvalidUpdate = errors.New("invalid update archive")
)

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
	Tag       string `json:"tag"`
	IsProd    bool   `json:"is-prod"`
	Labels    Labels `json:"labels"`
}

type Device struct {
	DeviceListItem

	Apps       []string `json:"apps"`
	OstreeHash string   `json:"ostree-hash"`
	PubKey     string   `json:"pubkey"`
	UpdateName string   `json:"update-name"`

	Aktoml  string `json:"aktualizr-toml"`
	HwInfo  string `json:"hardware-info"`
	NetInfo string `json:"network-info"`

	storage Storage
}

type Rollout struct {
	Uuids  []string `json:"uuids,omitempty"`
	Groups []string `json:"groups,omitempty"`
	Effect []string `json:"effective-uuids,omitempty"`
	Commit bool     `json:"committed"`
}

type Storage struct {
	db *storage.DbHandle
	fs *storage.FsHandle

	stmtDeviceGet       stmtDeviceGet
	stmtDeviceGetGroups stmtDeviceGetGroups
	stmtDeviceGetLabels stmtDeviceGetLabels
	stmtDeviceList      map[OrderBy]stmtDeviceList
	stmtDeviceSetLabels stmtDeviceSetLabels
	stmtDeviceSetUpdate stmtDeviceSetUpdate
}

func (d Device) Updates() ([]string, error) {
	names, err := d.storage.fs.Devices.ListFiles(d.Uuid, storage.EventsPrefix, true)
	if err != nil {
		return nil, err
	}
	for i, name := range names {
		names[i] = name[len(storage.EventsPrefix)+1:]
	}
	slices.Reverse(names)
	return names, nil
}

func (d Device) Events(updateId string) ([]DeviceUpdateEvent, error) {
	name := fmt.Sprintf("%s-%s", storage.EventsPrefix, updateId)
	content, err := d.storage.fs.Devices.ReadFile(d.Uuid, name)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(content, "\n")
	events := make([]DeviceUpdateEvent, 0, len(lines))
	for _, line := range lines {
		if len(line) > 0 {
			var evt DeviceUpdateEvent
			if err := json.Unmarshal([]byte(line), &evt); err != nil {
				return nil, fmt.Errorf("unexpected error unmarshalling event json: %w", err)
			}
			events = append(events, evt)
		}
	}
	return events, nil
}

func (d Device) AppsStates() ([]AppsStates, error) {
	names, err := d.storage.fs.Devices.ListFiles(d.Uuid, storage.StatesPrefix, true)
	if err != nil {
		return nil, err
	}

	states := make([]AppsStates, len(names))
	for i, name := range names {
		content, err := d.storage.fs.Devices.ReadFile(d.Uuid, name)
		if err != nil {
			return nil, err
		}
		var s AppsStates
		if err := json.Unmarshal([]byte(content), &s); err != nil {
			return nil, fmt.Errorf("unexpected error unmarshalling apps states json: %w", err)
		}
		states[len(names)-1-i] = s //store in reverse order
	}
	return states, nil
}

func NewStorage(db *storage.DbHandle, fs *storage.FsHandle) (*Storage, error) {
	handle := Storage{db: db, fs: fs}

	if err := db.InitStmt(
		&handle.stmtDeviceGet,
		&handle.stmtDeviceGetGroups,
		&handle.stmtDeviceGetLabels,
		&handle.stmtDeviceSetLabels,
		&handle.stmtDeviceSetUpdate,
	); err != nil {
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
	var (
		err    error
		apps   string
		labels string
	)
	if err := s.stmtDeviceGet.run(
		uuid,
		&d.CreatedAt, &d.LastSeen,
		&d.PubKey, &d.UpdateName, &d.Tag, &d.Target, &d.OstreeHash,
		&apps, &labels, &d.IsProd,
	); err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return nil, err
	}
	for _, v := range strings.Split(apps, ",") {
		if v = strings.TrimSpace(v); len(v) > 0 {
			d.Apps = append(d.Apps, v)
		}
	}
	if err = json.Unmarshal([]byte(labels), &d.Labels); err != nil {
		return nil, fmt.Errorf("failed to parse device labels: %w", err)
	}

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

func (s Storage) ListUpdates(tag string, isProd bool) (map[string][]string, error) {
	return s.getRolloutsFsHandle(isProd).ListUpdates(tag)
}

func (s Storage) GetUpdateTufMetadata(tag, updateName string, isProd bool) (map[string]map[string]any, error) {
	handle := s.fs.Updates.Ci
	if isProd {
		handle = s.fs.Updates.Prod
	}

	latestRoot, err := handle.Tuf.LatestRootMetaName(tag, updateName)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]map[string]any)
	for _, x := range []string{"targets.json", "snapshot.json", "timestamp.json", latestRoot} {
		metaStr, err := handle.Tuf.ReadFile(tag, updateName, x)
		if err != nil {
			return nil, err
		}
		var metaDict map[string]any
		if err := json.Unmarshal([]byte(metaStr), &metaDict); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", x, err)
		}
		if x == latestRoot {
			x = "root.json"
		}
		meta[x] = metaDict
	}

	return meta, nil
}

func (s Storage) ListRollouts(tag, updateName string, isProd bool) ([]string, error) {
	return s.getRolloutsFsHandle(isProd).ListFiles(tag, updateName)
}

func (s Storage) GetRollout(tag, updateName, rolloutName string, isProd bool) (res Rollout, err error) {
	var content string
	content, err = s.getRolloutsFsHandle(isProd).ReadFile(tag, updateName, rolloutName)
	if err == nil {
		err = json.Unmarshal([]byte(content), &res)
	}
	return
}

func (s Storage) SaveRollout(tag, updateName, rolloutName string, isProd bool, rollout Rollout) error {
	if data, err := json.Marshal(rollout); err != nil {
		return err
	} else {
		return s.getRolloutsFsHandle(isProd).WriteFile(tag, updateName, rolloutName, string(data))
	}
}

func (s Storage) CreateRollout(tag, updateName, rolloutName string, isProd bool, rollout Rollout) error {
	h := s.getRolloutsFsHandle(isProd)
	log := strings.Join([]string{tag, updateName, rolloutName}, "|")
	if data, err := json.Marshal(rollout); err != nil {
		return err
	} else if err := h.AppendJournal(log); err != nil {
		return err
	} else {
		return h.WriteFile(tag, updateName, rolloutName, string(data))
	}
}

func (s Storage) CommitRollout(tag, updateName, rolloutName string, isProd bool, rollout Rollout) (err error) {
	if rollout.Effect, err = s.SetUpdateName(tag, updateName, isProd, rollout.Uuids, rollout.Groups); err != nil {
		return err
	} else {
		rollout.Commit = true
		return s.SaveRollout(tag, updateName, rolloutName, isProd, rollout)
	}
}

func (s Storage) ReadRolloutJournal(isProd bool) iter.Seq2[*[3]string, error] {
	h := s.getRolloutsFsHandle(isProd)
	return func(yield func(*[3]string, error) bool) {
		for log, err := range h.ReadJournal() {
			if err != nil {
				yield(nil, err)
				break
			}
			parts := strings.Split(log, "|")
			if len(parts) != 3 {
				// This is impossible; just a sanity check.
				yield(nil, fmt.Errorf("corrupted journal file line: %s", log))
				break
			}
			// parts are tag, updateName, rolloutName
			if !yield(&[3]string{parts[0], parts[1], parts[2]}, nil) {
				break
			}
		}
	}
}

func (s Storage) RolloverRolloutJournal(isProd bool) error {
	return s.getRolloutsFsHandle(isProd).RolloverJournal()
}

func (s Storage) GetKnownDeviceGroupNames() ([]string, error) {
	return s.stmtDeviceGetGroups.run()
}

func (s Storage) GetKnownDeviceLabelNames() ([]string, error) {
	return s.stmtDeviceGetLabels.run()
}

func (s Storage) PatchDeviceLabels(labels map[string]*string, uuids []string) error {
	// This function applies a merge-patch on top of existing labels:
	// new labels are added, updated labels are replaced, null labels are removed, missing labels are left intact.
	return s.stmtDeviceSetLabels.run(labels, uuids)
}

func (s Storage) SetUpdateName(tag, updateName string, isProd bool, uuids, groups []string) (effectiveUuids []string, err error) {
	err = s.stmtDeviceSetUpdate.run(tag, updateName, isProd, uuids, groups, &effectiveUuids)
	return
}

func (s Storage) TailRolloutsLog(tag, updateName string, isProd bool, stop storage.DoneChan) iter.Seq2[string, error] {
	fs := s.fs.Updates.Ci.Logs
	if isProd {
		fs = s.fs.Updates.Prod.Logs
	}
	return fs.TailFileLines(tag, updateName, storage.LogRolloutsFile, stop)
}

func (s Storage) getRolloutsFsHandle(isProd bool) storage.RolloutsFsHandle {
	if isProd {
		return s.fs.Updates.Prod.Rollouts
	} else {
		return s.fs.Updates.Ci.Rollouts
	}
}

type stmtDeviceGet storage.DbStmt

func (s *stmtDeviceGet) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceGet", `
		SELECT
			created_at, last_seen, pubkey, update_name, tag, target_name, ostree_hash, apps, json(labels), is_prod
		FROM devices
		WHERE uuid = ? AND deleted=false`,
	)
	return
}

func (s *stmtDeviceGet) run(
	uuid string,
	createdAt, lastSeen *int64,
	pubkey, updateName, tag, targetName, ostreeHash, apps, labels *string,
	isProd *bool,
) error {
	return s.Stmt.QueryRow(uuid).Scan(
		createdAt, lastSeen, pubkey, updateName, tag, targetName, ostreeHash, apps, labels, isProd)
}

type stmtDeviceList storage.DbStmt

func (s *stmtDeviceList) Init(db storage.DbHandle, orderBy string) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceList", fmt.Sprintf(`
		SELECT
			uuid, created_at, last_seen, target_name, tag, is_prod, json(labels)
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
		defer func() {
			if err := rows.Close(); err != nil {
				slog.Error("failed to close rows in device list", "error", err)
			}
		}()
		for rows.Next() {
			var (
				d      DeviceListItem
				labels []byte
			)
			if err = rows.Scan(
				&d.Uuid, &d.CreatedAt, &d.LastSeen, &d.Target, &d.Tag, &d.IsProd, &labels,
			); err != nil {
				return err
			}
			if err = json.Unmarshal(labels, &d.Labels); err != nil {
				return fmt.Errorf("failed to parse device labels: %w", err)
			}
			*dl = append(*dl, d)
		}
		if err = rows.Err(); err != nil {
			return err
		}
	}
	return nil
}

type stmtDeviceSetLabels storage.DbStmt

func (s *stmtDeviceSetLabels) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceSetName", `
		UPDATE devices
		SET labels=jsonb_patch(labels,?)
		WHERE uuid IN (SELECT value from json_each(?))`,
	)
	return
}

func (s *stmtDeviceSetLabels) run(labels map[string]*string, uuids []string) error {
	labelsStr, err := json.Marshal(labels)
	if err != nil {
		return fmt.Errorf("unexpected error marshalling labels to JSON: %w", err)
	}
	uuidsStr, err := json.Marshal(uuids)
	if err != nil {
		return fmt.Errorf("unexpected error marshalling UUIDs to JSON: %w", err)
	}
	_, err = s.Stmt.Exec(labelsStr, uuidsStr)
	return err
}

type stmtDeviceGetLabels storage.DbStmt

func (s *stmtDeviceGetLabels) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceGetLabelNames", `SELECT json_group_array(label) FROM device_labels`)
	return
}

func (s *stmtDeviceGetLabels) run() (labels []string, err error) {
	var labelsStr []byte
	if err = s.Stmt.QueryRow().Scan(&labelsStr); err == nil {
		err = json.Unmarshal(labelsStr, &labels)
	}
	return
}

type stmtDeviceGetGroups storage.DbStmt

func (s *stmtDeviceGetGroups) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceGetLabelNames", `
		SELECT json_group_array(DISTINCT group_name) FROM devices
		WHERE group_name != ""
	`)
	return
}

func (s *stmtDeviceGetGroups) run() (groups []string, err error) {
	var groupsStr []byte
	if err = s.Stmt.QueryRow().Scan(&groupsStr); err == nil {
		err = json.Unmarshal(groupsStr, &groups)
	}
	return
}

type stmtDeviceSetUpdate storage.DbStmt

func (s *stmtDeviceSetUpdate) Init(db storage.DbHandle) (err error) {
	s.Stmt, err = db.Prepare("apiDeviceSetUpdateName", `
		UPDATE devices
		SET update_name=?
		WHERE tag=? AND is_prod=? AND (
			uuid IN (SELECT value from json_each(?))
			OR
			group_name IN (SELECT value from json_each(?))
		) RETURNING uuid`,
	)
	return
}

func (s *stmtDeviceSetUpdate) run(tag, updateName string, isProd bool, uuids, groups []string, effectiveUuids *[]string) error {
	uuidsStr, err := json.Marshal(uuids)
	if err != nil {
		return fmt.Errorf("unexpected error marshalling UUIDs to JSON: %w", err)
	}
	groupsStr, err := json.Marshal(groups)
	if err != nil {
		return fmt.Errorf("unexpected error marshalling groups to JSON: %w", err)
	}
	if rows, err := s.Stmt.Query(updateName, tag, isProd, uuidsStr, groupsStr); err != nil {
		return err
	} else {
		var resUuid string
		for rows.Next() {
			if err = rows.Scan(&resUuid); err != nil {
				return err
			}
			*effectiveUuids = append(*effectiveUuids, resUuid)
		}
		if err = rows.Err(); err != nil {
			return err
		}
	}
	return nil
}

// CreateUpdate extracts a tar stream into the update storage directory for the
// given tag and update name. The caller is responsible for decompressing the
// stream (e.g. gzip) before passing it in.
func (s Storage) CreateUpdate(tag, updateName string, isProd bool, body io.Reader) error {
	baseDir := s.fs.Config.UpdatesCiDir()
	if isProd {
		baseDir = s.fs.Config.UpdatesProdDir()
	}
	destDir := filepath.Join(baseDir, tag, updateName)
	if err := extractTar(body, destDir); err != nil {
		err2 := os.RemoveAll(destDir)
		if err2 != nil {
			return errors.Join(err, err2)
		}
		return err
	}
	return nil
}

func extractTar(r io.Reader, destDir string) error {
	tr := tar.NewReader(r)

	var sawTuf, sawOstree, sawApps bool

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		name := filepath.Clean(header.Name)
		target := filepath.Join(destDir, name)

		// Security: prevent directory traversal
		if rel, err := filepath.Rel(destDir, target); err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("tar entry would escape destination directory: %s", header.Name)
		}

		// Track top-level directories for validation
		topDir := strings.SplitN(name, string(filepath.Separator), 2)[0]
		switch topDir {
		case storage.UpdatesTufDir:
			sawTuf = true
		case storage.UpdatesOstreeDir:
			sawOstree = true
		case storage.UpdatesAppsDir:
			sawApps = true
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", header.Name, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", header.Name, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", header.Name, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				if err := f.Close(); err != nil {
					return fmt.Errorf("failed to close file %s after copy error: %w", header.Name, err)
				}
				return fmt.Errorf("failed to write file %s: %w", header.Name, err)
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("failed to close file %s: %w", header.Name, err)
			}
		default:
			return fmt.Errorf("unsupported tar entry type for %s: %c", header.Name, header.Typeflag)
		}
	}

	if !sawTuf {
		return fmt.Errorf("%w: missing required %q directory", ErrInvalidUpdate, storage.UpdatesTufDir)
	}
	if !sawOstree && !sawApps {
		return fmt.Errorf("%w: must contain %q and/or %q directory", ErrInvalidUpdate, storage.UpdatesOstreeDir, storage.UpdatesAppsDir)
	}

	return nil
}
