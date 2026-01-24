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

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices",
	Long:  `List all devices known to the server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		api := cmd.Context().Value(api.ContextKey).(*api.Api)
		columns, _ := cmd.Flags().GetString("columns")
		return listDevices(api, columns)
	},
}

func init() {
	DevicesCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("columns", "", "uuid,target,last-seen",
		"Comma-separated list of columns to display (available: uuid,name,group,target,last-seen,created-at,is-prod,tag,labels)")
}

func listDevices(api *api.Api, columnsStr string) error {
	var devices []rest.DeviceListItem
	err := api.Get("/v1/devices", &devices)
	cobra.CheckErr(err)

	columns := strings.Split(columnsStr, ",")
	for i, col := range columns {
		columns[i] = strings.TrimSpace(col)
	}

	t := tabby.New()
	headers := make([]interface{}, 0, len(columns))
	for _, col := range columns {
		headers = append(headers, strings.ToUpper(strings.ReplaceAll(col, "-", " ")))
	}
	t.AddHeader(headers...)

	for _, device := range devices {
		row := make([]any, 0, len(columns))
		for _, col := range columns {
			row = append(row, getColumnValue(&device, col))
		}
		t.AddLine(row...)
	}

	t.Print()
	return nil
}

func getColumnValue(device *rest.DeviceListItem, column string) string {
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
				labelStrs += ","
			}
			labelStrs += fmt.Sprintf("%s=%s", k, v)
		}
		return labelStrs
	default:
		return ""
	}
}
