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
	Aktoml       = "aktoml"
	HwInfo       = "hardware-info"
	NetInfo      = "network-info"
	EventsPrefix = "events"
)

type FsHandle struct {
	root string
}

func NewFs(root string) (*FsHandle, error) {
	if err := os.MkdirAll(root, 0o744); err != nil {
		return nil, fmt.Errorf("unable to initialize file storage: %w", err)
	}
	return &FsHandle{root: root}, nil
}

func (s FsHandle) ReadFile(uuid, name string) (string, error) {
	if content, err := os.ReadFile(filepath.Join(s.root, uuid, name)); err == nil {
		return string(content), nil
	} else if os.IsNotExist(err) {
		return "", nil
	} else {
		return "", fmt.Errorf("unexpected error reading file %s for device %s: %w", name, uuid, err)
	}
}

func (s FsHandle) WriteFile(uuid, name, content string) error {
	path := filepath.Join(s.root, uuid)
	if err := os.MkdirAll(path, 0o744); err != nil {
		return fmt.Errorf("unable to create file storage for device %s: %w", uuid, err)
	}
	path = filepath.Join(path, name)
	if err := os.WriteFile(path, []byte(content), 0o744); err != nil {
		return fmt.Errorf("error writing file %s for device %s: %w", name, uuid, err)
	}
	return nil
}

func (s FsHandle) AppendFile(uuid, name, content string) error {
	path := filepath.Join(s.root, uuid)
	if err := os.MkdirAll(path, 0o744); err != nil {
		return fmt.Errorf("unable to create file storage for device %s: %w", uuid, err)
	}
	path = filepath.Join(path, name)
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o744)
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
	path := filepath.Join(s.root, uuid)
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
	path := filepath.Join(s.root, uuid)
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
