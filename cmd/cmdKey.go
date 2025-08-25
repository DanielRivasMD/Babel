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
	"os"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var keyCmd = &cobra.Command{
	Use:     "key",
	Short:   "Generate keybinding docs as Markdown",
	Long:    helpKey,
	Example: exampleKey,

	Run: runKey,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	ednFile string
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(keyCmd)
	keyCmd.Flags().StringVarP(&ednFile, "file", "f", "", "Path to your EDN file")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type Row struct {
	Action      string
	Description string
	Command     string
	Program     string

	Trigger    string
	Keybinding string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runKey(cmd *cobra.Command, args []string) {
	var rows []Row

	// TODO: error handler needed => one-liner
	if ednFile == "" {
		log.Fatal("please pass --file <path>.edn")
	}

	// TODO: reader needed
	data, err := os.ReadFile(ednFile)
	if err != nil {
		log.Fatalf("failed to read %s: %v", ednFile, err)
	}
	text := string(data)

	totalCarets := strings.Count(text, "^")
	if verbose {
		fmt.Printf("Debug: found %d '^' carets in %s\n\n", totalCarets, ednFile)
	}

	pos := 0
	for {
		// 1) find next '^'
		delta := strings.IndexRune(text[pos:], '^')
		if delta < 0 {
			break
		}
		i := pos + delta

		// 2) skip whitespace, expect '{'
		j := i + 1
		for j < len(text) && unicode.IsSpace(rune(text[j])) {
			j++
		}
		if j >= len(text) || text[j] != '{' {
			pos = i + 1
			continue
		}

		// 3) extract metadata map literal
		metaStart := j
		braceCount := 0
		k := j
	metaLoop:
		for ; k < len(text); k++ {
			switch text[k] {
			case '{':
				braceCount++
			case '}':
				braceCount--
				if braceCount == 0 {
					k++ // include closing brace
					break metaLoop
				}
			}
		}
		if braceCount != 0 || k > len(text) {
			break
		}
		metaEnd := k
		metadataStr := text[metaStart:metaEnd]

		// 4) skip to '['
		p := metaEnd
		for p < len(text) && unicode.IsSpace(rune(text[p])) {
			p++
		}
		if p >= len(text) || text[p] != '[' {
			pos = metaEnd
			continue
		}

		// 5) extract the vector literal
		vecStart := p
		bracketCount := 0
		q := p
	vecLoop:
		for ; q < len(text); q++ {
			switch text[q] {
			case '[':
				bracketCount++
			case ']':
				bracketCount--
				if bracketCount == 0 {
					q++ // include closing bracket
					break vecLoop
				}
			}
		}
		if bracketCount != 0 || q > len(text) {
			break
		}
		vecEnd := q
		ruleStr := text[vecStart:vecEnd]
		pos = vecEnd

		// 6) unmarshal metadata
		var rawMeta map[edn.Keyword]any
		// TODO: handle with horus
		if err := edn.Unmarshal([]byte(metadataStr), &rawMeta); err != nil {
			log.Fatalf("EDN metadata unmarshal error: %v", err)
		}

		// 7) decode the vector form
		var raw any
		dec := edn.NewDecoder(strings.NewReader(ruleStr))
		// TODO: handle with horus
		if err := dec.Decode(&raw); err != nil {
			log.Fatalf("EDN rule decode error: %v", err)
		}
		vec, ok := raw.([]any)
		if !ok || len(vec) < 2 {
			continue
		}

		// 8) human-readable trigger
		triggerRaw, _ := vec[0].(edn.Keyword)
		t := string(triggerRaw)
		t = strings.TrimPrefix(t, "!")
		parts := strings.SplitN(t, "#P", 2) // ["TC", "Pleft_arrow"]
		group := parts[0]
		namePart := ""
		if len(parts) > 1 {
			namePart = parts[1]
		}
		trigger := fmt.Sprintf("%s %s", group, namePart)
		trigger = strings.ReplaceAll(trigger, "right_control", "<W>")

		// 9) keybinding sequence
		var keySeq string
		if kv, ok := vec[1].([]any); ok {
			seq := make([]string, len(kv))
			for i, e := range kv {
				seq[i] = fmt.Sprint(e)
			}
			keySeq = strings.Join(seq, " ")
		} else {
			keySeq = fmt.Sprint(vec[1])
		}
		keySeq = strings.ReplaceAll(keySeq, ":", "")
		keySeq = strings.ReplaceAll(keySeq, "!", "")

		// 10) collect rows for each :doc/actions
		if rawActs, found := rawMeta[edn.Keyword("doc/actions")]; found {
			if acts, ok := rawActs.([]any); ok {
				for _, a := range acts {
					// 1) Assert to the raw generic map
					rawMap, ok := a.(map[any]any)
					if !ok {
						continue
					}

					// 2) Extract your fields by converting each key
					var action, prog, description, command string

					// helper to fetch and fmt.Sprint any value
					fetch := func(k any) (string, bool) {
						if v, exists := rawMap[k]; exists {
							return fmt.Sprint(v), true
						}
						return "", false
					}

					// check both edn.Keyword and string forms
					if name, ok := fetch(edn.Keyword("name")); ok {
						action = name
					} else if name, ok := fetch("name"); ok {
						action = name
					}

					if pr, ok := fetch(edn.Keyword("program")); ok {
						prog = pr
					} else if pr, ok := fetch("program"); ok {
						prog = pr
					}

					if ds, ok := fetch(edn.Keyword("description")); ok {
						description = ds
					} else if ds, ok := fetch("description"); ok {
						description = ds
					}

					if ex, ok := fetch(edn.Keyword("exec")); ok {
						command = ex
					} else if ex, ok := fetch("exec"); ok {
						command = ex
					}

					rows = append(rows, Row{
						Action:      action,
						Description: description,
						Command:     command,
						Program:     prog,

						Trigger:    trigger,
						Keybinding: keySeq,
					})
				}
			}
		}
	}

	// 11) emit the Markdown table
	if len(rows) == 0 {
		fmt.Println("No keybindings found.")
		return
	}
	fmt.Println("| Program      | Action            | Trigger  | Keybinding | Description                                        |")
	fmt.Println("|--------------|-------------------|----------|------------|----------------------------------------------------|")
	for _, r := range rows {
		fmt.Printf("| %-12s | %-17s | %-8s | %-10s | %-50s |\n",
			r.Program, r.Action, r.Trigger, r.Keybinding, r.Description)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
