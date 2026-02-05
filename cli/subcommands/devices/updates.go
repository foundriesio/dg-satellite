// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package devices

import (
	"fmt"
	"strings"
	"time"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/foundriesio/dg-satellite/cli/subcommands"
	"github.com/spf13/cobra"
)

var updatesCmd = &cobra.Command{
	Use:   "updates <uuid> [update-id]",
	Short: "Show device updates",
	Long:  `List all updates for a device, or show details for a specific update`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		api := api.CtxGetApi(cmd.Context())
		if len(args) == 1 {
			return listUpdates(api.Devices(), args[0])
		}
		return showUpdate(api.Devices(), args[0], args[1])
	},
}

func init() {
	DevicesCmd.AddCommand(updatesCmd)
}

func listUpdates(devices api.DeviceApi, uuid string) error {
	updates, err := devices.Updates(uuid)
	cobra.CheckErr(err)

	if len(updates) == 0 {
		fmt.Println("No updates found for this device")
		return nil
	}

	t := subcommands.NewTableWriter([]string{"UPDATE ID"})

	for _, updateId := range updates {
		t.AddRow(updateId)
	}
	t.Render()
	return nil
}

func showUpdate(devices api.DeviceApi, uuid, updateId string) error {
	events, err := devices.UpdateEvents(uuid, updateId)
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
