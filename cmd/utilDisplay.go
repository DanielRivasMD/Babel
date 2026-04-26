/*
Copyright © 2026 Daniel Rivas <danielrivasmd@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"strings"

	"github.com/ttacon/chalk"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

type tableRow struct {
	Program string
	Action  string
	Trigger string
	Binding string
	Empty   bool
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func printTable(rows []tableRow) {
	if len(rows) == 0 {
		fmt.Println("No bindings found.")
		return
	}
	fmt.Println(tableBorder)
	fmt.Println(tableHeader)
	fmt.Println(tableDivider)
	for _, r := range rows {
		var progColor *chalk.Color
		if c, ok := programColors[r.Program]; ok {
			progColor = &c
		}
		row := fmt.Sprintf("| %s | %s | %s | %s |\n",
			renderCell(r.Program, 15, progColor),
			renderCell(r.Action, 30, nil),
			renderCell(r.Trigger, 20, nil),
			renderCell(r.Binding, 20, nil),
		)
		if r.Empty {
			row = chalk.Dim.TextStyle(row)
		}
		fmt.Print(row)
	}
	fmt.Println(tableBorder)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func isEmptyEntry(e BindingEntry) bool {
	if len(e.Actions) == 0 {
		return true
	}
	for _, a := range e.Actions {
		prog := strings.TrimSpace(fmt.Sprint(a.Program))
		act := strings.TrimSpace(fmt.Sprint(a.Action))
		cmd := strings.TrimSpace(fmt.Sprint(a.Command))
		if prog != "" && prog != "<nil>" {
			return false
		}
		if act != "" && act != "<nil>" {
			return false
		}
		if cmd != "" && cmd != "<nil>" {
			return false
		}
	}
	return true
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func renderCell(val string, width int, color *chalk.Color) string {
	raw := fmt.Sprintf("%-*s", width, val)
	if color != nil {
		return color.Color(raw)
	}
	return raw
}

////////////////////////////////////////////////////////////////////////////////////////////////////
