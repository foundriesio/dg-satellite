// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package devices

import (
	"fmt"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices",
	Long:  `List all devices known to the server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		api := api.CtxGetApi(cmd.Context())
		return listDevices(api.Devices())
	},
}

func init() {
	DevicesCmd.AddCommand(listCmd)
}

func listDevices(dapi api.DeviceApi) error {
	devices, err := dapi.List()
	cobra.CheckErr(err)

	fmt.Println("TODO", devices)
	return nil
}
