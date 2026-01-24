// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package devices

import (
	"fmt"
	"strings"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/dg-satellite/cli/api"
	rest "github.com/foundriesio/dg-satellite/storage/api"
	"github.com/spf13/cobra"
)

var updatesCmd = &cobra.Command{
	Use:   "updates <uuid> [update-id]",
	Short: "Show device updates",
	Long:  `List all updates for a device, or show details for a specific update`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := cmd.Context().Value(api.ContextKey).(*api.Api)
		if len(args) == 1 {
			return listUpdates(api, args[0])
		}
		return showUpdate(api, args[0], args[1])
	},
}

func init() {
	DevicesCmd.AddCommand(updatesCmd)
}

func listUpdates(api *api.Api, uuid string) error {
	var updates []string
	err := api.Get(fmt.Sprintf("/v1/devices/%s/updates", uuid), &updates)
	cobra.CheckErr(err)

	if len(updates) == 0 {
		fmt.Println("No updates found for this device")
		return nil
	}

	t := tabby.New()
	t.AddHeader("UPDATE ID")

	for _, updateId := range updates {
		t.AddLine(updateId)
	}

	t.Print()
	return nil
}

func showUpdate(api *api.Api, uuid, updateId string) error {
	var events []rest.DeviceUpdateEvent
	err := api.Get(fmt.Sprintf("/v1/devices/%s/updates/%s", uuid, updateId), &events)
	cobra.CheckErr(err)

	if len(events) == 0 {
		fmt.Printf("No events found for update %s\n", updateId)
		return nil
	}

	fmt.Printf("Update: %s\n", updateId)
	fmt.Printf("Device: %s\n\n", uuid)

	for _, event := range events {
		timestamp := "-"
		if event.DeviceTime != "" {
			if t, err := time.Parse(time.RFC3339, event.DeviceTime); err == nil {
				timestamp = t.Format("2006-01-02 15:04:05")
			} else {
				timestamp = event.DeviceTime
			}
		}

		status := ""
		if event.Event.Success != nil {
			if *event.Event.Success {
				status = "-> Succeeded"
			} else {
				status = "-> Failed"
			}
		}

		fmt.Printf("%s: %s(%s) %s\n", timestamp, event.EventType.Id, event.Event.TargetName, status)
		if len(event.Event.Details) > 0 {
			fmt.Println(" Details:")
			for line := range strings.SplitSeq(event.Event.Details, "\n") {
				fmt.Printf(" | %s\n", line)
			}
		}
	}

	return nil
}
