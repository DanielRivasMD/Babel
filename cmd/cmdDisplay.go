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
	"olympos.io/encoding/edn"
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

// TODO: add debug flag, or use verbose, telling which file & line we are currently reading
// TODO: update error handlers
// TODO: simplify run call
func runDisplay(cmd *cobra.Command, args []string) {
	// Resolve EDN file paths
	paths := resolveEDNFiles(flags.ednFile, flags.rootDir)

	// Parse all EDN files into structured bindings
	allEntries, err := gatherRowsFromPaths(paths)
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
			if len(e.Actions) == 0 {
				final = append(final, e)
			}
		}
	default: // "DEFAULT"
		for _, e := range filtered {
			if len(e.Actions) > 0 {
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

func collectRows(rawMeta map[edn.Keyword]any, trigger, binding KeySeq) []BindingEntry {
	var out []BindingEntry
	acts, ok := rawMeta[edn.Keyword("doc/actions")].([]any)
	if !ok {
		return out
	}

	var actions []ProgramAction
	for _, a := range acts {
		m, ok := a.(map[any]any)
		if !ok {
			continue
		}
		fetch := func(k edn.Keyword) string {
			if v, ok := m[k]; ok {
				return fmt.Sprint(v)
			}
			return ""
		}
		actions = append(actions, ProgramAction{
			Action:  fetch("name"),
			Command: fetch("exec"),
			Program: fetch("program"),
		})
	}

	return []BindingEntry{{
		Trigger:  trigger,
		Binding:  binding,
		Sequence: "", // optional: fetch(edn.Keyword("sequence"))
		Actions:  actions,
	}}
}

// emitTable prints all rows as a Markdown table, sorted by --sort
func emitTable(entries []BindingEntry) {
	if len(entries) == 0 {
		fmt.Println("No bindings found.")
		return
	}

	// Optional: sort by flags.sortBy
	// You can implement sorting later if needed

	fmt.Println("===================================================================================")
	fmt.Println("| Program      | Action                         | Trigger         | Binding        |")
	fmt.Println("|--------------|--------------------------------|------------------|----------------|")

	for _, e := range entries {
		for _, a := range e.Actions {
			bind := e.Sequence
			if bind == "" {
				bind = e.Binding.Key
			}
			fmt.Printf(
				"| %-12s | %-30s | %-16s | %-14s |\n",
				a.Program,
				a.Action,
				e.Trigger.Modifier+"-"+e.Trigger.Key,
				e.Binding.Modifier+"-"+bind,
			)
		}
	}

	fmt.Println("===================================================================================")
}

// extractMode finds the first symbol immediately under :rules,
// e.g. [:q-mode …], trims the leading ':', splits on '-'
// and returns the first character as a lowercase string
func extractMode(text string) string {
	ixSpace := 20 // TODO: random hardcode number
	// locate the ":rules" clause
	ruleStart := strings.Index(text, ":rules")
	if ruleStart < 0 {
		return ""
	}
	// find the '[' that starts the rules vector
	sliceRule := text[ruleStart : ruleStart+ixSpace]
	brOpen := strings.Index(sliceRule, "[")
	if brOpen < 0 {
		return ""
	}
	if sliceRule[brOpen+1:brOpen+2] != ":" {
		return ""
	} else {
		sliceMode := sliceRule[brOpen:]
		startMode := strings.Index(sliceMode, ":")
		endMode := strings.Index(sliceMode, "-")
		if startMode < 0 || endMode < 0 {
			return ""
		}
		mode := sliceRule[brOpen:][startMode:endMode]
		mode = strings.TrimPrefix(mode, ":")
		return mode
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
