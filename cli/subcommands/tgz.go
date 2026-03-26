// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package subcommands

import (
	"compress/gzip"
	"io"
)

type sourceWriter = func(io.Writer) error

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
		if err = source(gw); err != nil {
			_ = gw.Close()
		} else {
			err = gw.Close()
		}
	}()
	return pr
}
