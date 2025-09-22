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
*/
package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var displayCmd = &cobra.Command{
	Use:     "display",
	Short:   "Display current bindings",
	Long:    helpDisplay,
	Example: exampleDisplay,

	Run: runDisplay,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	tableBorder  = "==============================================================================================="
	tableHeader  = "| Program      | Action                         | Trigger              | Binding              |"
	tableDivider = "|--------------|--------------------------------|----------------------|----------------------|"
	tableRowFmt  = "| %-12s | %-30s | %-20s | %-20s |\n"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(displayCmd)

	displayCmd.Flags().StringVarP(&flags.ednFile, "file", "f", "", "Path to your EDN file")
	displayCmd.Flags().StringVarP(&flags.renderMode, "render", "m", "DEFAULT", "Which rows to render: EMPTY (only empty program+action), FULL (all), DEFAULT (non-empty program+action)")
	displayCmd.Flags().StringVarP(&flags.sortBy, "sort", "s", "trigger", "Sort output by one of: program, action, trigger, binding")

	horus.CheckErr(
		displayCmd.RegisterFlagCompletionFunc("render", completeRenderType),
		horus.WithOp("display.init"),
		horus.WithMessage("registering config completion for flag program"),
	)

	horus.CheckErr(
		displayCmd.RegisterFlagCompletionFunc("sort", completeSortType),
		horus.WithOp("display.init"),
		horus.WithMessage("registering config completion for flag sort"),
	)

}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add debug flag, or use verbose, telling which file & line we are currently reading
// TODO: update error handlers
// TODO: simplify run call
func runDisplay(cmd *cobra.Command, args []string) {
	// Resolve EDN file paths
	paths := resolveEDNFiles(flags.ednFile, flags.rootDir)

	// Parse all EDN files into structured bindings
	allEntries, err := parseEDNFiles(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}

	// Filter by program
	filtered := filterByProgram(allEntries, flags.program)

	// Apply render mode
	var final []BindingEntry
	switch strings.ToUpper(flags.renderMode) {
	case "FULL":
		final = filtered
	case "EMPTY":
		for _, e := range filtered {
			fmt.Println(e)
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

	emitTable(final)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// tableRow is a flattened view of one binding row
type tableRow struct {
	Program string
	Action  string
	Trigger string
	Binding string
}

// buildKeySequence joins the second element of the rule vector into a string
func buildKeySequence(x any) string {
	// if x == nil {
	// 	return ""
	// }
	switch kv := x.(type) {
	case []any:
		parts := make([]string, len(kv))
		for i, e := range kv {
			parts[i] = fmt.Sprint(e)
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprint(kv)
	}
}

// emitTable prints all rows as a Markdown table, sorted by --sort
func emitTable(entries []BindingEntry) {
	if len(entries) == 0 {
		fmt.Println("No bindings found.")
		return
	}

	// Flatten entries into rows
	var rows []tableRow
	for _, entry := range entries {
		for _, action := range entry.Actions {
			trigger := formatKeySeq(entry.Trigger, lookups.displayTrigger, action.Program)
			binding := formatBindingEntry(entry, lookups.displayBinding, action.Program)
			rows = append(rows, tableRow{
				Program: action.Program,
				Action:  action.Action,
				Trigger: trigger,
				Binding: binding,
			})
		}
	}

	// Sort rows based on --sort flag
	sort.Slice(rows, func(i, j int) bool {
		switch strings.ToLower(flags.sortBy) {
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
	fmt.Println(tableBorder)
	fmt.Println(tableHeader)
	fmt.Println(tableDivider)

	for _, r := range rows {
		fmt.Printf(tableRowFmt, r.Program, r.Action, r.Trigger, r.Binding)
	}

	fmt.Println(tableBorder)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func isEmptyEntry(e BindingEntry) bool {
	if len(e.Actions) == 0 {
		return true
	}
	for _, a := range e.Actions {
		// Defensive: handle <nil> or empty
		prog := strings.TrimSpace(fmt.Sprint(a.Program))
		act := strings.TrimSpace(fmt.Sprint(a.Action))
		cmd := strings.TrimSpace(fmt.Sprint(a.Command))

		// If any field is meaningful, it's not empty
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

func completeSortType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	options := []string{"program", "action", "trigger", "binding"}
	var completions []string
	for _, opt := range options {
		if strings.HasPrefix(opt, toComplete) {
			completions = append(completions, opt)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

////////////////////////////////////////////////////////////////////////////////////////////////////
