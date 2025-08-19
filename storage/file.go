// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package storage

import (
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

	// Per device files/dirs
	AktomlFile   = "aktoml"
	HwInfoFile   = "hardware-info"
	NetInfoFile  = "network-info"
	EventsPrefix = "events"
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

type FsHandle struct {
	Config FsConfig
}

func NewFs(root string) (*FsHandle, error) {
	if err := os.MkdirAll(root, 0o744); err != nil {
		return nil, fmt.Errorf("unable to initialize file storage: %w", err)
	}
	return &FsHandle{Config: FsConfig(root)}, nil
}

func (s FsHandle) devicePath(uuid, name string) string {
	if len(name) == 0 {
		return filepath.Join(s.Config.DevicesDir(), uuid)
	}
	return filepath.Join(s.Config.DevicesDir(), uuid, name)
}

func (s FsHandle) assertDevicePath(uuid string) error {
	if err := os.MkdirAll(s.devicePath(uuid, ""), 0o744); err != nil {
		return fmt.Errorf("unable to create file storage for device %s: %w", uuid, err)
	}
	return nil
}

func (s FsHandle) ReadFile(uuid, name string) (string, error) {
	if content, err := os.ReadFile(s.devicePath(uuid, name)); err == nil {
		return string(content), nil
	} else if os.IsNotExist(err) {
		return "", nil
	} else {
		return "", fmt.Errorf("unexpected error reading file %s for device %s: %w", name, uuid, err)
	}
}

func (s FsHandle) WriteFile(uuid, name, content string) error {
	if err := s.assertDevicePath(uuid); err != nil {
		return err
	}
	if err := os.WriteFile(s.devicePath(uuid, name), []byte(content), 0o744); err != nil {
		return fmt.Errorf("error writing file %s for device %s: %w", name, uuid, err)
	}
	return nil
}

func (s FsHandle) AppendFile(uuid, name, content string) error {
	if err := s.assertDevicePath(uuid); err != nil {
		return err
	}
	fd, err := os.OpenFile(s.devicePath(uuid, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o744)
	if err == nil {
		_, err = fd.Write([]byte(content))
		if err != nil {
			_ = fd.Close()
		} else {
			err = fd.Close()
		}
	}
	if err != nil {
		return fmt.Errorf("error writing file %s for device %s: %w", name, uuid, err)
	}
	return nil
}

func (s FsHandle) ListFiles(uuid, prefix string, sortByModTime bool) ([]string, error) {
	names, err := s.matchFiles(uuid, prefix, sortByModTime)
	if err != nil {
		err = fmt.Errorf("error listing %s files for device %s: %w", prefix, uuid, err)
	}
	return names, err
}

func (s FsHandle) RolloverFiles(uuid, prefix string, max int) error {
	path := s.devicePath(uuid, "")
	names, err := s.matchFiles(uuid, prefix, true)
	if err == nil {
		for i := 0; i < len(names)-max; i++ {
			if err = os.Remove(filepath.Join(path, names[i])); err != nil {
				break
			}
		}
	}
	if err != nil {
		err = fmt.Errorf("error rolling over %s files for device %s: %w", prefix, uuid, err)
	}
	return err
}

func (s FsHandle) matchFiles(uuid, prefix string, sortByModTime bool) ([]string, error) {
	path := s.devicePath(uuid, "")
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		if info, err := entry.Info(); err != nil {
			return nil, err
		} else if strings.HasPrefix(info.Name(), prefix) {
			infos = append(infos, info)
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
