// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/foundriesio/dg-satellite/cli/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "satcli",
	Short: "A command line interface to the Satellite Server",
	Long: `satcli is a command-line interface for managing devices, updates,
and other resources on a Satellite server.

Configuration is stored in $HOME/.config/satcli.yaml`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		contextName, err := cmd.Flags().GetString("context")
		if err != nil {
			return fmt.Errorf("failed to get context flag: %w", err)
		}

		appctx, err := cfg.GetContext(contextName)
		if err != nil {
			return fmt.Errorf("failed to get current context: %w", err)
		}

		client := api.NewClient(*appctx)

		ctx := context.WithValue(cmd.Context(), api.ContextKey, client)
		cmd.SetContext(ctx)

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringP("context", "c", "", "Specify the context to use from the configuration file")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
