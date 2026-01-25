// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package updates

import (
	"github.com/cheynewallace/tabby"
	"github.com/foundriesio/dg-satellite/cli/api"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all updates",
	Long:  `List all CI and production updates`,
	RunE: func(cmd *cobra.Command, args []string) error {
		api := cmd.Context().Value(api.ContextKey).(*api.Api)
		return listUpdates(api)
	},
}

func init() {
	UpdatesCmd.AddCommand(listCmd)
}

func listUpdates(api *api.Api) error {
	var ciUpdates map[string][]string
	err := api.Get("/v1/updates/ci", &ciUpdates)
	cobra.CheckErr(err)

	var prodUpdates map[string][]string
	err = api.Get("/v1/updates/prod", &prodUpdates)
	cobra.CheckErr(err)

	t := tabby.New()
	t.AddHeader("TYPE", "TAG", "NAME")

	for tag, names := range ciUpdates {
		for _, name := range names {
			t.AddLine("ci", tag, name)
		}
	}

	for tag, names := range prodUpdates {
		for _, name := range names {
			t.AddLine("prod", tag, name)
		}
	}

	t.Print()
	return nil
}
