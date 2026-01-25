// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package updates

import (
	"fmt"
	"strings"

	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <ci|prod> <tag> <update-name>",
	Short: "Show rollouts for an update",
	Long:  `Display all rollouts for a specific update`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := cmd.Context().Value(api.ContextKey).(*api.Api)
		prodType := args[0]

		if prodType != "ci" && prodType != "prod" {
			return fmt.Errorf("first argument must be 'ci' or 'prod', got '%s'", prodType)
		}

		return showUpdate(api, prodType, args[1], args[2])
	},
}

func init() {
	UpdatesCmd.AddCommand(showCmd)
}

func showUpdate(api *api.Api, prodType, tag, updateName string) error {
	var rollouts []string
	endpoint := fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts", prodType, tag, updateName)
	err := api.Get(endpoint, &rollouts)
	cobra.CheckErr(err)

	if len(rollouts) == 0 {
		fmt.Printf("No rollouts found for %s update %s/%s\n", prodType, tag, updateName)
		return nil
	}

	fmt.Printf("Update: %s (%s)\n", updateName, strings.ToUpper(prodType))
	fmt.Printf("Tag: %s\n\n", tag)

	t := tabby.New()
	t.AddHeader("ROLLOUT NAME")

	for _, rollout := range rollouts {
		t.AddLine(rollout)
	}

	t.Print()
	return nil
}
