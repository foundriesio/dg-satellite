// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package subcommands

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
)

type (
	ArchiveEntry struct {
		RealPath string      // A real path the original file
		Path     string      // A path to that file inside an archive
		Info     os.FileInfo // A file info; if nil, will be taken from RealPath.
	}

	sourceWriter = func(io.Writer) error
	fileSourcer  = iter.Seq2[*ArchiveEntry, error]
	fileSkipper  = func(ArchiveEntry) error
)

var SkipEntry = errors.New("skip this entry") //nolint:staticcheck // Ignore ST1012 rule for error names, as we mimic filepath.SkipXXX.

func GzipStream(source sourceWriter) io.ReadCloser {
	// Allows to gzip any data provided by the source.
	// A `source` function is supposed to write data into the gzip writer, passed to it as an argument.
	// An idea is to open a pipe to write raw data as an input and read a gzipped data as an output.
	// Gzip writer consumes data from source in a separate goroutine, and any error is propagated into the reader via a pipe.
	pr, pw := io.Pipe()
	go func() {
		var err error
		defer func() {
			// This makes sure that an I/O error gets proliferated to the reader goroutine
			_ = pw.CloseWithError(err)
		}()
		gw := gzip.NewWriter(pw)
		err = source(gw)
		err = closeWithError(gw, err)
	}()
	return pr
}

func TarStream(source fileSourcer) sourceWriter {
	// Allows to tar any sequence of files provided by the source into an output writer.
	// A `source` is supposed to yield rairs of `[ArchiveEntry, nil]` to iterate over files, or `[nil, error]` to interrupt with error.
	// Typical use is in conjunction with GzipStream and ArchiveSourcer as: `reader := GzipStream(TarStream(ArchiveSourcer(pathToDir)))`.
	return func(writer io.Writer) (err error) {
		tw := tar.NewWriter(writer)
		defer func() {
			err = closeWithError(tw, err)
		}()

		var entry *ArchiveEntry
		for entry, err = range source {
			if err != nil {
				break
			}
			if err = entry.EnsureInfo(); err != nil {
				break
			}
			var header *tar.Header
			if header, err = tar.FileInfoHeader(entry.Info, ""); err != nil {
				err = fmt.Errorf("failed to create tar header for '%s': %w", entry.Path, err)
				break
			}
			header.Name = entry.Path
			if err = tw.WriteHeader(header); err != nil {
				err = fmt.Errorf("failed to write tar header for '%s': %w", entry.Path, err)
				break
			}

			if !entry.Info.IsDir() {
				var fd io.ReadCloser
				if fd, err = os.Open(entry.RealPath); err != nil {
					err = fmt.Errorf("failed to open '%s': %w", entry.Path, err)
					break
				}
				defer fd.Close() //nolint:errcheck
				if _, err = io.Copy(tw, fd); err != nil {
					err = fmt.Errorf("failed to write '%s' to tar: %w", entry.Path, err)
					break
				}
			}
		}
		return
	}

}

func ArchiveSourcer(dir string, skipper ...fileSkipper) fileSourcer {
	// Ranges through directory entries, yielding ArchiveEntry items.
	// For each item, a Path is a RealPath relative to the `dir`.
	// A caller can supply one or more `skipper` functions to filter out speficic entries.
	return func(yield func(*ArchiveEntry, error) bool) {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			if rel == "." {
				// Skip root directory
				return nil
			}
			entry := ArchiveEntry{RealPath: path, Path: rel, Info: info}
			for _, skip := range skipper {
				if err = skip(entry); err != nil {
					if errors.Is(err, SkipEntry) {
						err = nil
					}
					return err
				}
			}
			if !yield(&entry, nil) {
				return filepath.SkipAll
			}
			return nil
		})
		if err != nil {
			_ = yield(nil, err)
		}
	}
}

func (entry *ArchiveEntry) EnsureInfo() (err error) {
	if entry.Info == nil {
		if entry.Info, err = os.Stat(entry.RealPath); err != nil {
			err = fmt.Errorf("failed to stat '%s': %w", entry.Path, err)
		}
	}
	return
}

func closeWithError(closer io.Closer, err error) error {
	if err == nil {
		err = closer.Close()
	} else {
		_ = closer.Close()
	}
	return err
}
