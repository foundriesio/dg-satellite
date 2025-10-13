// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// Global files/dirs
	CertsDir   = "certs"
	DbFile     = "db.sqlite"
	DevicesDir = "devices"
	UpdatesDir = "updates"

	partialFileSuffix = "..part"

	CertsCasPemFile = "cas.pem"
	CertsTlsCsrFile = "tls.csr"
	CertsTlsKeyFile = "tls.key"
	CertsTlsPemFile = "tls.pem"

	// Per device files/dirs
	AktomlFile   = "aktoml"
	HwInfoFile   = "hardware-info"
	NetInfoFile  = "network-info"
	EventsPrefix = "events"
	StatesPrefix = "apps-states"

	// Per update files/dirs
	// Update roots
	UpdatesCiDir   = "ci"
	UpdatesProdDir = "prod"
	// Update categories
	UpdatesTufDir      = "tuf"
	UpdatesOstreeDir   = "ostree_repo"
	UpdatesRolloutsDir = "rollouts"
	// TUF category files
	TufRootFile      = "root.json"
	TufTimestampFile = "timestamp.json"
	TufSnapshotFile  = "snapshot.json"
	TufTargetsFile   = "targets.json"
)

type FsConfig string

func (c FsConfig) RootDir() string {
	return string(c)
}

func (c FsConfig) DbFile() string {
	return filepath.Join(string(c), DbFile)
}

func (c FsConfig) CertsDir() string {
	return filepath.Join(string(c), CertsDir)
}

func (c FsConfig) DevicesDir() string {
	return filepath.Join(string(c), DevicesDir)
}

func (c FsConfig) UpdatesDir() string {
	return filepath.Join(string(c), UpdatesDir)
}

func (c FsConfig) UpdatesCiDir() string {
	return filepath.Join(c.UpdatesDir(), UpdatesCiDir)
}

func (c FsConfig) UpdatesProdDir() string {
	return filepath.Join(c.UpdatesDir(), UpdatesProdDir)
}

type FsHandle struct {
	Config FsConfig

	Certs   CertsFsHandle
	Devices DevicesFsHandle
	Updates struct {
		Ci struct {
			Ostree   UpdatesFsHandle
			Tuf      UpdatesFsHandle
			Rollouts RolloutsFsHandle
		}
		Prod struct {
			Ostree   UpdatesFsHandle
			Tuf      UpdatesFsHandle
			Rollouts RolloutsFsHandle
		}
	}
}

func NewFs(root string) (*FsHandle, error) {
	fs := &FsHandle{Config: FsConfig(root)}
	fs.Certs.root = fs.Config.CertsDir()
	fs.Devices.root = fs.Config.DevicesDir()
	fs.Updates.Ci.Ostree.root = fs.Config.UpdatesCiDir()
	fs.Updates.Ci.Ostree.category = UpdatesOstreeDir
	fs.Updates.Ci.Rollouts.root = fs.Config.UpdatesCiDir()
	fs.Updates.Ci.Rollouts.category = UpdatesRolloutsDir
	fs.Updates.Ci.Tuf.root = fs.Config.UpdatesCiDir()
	fs.Updates.Ci.Tuf.category = UpdatesTufDir
	fs.Updates.Prod.Ostree.root = fs.Config.UpdatesProdDir()
	fs.Updates.Prod.Ostree.category = UpdatesOstreeDir
	fs.Updates.Prod.Rollouts.root = fs.Config.UpdatesProdDir()
	fs.Updates.Prod.Rollouts.category = UpdatesRolloutsDir
	fs.Updates.Prod.Tuf.root = fs.Config.UpdatesProdDir()
	fs.Updates.Prod.Tuf.category = UpdatesTufDir

	for _, h := range []struct {
		handle baseFsHandle
		mode   os.FileMode
	}{
		{fs.Certs.baseFsHandle, 0o744},
		{fs.Devices.baseFsHandle, 0o740},
		// All updates categories have the same base dir, so only one of Ci/prod is needed.
		{fs.Updates.Ci.Tuf.baseFsHandle, 0o744},
		{fs.Updates.Prod.Tuf.baseFsHandle, 0o744},
	} {
		if err := h.handle.mkdirs(h.mode, true); err != nil {
			return nil, fmt.Errorf("unable to initialize file storage: %w", err)
		}
	}
	return fs, nil
}

