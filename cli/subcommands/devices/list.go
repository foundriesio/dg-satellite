// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package devices

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/foundriesio/dg-satellite/cli/subcommands"
	"github.com/spf13/cobra"
)

var allColumns = []string{
	"uuid",
	"name",
	"group",
	"target",
	"last-seen",
	"created-at",
	"is-prod",
	"tag",
	"labels",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices",
	Long:  `List all devices known to the server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		columns, _ := cmd.Flags().GetString("columns")
		api := api.CtxGetApi(cmd.Context())
		return listDevices(api.Devices(), columns)
	},
}

func init() {
	colmnsStr := strings.Join(allColumns, ",")
	DevicesCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("columns", "", "uuid,target,last-seen",
		"Comma-separated list of columns to display (available: "+colmnsStr+")")
}

func listDevices(dapi api.DeviceApi, columnsStr string) error {
	devices, err := dapi.List()
	cobra.CheckErr(err)

	columns := strings.Split(columnsStr, ",")
	for i, col := range columns {
		columns[i] = strings.TrimSpace(col)
		if slices.Index(allColumns, col) < 0 {
			return fmt.Errorf("invalid column: %s", col)
		}
	}

	headers := make([]string, 0, len(columns))
	for _, col := range columns {
		headers = append(headers, strings.ToUpper(strings.ReplaceAll(col, "-", " ")))
	}
	table := subcommands.NewTableWriter(headers)

	for _, device := range devices {
		row := make([]any, 0, len(columns))
		for _, col := range columns {
			row = append(row, getColumnValue(&device, col))
		}
		table.AddRow(row...)
	}

	table.Render()
	return nil
}

func getColumnValue(device *api.DeviceListItem, column string) string {
	switch column {
	case "uuid":
		return device.Uuid
	case "target":
		return device.Target
	case "last-seen":
		if device.LastSeen > 0 {
			return time.Unix(device.LastSeen, 0).Format("2006-01-02 15:04:05")
		}
		return "-"
	case "created-at":
		if device.CreatedAt > 0 {
			return time.Unix(device.CreatedAt, 0).Format("2006-01-02 15:04:05")
		}
		return "-"
	case "group":
		if group, ok := device.Labels["group"]; ok {
			return group
		}
		return "-"
	case "name":
		if name, ok := device.Labels["name"]; ok {
			return name
		}
		return "-"
	case "is-prod":
		if device.IsProd {
			return "true"
		}
		return "false"
	case "tag":
		return device.Tag
	case "labels":
		if len(device.Labels) == 0 {
			return ""
		}
		labelStrs := ""
		for k, v := range device.Labels {
			if len(labelStrs) > 0 {
				labelStrs += "\n"
			}
			labelStrs += fmt.Sprintf("%s=%s", k, v)
		}
		return labelStrs
	default:
		return ""
	}
}
