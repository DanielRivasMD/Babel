/*
Copyright © 2025 Daniel Rivas <danielrivasmd@gmail.com>

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
	"sort"
	"strings"

	"github.com/DanielRivasMD/domovoi"
	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var displayFlags struct {
	ednFile    string
	renderMode string
	sortBy     string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func DisplayCmd() *cobra.Command {
	d := horus.Must(domovoi.GlobalDocs())
	cmd := horus.Must(d.MakeCmd("display", runDisplay))

	cmd.Flags().StringVarP(&displayFlags.ednFile, "file", "f", "", "Path to your EDN file")
	cmd.Flags().StringVarP(&displayFlags.renderMode, "render", "m", "DEFAULT", "Which rows to render: EMPTY (only empty program+action), FULL (all), DEFAULT (non-empty program+action)")
	cmd.Flags().StringVarP(&displayFlags.sortBy, "sort", "s", "trigger", "Sort output by one of: program, action, trigger, binding")

	horus.CheckErr(cmd.RegisterFlagCompletionFunc("render", completeRenderType),
		horus.WithOp("display.init"), horus.WithMessage("registering render completion"))
	horus.CheckErr(cmd.RegisterFlagCompletionFunc("sort", completeSortType),
		horus.WithOp("display.init"), horus.WithMessage("registering sort completion"))

	return cmd
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runDisplay(cmd *cobra.Command, args []string) {
	paths := resolveEDNFiles(displayFlags.ednFile, rootFlags.rootDir)
	allEntries, err := parseEDNFiles(paths)
	horus.CheckErr(
		err,
		horus.WithExitCode(2),
		horus.WithFormatter(func(he *horus.Herror) string {
			return horus.OneLineErr(he.Message)
		}),
	)

	filtered := filterByProgram(allEntries, rootFlags.program)

	var final []BindingEntry
	switch strings.ToUpper(displayFlags.renderMode) {
	case "FULL":
		final = filtered
	case "EMPTY":
		for _, e := range filtered {
			if isEmptyEntry(e) {
				final = append(final, e)
			}
		}
	default: // "DEFAULT"
		for _, e := range filtered {
			if !isEmptyEntry(e) {
				final = append(final, e)
			}
		}
	}

	// Build rows
	var rows []tableRow
	for _, entry := range final {
		for _, action := range entry.Actions {
			trigger := formatKeySeq(entry.Trigger, lookups.displayTrigger, action.Program, "-")
			binding := formatBindingEntry(entry, lookups.displayBinding, action.Program)
			rows = append(rows, tableRow{
				Program: action.Program,
				Action:  action.Action,
				Trigger: trigger,
				Binding: binding,
				Empty:   isEmptyEntry(entry),
			})
		}
	}

	// Sort rows
	sort.Slice(rows, func(i, j int) bool {
		switch strings.ToLower(displayFlags.sortBy) {
		case "program":
			return rows[i].Program < rows[j].Program
		case "action":
			return rows[i].Action < rows[j].Action
		case "binding":
			return rows[i].Binding < rows[j].Binding
		default: // "trigger"
			return rows[i].Trigger < rows[j].Trigger
		}
	})

	// Print table
	printTable(rows)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
