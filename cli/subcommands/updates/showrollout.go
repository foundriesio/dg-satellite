// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package updates

import (
	"fmt"
	"strings"

	"github.com/foundriesio/dg-satellite/cli/api"
	rest "github.com/foundriesio/dg-satellite/storage/api"
	"github.com/spf13/cobra"
)

var showRolloutCmd = &cobra.Command{
	Use:   "show-rollout <ci|prod> <tag> <update-name> <rollout>",
	Short: "Show details for a specific rollout",
	Long:  `Display detailed information about a rollout including UUIDs, groups, and effective devices`,
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := cmd.Context().Value(api.ContextKey).(*api.Api)
		prodType := args[0]

		// Validate prod type
		if prodType != "ci" && prodType != "prod" {
			return fmt.Errorf("first argument must be 'ci' or 'prod', got '%s'", prodType)
		}

		return showRollout(api, prodType, args[1], args[2], args[3])
	},
}

func init() {
	UpdatesCmd.AddCommand(showRolloutCmd)
}

func showRollout(api *api.Api, prodType, tag, updateName, rollout string) error {
	var rolloutData rest.Rollout
	endpoint := fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts/%s", prodType, tag, updateName, rollout)
	err := api.Get(endpoint, &rolloutData)
	cobra.CheckErr(err)

	fmt.Printf("Rollout: %s\n", rollout)
	fmt.Printf("Update: %s (%s)\n", updateName, strings.ToUpper(prodType))
	fmt.Printf("Tag: %s\n", tag)
	fmt.Printf("Committed: %v\n\n", rolloutData.Commit)

	if len(rolloutData.Groups) > 0 {
		fmt.Println("Groups:")
		for _, group := range rolloutData.Groups {
			fmt.Printf("  - %s\n", group)
		}
		fmt.Println()
	}

	if len(rolloutData.Uuids) > 0 {
		fmt.Println("Device UUIDs:")
		for _, uuid := range rolloutData.Uuids {
			fmt.Printf("  - %s\n", uuid)
		}
		fmt.Println()
	}

	if len(rolloutData.Effect) > 0 {
		fmt.Printf("Effective UUIDs (%d devices):\n", len(rolloutData.Effect))
		for _, uuid := range rolloutData.Effect {
			fmt.Printf("  - %s\n", uuid)
		}
	}

	return nil
}
