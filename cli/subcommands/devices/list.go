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
	Short: "List devices",
	Long:  `List devices known to the server. By default shows the first page of results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		api := api.CtxGetApi(cmd.Context())
		return listDevices(api.Devices(), columnsStr, page)
	},
}

var columnsStr string

func init() {
	allColsStr := strings.Join(allColumns, ",")
	DevicesCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&columnsStr, "columns", "uuid,target,last-seen",
		"Comma-separated list of columns to display (available: "+allColsStr+")")
	listCmd.Flags().IntP("page", "p", 0, "Page number to display (0-indexed)")
}

const defaultPageLimit = 1000

func listDevices(dapi api.DeviceApi, cols string, page int) error {
	devices, hasMore, err := dapi.ListPage(page, defaultPageLimit)
	if err != nil {
		return err
	}

	columns := strings.Split(cols, ",")
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
	if hasMore {
		fmt.Printf("\nMore results available. Use --page %d for the next page.\n", page+1)
	}
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
