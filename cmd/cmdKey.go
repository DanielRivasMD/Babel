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
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/DanielRivasMD/horus"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
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
	rootDir string
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func init() {
	rootCmd.AddCommand(keyCmd)
	keyCmd.Flags().StringVarP(&ednFile, "file", "f", "", "Path to your EDN file")
	keyCmd.Flags().StringVarP(&rootDir, "root", "R", defaultRootDir(), "Configuration root directory (will scan all .edn under here)")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type Row struct {
	Action  string
	Command string
	Program string

	Trigger    string
	Keybinding string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: add filter for program
// TODO: add filter for non-empty (default), empty & full
func runKey(cmd *cobra.Command, args []string) {
	// 0) make sure we have at least one input source
	if ednFile == "" && rootDir == "" {
		log.Fatal("please pass --file <path>.edn or --root <config-dir>")
	}

	// 1) build list of files to process
	files := resolveFiles(ednFile, rootDir)
	fmt.Println(files)

	// 2) process each file and collect rows
	var allRows []Row
	for _, f := range files {
		text := loadEDNFile(f)
		allRows = append(allRows, parseBindings(text)...)
	}

	// 3) emit a single table from allRows
	emitTable(allRows)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// // 1) validateArgs ensures --file was provided
// func validateArgs() {
// 	if ednFile == "" {
// 		horus.CheckErr(
// 			fmt.Errorf(""),
// 			horus.WithMessage(""),
// 			horus.WithExitCode(2),
// 			horus.WithFormatter(func(he *horus.Herror) string {
// 				return "please pass --file <path>.edn"
// 			}),
// 		)
// 	}
// }

// 2) loadEDNFile reads the entire EDN file into a string
func loadEDNFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		horus.CheckErr(
			err,
			horus.WithMessage(path),
			horus.WithExitCode(2),
			horus.WithFormatter(func(he *horus.Herror) string {
				return "failed to read: " + chalk.Red.Color(he.Message)
			}),
		)
	}
	return string(data)
}

// 1–5) extractEntry finds the next ^{…}[…] pair, returns meta & vector & new position
func extractEntry(text string, startPos int) (metaStr, vecStr string, nextPos int, ok bool) {
	// 1) find next caret
	delta := strings.IndexRune(text[startPos:], '^')
	if delta < 0 {
		return "", "", 0, false
	}
	i := startPos + delta

	// 2) skip whitespace, expect '{'
	j := i + 1
	for j < len(text) && unicode.IsSpace(rune(text[j])) {
		j++
	}
	if j >= len(text) || text[j] != '{' {
		return extractEntry(text, i+1)
	}

	// 3) extract metadata map literal
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
				k++ // include closing
				break metaLoop
			}
		}
	}
	if braceCount != 0 {
		return "", "", 0, false
	}
	metaEnd := k
	metaStr = text[j:metaEnd]

	// 4) skip to '['
	p := metaEnd
	for p < len(text) && unicode.IsSpace(rune(text[p])) {
		p++
	}
	if p >= len(text) || text[p] != '[' {
		return extractEntry(text, metaEnd)
	}

	// 5) extract the vector literal
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
				q++ // include closing
				break vecLoop
			}
		}
	}
	if bracketCount != 0 {
		return "", "", 0, false
	}
	vecEnd := q
	vecStr = text[p:vecEnd]
	return metaStr, vecStr, vecEnd, true
}

// 6) decodeMetadata turns the EDN map string into Go map
func decodeMetadata(metaStr string) (map[edn.Keyword]any, error) {
	var rawMeta map[edn.Keyword]any
	err := edn.Unmarshal([]byte(metaStr), &rawMeta)
	return rawMeta, err
}

// 7) decodeRule parses the EDN vector into []any
func decodeRule(vecStr string) ([]any, error) {
	var raw any
	dec := edn.NewDecoder(strings.NewReader(vecStr))
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	vec, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid rule form")
	}
	return vec, nil
}

// 8) humanReadableTrigger rewrites a Keyword like ":!Tpage_up" → "T page_up"
func humanReadableTrigger(raw edn.Keyword) string {
	s := string(raw)
	s = strings.TrimPrefix(s, ":!")
	parts := strings.SplitN(s, "#P", 2) // group#Pname
	group := parts[0]
	name := ""
	if len(parts) > 1 {
		name = parts[1]
	}
	// replace arrows & modifiers
	r := strings.NewReplacer(
		"right_arrow", "→",
		"left_arrow", "←",
		"right_control", "<W>",
		"left_control", "<T>",
		"right_option", "<E>",
		"left_option", "<O>",
		"right_shift", "<R>",
		"left_shift", "<S>",
		"tab", "TAB",
		"caps_lock", "<P>",
		"spacebar", "<_>",
	)
	return r.Replace(fmt.Sprintf("%s %s", group, name))
}

// 9) buildKeySequence joins the second element of the rule vector into a string
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

// 10) collectRows expands each :doc/actions into one or more Rows
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
			Action:     fetch(edn.Keyword("name")),
			Command:    fetch(edn.Keyword("exec")),
			Program:    fetch(edn.Keyword("program")),
			Trigger:    trigger,
			Keybinding: strings.ReplaceAll(strings.ReplaceAll(keySeq, ":", ""), "!", ""),
		})
	}
	return out
}

// 11) emitTable prints all rows as a Markdown table
func emitTable(rows []Row) {
	if len(rows) == 0 {
		fmt.Println("No keybindings found.")
		return
	}
	fmt.Println("| Program      | Action                         | Trigger    | Keybinding |")
	fmt.Println("|--------------|--------------------------------|------------|------------|")
	for _, r := range rows {
		fmt.Printf(
			"| %-12s | %-30s | %-10s | %-10s |\n",
			r.Program, r.Action, r.Trigger, r.Keybinding,
		)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func defaultRootDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // will be caught later
	}
	return filepath.Join(home, ".saiyajin", "frag")
}

// resolveFiles returns either the single --file or all .edn under --root
func resolveFiles(file, root string) []string {
	if file != "" {
		return []string{file}
	}

	var ednFiles []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".edn") {
			ednFiles = append(ednFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to scan %s: %v", root, err)
	}
	return ednFiles
}

// parseBindings wraps your existing pos-loop and subfunctions (1–10)
func parseBindings(text string) []Row {
	var rows []Row
	pos := 0
	for {
		metaStr, vecStr, nextPos, ok := extractEntry(text, pos)
		if !ok {
			break
		}
		pos = nextPos

		rawMeta, err := decodeMetadata(metaStr)
		if err != nil {
			log.Fatalf("EDN metadata unmarshal error: %v", err)
		}

		vec, err := decodeRule(vecStr)
		if err != nil {
			log.Fatalf("EDN rule decode error: %v", err)
		}

		fmt.Println(text)
		fmt.Println(vec)

		trigger := humanReadableTrigger(vec[0].(edn.Keyword))
		keySeq := buildKeySequence(vec[1])
		rows = append(rows, collectRows(rawMeta, trigger, keySeq)...)
	}
	return rows
}

////////////////////////////////////////////////////////////////////////////////////////////////////
