// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package updates

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/spf13/cobra"
)

var tailCmd = &cobra.Command{
	Use:   "tail <ci|prod> <tag> <update-name>",
	Short: "Tail update logs",
	Long:  `Follow server-side events for an update or specific rollout`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := cmd.Context().Value(api.ContextKey).(*api.Api)
		prodType := args[0]

		// Validate prod type
		if prodType != "ci" && prodType != "prod" {
			return fmt.Errorf("first argument must be 'ci' or 'prod', got '%s'", prodType)
		}

		rollout, _ := cmd.Flags().GetString("rollout")
		return tailUpdate(cmd, api, prodType, args[1], args[2], rollout)
	},
}

func init() {
	UpdatesCmd.AddCommand(tailCmd)
	tailCmd.Flags().String("rollout", "", "Specific rollout to tail (optional)")
}

func tailUpdate(cmd *cobra.Command, api *api.Api, prodType, tag, updateName, rollout string) error {
	var endpoint string
	if rollout != "" {
		endpoint = fmt.Sprintf("/v1/updates/%s/%s/%s/rollouts/%s/tail", prodType, tag, updateName, rollout)
		fmt.Printf("Tailing rollout '%s' for update %s/%s (%s)\n", rollout, tag, updateName, strings.ToUpper(prodType))
	} else {
		endpoint = fmt.Sprintf("/v1/updates/%s/%s/%s/tail", prodType, tag, updateName)
		fmt.Printf("Tailing all rollouts for update %s/%s (%s)\n", tag, updateName, strings.ToUpper(prodType))
	}
	fmt.Println("Press Ctrl+C to stop...")

	resp, err := api.GetStream(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var eventType, data string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line marks end of event
			if eventType == "log" && data != "" {
				fmt.Println(data)
			} else if eventType == "error" && data != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: %s\n", data)
			}
			eventType = ""
			data = ""
			continue
		}

		if after, ok := strings.CutPrefix(line, "event: "); ok {
			eventType = after
		} else if after, ok := strings.CutPrefix(line, "data: "); ok {
			data = after
		}
		// Ignore id and retry fields
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}
