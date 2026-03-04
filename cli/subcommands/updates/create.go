// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package updates

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/spf13/cobra"
)

const AnsiClearLine = "\033[2K\r"

var createCmd = &cobra.Command{
	Use:   "create <ci|prod> <tag> <update-name> <directory>",
	Short: "Create an update from an offline update",
	Long:  `Create an update by uploading the offline update found in the directory.`,
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		a := api.CtxGetApi(cmd.Context())
		prodType := args[0]

		if prodType != "ci" && prodType != "prod" {
			return fmt.Errorf("first argument must be 'ci' or 'prod', got '%s'", prodType)
		}

		tag := args[1]
		updateName := args[2]
		dir := args[3]

		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("cannot access directory %q: %w", dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%q is not a directory", dir)
		}

		pr, pw := io.Pipe()
		errCh := make(chan error, 1)

		// Track compressed bytes produced by the tar+gz writer
		var compressedBytes atomic.Int64
		// Track uploaded bytes consumed by the HTTP client
		var uploadedBytes atomic.Int64
		// Signal when tar creation is done so total size is known
		var totalSize atomic.Int64

		go func() {
			cw := &countingWriter{pw: pw, count: &compressedBytes}
			err := createTarGz(cw, dir)
			if err == nil {
				totalSize.Store(compressedBytes.Load())
			}
			errCh <- err
		}()

		// Progress reporter
		stopProgress := make(chan struct{})
		progressDone := make(chan struct{})
		go func() {
			defer close(progressDone)
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					uploaded := uploadedBytes.Load()
					compressed := compressedBytes.Load()
					total := totalSize.Load()
					rate := formatBytes(uploaded * 2) // ~per second at 500ms tick
					if total > 0 {
						pct := float64(uploaded) / float64(total) * 100
						fmt.Fprintf(os.Stderr, AnsiClearLine+"Uploading: %s / %s (%.1f%%) [%s/s]",
							formatBytes(uploaded), formatBytes(total), pct, rate)
					} else {
						fmt.Fprintf(os.Stderr, AnsiClearLine+"Compressing: %s compressed, %s uploaded [%s/s]",
							formatBytes(compressed), formatBytes(uploaded), rate)
					}
				case <-stopProgress:
					return
				}
			}
		}()

		uploadReader := &countingReader{r: pr, count: &uploadedBytes}
		uploadErr := a.Updates(prodType).CreateUpdate(tag, updateName, uploadReader)

		tarErr := <-errCh

		close(stopProgress)
		<-progressDone
		total := totalSize.Load()
		if total > 0 {
			fmt.Fprintf(os.Stderr, AnsiClearLine+"Uploaded: %s / %s (100%%)\n", formatBytes(total), formatBytes(total))
		} else {
			fmt.Fprintf(os.Stderr, AnsiClearLine+"Uploaded: %s\n", formatBytes(uploadedBytes.Load()))
		}

		if uploadErr != nil {
			return fmt.Errorf("upload failed: %w", uploadErr)
		}
		if tarErr != nil {
			return fmt.Errorf("tar creation failed: %w", tarErr)
		}

		fmt.Printf("Update %s/%s/%s created successfully\n", prodType, tag, updateName)
		return nil
	},
}

func init() {
	UpdatesCmd.AddCommand(createCmd)
}

// countingWriter wraps an io.PipeWriter and counts bytes written through it.
type countingWriter struct {
	pw    *io.PipeWriter
	count *atomic.Int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n, err := w.pw.Write(p)
	w.count.Add(int64(n))
	return n, err
}

func (w *countingWriter) CloseWithError(err error) error {
	return w.pw.CloseWithError(err)
}

// countingReader wraps an io.Reader and counts bytes read through it.
type countingReader struct {
	r     io.Reader
	count *atomic.Int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.count.Add(int64(n))
	return n, err
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func createTarGz(cw *countingWriter, dir string) error {
	gw := gzip.NewWriter(cw)
	tw := tar.NewWriter(gw)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", rel, err)
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", rel, err)
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", rel, err)
			}
			defer func() { _ = f.Close() }()
			if _, err := io.Copy(tw, f); err != nil {
				return fmt.Errorf("failed to write %s to tar: %w", rel, err)
			}
		}

		return nil
	})

	if closeErr := tw.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if closeErr := gw.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	// CloseWithError signals the pipe reader; use nil on success so the reader gets EOF
	if err2 := cw.CloseWithError(err); err2 != nil && err == nil {
		return errors.Join(err, err2)
	}
	return err
}
