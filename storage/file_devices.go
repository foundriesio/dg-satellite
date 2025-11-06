// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import (
	"fmt"
	"path/filepath"
)

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
