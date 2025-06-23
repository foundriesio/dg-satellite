// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	Aktoml  = "aktoml"
	HwInfo  = "hardware-info"
	NetInfo = "network-info"
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
