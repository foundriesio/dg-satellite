// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package configs

import (
	"github.com/spf13/cobra"
)

var ConfigsCmd = &cobra.Command{
	Use:   "configs",
	Short: "Manage configs",
	Long:  `Commands for managing configs in the Satellite server`,
}
