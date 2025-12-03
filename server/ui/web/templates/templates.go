// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package templates

import (
	"embed"
	"html/template"
	"time"
)

//go:embed *.html *.css
var Assets embed.FS
var Templates *template.Template

func init() {
	funcMap := template.FuncMap{
		"tsToString": func(ts int64) string {
			return time.Unix(ts, 0).Format(time.RFC3339)
		},
	}

	Templates = template.Must(template.New("").Funcs(funcMap).ParseFS(Assets, "*.html", "*.css"))
}
