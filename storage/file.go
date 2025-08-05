// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	Aktoml              = "aktoml"
	HwInfo              = "hardware-info"
	NetInfo             = "network-info"
	EventsPrefix        = "events"
	TestsPrefix         = "tests"
	TestArtifactsPrefix = "test-artifacts"
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

func (s FsHandle) ReadFileStream(uuid, name string) (io.ReadCloser, error) {
	fd, err := os.Open(filepath.Join(s.root, uuid, name))
	if err != nil {
		return nil, fmt.Errorf("error reading file %s for device %s: %w", name, uuid, err)
	}
	return fd, nil
}

func (s FsHandle) ReadFile(uuid, name string) (string, error) {
	if fd, err := s.ReadFileStream(uuid, name); err == nil {
		content, err := io.ReadAll(fd)
		if err != nil {
			_ = fd.Close()
			return "", fmt.Errorf("error reading file content %s for device %s: %w", name, uuid, err)
		}
		return string(content), fd.Close()
	} else if errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else {
		return "", fmt.Errorf("unexpected error reading file %s for device %s: %w", name, uuid, err)
	}
}

func (s FsHandle) ReadAsJson(uuid, name string, value any) error {
	if content, err := os.ReadFile(filepath.Join(s.root, uuid, name)); err == nil {
		if err = json.Unmarshal(content, value); err != nil {
			return fmt.Errorf("unexpected error unmarshalling file %s for device %s: %w", name, uuid, err)
		}
	} else {
		return fmt.Errorf("unexpected error reading file %s for device %s: %w", name, uuid, err)
	}
	return nil
}

func (s FsHandle) WriteFileStream(uuid, name string, src io.Reader) error {
	path := filepath.Join(s.root, uuid)
	if err := os.MkdirAll(path, 0o744); err != nil {
		return fmt.Errorf("unable to create file storage for device %s: %w", uuid, err)
	}
	path = filepath.Join(path, name)
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o744)
	if err != nil {
		return fmt.Errorf("error writing file %s for device %s: %w", name, uuid, err)
	}
	if _, err = io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return fmt.Errorf("error writing file content %s for device %s: %w", name, uuid, err)
	}
	return dst.Close()
}

func (s FsHandle) WriteFile(uuid, name string, content []byte) error {
	return s.WriteFileStream(uuid, name, bytes.NewReader(content))
}

func (s FsHandle) AppendFile(uuid, name string, content []byte) error {
	path := filepath.Join(s.root, uuid)
	if err := os.MkdirAll(path, 0o744); err != nil {
		return fmt.Errorf("unable to create file storage for device %s: %w", uuid, err)
	}
	path = filepath.Join(path, name)
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o744)
	if err == nil {
		_, err = fd.Write(content)
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
