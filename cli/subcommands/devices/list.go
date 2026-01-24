// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package devices

import (
	"fmt"

	"github.com/foundriesio/dg-satellite/cli/api"
	rest "github.com/foundriesio/dg-satellite/storage/api"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices",
	Long:  `List all devices known to the server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		api := cmd.Context().Value(api.ContextKey).(*api.Api)
		return listDevices(api)
	},
}

func init() {
	DevicesCmd.AddCommand(listCmd)
}

func listDevices(api *api.Api) error {
	var devices []rest.DeviceListItem
	err := api.Get("/v1/devicess", &devices)
	cobra.CheckErr(err)

	fmt.Println("TODO", devices)
	return nil
}
