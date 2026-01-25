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

var createRolloutCmd = &cobra.Command{
	Use:   "create-rollout <ci|prod> <tag> <update-name> <rollout-name>",
	Short: "Create a new rollout for an update",
	Long:  `Create a new rollout specifying device UUIDs and/or groups to target`,
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := cmd.Context().Value(api.ContextKey).(*api.Api)
		prodType := args[0]

		// Validate prod type
		if prodType != "ci" && prodType != "prod" {
			return fmt.Errorf("first argument must be 'ci' or 'prod', got '%s'", prodType)
		}

		uuids, _ := cmd.Flags().GetString("uuids")
		groups, _ := cmd.Flags().GetString("groups")

		return createRollout(api, prodType, args[1], args[2], args[3], uuids, groups)
	},
}

func init() {
	UpdatesCmd.AddCommand(createRolloutCmd)
	createRolloutCmd.Flags().String("uuids", "", "Comma-separated list of device UUIDs")
	createRolloutCmd.Flags().String("groups", "", "Comma-separated list of device groups")
}

func createRollout(api *api.Api, prodType, tag, updateName, rolloutName, uuidsStr, groupsStr string) error {
	if uuidsStr == "" && groupsStr == "" {
		return fmt.Errorf("at least one of --uuids or --groups must be specified")
	}

	var uuids []string
	if uuidsStr != "" {
		for uuid := range strings.SplitSeq(uuidsStr, ",") {
			trimmed := strings.TrimSpace(uuid)
			if trimmed != "" {
				uuids = append(uuids, trimmed)
			}
		}
	}

	var groups []string
	if groupsStr != "" {
		for group := range strings.SplitSeq(groupsStr, ",") {
			trimmed := strings.TrimSpace(group)
			if trimmed != "" {
				groups = append(groups, trimmed)
			}
		}
	}

	rollout := rest.Rollout{
		Uuids:  uuids,
		Groups: groups,
	}

	endpoint := fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts/%s", prodType, tag, updateName, rolloutName)
	_, err := api.Put(endpoint, rollout)
	cobra.CheckErr(err)
	return nil
}
