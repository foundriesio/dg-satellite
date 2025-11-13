// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import (
	"bufio"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

const (
	// Global files/dirs
	CertsDir   = "certs"
	DbFile     = "db.sqlite"
	DevicesDir = "devices"
	UpdatesDir = "updates"

	partialFileSuffix  = "..part"
	rolloutJournalFile = "rollouts.journal"

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
	UpdatesAppsDir     = "apps"
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
			Apps     UpdatesFsHandle
			Ostree   UpdatesFsHandle
			Tuf      UpdatesFsHandle
			Rollouts RolloutsFsHandle
		}
		Prod struct {
			Apps     UpdatesFsHandle
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
	fs.Updates.Ci.Apps.root = fs.Config.UpdatesCiDir()
	fs.Updates.Ci.Apps.category = UpdatesAppsDir
	fs.Updates.Ci.Ostree.root = fs.Config.UpdatesCiDir()
	fs.Updates.Ci.Ostree.category = UpdatesOstreeDir
	fs.Updates.Ci.Rollouts.root = fs.Config.UpdatesCiDir()
	fs.Updates.Ci.Rollouts.category = UpdatesRolloutsDir
	fs.Updates.Ci.Tuf.root = fs.Config.UpdatesCiDir()
	fs.Updates.Ci.Tuf.category = UpdatesTufDir
	fs.Updates.Prod.Apps.root = fs.Config.UpdatesProdDir()
	fs.Updates.Prod.Apps.category = UpdatesAppsDir
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

func (s baseFsHandle) readFileLines(name string, ignoreNotExist bool) iter.Seq2[string, error] {
	// memory efficient way to read lines from a potentially large file
	return func(yield func(string, error) bool) {
		if fd, err := os.OpenFile(filepath.Join(s.root, name), os.O_RDONLY, 0); err != nil {
			if !ignoreNotExist || !errors.Is(err, os.ErrNotExist) {
				yield("", err)
			}
		} else {
			defer fd.Close()                // nolint:errcheck
			scanner := bufio.NewScanner(fd) // line reader
			for scanner.Scan() {
				if !yield(scanner.Text(), nil) {
					return
				}
			}
			if err = scanner.Err(); err != nil {
				yield("", err)
			}
		}
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
	// O_APPEND + O_SYNC on Linux warrants that concurrent file appends up to 1MB are serialized.
	fd, err := os.OpenFile(filepath.Join(s.root, name),
		os.O_CREATE|os.O_APPEND|syscall.O_SYNC|os.O_WRONLY, mode)
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
