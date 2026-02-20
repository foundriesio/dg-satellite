// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package updates

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/spf13/cobra"
)

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

		go func() {
			errCh <- createTarGz(pw, dir)
		}()

		uploadErr := a.Updates(prodType).CreateUpdate(tag, updateName, pr)

		// Wait for the tar writer goroutine to finish
		tarErr := <-errCh
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

func createTarGz(pw *io.PipeWriter, dir string) error {
	gw := gzip.NewWriter(pw)
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
	pw.CloseWithError(err)
	return err
}
