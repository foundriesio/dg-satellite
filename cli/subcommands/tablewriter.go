// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package subcommands

import (
	"fmt"
	"strings"
)

type TableWriter struct {
	headers []string
	rows    [][]string
}

func NewTableWriter(headers []string) *TableWriter {
	return &TableWriter{
		headers: headers,
		rows:    make([][]string, 0),
	}
}

func (t *TableWriter) AddRow(columns ...any) {
	strColumns := make([]string, len(columns))
	for i, col := range columns {
		strColumns[i] = fmt.Sprintf("%v", col)
	}
	t.rows = append(t.rows, strColumns)
}

func (t *TableWriter) Render() {
	if len(t.headers) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(t.headers))
	for i, header := range t.headers {
		colWidths[i] = len(header)
	}

	// Check all rows for maximum width (considering multiline content)
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(colWidths) {
				break
			}
			for line := range strings.SplitSeq(cell, "\n") {
				if len(line) > colWidths[i] {
					colWidths[i] = len(line)
				}
			}
		}
	}

	// Print header
	for i, header := range t.headers {
		fmt.Print(header)
		if i < len(t.headers)-1 {
			padding := colWidths[i] - len(header) + 2
			fmt.Print(strings.Repeat(" ", padding))
		}
	}
	fmt.Println()

	// Print rows
	for _, columns := range t.rows {
		// Split all cells into lines
		cellLines := make([][]string, len(columns))
		maxLines := 0
		for i, cell := range columns {
			cellLines[i] = strings.Split(cell, "\n")
			if len(cellLines[i]) > maxLines {
				maxLines = len(cellLines[i])
			}
		}

		for lineNum := 0; lineNum < maxLines; lineNum++ {
			for colNum := 0; colNum < len(t.headers); colNum++ {
				var content string
				if colNum < len(cellLines) && lineNum < len(cellLines[colNum]) {
					content = cellLines[colNum][lineNum]
				}

				fmt.Print(content)
				if colNum < len(t.headers)-1 {
					// Add padding to align columns
					padding := colWidths[colNum] - len(content) + 2
					fmt.Print(strings.Repeat(" ", padding))
				}
			}
			fmt.Println()
		}
	}
}
