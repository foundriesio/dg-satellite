// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package updates

import (
	"fmt"
	"strings"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/foundriesio/dg-satellite/cli/subcommands"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <ci|prod> <tag> <update-name>",
	Short: "Show rollouts for an update",
	Long:  `Display all rollouts for a specific update`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := api.CtxGetApi(cmd.Context())
		prodType := args[0]

		if prodType != "ci" && prodType != "prod" {
			return fmt.Errorf("first argument must be 'ci' or 'prod', got '%s'", prodType)
		}

		updates := api.Updates(prodType)
		return showUpdate(updates, args[1], args[2])
	},
}

func init() {
	UpdatesCmd.AddCommand(showCmd)
}

func showUpdate(updates api.UpdatesApi, tag, updateName string) error {
	rollouts, err := updates.Get(tag, updateName)
	cobra.CheckErr(err)

	if len(rollouts) == 0 {
		fmt.Printf("No rollouts found for %s update %s/%s\n", updates.Type, tag, updateName)
		return nil
	}

	fmt.Printf("Update: %s (%s)\n", updateName, strings.ToUpper(updates.Type))
	fmt.Printf("Tag: %s\n\n", tag)

	t := subcommands.NewTableWriter([]string{"ROLLOUT NAME"})

	for _, rollout := range rollouts {
		t.AddRow(rollout)
	}

	t.Render()
	return nil
}
