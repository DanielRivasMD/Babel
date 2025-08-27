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

var (
	ednFile    string
	renderMode string
	sortBy     string
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(displayCmd)

	displayCmd.Flags().StringVarP(&ednFile, "file", "f", "", "Path to your EDN file")
	displayCmd.Flags().StringVarP(&renderMode, "render", "m", "DEFAULT", "Which rows to render: EMPTY (only empty program+action), FULL (all), DEFAULT (non-empty program+action)")
	displayCmd.Flags().StringVarP(&sortBy, "sort", "s", "trigger", "Sort output by one of: program, action, trigger, binding")

	horus.CheckErr(
		displayCmd.RegisterFlagCompletionFunc("render", completeRenderType),
		horus.WithOp("display.init"),
		horus.WithMessage("registering config completion for flag program"),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add debug flag, or use verbose, telling which file & line we are currently reading
// TODO: update error handlers
func runDisplay(cmd *cobra.Command, args []string) {
	allRows := gatherRows(ednFile, rootDir)

	progFiltered := filterByProgram(allRows, program)

	var finalRows []Row
	mode := strings.ToUpper(renderMode)
	for _, r := range progFiltered {
		switch mode {
		case "FULL":
			finalRows = append(finalRows, r)

		case "EMPTY":
			if r.Program == "" && r.Action == "" {
				finalRows = append(finalRows, r)
			}

		default: // "DEFAULT"
			if r.Program != "" && r.Action != "" {
				finalRows = append(finalRows, r)
			}
		}
	}

	// emit a single table from allRows
	emitTable(finalRows)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// humanReadableTrigger rewrites a Keyword like ":!Tpage_up" → "T page_up"
func humanReadableTrigger(raw edn.Keyword) string {
	s := string(raw)
	s = strings.TrimPrefix(s, ":")
	s = strings.TrimPrefix(s, "!")
	parts := strings.SplitN(s, "#P", 2) // group#Pname
	group := parts[0]
	name := ""
	if len(parts) > 1 {
		name = parts[1]
	}
	// replace arrows & modifiers
	r := strings.NewReplacer(
		"up_arrow", "↑",
		"down_arrow", "↓",
		"right_arrow", "→",
		"left_arrow", "←",
		"right_control", "<W>",
		"left_control", "<T>",
		"right_option", "<E>",
		"left_option", "<O>",
		"right_command", "<Q>",
		"left_command", "<C>",
		"right_shift", "<R>",
		"left_shift", "<S>",
		"tab", "TAB",
		"delete_or_backspace", "DEL",
		"return_or_enter", "RET",
		"caps_lock", "<P>",
		"spacebar", "<_>",

		"open_bracket", "[",
		"close_bracket", "]",
		"semicolon", ";",
		"quote", "'",
		"backslash", "\\",
		"comma", ",",
		"period", ".",
		"slash", "/",
	)
	return r.Replace(fmt.Sprintf("%s %s", group, name))
}

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
func collectRows(rawMeta map[edn.Keyword]any, trigger, keySeq string) []Row {
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
			Action:   fetch(edn.Keyword("name")),
			Command:  fetch(edn.Keyword("exec")),
			Program:  fetch(edn.Keyword("program")),
			Trigger:  trigger,
			Binding:  strings.ReplaceAll(strings.ReplaceAll(keySeq, ":", ""), "!", ""),
			Sequence: fetch(edn.Keyword("sequence")),
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
	key := sortBy
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
			return a.Program < b.Program
		case "action":
			return a.Action < b.Action
		case "binding":
			// compare Sequence if you prefer it over Binding when present:
			bi, bj := a.Binding, b.Binding
			if a.Sequence != "" {
				bi = a.Sequence
			}
			if b.Sequence != "" {
				bj = b.Sequence
			}
			return bi < bj
		default: // "trigger"
			return a.Trigger < b.Trigger
		}
	})

	// 2) print header
	fmt.Println("| Program      | Action                         | Trigger    | Binding    |")
	fmt.Println("|--------------|--------------------------------|------------|------------|")

	// 3) print rows
	for _, r := range rows {
		val := r.Binding
		if r.Sequence != "" {
			val = r.Sequence
		}
		fmt.Printf(
			"| %-12s | %-30s | %-10s | %-10s |\n",
			r.Program, r.Action, r.Trigger, val,
		)
	}
}

// take the raw EDN text + mode letter.
func parseBindings(text, modeLetter string) []Row {
	var rows []Row
	pos := 0

	for {
		// find the next ^{…}[…] block
		metaStr, vecStr, nextPos, ok := extractEntry(text, pos)
		if !ok {
			break
		}
		pos = nextPos

		// decode metadata
		rawMeta, err := decodeMetadata(metaStr)
		if err != nil {
			log.Fatalf("EDN metadata unmarshal error: %v", err)
		}

		// decode the rule vector
		vec, err := decodeRule(vecStr)
		if err != nil {
			log.Fatalf("EDN rule decode error: %v", err)
		}

		// human‐readable trigger (e.g. 'T page_up'), then override with modeLetter
		trigger := humanReadableTrigger(vec[0].(edn.Keyword))
		if modeLetter != "" {
			trigger = modeLetter + trigger
		}

		// build the key sequence string
		keySeq := buildKeySequence(vec[1])

		// expand each :doc/actions entry into one Row
		rows = append(rows, collectRows(rawMeta, trigger, keySeq)...)
	}

	return rows
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
