// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package testing

import (
	"archive/tar"
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func CreateTarBuffer(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer func() { require.NoError(t, tw.Close()) }()

	// Collect and create directories first
	dirs := map[string]bool{}
	for name := range files {
		dir := filepath.Dir(name)
		for dir != "." && !dirs[dir] {
			dirs[dir] = true
			dir = filepath.Dir(dir)
		}
	}
	for dir := range dirs {
		err := tw.WriteHeader(&tar.Header{
			Name:     dir + "/",
			Typeflag: tar.TypeDir,
			Mode:     0o755,
		})
		require.NoError(t, err)
	}

	for name, content := range files {
		err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0o644,
			Typeflag: tar.TypeReg,
		})
		require.NoError(t, err)
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}
	return &buf
}
