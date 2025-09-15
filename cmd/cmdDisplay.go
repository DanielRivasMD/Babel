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
	// resolve EDN file paths
	paths := resolveEDNFiles(flags.ednFile, flags.rootDir)
	// if err != nil {
	// 	log.Fatalf("file resolution error: %v", err)
	// }

	// parse all EDN files into rows
	allRows, err := gatherRowsFromPaths(paths)
	if err != nil {
		log.Fatalf("EDN parsing error: %v", err)
	}

	progFiltered := filterByProgram(allRows, flags.program)

	var finalRows []Row
	mode := strings.ToUpper(flags.renderMode)
	for _, r := range progFiltered {
		switch mode {
		case "FULL":
			finalRows = append(finalRows, r)

		case "EMPTY":
			if r.program == "" && r.action == "" {
				finalRows = append(finalRows, r)
			}

		default: // "DEFAULT"
			if r.program != "" && r.action != "" {
				finalRows = append(finalRows, r)
			}
		}
	}

	// emit a single table from allRows
	emitTable(finalRows)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// buildKeySequence joins the second element of the rule vector into a string
func buildKeySequence(x any) string {
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

// collectRows expands each :doc/actions into one or more Rows
func collectRows(rawMeta map[edn.Keyword]any, rawTrigger, formatTrigger, rawBinding, formatBinding string) []Row {
	var out []Row
	acts, ok := rawMeta[edn.Keyword("doc/actions")].([]any)
	if !ok {
		return out
	}
	for _, a := range acts {
		m, ok := a.(map[any]any)
		if !ok {
			continue
		}
		fetch := func(k any) string {
			if v, ok := m[k]; ok {
				return fmt.Sprint(v)
			}
			return ""
		}
		out = append(out, Row{
			action:        fetch(edn.Keyword("name")),
			command:       fetch(edn.Keyword("exec")),
			program:       fetch(edn.Keyword("program")),
			rawTrigger:    rawTrigger,
			formatTrigger: formatTrigger,
			rawBinding:    rawBinding,
			formatBinding: formatBinding,
			sequence:      fetch(edn.Keyword("sequence")),
		})
	}
	return out
}

// emitTable prints all rows as a Markdown table, sorted by --sort
func emitTable(rows []Row) {
	if len(rows) == 0 {
		fmt.Println("No bindings found.")
		return
	}

	// 0) normalize sort key
	key := flags.sortBy
	switch key {
	case "program", "action", "trigger", "binding":
		// ok
	default:
		log.Printf("warning: unknown sort key %q, defaulting to 'trigger'", key)
		key = "trigger"
	}

	// 1) sort in place
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch key {
		case "program":
			return a.program < b.program
		case "action":
			return a.action < b.action
		case "binding":
			// compare Sequence if you prefer it over Binding when present:
			bi, bj := a.rawBinding, b.rawBinding
			if a.sequence != "" {
				bi = a.sequence
			}
			if b.sequence != "" {
				bj = b.sequence
			}
			return bi < bj
		default: // "trigger"
			return a.rawTrigger < b.rawTrigger
		}
	})

	// 2) print header
	fmt.Println("===========================================================================")
	fmt.Println("| Program      | Action                         | Trigger    | Binding    |")
	fmt.Println("|--------------|--------------------------------|------------|------------|")

	// 3) print rows
	for _, r := range rows {
		val := r.rawBinding
		if r.sequence != "" {
			val = r.sequence
		}
		fmt.Printf(
			"| %-12s | %-30s | %-10s | %-10s |\n",
			r.program, r.action, r.rawTrigger, val,
		)
	}
	fmt.Println("===========================================================================")
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
