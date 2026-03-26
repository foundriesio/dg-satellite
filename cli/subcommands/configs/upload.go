// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package configs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/foundriesio/dg-satellite/cli/subcommands"
)

var uploadCmd = &cobra.Command{
	Use:   "upload <configs.tgz>",
	Short: "Upload configs",
	Long: `Upload configs to the Satellite server.

	Supported file formats are .tar, .tar.gz, and .tgz.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		api := api.CtxGetApi(cmd.Context())
		cobra.CheckErr(uploadConfigs(api.Configs(), path))
	},
}

func init() {
	ConfigsCmd.AddCommand(uploadCmd)
}

func uploadConfigs(capi api.ConfigsApi, path string) error {
	var isGzip bool
	switch ext := filepath.Ext(path); ext {
	case ".tar":
		break
	case ".tar.gz", ".tgz":
		isGzip = true
	default:
		return fmt.Errorf("supported file types are '.tar, .tar.gz, .tgz', but '%s' given", ext)
	}

	fd, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to read file '%s': %w", path, err)
	}
	defer fd.Close() //nolint:errcheck

	if stat, err := fd.Stat(); err != nil {
		return fmt.Errorf("failed to read file '%s': %w", path, err)
	} else if !stat.Mode().IsRegular() {
		return fmt.Errorf("a '%s' is neither a regular file nor a symlink to a regular file", path)
	}

	var reader io.ReadCloser = fd
	if !isGzip {
		// Gzip raw tar files on-the-fly to save network traffic
		reader = subcommands.GzipStream(func(gzipper io.Writer) error {
			_, err = io.Copy(gzipper, fd)
			return err
		})
		defer reader.Close() //nolint:errcheck
	}

	return capi.Upload(reader,
		api.HttpHeader("Content-Type", "application/x-tar"),
		api.HttpHeader("Content-Encoding", "gzip"),
	)
}