type CertsFsHandle struct {
	baseFsHandle
}

func (s CertsFsHandle) FilePath(name string) string {
	return filepath.Join(s.root, name)
}

func (s CertsFsHandle) ReadFile(name string) ([]byte, error) {
	content, err := s.readFile(name, false)
	if err != nil {
		err = fmt.Errorf("error reading file %s: %w", name, err)
	}
	return []byte(content), err
}

func (s CertsFsHandle) WriteFile(name string, content []byte) error {
	if err := s.writeFile(name, string(content), 0o740); err != nil {
		return fmt.Errorf("error writing file %s: %w", name, err)
	}
	return nil
}

func (s CertsFsHandle) AssertCleanTls() error {
	for _, name := range []string{
		CertsTlsCsrFile, CertsTlsKeyFile, CertsTlsPemFile,
	} {
		if _, err := os.Stat(filepath.Join(s.root, name)); err == nil {
			return fmt.Errorf("a TLS file %s already exists: %w", name, os.ErrExist)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to check if a TLS file %s exists: %w", name, err)
		}
	}
	return nil
}

type DevicesFsHandle struct {
	baseFsHandle
}

func (s DevicesFsHandle) ReadFile(uuid, name string) (string, error) {
	h, _ := s.deviceLocalHandle(uuid, false)
	content, err := h.readFile(name, true)
	if err != nil {
		err = fmt.Errorf("unexpected error reading file %s for device %s: %w", name, uuid, err)
	}
	return content, err
}

func (s DevicesFsHandle) WriteFile(uuid, name, content string) error {
	if h, err := s.deviceLocalHandle(uuid, true); err != nil {
		return err
	} else if err = h.writeFile(name, content, 0o744); err != nil {
		return fmt.Errorf("error writing file %s for device %s: %w", name, uuid, err)
	}
	return nil
}

func (s DevicesFsHandle) AppendFile(uuid, name, content string) error {
	if h, err := s.deviceLocalHandle(uuid, true); err != nil {
		return err
	} else if err = h.appendFile(name, content, 0o744); err != nil {
		return fmt.Errorf("error writing file %s for device %s: %w", name, uuid, err)
	}
	return nil
}

func (s DevicesFsHandle) ListFiles(uuid, prefix string, sortByModTime bool) ([]string, error) {
	h, _ := s.deviceLocalHandle(uuid, false)
	names, err := h.matchFiles(prefix, sortByModTime)
	if err != nil {
		err = fmt.Errorf("error listing %s files for device %s: %w", prefix, uuid, err)
	}
	return names, err
}

func (s DevicesFsHandle) RolloverFiles(uuid, prefix string, max int) error {
	if h, err := s.deviceLocalHandle(uuid, true); err != nil {
		return err
	} else if err = h.rolloverFiles(prefix, max); err != nil {
		return fmt.Errorf("error rolling over %s files for device %s: %w", prefix, uuid, err)
	}
	return nil
}

func (s DevicesFsHandle) deviceLocalHandle(uuid string, forUpdate bool) (h baseFsHandle, err error) {
	h.root = filepath.Join(s.root, uuid)
	if forUpdate {
		if err = h.mkdirs(0o744, true); err != nil {
			err = fmt.Errorf("unable to create file storage for device %s: %w", uuid, err)
		}
	}
	return
}

type UpdatesFsHandle struct {
	baseFsHandle
	category string
}

func (s UpdatesFsHandle) FilePath(tag, update, name string) string {
	return filepath.Join(s.root, tag, update, s.category, name)
}

func (s UpdatesFsHandle) ReadFile(tag, update, name string) (string, error) {
	h, _ := s.updateLocalHandle(tag, update, false)
	content, err := h.readFile(name, false)
	if err != nil {
		err = fmt.Errorf("unexpected error reading %s file for tag %s update %s: %w", s.category, tag, update, err)
	}
	return content, err
}

func (s UpdatesFsHandle) WriteFile(tag, update, name, content string) error {
	if h, err := s.updateLocalHandle(tag, update, true); err != nil {
		return err
	} else if err = h.writeFile(name, content, 0o744); err != nil {
		return fmt.Errorf("unexpected error writing %s file for tag %s update %s: %w", s.category, tag, update, err)
	}
	return nil
}

