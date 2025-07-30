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
	"github.com/ttacon/chalk"
	"olympos.io/encoding/edn"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	ednFile string
	verbose bool
)

////////////////////////////////////////////////////////////////////////////////////////////////////

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: chalk.Yellow.Color("Generate keybinding docs as Markdown"),
	Long:  chalk.Green.Color(chalk.Bold.TextStyle("babel key")) + " scans your EDN metadata + vector rules and emits a 4-column Markdown table.\n",
	Example: `
  babel key --file ~/.saiyajin/frag/simple/lctlcmd.edn`,

	////////////////////////////////////////////////////////////////////////////////////////////////////

	Run: func(cmd *cobra.Command, args []string) {
		generateKeyDocs()
	},
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(keyCmd)
	keyCmd.Flags().StringVarP(&ednFile, "file", "f", "", "Path to your EDN file")
	keyCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose debug output")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func generateKeyDocs() {
	if ednFile == "" {
		log.Fatal("🚨 please pass --file <path>.edn")
	}

	data, err := os.ReadFile(ednFile)
	if err != nil {
		log.Fatalf("failed to read %s: %v", ednFile, err)
	}
	text := string(data)

	totalCarets := strings.Count(text, "^")
	if verbose {
		fmt.Printf("🛠️  Debug: found %d '^' carets in %s\n\n", totalCarets, ednFile)
	}

	type Row struct {
		Program    string
		Action     string
		Trigger    string
		Keybinding string
	}
	var rows []Row

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
		var rawMeta map[edn.Keyword]interface{}
		if err := edn.Unmarshal([]byte(metadataStr), &rawMeta); err != nil {
			log.Fatalf("EDN metadata unmarshal error: %v", err)
		}

		// // DEBUG: show exactly what metadataStr and rawMeta we got
		// fmt.Println("-------")
		// fmt.Println("metadataStr:", metadataStr)
		// fmt.Println("rawMeta:")
		// for key, val := range rawMeta {
		// 	fmt.Printf("  %s => %#v\n", string(key), val)
		// }
		// fmt.Println()

		// 7) decode the vector form
		var raw interface{}
		dec := edn.NewDecoder(strings.NewReader(ruleStr))
		if err := dec.Decode(&raw); err != nil {
			log.Fatalf("EDN rule decode error: %v", err)
		}
		vec, ok := raw.([]interface{})
		if !ok || len(vec) < 2 {
			continue
		}

		// 8) human-readable trigger
		triggerRaw, _ := vec[0].(edn.Keyword)
		t := string(triggerRaw)
		t = strings.TrimPrefix(t, "!")
		parts := strings.SplitN(t, "#", 2) // ["TC", "Pleft_arrow"]
		group := parts[0]
		namePart := ""
		if len(parts) > 1 {
			namePart = parts[1]
		}
		namePart = strings.TrimPrefix(namePart[1:], "P")
		trigger := fmt.Sprintf("%s %s", group, namePart)

		// 9) keybinding sequence
		var keySeq string
		if kv, ok := vec[1].([]interface{}); ok {
			seq := make([]string, len(kv))
			for i, e := range kv {
				seq[i] = fmt.Sprint(e)
			}
			keySeq = strings.Join(seq, " ")
		} else {
			keySeq = fmt.Sprint(vec[1])
		}

		// 10) collect rows for each :doc/actions
		if rawActs, found := rawMeta[edn.Keyword("doc/actions")]; found {
			if acts, ok := rawActs.([]interface{}); ok {
				for _, a := range acts {
					// 1) Assert to the raw generic map
					rawMap, ok := a.(map[interface{}]interface{})
					if !ok {
						continue
					}

					// 2) Extract your fields by converting each key
					var actionName, prog string

					// helper to fetch and fmt.Sprint any value
					fetch := func(k interface{}) (string, bool) {
						if v, exists := rawMap[k]; exists {
							return fmt.Sprint(v), true
						}
						return "", false
					}

					// check both edn.Keyword and string forms
					if name, ok := fetch(edn.Keyword("name")); ok {
						actionName = name
					} else if name, ok := fetch("name"); ok {
						actionName = name
					}

					if pr, ok := fetch(edn.Keyword("program")); ok {
						prog = pr
					} else if pr, ok := fetch("program"); ok {
						prog = pr
					}

					rows = append(rows, Row{
						Program:    prog,
						Action:     actionName,
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
	fmt.Println("| Program | Action      | Trigger        | Keybinding |")
	fmt.Println("|---------|-------------|----------------|------------|")
	for _, r := range rows {
		fmt.Printf("| %-7s | %-11s | %-14s | %-10s |\n",
			r.Program, r.Action, r.Trigger, r.Keybinding)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
