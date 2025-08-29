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

	CertsCasPemFile = "cas.pem"
	CertsTlsCsrFile = "tls.csr"
	CertsTlsKeyFile = "tls.key"
	CertsTlsPemFile = "tls.pem"

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

	Certs   CertsFsHandle
	Devices DevicesFsHandle
}

func NewFs(root string) (*FsHandle, error) {
	fs := &FsHandle{Config: FsConfig(root)}
	fs.Certs.root = fs.Config.CertsDir()
	fs.Devices.root = fs.Config.DevicesDir()

	for _, h := range []struct {
		handle baseFsHandle
		mode   os.FileMode
	}{
		{fs.Certs.baseFsHandle, 0o744},
		{fs.Devices.baseFsHandle, 0o740},
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
		} else if !os.IsNotExist(err) {
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
	} else if ignoreNotExist && os.IsNotExist(err) {
		return "", nil
	} else {
		return "", err
	}
}

func (s baseFsHandle) writeFile(name, content string, mode os.FileMode) error {
	return os.WriteFile(filepath.Join(s.root, name), []byte(content), mode)
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
		if os.IsNotExist(err) {
			return []string{}, nil
		}
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