func (s UpdatesFsHandle) updateLocalHandle(tag, update string, forUpdate bool) (h baseFsHandle, err error) {
	h.root = filepath.Join(s.root, tag, update, s.category)
	if forUpdate {
		if err = h.mkdirs(0o744, true); err != nil {
			err = fmt.Errorf("unable to create %s file storage for tag %s update %s: %w", s.category, tag, update, err)
		}
	}
	return
}

type RolloutsFsHandle struct {
	UpdatesFsHandle
}

func (s RolloutsFsHandle) ListUpdates(tag string) (map[string][]string, error) {
	// An assumption is that we will have a limited amount of tags.
	// In this case it is just fine to list all available updates for all tags at once.
	var tagDirs []string
	if len(tag) > 0 {
		tagDirs = []string{tag}
	} else if dirs, err := os.ReadDir(s.root); err == nil {
		for _, d := range dirs {
			if d.IsDir() {
				tagDirs = append(tagDirs, d.Name())
			}
		}
	} else if os.IsNotExist(err) {
		return nil, nil
	} else {
		return nil, err
	}

	res := make(map[string][]string, len(tagDirs))
	for _, tag = range tagDirs {
		if dirs, err := os.ReadDir(filepath.Join(s.root, tag)); err == nil {
			res[tag] = make([]string, 0, len(dirs))
			for _, d := range dirs {
				if d.IsDir() {
					res[tag] = append(res[tag], d.Name())
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return res, nil
}

func (s RolloutsFsHandle) ListFiles(tag, update string) ([]string, error) {
	h, _ := s.updateLocalHandle(tag, update, false)
	return h.matchFiles("", true)
}

type baseFsHandle struct {
	root string
}

func (s baseFsHandle) mkdirs(mode os.FileMode, ignoreExists bool) error {
	if ignoreExists {
		return os.MkdirAll(s.root, mode)
	} else {
		return os.Mkdir(s.root, mode)
	}
}

func (s baseFsHandle) readFile(name string, ignoreNotExist bool) (string, error) {
	if content, err := os.ReadFile(filepath.Join(s.root, name)); err == nil {
		return string(content), nil
	} else if ignoreNotExist && errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else {
		return "", err
	}
}

func (s baseFsHandle) writeFile(name, content string, mode os.FileMode) error {
	path := filepath.Join(s.root, name)
	partial := filepath.Join(s.root, name+partialFileSuffix)
	if fd, err := os.OpenFile(partial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode); err != nil {
		return err
	} else if _, err = fd.WriteString(content); err != nil {
		_ = fd.Close()
		return err
	} else if err = fd.Sync(); err != nil {
		_ = fd.Close()
		return err
	} else if err = fd.Close(); err != nil {
		return err
	} else {
		return os.Rename(partial, path)
	}
}

func (s baseFsHandle) appendFile(name, content string, mode os.FileMode) error {
	fd, err := os.OpenFile(filepath.Join(s.root, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, mode)
	if err == nil {
		_, err = fd.Write([]byte(content))
		if err != nil {
			_ = fd.Close()
		} else {
			err = fd.Close()
		}
	}
	return err
}

func (s baseFsHandle) rolloverFiles(prefix string, max int) error {
	names, err := s.matchFiles(prefix, true)
	if err == nil {
		for i := 0; i < len(names)-max; i++ {
			if err = os.Remove(filepath.Join(s.root, names[i])); err != nil {
				break
			}
		}
	}
	return err
}

func (s baseFsHandle) matchFiles(prefix string, sortByModTime bool) ([]string, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		if info, err := entry.Info(); err != nil {
			return nil, err
		} else {
			name := info.Name()
			if strings.HasSuffix(name, partialFileSuffix) {
				// Filter out partial files - uploads in progress or data corruptions
				continue
			} else if len(prefix) == 0 || strings.HasPrefix(name, prefix) {
				infos = append(infos, info)
			}
		}
	}
	if sortByModTime {
		slices.SortFunc(infos, func(a, b os.FileInfo) int {
			// UnixMilli is int64, but in our universe UnixMilli difference of two events files of the same device is int.
			return int(a.ModTime().UnixMilli() - b.ModTime().UnixMilli())
		})
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name())
	}
	return names, nil
}
