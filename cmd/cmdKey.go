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

var (
	ednFile string
	verbose bool
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: chalk.Yellow.Color("Generate keybinding docs"),
	Long: chalk.Green.Color(chalk.Bold.TextStyle("Daniel Rivas ")) +
		chalk.Dim.TextStyle(chalk.Italic.TextStyle("<danielrivasmd@gmail.com>")) + "\n",
	Example: `
  babel key --file ~/.saiyajin/frag/simple/lctlcmd.edn`,
	Run: func(cmd *cobra.Command, args []string) {
		generateKeyDocs()
	},
}

func init() {
	rootCmd.AddCommand(keyCmd)
	keyCmd.Flags().StringVarP(&ednFile, "file", "f", "", "Path to your EDN file")
	keyCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

// generateKeyDocs reads the EDN file, scans out ^{…} metadata + the next
// vector, dumps both for you, then emits Markdown.
func generateKeyDocs() {
	if ednFile == "" {
		log.Fatal("🚨 please pass --file <path>.edn")
	}

	// 1) Slurp the file
	data, err := os.ReadFile(ednFile)
	if err != nil {
		log.Fatalf("failed to read %s: %v", ednFile, err)
	}
	text := string(data)

	// DEBUG: how many carets did we see at all?
	totalCarets := strings.Count(text, "^")
	fmt.Printf("🛠️  Debug: found %d '^' carets in %s\n\n", totalCarets, ednFile)

	pos := 0
	for {
		// 2) locate the next caret
		delta := strings.IndexRune(text[pos:], '^')
		if delta < 0 {
			break
		}
		i := pos + delta

		// 3) skip any whitespace after '^'
		j := i + 1
		for j < len(text) && unicode.IsSpace(rune(text[j])) {
			j++
		}
		if j >= len(text) || text[j] != '{' {
			// not actually metadata
			pos = i + 1
			continue
		}

		// 4) extract the metadata map literal
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
					k++ // include the closing brace
					break metaLoop
				}
			}
		}
		if braceCount != 0 || k > len(text) {
			break // unmatched braces, bail out
		}
		metaEnd := k
		metadataStr := text[metaStart:metaEnd]

		// 5) skip whitespace (newlines, spaces) to the vector '['
		p := metaEnd
		for p < len(text) && unicode.IsSpace(rune(text[p])) {
			p++
		}
		if p >= len(text) || text[p] != '[' {
			pos = metaEnd
			continue
		}

		// 6) extract the vector literal
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
					q++ // include the closing bracket
					break vecLoop
				}
			}
		}
		if bracketCount != 0 || q > len(text) {
			break // unmatched brackets
		}
		vecEnd := q
		ruleStr := text[vecStart:vecEnd]

		// advance scan cursor
		pos = vecEnd

		// ------------------------------------------------------------
		// Now decode + dump what we found:
		// ------------------------------------------------------------

		// A) Unmarshal metadata
		var rawMeta map[edn.Keyword]interface{}
		if err := edn.Unmarshal([]byte(metadataStr), &rawMeta); err != nil {
			log.Fatalf("EDN metadata unmarshal error: %v", err)
		}

		// DEBUG: show exactly what metadataStr and rawMeta we got
		fmt.Println("-------")
		fmt.Println("metadataStr:", metadataStr)
		fmt.Println("rawMeta:")
		for key, val := range rawMeta {
			fmt.Printf("  %s => %#v\n", string(key), val)
		}
		fmt.Println()

		// B) Decode the vector form
		var raw interface{}
		dec := edn.NewDecoder(strings.NewReader(ruleStr))
		if err := dec.Decode(&raw); err != nil {
			log.Fatalf("EDN rule decode error: %v", err)
		}
		vec, ok := raw.([]interface{})
		if !ok || len(vec) < 2 {
			continue
		}

		// 1) Trigger keyword
		trigger, _ := vec[0].(edn.Keyword)
		fmt.Printf("## %s\n\n", string(trigger))

		// 2) Key sequence
		keySeqVal := vec[1]
		var keySeq string
		if kv, ok := keySeqVal.([]interface{}); ok {
			parts := make([]string, len(kv))
			for i, e := range kv {
				if kw, ok := e.(edn.Keyword); ok {
					parts[i] = string(kw)
				} else {
					parts[i] = fmt.Sprint(e)
				}
			}
			keySeq = strings.Join(parts, " ")
		} else {
			keySeq = fmt.Sprint(keySeqVal)
		}
		fmt.Printf("- Keys: `%s`\n", keySeq)

		// 3) :doc/actions
		rawActs, has := rawMeta[edn.Keyword(":doc/actions")]
		if has {
			if acts, ok := rawActs.([]interface{}); ok {
				fmt.Println("- Actions:")
				for _, a := range acts {
					if amap, ok := a.(map[edn.Keyword]interface{}); ok {
						name, prog := "", ""
						if v, ok := amap[edn.Keyword(":name")]; ok {
							name = fmt.Sprint(v)
						}
						if v, ok := amap[edn.Keyword(":program")]; ok {
							prog = fmt.Sprint(v)
						}
						fmt.Printf("  - `%s` via `%s`\n", name, prog)
					}
				}
			}
		}
		fmt.Println()
	}
}
