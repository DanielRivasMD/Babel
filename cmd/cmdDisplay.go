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
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// BUG: rendering not detecting empty
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

	fmt.Println("===============================================================================================")
	fmt.Println("| Program      | Action                         | Trigger              | Binding              |")
	fmt.Println("|--------------|--------------------------------|----------------------|----------------------|")

	for _, entry := range entries {
		for _, action := range entry.Actions {
			trigger := formatKeySeq(entry.Trigger, lookups.displayTrigger, action.Program)
			binding := formatBindingEntry(entry, lookups.displayBinding, action.Program)
			fmt.Printf(
				"| %-12s | %-30s | %-20s | %-20s |\n",
				action.Program,
				action.Action,
				trigger,
				binding,
			)
		}
	}

	fmt.Println("===============================================================================================")
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
