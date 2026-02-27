// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type tarUnpackConfig struct {
	createDest  bool
	replaceDest bool
	mergeDest   bool
	dirAccess   os.FileMode
	fileAccess  os.FileMode
}

type TarUnpackOption func(tarUnpackConfig) tarUnpackConfig

func TarUnpackDirAccess(mode os.FileMode) TarUnpackOption {
	return func(cfg tarUnpackConfig) tarUnpackConfig {
		cfg.dirAccess = mode
		return cfg
	}
}

func TarUnpackFileAccess(mode os.FileMode) TarUnpackOption {
	return func(cfg tarUnpackConfig) tarUnpackConfig {
		cfg.fileAccess = mode
		return cfg
	}
}

func TarUnpackCreateDest(val bool) TarUnpackOption {
	return func(cfg tarUnpackConfig) tarUnpackConfig {
		cfg.createDest = val
		return cfg
	}
}

func TarUnpackMergeDest(val bool) TarUnpackOption {
	return func(cfg tarUnpackConfig) tarUnpackConfig {
		cfg.mergeDest = val
		return cfg
	}
}

func TarUnpackReplaceDest(val bool) TarUnpackOption {
	return func(cfg tarUnpackConfig) tarUnpackConfig {
		cfg.replaceDest = val
		return cfg
	}
}

func (s baseFsHandle) unpackTar(srcReader io.Reader, destDir string, opts ...TarUnpackOption) error {
	cfg := tarUnpackConfig{
		createDest:  true,
		mergeDest:   false,
		replaceDest: false,
		dirAccess:   defaultDirAccess,
		fileAccess:  defaultFileAccess,
	}
	for _, opt := range opts {
		cfg = opt(cfg)
	}
	// A filepath.Join warrants that destDirPath is clean, so that we can use absPathNoEscape freely below.
	destDirPath := filepath.Join(s.root, destDir)

	var destExists, destEmpty bool
	if destItems, err := os.ReadDir(destDirPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to check if destination '%s' exists: %w", destDir, err)
		}
	} else {
		destExists = true
		destEmpty = len(destItems) == 0
	}

	if !destExists {
		if cfg.createDest {
			if err := os.MkdirAll(destDirPath, cfg.dirAccess); err != nil {
				return fmt.Errorf("failed to create destination '%s': %w", destDir, err)
			}
		}
	} else if !destEmpty {
		if cfg.replaceDest {
			if err := os.RemoveAll(destDirPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to clean destination '%s': %w", destDir, err)
			}
			if err := os.MkdirAll(destDirPath, cfg.dirAccess); err != nil {
				return fmt.Errorf("failed to create destination '%s': %w", destDir, err)
			}
		} else if !cfg.mergeDest {
			return fmt.Errorf("destination '%s' already exists and is not empty", destDir)
		}
	}

	// Unpack config upload tarball; if it fails - halt.
	tarReader := tar.NewReader(srcReader)
	for hdr, err := tarReader.Next(); err != nil && errors.Is(err, io.EOF); {
		if err != nil {
			return fmt.Errorf("failed to read tarball header: %w", err)
		}
		switch hdr.Typeflag {
		case tar.TypeReg:
			// Logic continues after the switch.
		case tar.TypeDir:
			dirPath, err := absPathNoEscape(destDirPath, hdr.Name)
			if err == nil {
				err = os.MkdirAll(dirPath, cfg.dirAccess)
			}
			if err != nil {
				return fmt.Errorf("failed to unpack directory '%s': %w", hdr.Name, err)
			}
			continue
		default:
			return fmt.Errorf("failed to unpack file '%s': unsupported file type %d", hdr.Name, hdr.Typeflag)
		}
		if len(hdr.Name) == 0 {
			return errors.New("filed to unpack file with empty name")
		}
		filePath, err := absPathNoEscape(destDirPath, hdr.Name)
		if err != nil {
			return fmt.Errorf("failed to unpack file '%s': %w", hdr.Name, err)
		}
		dirPath := filepath.Dir(filePath)
		if err = os.MkdirAll(dirPath, defaultDirAccess); err != nil {
			return fmt.Errorf("failed to unpack file '%s': %w", hdr.Name, err)
		}
		if file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, cfg.fileAccess); err != nil {
			return fmt.Errorf("failed to unpack file '%s': %w", hdr.Name, err)
		} else if _, err = io.Copy(file, tarReader); err != nil {
			return fmt.Errorf("failed to unpack file '%s': %w", hdr.Name, err)
		}
	}
	return nil
}

func absPathNoEscape(root, path string) (absPath string, err error) {
	// Assume that the root is clean (a caller is responsible for this for performance optimization).
	// A filepath.Join warrants that the absPath is also clean.
	// So, in order to check for the root path escape attempt, we only need to check for the path prefix.
	absPath = filepath.Join(root, path)
	var isEscaping bool
	if len(root) >= len(absPath) {
		// Special case: path was technically empty.
		isEscaping = root != absPath
	} else if root != "/" {
		// It is not possible to escape outside the top directory.
		// In all other cases filepath.Clean warrants that root does not end in a slash.
		isEscaping = !strings.HasPrefix(absPath, root+string(filepath.Separator))
	}
	if isEscaping {
		err = errors.New("directory escape attempt")
	}
	return

}
